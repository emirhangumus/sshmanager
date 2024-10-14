package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/manifoldco/promptui"
)

// Connect to SSH
func connectSSH(conn map[string]string) {
	fmt.Println("Connecting to SSH...")

	// whereis sshpass
	sshpass, err := exec.LookPath("sshpass")
	if err != nil {
		fmt.Println("sshpass not found. Please install sshpass.")
		return
	}

	if err := syscall.Exec(sshpass, []string{"sshpass", "-p", conn["password"], "ssh", conn["username"] + "@" + conn["host"]}, os.Environ()); err != nil {
		fmt.Println(err)
	}
}

// Prompt user to add SSH connection
func addSSHConnection(defaultFilePath, keyFilePath string) error {
	sshConnection, err := addSSHConnectionPrompt()

	if err != nil {
		return err
	}

	key, err := loadKey(keyFilePath)

	if err != nil {
		return err
	}

	err = storeFile(sshConnection, defaultFilePath, key)
	if err != nil {
		return err
	}

	fmt.Println("SSH connection details added successfully!")
	return nil
}

func connToStrSlice(conns map[string]map[string]string) []string {
	items := []string{"Exit"}

	for _, value := range conns {
		items = append(items, value["username"]+"@"+value["host"])
	}

	return items
}

func findKeyOfSelectedSSHOption(choice string, conns map[string]map[string]string) (error, string) {
	data := strings.Split(choice, "@") // ['username', 'host]

	for key, value := range conns {
		if data[0] == value["username"] && data[1] == value["host"] {
			return nil, key
		}
	}

	return errors.New("The selected options not in connections"), ""
}

// Show SSH connections and prompt for action
func showConnections(defaultFilePath, keyFilePath string) {
	connections, err := readAllConnections(defaultFilePath, keyFilePath)
	if err != nil {
		fmt.Println("No SSH connections found.")
		return
	}

	items := connToStrSlice(connections)

	prompt := promptui.Select{
		Label: "Select an SSH connection",
		Items: items,
	}

	_, result, err := prompt.Run()
	if err != nil {
		fmt.Println("Invalid connection selection. Please try again.")
		return
	}

	switch result {
	case "Exit":
		return
	default:
		err, index := findKeyOfSelectedSSHOption(result, connections)
		if err != nil {
			return
		}
		conn := connections[index]
		connectSSH(conn)
	}
}

// Main menu options
func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Could not determine home directory: %v", err)
	}

	defaultFilePath := filepath.Join(homeDir, ".sshmanager", "conn")
	keyFilePath := filepath.Join(homeDir, ".sshmanager", "secret.key")

	menuOptions := []string{"Exit", "Connect to SSH", "Add SSH Connection"}

	for {
		prompt := promptui.Select{
			Label: "Menu Options",
			Items: menuOptions,
		}

		_, choice, err := prompt.Run()
		if err != nil {
			fmt.Println("Invalid option selected. Please try again.")
			// print err
			continue
		}

		switch choice {
		case "Exit":
			return
		case "Connect to SSH":
			showConnections(defaultFilePath, keyFilePath)
		case "Add SSH Connection":
			addSSHConnection(defaultFilePath, keyFilePath)
		}
	}
}
