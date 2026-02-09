package store

import (
	"fmt"
	"os"

	cryptoutil "github.com/emirhangumus/sshmanager/internal/crypto"
	"github.com/emirhangumus/sshmanager/internal/storage"
)

func encryptAndStoreFile(data, filePath string, key []byte) error {
	encryptedData, err := cryptoutil.EncryptData(data, key)
	if err != nil {
		return err
	}

	if err := storage.WriteFileAtomic(filePath, encryptedData, 0o600); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}
	return nil
}

func decryptAndReadFile(filePath string, key []byte) (string, error) {
	encryptedData, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	if len(encryptedData) == 0 {
		return "", nil
	}

	content, err := cryptoutil.DecryptData(encryptedData, key)
	if err != nil {
		return "", err
	}
	return content, nil
}
