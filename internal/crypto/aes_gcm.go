package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	keySize        = 32
	nonceSize      = 12
	passphraseSalt = 16
)

const (
	passphraseEnvVar       = "SSHMANAGER_MASTER_PASSPHRASE"
	passphraseKeyFileMode  = "passphrase"
	passphraseKeyFileKDF   = "pbkdf2-sha256"
	passphraseKeyFileV1    = 1
	passphraseIterationsV1 = 600_000
)

func generateKey() ([]byte, error) {
	key := make([]byte, keySize)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

type passphraseKeyFile struct {
	Version    int    `json:"version"`
	Mode       string `json:"mode"`
	KDF        string `json:"kdf"`
	Iterations int    `json:"iterations"`
	Salt       string `json:"salt"`
}

// LoadKey loads an existing AES-256 key, or creates one when missing.
func LoadKey(filePath string) ([]byte, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return createKeyFile(filePath)
	}

	key, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}
	if len(key) != keySize {
		return loadPassphraseKey(key)
	}
	return key, nil
}

func createKeyFile(filePath string) ([]byte, error) {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o700); err != nil {
		return nil, fmt.Errorf("failed to create key directory: %w", err)
	}

	passphrase := strings.TrimSpace(os.Getenv(passphraseEnvVar))
	if passphrase == "" {
		key, err := generateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate key: %w", err)
		}
		if err := os.WriteFile(filePath, key, 0o600); err != nil {
			return nil, fmt.Errorf("failed to write key file: %w", err)
		}
		return key, nil
	}

	return createPassphraseKeyFile(filePath, passphrase)
}

func createPassphraseKeyFile(filePath, passphrase string) ([]byte, error) {
	salt := make([]byte, passphraseSalt)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate passphrase salt: %w", err)
	}

	key, err := derivePassphraseKey(passphrase, salt, passphraseIterationsV1)
	if err != nil {
		return nil, err
	}

	meta := passphraseKeyFile{
		Version:    passphraseKeyFileV1,
		Mode:       passphraseKeyFileMode,
		KDF:        passphraseKeyFileKDF,
		Iterations: passphraseIterationsV1,
		Salt:       base64.StdEncoding.EncodeToString(salt),
	}
	data, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("failed to encode passphrase key metadata: %w", err)
	}
	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		return nil, fmt.Errorf("failed to write key file: %w", err)
	}
	return key, nil
}

func loadPassphraseKey(data []byte) ([]byte, error) {
	var meta passphraseKeyFile
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("invalid key file format: expected %d raw bytes or passphrase metadata", keySize)
	}

	if meta.Mode != passphraseKeyFileMode {
		return nil, fmt.Errorf("unsupported key file mode: %q", meta.Mode)
	}
	if meta.Version != passphraseKeyFileV1 {
		return nil, fmt.Errorf("unsupported key file version: %d", meta.Version)
	}

	passphrase := strings.TrimSpace(os.Getenv(passphraseEnvVar))
	if passphrase == "" {
		return nil, fmt.Errorf("passphrase key file detected; set %s", passphraseEnvVar)
	}

	salt, err := base64.StdEncoding.DecodeString(meta.Salt)
	if err != nil {
		return nil, fmt.Errorf("invalid key file salt: %w", err)
	}

	iterations := meta.Iterations
	if iterations <= 0 {
		iterations = passphraseIterationsV1
	}

	key, err := derivePassphraseKey(passphrase, salt, iterations)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func derivePassphraseKey(passphrase string, salt []byte, iterations int) ([]byte, error) {
	key, err := pbkdf2.Key(sha256.New, passphrase, salt, iterations, keySize)
	if err != nil {
		return nil, fmt.Errorf("failed to derive passphrase key: %w", err)
	}
	if len(key) != keySize {
		return nil, fmt.Errorf("invalid derived key size: got %d, want %d", len(key), keySize)
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
