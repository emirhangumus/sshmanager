package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
)

const (
	keySize   = 32
	nonceSize = 12
)

func generateKey() ([]byte, error) {
	key := make([]byte, keySize)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

// LoadKey loads an existing AES-256 key, or creates one when missing.
func LoadKey(filePath string) ([]byte, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		key, err := generateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate key: %w", err)
		}

		if err := os.MkdirAll(filepath.Dir(filePath), 0o700); err != nil {
			return nil, fmt.Errorf("failed to create key directory: %w", err)
		}
		if err := os.WriteFile(filePath, key, 0o600); err != nil {
			return nil, fmt.Errorf("failed to write key file: %w", err)
		}
		return key, nil
	}

	key, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}
	if len(key) != keySize {
		return nil, fmt.Errorf("invalid key size: got %d, want %d", len(key), keySize)
	}
	return key, nil
}

// EncryptData encrypts plain text with AES-GCM and prefixes nonce bytes.
func EncryptData(data string, key []byte) ([]byte, error) {
	if len(key) != keySize {
		return nil, fmt.Errorf("invalid key size: got %d, want %d", len(key), keySize)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	ciphertext := aead.Seal(nil, nonce, []byte(data), nil)
	return append(nonce, ciphertext...), nil
}

// DecryptData decrypts AES-GCM payloads where nonce is prefixed.
func DecryptData(encryptedData, key []byte) (string, error) {
	if len(key) != keySize {
		return "", fmt.Errorf("invalid key size: got %d, want %d", len(key), keySize)
	}
	if len(encryptedData) < nonceSize {
		return "", fmt.Errorf("invalid data format: encrypted payload too short")
	}

	nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt data: %w", err)
	}
	return string(plaintext), nil
}
