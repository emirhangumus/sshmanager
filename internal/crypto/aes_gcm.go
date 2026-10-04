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

	"github.com/emirhangumus/sshmanager/v2/internal/storage"
)

const (
	keySize        = 32
	nonceSize      = 12
	passphraseSalt = 16
)

const (
	passphraseEnvVar       = "SSHMANAGER_MASTER_PASSPHRASE" //nolint:gosec // env var name, not a credential value
	passphraseKeyFileMode  = "passphrase"
	passphraseKeyFileKDF   = "pbkdf2-sha256" //nolint:gosec // KDF identifier string, not a credential value
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

	return LoadExistingKey(filePath)
}

// LoadExistingKey never creates a key. Existing ciphertext must not receive a replacement key.
func LoadExistingKey(filePath string) ([]byte, error) {
	key, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}
	return DecodeKeyFile(key)
}

// DecodeKeyFile resolves raw keys or passphrase metadata without writing files.
func DecodeKeyFile(data []byte) ([]byte, error) {
	if len(data) == keySize {
		return data, nil
	}
	return loadPassphraseKey(data)
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
		if err := storage.WriteFileAtomic(filePath, key, 0o600); err != nil {
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
	if err := storage.WriteFileAtomic(filePath, data, 0o600); err != nil {
		return nil, fmt.Errorf("failed to write key file: %w", err)
	}
	return key, nil
}

func loadPassphraseKey(data []byte) ([]byte, error) {
	if err := ValidatePassphraseMetadata(data); err != nil {
		return nil, err
	}
	var meta passphraseKeyFile
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("invalid key file format: expected %d raw bytes or passphrase metadata", keySize)
	}

	passphrase := strings.TrimSpace(os.Getenv(passphraseEnvVar))
	if passphrase == "" {
		return nil, fmt.Errorf("passphrase key file detected; set %s", passphraseEnvVar)
	}

	salt, err := base64.StdEncoding.DecodeString(meta.Salt)
	if err != nil {
		return nil, fmt.Errorf("invalid key file salt: %w", err)
	}

	key, err := derivePassphraseKey(passphrase, salt, meta.Iterations)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// ValidatePassphraseMetadata validates nonsecret metadata without needing a passphrase.
func ValidatePassphraseMetadata(data []byte) error {
	var meta passphraseKeyFile
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("invalid key file format: %w", err)
	}
	if meta.Mode != passphraseKeyFileMode || meta.Version != passphraseKeyFileV1 || meta.KDF != passphraseKeyFileKDF || meta.Iterations <= 0 {
		return fmt.Errorf("invalid passphrase KDF metadata")
	}
	salt, err := base64.StdEncoding.DecodeString(meta.Salt)
	if err != nil || len(salt) != passphraseSalt {
		return fmt.Errorf("invalid passphrase salt")
	}
	return nil
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
