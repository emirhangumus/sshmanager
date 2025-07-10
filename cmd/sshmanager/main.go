package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/emirhangumus/sshmanager/internal/cli"
	"github.com/emirhangumus/sshmanager/internal/encryption"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Could not determine home directory: %v", err)
	}

	defaultFile := filepath.Join(homeDir, ".sshmanager", "conn")
	keyFile := filepath.Join(homeDir, ".sshmanager", "secret.key")

	clean := flag.Bool("clean", false, "Reset all saved SSH connections and key file")
	flag.Parse()

	if *clean {
		encryption.SecureDelete(defaultFile)
		encryption.SecureDelete(keyFile)
		fmt.Println("All SSH connections and key files removed.")
		return
	}

	cli.ShowMainMenu(defaultFile, keyFile)
}
