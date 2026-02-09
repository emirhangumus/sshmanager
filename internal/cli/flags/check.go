package flags

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/emirhangumus/sshmanager/internal/completion"
	"github.com/emirhangumus/sshmanager/internal/config"
	"github.com/emirhangumus/sshmanager/internal/store"
	prompttext "github.com/emirhangumus/sshmanager/internal/ui/prompt"
	"github.com/manifoldco/promptui"
)

// Check parses and executes command-line flags.
// Returns true if the program should continue to menu/alias flow.
func Check(args []string, connectionFilePath, secretKeyFilePath, configFilePath, version string) (bool, error) {
	fs := flag.NewFlagSet("sshmanager", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	clean := fs.Bool("clean", false, "Reset all saved SSH connections and key file")
	setConfig := fs.Bool("set", false, "Set SSHManager configuration. Usage: sshmanager -set <config-name> <config-value>")
	versionFlag := fs.Bool("version", false, "Show build version")
	complete := fs.Bool("complete", false, "Output aliases for shell completion. Usage: sshmanager -complete [prefix]")
	completionFlag := fs.String("completion", "", "Completion action. Usage: sshmanager -completion <bash|zsh|install>")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return false, nil
		}
		return false, err
	}

	extraArgs := fs.Args()

	if *completionFlag != "" {
		if err := handleCompletion(*completionFlag, extraArgs); err != nil {
			return false, err
		}
		return false, nil
	}

	if *complete {
		prefix := ""
		if len(extraArgs) > 0 {
			prefix = extraArgs[0]
		}

		if err := printCompletionCandidates(connectionFilePath, secretKeyFilePath, prefix); err != nil {
			return false, err
		}
		return false, nil
	}

	if *clean {
		if err := CleanSSHFiles(connectionFilePath, secretKeyFilePath); err != nil {
			return false, err
		}
		return false, nil
	}

	if *setConfig {
		if len(extraArgs) < 2 {
			return false, errors.New("not enough arguments for -set; expected: sshmanager -set <config-name> <config-value>")
		}
		if err := config.SetConfig(configFilePath, extraArgs[0], extraArgs[1]); err != nil {
			return false, err
		}
		return false, nil
	}

	if *versionFlag {
		fmt.Println(version)
		return false, nil
	}

	return true, nil
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
			return errors.New("missing shell argument for install: usage: sshmanager -completion install <bash|zsh>")
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
		return fmt.Errorf("unknown -completion mode %q (use bash, zsh, install)", mode)
	}
}

func confirmInstall(shell string) (bool, error) {
	p := promptui.Prompt{Label: fmt.Sprintf("Install %s completion into your home directory? Type 'yes' to continue", shell)}
	v, err := p.Run()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(strings.ToLower(v)) == "yes", nil
}

func printCompletionCandidates(connectionFilePath, secretKeyFilePath, prefix string) error {
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
