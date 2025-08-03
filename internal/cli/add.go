package cli

import (
	"fmt"
	"github.com/emirhangumus/sshmanager/internal/gstructs/connectionfile"

	"github.com/emirhangumus/sshmanager/internal/prompt"
)

func HandleAdd(connectionFilePath string, secretKeyFilePath string) error {
	connStr, err := prompt.AddSSHConnectionPrompt()
	if err != nil {
		return err
	}

	connFile := connectionfile.NewConnectionFile(connectionFilePath, secretKeyFilePath)
	connFile.AddConnection(connStr)
	if err := connFile.Save(connectionFilePath, secretKeyFilePath); err != nil {
		return err
	}

	fmt.Println(prompt.DefaultPromptTexts.SuccessMessages.SSHConnectionSaved)
	return nil
}
