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

	"github.com/emirhangumus/sshmanager/internal/model"
	"github.com/emirhangumus/sshmanager/internal/store"
	prompttext "github.com/emirhangumus/sshmanager/internal/ui/prompt"
)

// scpArg is a single positional argument to the scp subcommand, classified
// as either a plain local path or a path on a saved sshmanager connection.
type scpArg struct {
	raw        string
	conn       *model.SSHConnection
	remotePath string
}

func (a scpArg) isRemote() bool {
	return a.conn != nil
}

func HandleScpArgs(connectionFilePath, secretKeyFilePath string, args []string) error {
	return handleScpArgs(connectionFilePath, secretKeyFilePath, args, os.Stdout)
}

func handleScpArgs(connectionFilePath, secretKeyFilePath string, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("scp", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	recursive := fs.Bool("r", false, "Copy directories recursively")
	fs.BoolVar(recursive, "recursive", false, "Copy directories recursively")

	if err := fs.Parse(args); err != nil {
		return err
	}

	positional := fs.Args()
	if len(positional) < 2 {
		return errors.New("scp: usage: sshmanager scp [-r] <src...> <alias:path | path> <dest>")
	}

	connStore := store.NewConnectionStore(connectionFilePath, secretKeyFilePath)
	connFile, err := connStore.Load()
	if err != nil {
		return err
	}

	scpArgs, err := classifyScpArgs(&connFile, positional)
	if err != nil {
		return err
	}

	conn, err := singleRemoteConnection(scpArgs)
	if err != nil {
		return err
	}

	sources := scpArgs[:len(scpArgs)-1]
	dest := scpArgs[len(scpArgs)-1]

	sourceSpecs := make([]string, len(sources))
	for i, s := range sources {
		sourceSpecs[i] = scpArgSpec(s, conn)
	}
	destSpec := scpArgSpec(dest, conn)

	bin, invocationArgs, env, err := buildScpInvocation(conn, *recursive, sourceSpecs, destSpec)
	if err != nil {
		return err
	}

	if err := runScp(bin, invocationArgs, env, conn.Host, conn.EffectivePort()); err != nil {
		return err
	}

	fmt.Fprintln(out, "Transfer complete.")
	return nil
}

// classifyScpArgs splits each positional token on its first ':'. A token is
// treated as remote only when the prefix before the colon matches a known
// connection alias — anything else (including local paths that happen to
// contain a colon) is left as a plain local path.
func classifyScpArgs(connFile *model.ConnectionFile, positional []string) ([]scpArg, error) {
	result := make([]scpArg, 0, len(positional))
	for _, raw := range positional {
		alias, remotePath, hasColon := strings.Cut(raw, ":")
		if hasColon {
			if conn := connFile.GetConnectionByAlias(alias); conn != nil {
				result = append(result, scpArg{raw: raw, conn: conn, remotePath: remotePath})
				continue
			}
		}
		result = append(result, scpArg{raw: raw})
	}
	return result, nil
}

// singleRemoteConnection ensures exactly one distinct sshmanager connection
// is referenced across all scp arguments, and returns it.
func singleRemoteConnection(args []scpArg) (*model.SSHConnection, error) {
	var conn *model.SSHConnection
	for _, a := range args {
		if !a.isRemote() {
			continue
		}
		if conn == nil {
			conn = a.conn
		} else if conn.ID != a.conn.ID {
			return nil, errors.New("scp: transferring between two sshmanager aliases in a single invocation is not supported")
		}
	}
	if conn == nil {
		return nil, errors.New("scp: no sshmanager alias found in arguments (expected one path like \"alias:/remote/path\")")
	}
	return conn, nil
}

// scpArgSpec renders a scpArg back into the literal token scp expects:
// "user@host:remotepath" for the remote argument, or the original local
// path unchanged.
func scpArgSpec(a scpArg, conn *model.SSHConnection) string {
	if !a.isRemote() {
		return a.raw
	}
	return fmt.Sprintf("%s@%s:%s", strings.TrimSpace(conn.Username), strings.TrimSpace(conn.Host), a.remotePath)
}

func buildScpInvocation(conn *model.SSHConnection, recursive bool, sources []string, dest string) (string, []string, []string, error) {
	username := strings.TrimSpace(conn.Username)
	host := strings.TrimSpace(conn.Host)
	if username == "" || host == "" {
		return "", nil, nil, fmt.Errorf("username and host are required")
	}

	port := strconv.Itoa(conn.EffectivePort())
	authMode := conn.EffectiveAuthMode()
	advancedArgs, advancedEnv, err := buildAdvancedScpArgs(conn)
	if err != nil {
		return "", nil, nil, err
	}

	pathArgs := make([]string, 0, len(sources)+1)
	if recursive {
		pathArgs = append(pathArgs, "-r")
	}
	pathArgs = append(pathArgs, sources...)
	pathArgs = append(pathArgs, dest)

	switch authMode {
	case model.AuthModePassword:
		password := conn.Password
		if password == "" {
			return "", nil, nil, fmt.Errorf("password is required when auth mode is %q", model.AuthModePassword)
		}
		scpArgs := []string{"-P", port}
		scpArgs = append(scpArgs, advancedArgs...)
		scpArgs = append(scpArgs, pathArgs...)
		env := append([]string{"SSHPASS=" + password}, advancedEnv...)
		return "sshpass", append([]string{"-e", "scp"}, scpArgs...), env, nil
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
		scpArgs := []string{"-P", port, "-i", identity}
		scpArgs = append(scpArgs, advancedArgs...)
		scpArgs = append(scpArgs, pathArgs...)
		return "scp", scpArgs, advancedEnv, nil
	case model.AuthModeAgent:
		scpArgs := []string{"-P", port}
		scpArgs = append(scpArgs, advancedArgs...)
		scpArgs = append(scpArgs, pathArgs...)
		return "scp", scpArgs, advancedEnv, nil
	default:
		return "", nil, nil, fmt.Errorf("unsupported auth mode: %s", authMode)
	}
}

// buildAdvancedScpArgs mirrors buildAdvancedSSHArgs (connect.go) for scp:
// ProxyJump and ExtraSSHArgs apply the same way, but LocalForwards /
// RemoteForwards are meaningless for a one-shot file copy and are omitted.
func buildAdvancedScpArgs(conn *model.SSHConnection) ([]string, []string, error) {
	proxyJump := strings.TrimSpace(conn.ProxyJump)
	extraArgs := model.NormalizeStringList(conn.ExtraSSHArgs)

	if err := model.ValidateProxyJump(proxyJump); err != nil {
		return nil, nil, fmt.Errorf("invalid proxy jump: %w", err)
	}
	if err := model.ValidateExtraSSHArgs(extraArgs); err != nil {
		return nil, nil, fmt.Errorf("invalid extra ssh args: %w", err)
	}

	proxyJumpArgs, proxyJumpEnv, err := buildProxyJumpArgs(conn, proxyJump)
	if err != nil {
		return nil, nil, err
	}

	args := make([]string, 0, len(proxyJumpArgs)+len(extraArgs))
	args = append(args, proxyJumpArgs...)
	args = append(args, extraArgs...)
	return args, proxyJumpEnv, nil
}

func runScp(bin string, args []string, envAdd []string, host string, port int) error {
	binPath, err := exec.LookPath(bin)
	if err != nil {
		if bin == "sshpass" {
			return errors.New(prompttext.DefaultPromptTexts.ErrorMessages.SSHPassNotFound)
		}
		return fmt.Errorf("required command %q not found in PATH", bin)
	}

	if bin == "sshpass" {
		if err := ensureHostKeyAccepted(strings.TrimSpace(host), port); err != nil {
			return err
		}
	}

	cmd := exec.Command(binPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), envAdd...)

	err = cmd.Run()
	if bin == "sshpass" {
		err = translateSSHPassError(err, host)
	}
	return err
}
