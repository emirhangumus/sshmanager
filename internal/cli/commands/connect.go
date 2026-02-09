package commands

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/emirhangumus/sshmanager/internal/config"
	"github.com/emirhangumus/sshmanager/internal/model"
	"github.com/emirhangumus/sshmanager/internal/store"
	prompttext "github.com/emirhangumus/sshmanager/internal/ui/prompt"
)

// HandleConnect returns true when caller should exit app after SSH command exits.
func HandleConnect(connectionFilePath, secretKeyFilePath string, cfg *config.SSHManagerConfig) (bool, error) {
	connStore := store.NewConnectionStore(connectionFilePath, secretKeyFilePath)
	connFile, err := connStore.Load()
	if err != nil {
		return false, err
	}
	if len(connFile.Connections) == 0 {
		fmt.Println(prompttext.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return false, nil
	}

	items := connFile.SelectItems()
	labels := make([]string, len(items))
	for i := range items {
		labels[i] = items[i].Label
	}
	idx, _, err := prompttext.SelectPrompt(prompttext.DefaultPromptTexts.SelectAnSSHConnection, labels)
	if err != nil {
		if prompttext.IsCancelError(err) {
			fmt.Println(prompttext.DefaultPromptTexts.SuccessMessages.OperationCancelled)
		}
		return false, nil
	}

	selectedID := items[idx].ConnectionID
	conn := connFile.GetConnectionByID(selectedID)
	if conn == nil {
		fmt.Println(prompttext.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return false, nil
	}

	printCredentialsIfEnabled(conn, cfg)

	if err := connect(conn); err != nil {
		fmt.Printf(prompttext.DefaultPromptTexts.ErrorMessages.ConnectionToXFailedX+"\n", fmt.Sprintf("%s@%s", conn.Username, conn.Host), err)
		return false, nil
	}

	return !cfg.Behaviour.ContinueAfterSSHExit, nil
}

func HandleConnectArgs(connectionFilePath, secretKeyFilePath, configFilePath string, args []string) error {
	return handleConnectArgs(connectionFilePath, secretKeyFilePath, configFilePath, args)
}

func handleConnectArgs(connectionFilePath, secretKeyFilePath, configFilePath string, args []string) error {
	fs := flag.NewFlagSet("connect", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	alias := fs.String("alias", "", "Connection alias")
	id := fs.String("id", "", "Connection ID")

	if err := fs.Parse(args); err != nil {
		return err
	}

	selectedAlias, selectedID, err := resolveSelector(*alias, *id, fs.Args(), "connect")
	if err != nil {
		return err
	}

	if selectedAlias == "" && selectedID == "" {
		cfg, err := config.LoadConfig(configFilePath)
		if err != nil {
			return err
		}
		_, err = HandleConnect(connectionFilePath, secretKeyFilePath, &cfg)
		return err
	}

	if selectedID != "" {
		return FindAndConnectByID(connectionFilePath, secretKeyFilePath, configFilePath, selectedID)
	}
	return FindAndConnect(connectionFilePath, secretKeyFilePath, configFilePath, selectedAlias)
}

func connect(conn *model.SSHConnection) error {
	bin, args, envAdd, err := buildConnectInvocation(conn)
	if err != nil {
		return err
	}

	binPath, err := exec.LookPath(bin)
	if err != nil {
		if bin == "sshpass" {
			return fmt.Errorf(prompttext.DefaultPromptTexts.ErrorMessages.SSHPassNotFound)
		}
		return fmt.Errorf("required command %q not found in PATH", bin)
	}

	cmd := exec.Command(binPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), envAdd...)

	return cmd.Run()
}

func buildConnectInvocation(conn *model.SSHConnection) (string, []string, []string, error) {
	username := strings.TrimSpace(conn.Username)
	host := strings.TrimSpace(conn.Host)
	if username == "" || host == "" {
		return "", nil, nil, fmt.Errorf("username and host are required")
	}

	target := fmt.Sprintf("%s@%s", username, host)
	port := strconv.Itoa(conn.EffectivePort())
	authMode := conn.EffectiveAuthMode()
	advancedArgs, err := buildAdvancedSSHArgs(conn)
	if err != nil {
		return "", nil, nil, err
	}

	switch authMode {
	case model.AuthModePassword:
		password := conn.Password
		if password == "" {
			return "", nil, nil, fmt.Errorf("password is required when auth mode is %q", model.AuthModePassword)
		}
		sshArgs := []string{"-p", port}
		sshArgs = append(sshArgs, advancedArgs...)
		sshArgs = append(sshArgs, target)
		return "sshpass", append([]string{"-e", "ssh"}, sshArgs...), []string{"SSHPASS=" + password}, nil
	case model.AuthModeKey:
		identity := strings.TrimSpace(conn.IdentityFile)
		if identity == "" {
			return "", nil, nil, fmt.Errorf("identityFile is required when auth mode is %q", model.AuthModeKey)
		}
		sshArgs := []string{"-p", port, "-i", identity}
		sshArgs = append(sshArgs, advancedArgs...)
		sshArgs = append(sshArgs, target)
		return "ssh", sshArgs, nil, nil
	case model.AuthModeAgent:
		sshArgs := []string{"-p", port}
		sshArgs = append(sshArgs, advancedArgs...)
		sshArgs = append(sshArgs, target)
		return "ssh", sshArgs, nil, nil
	default:
		return "", nil, nil, fmt.Errorf("unsupported auth mode: %s", authMode)
	}
}

func buildAdvancedSSHArgs(conn *model.SSHConnection) ([]string, error) {
	proxyJump := strings.TrimSpace(conn.ProxyJump)
	localForwards := model.NormalizeStringList(conn.LocalForwards)
	remoteForwards := model.NormalizeStringList(conn.RemoteForwards)
	extraArgs := model.NormalizeStringList(conn.ExtraSSHArgs)

	if err := model.ValidateProxyJump(proxyJump); err != nil {
		return nil, fmt.Errorf("invalid proxy jump: %w", err)
	}
	if err := model.ValidateForwardSpecs(localForwards); err != nil {
		return nil, fmt.Errorf("invalid local forwards: %w", err)
	}
	if err := model.ValidateForwardSpecs(remoteForwards); err != nil {
		return nil, fmt.Errorf("invalid remote forwards: %w", err)
	}
	if err := model.ValidateExtraSSHArgs(extraArgs); err != nil {
		return nil, fmt.Errorf("invalid extra ssh args: %w", err)
	}

	args := make([]string, 0, 2+2*len(localForwards)+2*len(remoteForwards)+len(extraArgs))
	if proxyJump != "" {
		args = append(args, "-J", proxyJump)
	}
	for _, spec := range localForwards {
		args = append(args, "-L", spec)
	}
	for _, spec := range remoteForwards {
		args = append(args, "-R", spec)
	}
	args = append(args, extraArgs...)
	return args, nil
}

func printCredentialsIfEnabled(conn *model.SSHConnection, cfg *config.SSHManagerConfig) {
	if cfg == nil || !cfg.Behaviour.ShowCredentialsOnConnect {
		return
	}

	fmt.Printf("Username: %s\n", conn.Username)
	fmt.Printf("Password: %s\n", conn.Password)
}
