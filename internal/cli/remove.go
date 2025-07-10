package cli

import (
	"fmt"
	"strings"

	"github.com/emirhangumus/sshmanager/internal/encryption"
	"github.com/emirhangumus/sshmanager/internal/storage"
	"github.com/manifoldco/promptui"
)

func HandleRemove(dataPath, keyPath string) {
	connections, err := storage.ReadAllConnections(dataPath, keyPath)
	if err != nil || len(connections) == 0 {
		fmt.Println("No SSH connections found.")
		return
	}

	items := ConnToStrSlice(connections)
	prompt := promptui.Select{Label: "Select a connection to remove", Items: items}
	_, result, err := prompt.Run()
	if err != nil || result == "Back to main menu" {
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
		fmt.Println("Error loading encryption key.")
		return
	}

	if err := storage.StoreFile(strings.Join(lines, "\n"), dataPath, keyBytes); err != nil {
		fmt.Println("Error updating file.")
		return
	}

	fmt.Println("SSH connection removed.")
}
