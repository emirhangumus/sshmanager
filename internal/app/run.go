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

	normalizedArgs := normalizeLegacyCommandArgs(args)

	if len(normalizedArgs) >= 2 {
		cmd := strings.TrimSpace(normalizedArgs[1])
		switch cmd {
		case "-h", "--help", "help":
			flags.PrintUsage(os.Stdout)
			return nil
		case "version":
			flags.HandleVersion(build.VersionString(), os.Stdout)
			return nil
		case "completion":
			return flags.HandleCompletion(normalizedArgs[2:])
		case "doctor":
			return commands.HandleDoctor(connectionFilePath, secretKeyFilePath, configFilePath, normalizedArgs[2:])
		case "clean":
			return flags.CleanSSHFiles(connectionFilePath, secretKeyFilePath)
		case "set":
			return flags.HandleSet(configFilePath, normalizedArgs[2:])
		case "complete":
			return flags.HandleComplete(connectionFilePath, secretKeyFilePath, normalizedArgs[2:])
		default:
			if strings.HasPrefix(cmd, "-") {
				return fmt.Errorf("unknown option %q (use 'sshmanager help')", cmd)
			}
		}
	}

	if err := startup.Setup(connectionFilePath, configFilePath, secretKeyFilePath); err != nil {
		return fmt.Errorf("startup failed: %w", err)
	}

	if len(normalizedArgs) >= 2 && !strings.HasPrefix(normalizedArgs[1], "-") {
		switch normalizedArgs[1] {
		case "add":
			return commands.HandleAddArgs(connectionFilePath, secretKeyFilePath, normalizedArgs[2:])
		case "edit":
			return commands.HandleEditArgs(connectionFilePath, secretKeyFilePath, normalizedArgs[2:])
		case "remove":
			return commands.HandleRemoveArgs(connectionFilePath, secretKeyFilePath, normalizedArgs[2:])
		case "rename":
			return commands.HandleRenameArgs(connectionFilePath, secretKeyFilePath, normalizedArgs[2:])
		case "connect":
			return commands.HandleConnectArgs(connectionFilePath, secretKeyFilePath, configFilePath, normalizedArgs[2:])
		case "list":
			return commands.HandleList(connectionFilePath, secretKeyFilePath, normalizedArgs[2:])
		case "export":
			return commands.HandleExport(connectionFilePath, secretKeyFilePath, normalizedArgs[2:])
		case "import":
			return commands.HandleImport(connectionFilePath, secretKeyFilePath, normalizedArgs[2:])
		case "backup":
			return commands.HandleBackup(connectionFilePath, secretKeyFilePath, configFilePath, normalizedArgs[2:])
		case "restore":
			return commands.HandleRestore(connectionFilePath, secretKeyFilePath, configFilePath, normalizedArgs[2:])
		default:
			if len(normalizedArgs) == 2 {
				if err := commands.FindAndConnect(connectionFilePath, secretKeyFilePath, configFilePath, normalizedArgs[1]); err != nil {
					return err
				}
				return nil
			}
		}
	}

	return cli.ShowMainMenu(connectionFilePath, secretKeyFilePath, configFilePath, build.VersionString())
}

func normalizeLegacyCommandArgs(args []string) []string {
	if len(args) < 2 {
		return args
	}

	mapped, ok := flags.MapLegacyDashCommand(args[1])
	if !ok {
		return args
	}

	normalized := make([]string, 0, len(args))
	normalized = append(normalized, args[0], mapped)
	normalized = append(normalized, args[2:]...)
	return normalized
}
