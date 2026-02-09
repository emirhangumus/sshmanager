package commands

import (
	"fmt"

	"github.com/emirhangumus/sshmanager/internal/store"
	prompttext "github.com/emirhangumus/sshmanager/internal/ui/prompt"
	"github.com/manifoldco/promptui"
)

func HandleRemove(connectionFilePath, secretKeyFilePath string) error {
	connStore := store.NewConnectionStore(connectionFilePath, secretKeyFilePath)
	connFile, err := connStore.Load()
	if err != nil {
		return err
	}
	if len(connFile.Connections) == 0 {
		fmt.Println(prompttext.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return nil
	}

	items := connFile.SelectItems()
	selector := promptui.Select{Label: prompttext.DefaultPromptTexts.SelectAConnectionToRemove, Items: items}
	idx, _, err := selector.Run()
	if err != nil {
		return nil
	}

	connID := items[idx].ConnectionID
	if !connFile.RemoveConnectionByID(connID) {
		fmt.Println(prompttext.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return nil
	}

	if err := connStore.Save(connFile); err != nil {
		return err
	}

	fmt.Println(prompttext.DefaultPromptTexts.SuccessMessages.SSHConnectionRemoved)
	return nil
}
