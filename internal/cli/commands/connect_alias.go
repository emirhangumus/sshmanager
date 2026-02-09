package commands

import (
	"fmt"

	"github.com/emirhangumus/sshmanager/internal/store"
	prompttext "github.com/emirhangumus/sshmanager/internal/ui/prompt"
)

func FindAndConnect(connectionFilePath, secretKeyFilePath, alias string) error {
	connStore := store.NewConnectionStore(connectionFilePath, secretKeyFilePath)
	connFile, err := connStore.Load()
	if err != nil {
		return err
	}
	if len(connFile.Connections) == 0 {
		fmt.Println(prompttext.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return nil
	}

	conn := connFile.GetConnectionByAlias(alias)
	if conn == nil {
		fmt.Printf(prompttext.DefaultPromptTexts.ErrorMessages.AliasNotFoundX+"\n", alias)
		return nil
	}

	fmt.Printf("Connecting to %s@%s...\n", conn.Username, conn.Host)
	if err := connect(conn); err != nil {
		fmt.Printf(prompttext.DefaultPromptTexts.ErrorMessages.ConnectionToXFailedX+"\n", fmt.Sprintf("%s@%s", conn.Username, conn.Host), err)
		return nil
	}

	return nil
}
