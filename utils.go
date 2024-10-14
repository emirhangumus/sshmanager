package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
