package flag

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/emirhangumus/sshmanager/internal/cli/flag/compScripts"
	"github.com/emirhangumus/sshmanager/internal/gstructs/connectionfile"
)

var SSHManagerVersion = "v1.1.0"

// Check function checks the command line flags for specific actions.
// Returns true if process should continue, false if it should exit.
func Check(connectionFilePath string, secretKeyFilePath string, configFilePath string) (bool, error) {
	clean := flag.Bool("clean", false, "Reset all saved SSH connections and key file")
	setConfig := flag.Bool("set", false, "Set a SSHManager configuration.\nUsage: sshmanager -set <config-name> <config-value>\nFor possible config names and values, see the documentation or README.")
	version := flag.Bool("version", false, "Show the version of SSHManager")
	complete := flag.Bool("complete", false, "Show complete list of hosts for tab completion.\nUsage: sshmanager -complete [prefix]\nIf prefix is provided, only hosts starting with that prefix will be shown.")
	completion := flag.String("completion", "", "Create a shell completion script for SSHManager.\nUsage: sshmanager -completion [shell]\nSupported shells: bash, zsh.")

	flag.Parse()

	if *completion != "" {
		switch shell := strings.ToLower(*completion); shell {
		case "bash":
			fmt.Print(compScripts.BashScript)
		case "zsh":
			fmt.Print(compScripts.ZshScript)
		default:
			fmt.Fprintf(os.Stderr, "unknown shell %q (want \"bash\" or \"zsh\")\n", shell)
			return false, errors.New("unknown shell for completion script")
		}
		return false, nil
	}
	if *complete {
		prefix := ""
		if len(os.Args) == 3 {
			prefix = os.Args[2]
		}

		connFile := connectionfile.NewConnectionFile(connectionFilePath, secretKeyFilePath)
		aliases := connFile.GeetAllAliases()

		for _, h := range aliases {
			if strings.HasPrefix(h, prefix) {
				fmt.Println(h)
			}
		}
		return false, nil
	}
	if *clean {
		CleanSSHFile(clean, connectionFilePath, secretKeyFilePath)
		return false, nil
	}
	if *setConfig {
		if len(flag.Args()) < 2 {
			return false, errors.New("not enough arguments for -set flag, expected: sshmanager -set <config-name> <config-value>")
		}
		configName := flag.Arg(0)
		configValue := flag.Arg(1)
		err := SetConfig(configFilePath, configName, configValue)
		if err != nil {
			return false, err
		}
		return false, nil
	}
	if *version {
		println(SSHManagerVersion)
		return false, nil
	}

	return true, nil
}
