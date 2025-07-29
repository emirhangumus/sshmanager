package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/emirhangumus/sshmanager/internal/cli/flag"
	"github.com/emirhangumus/sshmanager/internal/startup"

	"github.com/emirhangumus/sshmanager/internal/cli"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Could not determine home directory: %v", err)
	}

	connectionFilePath := filepath.Join(homeDir, ".sshmanager", "conn")
	secretKeyFilePath := filepath.Join(homeDir, ".sshmanager", "secret.key")
	configFilePath := filepath.Join(homeDir, ".sshmanager", "config.yaml")

	if err := startup.Startup(connectionFilePath, configFilePath, secretKeyFilePath); err != nil {
		log.Fatalf("Error during startup: %v", err)
	}

	continueExecution, err := flag.Check(connectionFilePath, secretKeyFilePath, configFilePath)
	if err != nil {
		log.Fatalf("Error checking flags: %v", err)
	}
	if !continueExecution {
		return
	}

	// That means, user provided an alias to connect to a specific connection
	if len(os.Args) == 2 {
		arg := os.Args[1]
		cli.FindAndConnect(connectionFilePath, secretKeyFilePath, configFilePath, arg)
		return
	}

	cli.ShowMainMenu(connectionFilePath, secretKeyFilePath, configFilePath)
}
