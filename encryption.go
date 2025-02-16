package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
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

// Load or create an encryption key
func loadKey(filePath string) ([]byte, error) {
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

// Encrypt data using AES-GCM
func encryptData(data string, key []byte) ([]byte, error) {
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

// Decrypt data using AES-GCM
func decryptData(encryptedData, key []byte) (string, error) {
	if len(encryptedData) < 12 {
		return "", fmt.Errorf("invalid data format")
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
		return "", fmt.Errorf("decryption failed: %v", err)
	}
	return string(plaintext), nil
}

// Securely delete a file by overwriting with random data before removal
func secureDelete(path string) {
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
