package flags

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/emirhangumus/sshmanager/internal/completion"
	"github.com/emirhangumus/sshmanager/internal/config"
	"github.com/emirhangumus/sshmanager/internal/store"
	prompttext "github.com/emirhangumus/sshmanager/internal/ui/prompt"
)

func PrintUsage(out io.Writer) {
	const usage = `Usage of sshmanager:
  sshmanager [command] [arguments]
  sshmanager <alias>

Connection Commands:
  add [flags]
        Create a new SSH connection (interactive if no flags)
        --host --username [--port] [--auth-mode password|key|agent] [--password] [--identity-file]
        [--proxy-jump] [--local-forward ...] [--remote-forward ...] [--extra-ssh-arg ...]
        [--group] [--tag ...] [--description] [--alias]
  edit [flags]
        Update an existing connection (interactive if no flags)
        Target: --alias <alias> | --id <connection-id>
        Updates: --new-host --new-username --new-port --new-auth-mode --new-password --new-identity-file
        --new-proxy-jump --new-local-forward ... --new-remote-forward ... --new-extra-ssh-arg ...
        --new-group --new-tag ... --new-description --new-alias
        Clears: --clear-alias --clear-description --clear-proxy-jump --clear-group
        --clear-local-forwards --clear-remote-forwards --clear-extra-ssh-args --clear-tags
  remove [flags]
        Remove a connection
        Target: --alias <alias> | --id <connection-id>
        Options: --yes (skip confirmation)
  rename [flags]
        Rename a connection alias
        Target: --alias <alias> | --id <connection-id>
        Required: --to <new-alias>
  connect [flags]
        Connect to a saved host (interactive if no flags)
        Target: --alias <alias> | --id <connection-id>
  list [flags]
        List saved connections
        --json
        --field id|alias|username|host|port|auth-mode|identity-file|proxy-jump|local-forwards|remote-forwards|extra-ssh-args|group|tags|description|target
        --group <name> --tag <tag> (repeatable)

Transfer / Recovery Commands:
  export --out <path> [--format yaml|json]
        Export decrypted connection data to file
  import --in <path> [--format auto|yaml|json] [--mode merge|replace]
        Import connection data from file
  backup --out <path> [--format yaml|json] [--include-config=true|false]
        Create recovery snapshot (connections + optional config)
  restore --in <path> [--format auto|yaml|json] [--mode merge|replace] [--with-config=true|false]
        Restore from recovery snapshot
  doctor [--json]
        Run consistency diagnostics for config/key/connection data

Utility Commands:
  clean
        Reset all saved SSH connections and key file
  set <config-name> <config-value>
        Set SSHManager configuration
  version
        Show build version
  complete [prefix]
        Output aliases for shell completion
  completion <bash|zsh>
        Print completion script
  completion install <bash|zsh>
        Install completion script
  help
        Show this help

Notes:
  - Running without a command opens the interactive menu.
  - Using a single non-command token tries alias connect (e.g. sshmanager prod).
  - Legacy dash commands (-clean, -set, -version, -complete, -completion) remain supported.`
	_, _ = fmt.Fprintln(out, usage)
}

func MapLegacyDashCommand(token string) (string, bool) {
	switch strings.TrimSpace(token) {
	case "-clean":
		return "clean", true
	case "-complete":
		return "complete", true
	case "-completion":
		return "completion", true
	case "-set":
		return "set", true
	case "-version":
		return "version", true
	default:
		return "", false
	}
}

func HandleVersion(version string, out io.Writer) {
	_, _ = fmt.Fprintln(out, version)
}

func HandleSet(configFilePath string, args []string) error {
	if len(args) < 2 {
		return errors.New("not enough arguments for set; expected: sshmanager set <config-name> <config-value>")
	}
	return config.SetConfig(configFilePath, args[0], args[1])
}

func HandleComplete(connectionFilePath, secretKeyFilePath string, args []string) error {
	if len(args) > 1 {
		return errors.New("too many arguments for complete; expected: sshmanager complete [prefix]")
	}

	prefix := ""
	if len(args) == 1 {
		prefix = args[0]
	}

	return printCompletionCandidates(connectionFilePath, secretKeyFilePath, prefix)
}

func HandleCompletion(args []string) error {
	if len(args) < 1 {
		return errors.New("missing completion mode: usage: sshmanager completion <bash|zsh|install>")
	}
	return handleCompletion(args[0], args[1:])
}

func handleCompletion(mode string, extraArgs []string) error {
	mode = strings.ToLower(strings.TrimSpace(mode))

	switch mode {
	case "bash", "zsh":
		script, err := completion.Script(mode)
		if err != nil {
			return err
		}
		fmt.Print(script)
		return nil
	case "install":
		if len(extraArgs) < 1 {
			return errors.New("missing shell argument for install: usage: sshmanager completion install <bash|zsh>")
		}
		shell := strings.ToLower(strings.TrimSpace(extraArgs[0]))

		confirmed, err := confirmInstall(shell)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println(prompttext.DefaultPromptTexts.SuccessMessages.OperationCancelled)
			return nil
		}

		installedPath, err := completion.Install(shell)
		if err != nil {
			return fmt.Errorf(prompttext.DefaultPromptTexts.ErrorMessages.FailedToInstallCompletionX, err)
		}
		fmt.Println(fmt.Sprintf(prompttext.DefaultPromptTexts.SuccessMessages.CompletionInstalledX, shell))
		fmt.Printf("Completion file: %s\n", installedPath)
		if shell == "bash" {
			fmt.Println("Reload your shell with: source ~/.bashrc")
		}
		return nil
	default:
		return fmt.Errorf("unknown completion mode %q (use bash, zsh, install)", mode)
	}
}

func confirmInstall(shell string) (bool, error) {
	v, err := prompttext.InputPrompt(
		fmt.Sprintf("Install %s completion into your home directory? Type 'yes' to continue", shell),
		"",
		false,
		nil,
	)
	if err != nil {
		if prompttext.IsCancelError(err) {
			return false, nil
		}
		return false, err
	}
	return strings.TrimSpace(strings.ToLower(v)) == "yes", nil
}

func printCompletionCandidates(connectionFilePath, secretKeyFilePath, prefix string) error {
	if _, err := os.Stat(connectionFilePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if _, err := os.Stat(secretKeyFilePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	connStore := store.NewConnectionStore(connectionFilePath, secretKeyFilePath)
	connFile, err := connStore.Load()
	if err != nil {
		return err
	}

	for _, alias := range connFile.AllAliases() {
		if strings.HasPrefix(alias, prefix) {
			fmt.Println(alias)
		}
	}
	return nil
}
