package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"golang.org/x/crypto/nacl/secretbox"
)

// Generate a key for encryption and decryption
func generateKey() ([]byte, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// Load an existing key or create a new one
func loadKey(filePath string) ([]byte, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		key, err := generateKey()
		if err != nil {
			return nil, err
		}
		os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
		err = os.WriteFile(filePath, key, 0600)
		if err != nil {
			return nil, err
		}
		return key, nil
	}
	key, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// Encrypt data
func encryptData(data string, key []byte) ([]byte, error) {
	var nonce [24]byte
	_, err := rand.Read(nonce[:])
	if err != nil {
		return nil, err
	}

	var secretKey [32]byte
	copy(secretKey[:], key[:32])

	encrypted := secretbox.Seal(nonce[:], []byte(data), &nonce, &secretKey)
	return encrypted, nil
}

// Decrypt data
func decryptData(encryptedData, key []byte) (string, error) {
	var nonce [24]byte
	copy(nonce[:], encryptedData[:24])

	var secretKey [32]byte
	copy(secretKey[:], key[:32])

	decrypted, ok := secretbox.Open(nil, encryptedData[24:], &nonce, &secretKey)
	if !ok {
		return "", fmt.Errorf("decryption failed")
	}
	return string(decrypted), nil
}

// Store encrypted data to a file
func storeFile(data, filePath string, key []byte) error {
	encryptedData, err := encryptData(data, key)
	if err != nil {
		return err
	}

	os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	return os.WriteFile(filePath, encryptedData, 0600)
}

// Read encrypted data from a file
func readFile(filePath string, key []byte) (string, error) {
	encryptedData, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return decryptData(encryptedData, key)
}

// Read all SSH connections from file
func readAllConnections(defaultFilePath, keyFilePath string) (map[string]map[string]string, error) {
	key, err := loadKey(keyFilePath)
	if err != nil {
		return nil, err
	}

	connections, err := readFile(defaultFilePath, key)
	if err != nil {
		return nil, err
	}

	conndict := make(map[string]map[string]string)
	connLines := strings.Split(connections, "\n")
	for i, line := range connLines {
		if line != "" {
			parts := strings.Split(line, "\t")
			userHost := strings.Split(parts[0], "@")
			conndict[fmt.Sprintf("%d", i+1)] = map[string]string{
				"username": userHost[0],
				"host":     userHost[1],
				"password": parts[1],
			}
		}
	}

	return conndict, nil
}

// Connect to SSH
func connectSSH(conn map[string]string) {
	fmt.Println("Connecting to SSH...")
	command := exec.Command("sshpass", "-p", conn["password"], "ssh", fmt.Sprintf("%s@%s", conn["username"], conn["host"]))
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin
	if err := command.Run(); err != nil {
		log.Fatal(err)
	}
}

// Prompt user to add SSH connection
func addSSHConnection(defaultFilePath, keyFilePath string) error {
	prompt := promptui.Prompt{
		Label: "Enter Host",
	}
	host, err := prompt.Run()
	if err != nil {
		return err
	}

	prompt = promptui.Prompt{
		Label: "Enter Username",
	}
	username, err := prompt.Run()
	if err != nil {
		return err
	}

	prompt = promptui.Prompt{
		Label: "Enter Password",
		Mask:  '*',
	}
	password, err := prompt.Run()
	if err != nil {
		return err
	}

	sshConnection := fmt.Sprintf("%s@%s\t%s\n", username, host, password)
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
