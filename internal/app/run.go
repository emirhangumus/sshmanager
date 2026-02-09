package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/emirhangumus/sshmanager/internal/cli"
	"github.com/emirhangumus/sshmanager/internal/cli/commands"
	"github.com/emirhangumus/sshmanager/internal/cli/flags"
	"github.com/emirhangumus/sshmanager/internal/startup"
)

type BuildInfo struct {
	Version   string
	Commit    string
	BuildTime string
}

func (b BuildInfo) VersionString() string {
	v := strings.TrimSpace(b.Version)
	if v == "" {
		return "dev"
	}
	return v
}

func Run(args []string, build BuildInfo) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %w", err)
	}

	connectionFilePath := filepath.Join(homeDir, ".sshmanager", "conn")
	secretKeyFilePath := filepath.Join(homeDir, ".sshmanager", "secret.key")
	configFilePath := filepath.Join(homeDir, ".sshmanager", "config.yaml")

	if len(args) >= 2 && !strings.HasPrefix(args[1], "-") && args[1] == "doctor" {
		return commands.HandleDoctor(connectionFilePath, secretKeyFilePath, configFilePath, args[2:])
	}

	if err := startup.Setup(connectionFilePath, configFilePath, secretKeyFilePath); err != nil {
		return fmt.Errorf("startup failed: %w", err)
	}

	continueExecution, err := flags.Check(args[1:], connectionFilePath, secretKeyFilePath, configFilePath, build.VersionString())
	if err != nil {
		return fmt.Errorf("flag handling failed: %w", err)
	}
	if !continueExecution {
		return nil
	}

	if len(args) >= 2 && !strings.HasPrefix(args[1], "-") {
		switch args[1] {
		case "add":
			return commands.HandleAddArgs(connectionFilePath, secretKeyFilePath, args[2:])
		case "edit":
			return commands.HandleEditArgs(connectionFilePath, secretKeyFilePath, args[2:])
		case "remove":
			return commands.HandleRemoveArgs(connectionFilePath, secretKeyFilePath, args[2:])
		case "rename":
			return commands.HandleRenameArgs(connectionFilePath, secretKeyFilePath, args[2:])
		case "connect":
			return commands.HandleConnectArgs(connectionFilePath, secretKeyFilePath, configFilePath, args[2:])
		case "list":
			return commands.HandleList(connectionFilePath, secretKeyFilePath, args[2:])
		case "export":
			return commands.HandleExport(connectionFilePath, secretKeyFilePath, args[2:])
		case "import":
			return commands.HandleImport(connectionFilePath, secretKeyFilePath, args[2:])
		case "backup":
			return commands.HandleBackup(connectionFilePath, secretKeyFilePath, configFilePath, args[2:])
		case "restore":
			return commands.HandleRestore(connectionFilePath, secretKeyFilePath, configFilePath, args[2:])
		default:
			if len(args) == 2 {
				if err := commands.FindAndConnect(connectionFilePath, secretKeyFilePath, configFilePath, args[1]); err != nil {
					return err
				}
				return nil
			}
		}
	}

	return cli.ShowMainMenu(connectionFilePath, secretKeyFilePath, configFilePath, build.VersionString())
}
