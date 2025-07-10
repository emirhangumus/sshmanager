package cli

import (
	"fmt"
	"strings"

	"github.com/emirhangumus/sshmanager/internal/encryption"
	"github.com/emirhangumus/sshmanager/internal/prompts"
	"github.com/emirhangumus/sshmanager/internal/storage"
	"github.com/manifoldco/promptui"
)

func HandleRemove(dataPath, keyPath string) {
	connections, err := storage.ReadAllConnections(dataPath, keyPath)
	if err != nil || len(connections) == 0 {
		fmt.Println(prompts.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return
	}

	items := ConnToStrSlice(connections)
	prompt := promptui.Select{Label: prompts.DefaultPromptTexts.SelectAConnectionToRemove, Items: items}
	_, result, err := prompt.Run()
	if err != nil || result == prompts.DefaultPromptTexts.BackToMainMenu {
		return
	}

	index := strings.Split(result, ".")[0]

	// Filter out the connection to be removed
	var updatedConnections []storage.SSHConnection
	for _, conn := range connections {
		if conn.Index != index {
			updatedConnections = append(updatedConnections, conn)
		}
	}

	// Convert updated list to text format
	var lines []string
	for _, conn := range updatedConnections {
		line := fmt.Sprintf("%s@%s\t%s\t%s", conn.Username, conn.Host, conn.Password, conn.Description)
		lines = append(lines, line)
	}

	keyBytes, err := encryption.LoadKey(keyPath)
	if err != nil {
		fmt.Println(prompts.DefaultPromptTexts.ErrorMessages.ErrorLoadingEncryptionKeyX, err)
		return
	}

	if err := storage.StoreFile(strings.Join(lines, "\n"), dataPath, keyBytes); err != nil {
		fmt.Println(prompts.DefaultPromptTexts.ErrorMessages.FailedToStoreUpdatedConnectionsX, err)
		return
	}

	fmt.Println(prompts.DefaultPromptTexts.SuccessMessages.SSHConnectionRemoved)
}
