package commands

import (
	"fmt"

	"github.com/emirhangumus/sshmanager/internal/store"
	prompttext "github.com/emirhangumus/sshmanager/internal/ui/prompt"
)

func HandleAdd(connectionFilePath, secretKeyFilePath string) error {
	conn, err := prompttext.AddSSHConnectionPrompt()
	if err != nil {
		return err
	}

	connStore := store.NewConnectionStore(connectionFilePath, secretKeyFilePath)
	connFile, err := connStore.Load()
	if err != nil {
		return err
	}

	connFile.AddConnection(conn)
	if err := connStore.Save(connFile); err != nil {
		return err
	}

	fmt.Println(prompttext.DefaultPromptTexts.SuccessMessages.SSHConnectionSaved)
	return nil
}
