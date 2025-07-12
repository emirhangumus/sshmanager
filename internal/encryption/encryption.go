package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"github.com/emirhangumus/sshmanager/internal/prompt"
)

// Generate a 32-byte AES key
func generateKey() ([]byte, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// LoadKey Load or create an encryption key
func LoadKey(filePath string) ([]byte, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		key, err := generateKey()
		if err != nil {
			return nil, err
		}
		os.MkdirAll(filepath.Dir(filePath), 0700) // Secure directory
		err = os.WriteFile(filePath, key, 0600)   // Restrict access
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

// EncryptData Encrypt data using AES-GCM
func EncryptData(data string, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, 12) // AES-GCM requires a 12-byte nonce
	_, err = rand.Read(nonce)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ciphertext := aesgcm.Seal(nil, nonce, []byte(data), nil)
	return append(nonce, ciphertext...), nil // Prepend nonce for decryption
}

// DecryptData Decrypt data using AES-GCM
func DecryptData(encryptedData, key []byte) (string, error) {
	if len(encryptedData) < 12 {
		return "", fmt.Errorf(prompt.DefaultPromptTexts.ErrorMessages.InvalidDataFormatX, "encrypted data too short")
	}
	nonce, ciphertext := encryptedData[:12], encryptedData[12:]
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf(prompt.DefaultPromptTexts.ErrorMessages.DecryptionDataFailedX, err)
	}
	return string(plaintext), nil
}
