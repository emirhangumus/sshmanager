package cli

import (
	"fmt"
	"strings"

	"github.com/emirhangumus/sshmanager/internal/gstructs/g_connectionfile"
	"github.com/emirhangumus/sshmanager/internal/prompt"
	"github.com/manifoldco/promptui"
)

func HandleEdit(connectionFilePath string, secretKeyFilePath string) error {
	connFile := g_connectionfile.NewConnectionFile(connectionFilePath, secretKeyFilePath)
	if len(connFile.Connections) == 0 {
		fmt.Println(prompt.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return nil
	}

	items := connFile.SafeConnectionListString()
	_prompt := promptui.Select{Label: "Select a connection to edit", Items: items}
	_, result, err := _prompt.Run()
	if err != nil || result == prompt.DefaultPromptTexts.BackToMainMenu {
		return nil
	}

	index := strings.Split(result, ".")[0]
	conn := connFile.GetConnection(index)
	if conn == nil {
		fmt.Println("No SSH connection found.")
		return nil
	}

	connStr, err := prompt.EditSSHConnectionPrompt(conn)
	if err != nil {
		return err
	}

	connFile.UpdateConnection(index, connStr)
	if err := connFile.Save(connectionFilePath, secretKeyFilePath); err != nil {
		fmt.Println(prompt.DefaultPromptTexts.ErrorMessages.FailedToStoreUpdatedConnectionsX, err)
		return err
	}

	fmt.Println("Connection updated successfully.")
	return nil
}
