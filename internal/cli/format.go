package cli

import (
	"errors"
	"fmt"

	"github.com/emirhangumus/sshmanager/internal/prompt"
	"github.com/emirhangumus/sshmanager/internal/storage"
)

func ConnToStrSlice(connections []storage.SSHConnection) []string {
	items := []string{prompt.DefaultPromptTexts.BackToMainMenu}
	for _, conn := range connections {
		display := fmt.Sprintf("%s. %s@%s - %s", conn.Index, conn.Username, conn.Host, conn.Description)
		items = append(items, display)
	}
	return items
}

func GetConnByIndex(index string, connections []storage.SSHConnection) (storage.SSHConnection, error) {
	for _, conn := range connections {
		if conn.Index == index {
			return conn, nil
		}
	}
	return storage.SSHConnection{}, errors.New(prompt.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
}
