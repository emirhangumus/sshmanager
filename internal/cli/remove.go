package cli

import (
	"fmt"
	"github.com/emirhangumus/sshmanager/internal/gstructs/connectionfile"
	"strings"

	"github.com/emirhangumus/sshmanager/internal/prompt"
	"github.com/manifoldco/promptui"
)

func HandleRemove(connectionFilePath string, secretKeyFilePath string) error {
	connFile := connectionfile.NewConnectionFile(connectionFilePath, secretKeyFilePath)
	if len(connFile.Connections) == 0 {
		fmt.Println(prompt.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return nil
	}

	items := connFile.SafeConnectionListString()
	_prompt := promptui.Select{Label: prompt.DefaultPromptTexts.SelectAConnectionToRemove, Items: items}
	_, result, err := _prompt.Run()
	if err != nil || result == prompt.DefaultPromptTexts.BackToMainMenu {
		return nil
	}

	index := strings.Split(result, ".")[0]

	connFile.RemoveConnection(index)
	if err := connFile.Save(connectionFilePath, secretKeyFilePath); err != nil {
		fmt.Println(prompt.DefaultPromptTexts.ErrorMessages.FailedToStoreUpdatedConnectionsX, err)
		return err
	}

	fmt.Println(prompt.DefaultPromptTexts.SuccessMessages.SSHConnectionRemoved)
	return nil
}
