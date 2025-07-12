package cli

import (
	"fmt"
	"strings"

	"github.com/emirhangumus/sshmanager/internal/encryption"
	"github.com/emirhangumus/sshmanager/internal/prompt"
	"github.com/emirhangumus/sshmanager/internal/storage"
	"github.com/manifoldco/promptui"
)

func HandleRemove(connectionFilePath string, secretKeyFilePath string) {
	connections, err := storage.ReadAllConnections(connectionFilePath, secretKeyFilePath)
	if err != nil || len(connections) == 0 {
		fmt.Println(prompt.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return
	}

	items := ConnToStrSlice(connections)
	_prompt := promptui.Select{Label: prompt.DefaultPromptTexts.SelectAConnectionToRemove, Items: items}
	_, result, err := _prompt.Run()
	if err != nil || result == prompt.DefaultPromptTexts.BackToMainMenu {
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

	keyBytes, err := encryption.LoadKey(secretKeyFilePath)
	if err != nil {
		fmt.Println(prompt.DefaultPromptTexts.ErrorMessages.ErrorLoadingEncryptionKeyX, err)
		return
	}

	if err := storage.EncryptAndStoreFile(strings.Join(lines, "\n"), connectionFilePath, keyBytes); err != nil {
		fmt.Println(prompt.DefaultPromptTexts.ErrorMessages.FailedToStoreUpdatedConnectionsX, err)
		return
	}

	fmt.Println(prompt.DefaultPromptTexts.SuccessMessages.SSHConnectionRemoved)
}
