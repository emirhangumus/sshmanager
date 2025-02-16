package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
				"index":       fmt.Sprintf("%d", i+1),
				"username":    userHost[0],
				"host":        userHost[1],
				"password":    parts[1],
				"description": parts[2],
			}
		}
	}

	return conndict, nil
}

func connToStrSlice(conns map[string]map[string]string) []string {
	items := []string{"Back to main menu"}

	for _, value := range conns {
		items = append(items, value["index"]+". "+value["username"]+"@"+value["host"]+" - "+value["description"])
	}

	return items
}
