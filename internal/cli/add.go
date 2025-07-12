package cli

import (
	"fmt"

	"github.com/emirhangumus/sshmanager/internal/encryption"
	"github.com/emirhangumus/sshmanager/internal/prompt"
	"github.com/emirhangumus/sshmanager/internal/storage"
)

func HandleAdd(connectionFilePath string, secretKeyFilePath string) error {
	connStr, err := prompt.AddSSHConnectionPrompt()
	if err != nil {
		return err
	}

	key, err := encryption.LoadKey(secretKeyFilePath)
	if err != nil {
		return err
	}

	content, _ := storage.DecryptAndReadFile(connectionFilePath, key)
	if content != "" {
		content += "\n" + connStr
	} else {
		content = connStr
	}

	if err := storage.EncryptAndStoreFile(content, connectionFilePath, key); err != nil {
		return err
	}

	fmt.Println(prompt.DefaultPromptTexts.SuccessMessages.SSHConnectionSaved)
	return nil
}
