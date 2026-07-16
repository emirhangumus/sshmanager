package commands

import (
	"errors"
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
			return errors.New(prompttext.DefaultPromptTexts.ErrorMessages.SSHPassNotFound)
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
	advancedArgs, advancedEnv, err := buildAdvancedSSHArgs(conn)
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
		env := append([]string{"SSHPASS=" + password}, advancedEnv...)
		return "sshpass", append([]string{"-e", "ssh"}, sshArgs...), env, nil
	case model.AuthModeKey:
		identity := strings.TrimSpace(conn.IdentityFile)
		if identity == "" {
			return "", nil, nil, fmt.Errorf("identityFile is required when auth mode is %q", model.AuthModeKey)
		}
		info, err := os.Stat(identity)
		if err != nil {
			return "", nil, nil, fmt.Errorf("identityFile %q is not accessible: %w", identity, err)
		}
		if info.IsDir() {
			return "", nil, nil, fmt.Errorf("identityFile %q is a directory, not a key file", identity)
		}
		sshArgs := []string{"-p", port, "-i", identity}
		sshArgs = append(sshArgs, advancedArgs...)
		sshArgs = append(sshArgs, target)
		return "ssh", sshArgs, advancedEnv, nil
	case model.AuthModeAgent:
		sshArgs := []string{"-p", port}
		sshArgs = append(sshArgs, advancedArgs...)
		sshArgs = append(sshArgs, target)
		return "ssh", sshArgs, advancedEnv, nil
	default:
		return "", nil, nil, fmt.Errorf("unsupported auth mode: %s", authMode)
	}
}

// proxyJumpPasswordEnvVar carries the ProxyJump hop's password into the
// ssh process's environment. It is only ever referenced by name inside a
// generated ProxyCommand string (see buildProxyJumpArgs) so the secret
// itself never appears in argv/ps output, matching the SSHPASS pattern
// used for the target host's password.
const proxyJumpPasswordEnvVar = "SSHMANAGER_PROXY_JUMP_SSHPASS"

func buildAdvancedSSHArgs(conn *model.SSHConnection) ([]string, []string, error) {
	proxyJump := strings.TrimSpace(conn.ProxyJump)
	localForwards := model.NormalizeStringList(conn.LocalForwards)
	remoteForwards := model.NormalizeStringList(conn.RemoteForwards)
	extraArgs := model.NormalizeStringList(conn.ExtraSSHArgs)

	if err := model.ValidateProxyJump(proxyJump); err != nil {
		return nil, nil, fmt.Errorf("invalid proxy jump: %w", err)
	}
	if err := model.ValidateForwardSpecs(localForwards); err != nil {
		return nil, nil, fmt.Errorf("invalid local forwards: %w", err)
	}
	if err := model.ValidateForwardSpecs(remoteForwards); err != nil {
		return nil, nil, fmt.Errorf("invalid remote forwards: %w", err)
	}
	if err := model.ValidateExtraSSHArgs(extraArgs); err != nil {
		return nil, nil, fmt.Errorf("invalid extra ssh args: %w", err)
	}

	proxyJumpArgs, proxyJumpEnv, err := buildProxyJumpArgs(conn, proxyJump)
	if err != nil {
		return nil, nil, err
	}

	args := make([]string, 0, len(proxyJumpArgs)+2*len(localForwards)+2*len(remoteForwards)+len(extraArgs))
	args = append(args, proxyJumpArgs...)
	for _, spec := range localForwards {
		args = append(args, "-L", spec)
	}
	for _, spec := range remoteForwards {
		args = append(args, "-R", spec)
	}
	args = append(args, extraArgs...)
	return args, proxyJumpEnv, nil
}

// buildProxyJumpArgs decides how to represent the ProxyJump hop in ssh argv.
//
// When the jump hop needs no dedicated credentials, it is passed through as
// native "-J <proxyJump>", identical to today's behavior (and this is the
// only path that supports multiple comma-separated hops).
//
// When the jump hop has its own password and/or identity file configured
// (distinct from the target host's), a single hop is instead wired up via
// "-o ProxyCommand=..." that shells out to a dedicated "ssh -W %h:%p" (via
// sshpass for password auth) so that hop can authenticate independently of
// the target host. The jump password is never placed in the ProxyCommand
// text or argv — it's referenced by environment variable name only, and the
// value travels via the returned env additions, mirroring the SSHPASS
// pattern used for the target host itself.
func buildProxyJumpArgs(conn *model.SSHConnection, proxyJump string) ([]string, []string, error) {
	if proxyJump == "" {
		return nil, nil, nil
	}

	jumpAuthMode := conn.EffectiveProxyJumpAuthMode()
	identityFile := strings.TrimSpace(conn.ProxyJumpIdentityFile)
	needsDedicatedHop := jumpAuthMode == model.AuthModePassword || identityFile != ""
	if !needsDedicatedHop {
		return []string{"-J", proxyJump}, nil, nil
	}

	if strings.Contains(proxyJump, ",") {
		return nil, nil, fmt.Errorf("proxy jump with a dedicated password or identity file is only supported for a single hop")
	}

	user, host, port, err := model.ParseProxyJumpHop(proxyJump)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid proxy jump: %w", err)
	}

	tokens := make([]string, 0, 8)
	var env []string
	if jumpAuthMode == model.AuthModePassword {
		password := conn.ProxyJumpPassword
		if password == "" {
			return nil, nil, fmt.Errorf("proxyJumpPassword is required when proxy jump auth mode is %q", model.AuthModePassword)
		}
		tokens = append(tokens, fmt.Sprintf(`SSHPASS="$%s"`, proxyJumpPasswordEnvVar), "sshpass", "-e")
		env = append(env, proxyJumpPasswordEnvVar+"="+password)
	}

	tokens = append(tokens, "ssh")
	if port != "" {
		tokens = append(tokens, "-p", shellQuoteSingle(port))
	}
	if identityFile != "" {
		info, statErr := os.Stat(identityFile)
		if statErr != nil {
			return nil, nil, fmt.Errorf("proxyJumpIdentityFile %q is not accessible: %w", identityFile, statErr)
		}
		if info.IsDir() {
			return nil, nil, fmt.Errorf("proxyJumpIdentityFile %q is a directory, not a key file", identityFile)
		}
		tokens = append(tokens, "-i", shellQuoteSingle(identityFile))
	}
	tokens = append(tokens, "-W", "%h:%p")

	hopTarget := host
	if user != "" {
		hopTarget = user + "@" + host
	}
	tokens = append(tokens, shellQuoteSingle(hopTarget))

	proxyCommand := strings.Join(tokens, " ")
	return []string{"-o", "ProxyCommand=" + proxyCommand}, env, nil
}

// shellQuoteSingle wraps s in single quotes so it is treated as one literal
// token by the /bin/sh that ssh uses to execute ProxyCommand values,
// regardless of any shell metacharacters s may contain.
func shellQuoteSingle(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func printCredentialsIfEnabled(conn *model.SSHConnection, cfg *config.SSHManagerConfig) {
	if cfg == nil || !cfg.Behaviour.ShowCredentialsOnConnect {
		return
	}

	fmt.Println("Warning: printing credentials to terminal (showCredentialsOnConnect=true)")
	fmt.Printf("Username: %s\n", conn.Username)
	fmt.Printf("Password: %s\n", conn.Password)
}
