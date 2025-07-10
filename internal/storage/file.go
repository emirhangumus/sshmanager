package storage

import (
	"os"
	"path/filepath"

	"github.com/emirhangumus/sshmanager/internal/encryption"
)

// StoreFile encrypts and stores the given data at filePath.
func StoreFile(data, filePath string, key []byte) error {
	encryptedData, err := encryption.EncryptData(data, key)
	if err != nil {
		return err
	}

	os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	return os.WriteFile(filePath, encryptedData, 0600)
}

// ReadFile decrypts and returns the content of the given file.
func ReadFile(filePath string, key []byte) (string, error) {
	encryptedData, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return encryption.DecryptData(encryptedData, key)
}
