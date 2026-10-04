package commands

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/emirhangumus/sshmanager/v2/internal/model"
	"github.com/emirhangumus/sshmanager/v2/internal/store"
	prompttext "github.com/emirhangumus/sshmanager/v2/internal/ui/prompt"
)

// ExecExitError carries the SSH process status to the top-level CLI. Detail
// is set only when sshmanager can add useful context to SSH's own stderr.
type ExecExitError struct {
	Code   int
	Detail string
}

func (e *ExecExitError) Error() string {
	if e.Detail != "" {
		return e.Detail
	}
	return fmt.Sprintf("remote command exited with status %d", e.Code)
}

type execOptions struct {
	alias   string
	id      string
	command string
	script  string
	shell   string
}

func HandleExecArgs(connectionFilePath, secretKeyFilePath string, args []string) error {
	opts, err := parseExecArgs(args)
	if err != nil {
		return err
	}

	connStore := store.NewConnectionStore(connectionFilePath, secretKeyFilePath)
	connFile, err := connStore.Load()
	if err != nil {
		return err
	}
	conn := findConnectionBySelector(&connFile, opts.alias, opts.id)
	if conn == nil {
		return fmt.Errorf("exec: %s", notFoundMessage(opts.alias, opts.id))
	}

	var stdin io.Reader = os.Stdin
	if opts.script != "" {
		file, err := os.Open(opts.script)
		if err != nil {
			return fmt.Errorf("exec: open script %q: %w", opts.script, err)
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			return fmt.Errorf("exec: inspect script %q: %w", opts.script, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("exec: script %q must be a regular file", opts.script)
		}
		stdin = file
	}

	remoteCommand := opts.command
	if opts.script != "" {
		remoteCommand = opts.shell + " -s"
	}
	bin, invocationArgs, env, err := buildExecInvocation(conn, remoteCommand)
	if err != nil {
		return fmt.Errorf("exec: %w", err)
	}
	return runExec(bin, invocationArgs, env, conn, stdin, os.Stdout, os.Stderr)
}

func parseExecArgs(args []string) (execOptions, error) {
	fs := flag.NewFlagSet("exec", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	alias := fs.String("alias", "", "Connection alias")
	id := fs.String("id", "", "Connection ID")
	script := fs.String("script", "", "Local script file")
	shell := fs.String("shell", "sh", "Remote script interpreter: sh|bash")
	if err := fs.Parse(args); err != nil {
		return execOptions{}, fmt.Errorf("exec: %w", err)
	}
	selectedAlias, selectedID, err := resolveSelector(*alias, *id, nil, "exec")
	if err != nil {
		return execOptions{}, err
	}
	if selectedAlias == "" && selectedID == "" {
		return execOptions{}, errors.New("exec: specify --alias or --id")
	}

	separator := false
	for _, arg := range args {
		if arg == "--" {
			separator = true
			break
		}
	}
	positional := fs.Args()
	if *script != "" {
		if separator || len(positional) != 0 {
			return execOptions{}, errors.New("exec: --script cannot be combined with a command")
		}
	} else {
		if !separator || len(positional) != 1 || strings.TrimSpace(positional[0]) == "" {
			return execOptions{}, errors.New("exec: expected one command string after --, or --script <file>")
		}
	}

	shellSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "shell" {
			shellSet = true
		}
	})
	if shellSet && *script == "" {
		return execOptions{}, errors.New("exec: --shell requires --script")
	}
	if *shell != "sh" && *shell != "bash" {
		return execOptions{}, fmt.Errorf("exec: unsupported shell %q (use sh or bash)", *shell)
	}
	return execOptions{alias: selectedAlias, id: selectedID, command: firstOrEmpty(positional), script: *script, shell: *shell}, nil
}

func firstOrEmpty(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func buildExecInvocation(conn *model.SSHConnection, remoteCommand string) (string, []string, []string, error) {
	if err := validateExecSSHArgs(conn.ExtraSSHArgs); err != nil {
		return "", nil, nil, err
	}
	// A one-shot command must not open listeners from a saved interactive profile.
	execConn := *conn
	execConn.LocalForwards = nil
	execConn.RemoteForwards = nil
	bin, args, env, err := buildConnectInvocation(&execConn)
	if err != nil {
		return "", nil, nil, err
	}
	// The destination is the final argument from buildConnectInvocation.
	destination := args[len(args)-1]
	args = append(args[:len(args)-1], "-T", destination, remoteCommand)
	return bin, args, env, nil
}

func validateExecSSHArgs(args []string) error {
	normalized := model.NormalizeStringList(args)
	for i, arg := range normalized {
		switch arg {
		case "-N", "-n", "-t":
			return fmt.Errorf("saved SSH option %q is incompatible with exec", arg)
		}
		option := ""
		if arg == "-o" && i+1 < len(normalized) {
			option = normalized[i+1]
		} else if strings.HasPrefix(arg, "-o") {
			option = strings.TrimPrefix(strings.TrimPrefix(arg, "-o"), "=")
		}
		key, value, _ := strings.Cut(option, "=")
		switch strings.ToLower(key) {
		case "requesttty", "stdinnull", "forkafterauthentication":
			if strings.EqualFold(value, "no") {
				continue
			}
			return fmt.Errorf("saved SSH option %q is incompatible with exec", key)
		case "sessiontype":
			if strings.EqualFold(value, "default") {
				continue
			}
			return fmt.Errorf("saved SSH option %q is incompatible with exec", key)
		case "remotecommand":
			return fmt.Errorf("saved SSH option %q is incompatible with exec", key)
		}
	}
	return nil
}

func runExec(bin string, args, envAdd []string, conn *model.SSHConnection, stdin io.Reader, stdout, stderr io.Writer) error {
	binPath, err := exec.LookPath(bin)
	if err != nil {
		if bin == "sshpass" {
			return errors.New(prompttext.DefaultPromptTexts.ErrorMessages.SSHPassNotFound)
		}
		return fmt.Errorf("required command %q not found in PATH", bin)
	}
	if bin == "sshpass" {
		if err := ensureHostKeyAccepted(strings.TrimSpace(conn.Host), conn.EffectivePort()); err != nil {
			return err
		}
	}
	cmd := exec.Command(binPath, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = append(os.Environ(), envAdd...)
	err = cmd.Run()
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() < 0 {
			return fmt.Errorf("exec: SSH process terminated: %w", err)
		}
		detail := ""
		if bin == "sshpass" {
			translated := translateSSHPassError(err, conn.Host)
			if translated != err {
				detail = translated.Error()
			}
		}
		return &ExecExitError{Code: exitErr.ExitCode(), Detail: detail}
	}
	return fmt.Errorf("exec: start SSH: %w", err)
}
