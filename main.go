package main

import (
	"errors"
	"flag"
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

	fileContents, _ := readFile(defaultFilePath, key)

	/**
	 * If fileContents is not empty, append the new SSH connection details
	 * to the existing file contents. Otherwise, set the new SSH connection
	 * details as the file contents.
	 */
	if fileContents != "" {
		fileContents = fileContents + "\n" + sshConnection
	} else {
		fileContents = sshConnection
	}

	err = storeFile(fileContents, defaultFilePath, key)
	if err != nil {
		return err
	}

	fmt.Println("SSH connection details added successfully!")
	return nil
}

func findKeyOfSelectedSSHOption(choice string, conns map[string]map[string]string) (error, string) {
	index := strings.Split(choice, ".")[0]

	for key, value := range conns {
		if value["index"] == index {
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
	case "Back to main menu":
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

// Remove SSH connection
func removeSSHConnection(defaultFilePath, keyFilePath string) {
	connections, err := readAllConnections(defaultFilePath, keyFilePath)
	if err != nil {
		fmt.Println("No SSH connections found.")
		return
	}

	items := connToStrSlice(connections)

	prompt := promptui.Select{
		Label: "Select an SSH connection to remove",
		Items: items,
	}

	_, result, err := prompt.Run()
	if err != nil {
		fmt.Println("Invalid connection selection. Please try again.")
		return
	}

	switch result {
	case "Back to main menu":
		return
	default:
		err, index := findKeyOfSelectedSSHOption(result, connections)
		if err != nil {
			return
		}

		delete(connections, index)

		var newConns []string
		for _, value := range connections {
			newConns = append(newConns, value["username"]+"@"+value["host"]+"\t"+value["password"])
		}

		key, err := loadKey(keyFilePath)
		if err != nil {
			fmt.Println("Failed to remove SSH connection.")
			return
		}

		err = storeFile(strings.Join(newConns, "\n"), defaultFilePath, key)
		if err != nil {
			fmt.Println("Failed to remove SSH connection.")
			return
		}

		fmt.Println("SSH connection removed successfully.")
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

	cleanFlag := flag.Bool("clean", false, "Resets the connections and key file")
	flag.Parse()

	if *cleanFlag {
		os.Remove(defaultFilePath)
		os.Remove(keyFilePath)
		fmt.Println("Connections and key file have been reset.")
		return
	}

	menuOptions := []string{"Exit", "Connect to SSH", "Add SSH Connection", "Remove SSH Connection"}

	for {
		prompt := promptui.Select{
			Label: "Menu Options | v0.1.0",
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
		case "Remove SSH Connection":
			removeSSHConnection(defaultFilePath, keyFilePath)
		}
	}
}
