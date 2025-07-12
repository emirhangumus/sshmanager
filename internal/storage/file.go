package storage

import (
	"crypto/rand"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"

	"github.com/emirhangumus/sshmanager/internal/encryption"
)

// EncryptAndStoreFile encrypts and stores the given data at filePath.
func EncryptAndStoreFile(data, filePath string, key []byte) error {
	encryptedData, err := encryption.EncryptData(data, key)
	if err != nil {
		return err
	}

	os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	return os.WriteFile(filePath, encryptedData, 0600)
}

// DecryptAndReadFile decrypts and returns the content of the given file.
func DecryptAndReadFile(filePath string, key []byte) (string, error) {
	encryptedData, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return encryption.DecryptData(encryptedData, key)
}

// StoreFile stores the given data at filePath without encryption.
func StoreFile(data, filePath string) error {
	os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	return os.WriteFile(filePath, []byte(data), 0600)
}

// ReadFile reads and returns the content of the given file without decryption.
func ReadFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SecureDelete Securely delete a file by overwriting with random data before removal
func SecureDelete(path string) {
	f, err := os.OpenFile(path, os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()

	info, err := os.Stat(path)
	if err == nil {
		size := info.Size()
		randomData := make([]byte, size)
		_, _ = rand.Read(randomData)
		f.Write(randomData) // Overwrite with random data
	}
	f.Close()
	os.Remove(path)
}

// ReadYAMLFile reads a YAML file and unmarshals it into the provided struct.
func ReadYAMLFile(filePath string, out interface{}) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	err = yaml.Unmarshal(data, out)
	if err != nil {
		return fmt.Errorf("failed to unmarshal YAML from file %s: %w", filePath, err)
	}
	return nil
}

// WriteYAMLFile marshals the provided struct into YAML and writes it to the specified file.
func WriteYAMLFile(filePath string, data interface{}) error {
	dataBytes, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data to YAML: %w", err)
	}

	err = os.WriteFile(filePath, dataBytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}
	return nil
}

func CreateFileIfNotExists(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", filePath, err)
		}
		defer file.Close()
	}
	return nil
}

// IsFileEmpty checks if the specified file is empty.
func IsFileEmpty(filePath string) (bool, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil // File does not exist, consider it empty
		}
		return false, fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}
	return fileInfo.Size() == 0, nil
}
