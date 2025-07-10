package cli

import (
	"fmt"

	"github.com/emirhangumus/sshmanager/internal/encryption"
	"github.com/emirhangumus/sshmanager/internal/prompts"
	"github.com/emirhangumus/sshmanager/internal/storage"
)

func HandleAdd(dataPath, keyPath string) error {
	connStr, err := prompts.AddSSHConnectionPrompt()
	if err != nil {
		return err
	}

	key, err := encryption.LoadKey(keyPath)
	if err != nil {
		return err
	}

	content, _ := storage.ReadFile(dataPath, key)
	if content != "" {
		content += "\n" + connStr
	} else {
		content = connStr
	}

	if err := storage.StoreFile(content, dataPath, key); err != nil {
		return err
	}

	fmt.Println("SSH connection saved.")
	return nil
}
