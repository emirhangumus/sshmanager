package cryptoutil

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
)

// Backup v1: magic (8), version/cipher/KDF/reserved (4), salt (16),
// nonce (12), ciphertext and GCM tag. The entire header is authenticated.
// Cipher 1 = AES-256-GCM; KDF 1 = PBKDF2-SHA256, 600,000 iterations.
const backupMagic = "SSHMBKUP"
const backupHeaderSize = 40
const MaxBackupSize = 64 * 1024 * 1024

func IsEncryptedBackup(data []byte) bool { return bytes.HasPrefix(data, []byte(backupMagic)) }

func backupAEAD(passphrase string, salt []byte) (cipher.AEAD, error) {
	if passphrase == "" {
		return nil, errors.New("backup passphrase must not be empty")
	}
	key, err := pbkdf2.Key(sha256.New, passphrase, salt, 600_000, 32)
	if err != nil {
		return nil, err
	}
	defer clear(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func EncryptBackup(plaintext []byte, passphrase string) ([]byte, error) {
	if len(plaintext) > MaxBackupSize-backupHeaderSize-16 {
		return nil, errors.New("backup exceeds size limit")
	}
	header := make([]byte, backupHeaderSize)
	copy(header, backupMagic)
	header[8], header[9], header[10] = 1, 1, 1
	if _, err := rand.Read(header[12:]); err != nil {
		return nil, err
	}
	aead, err := backupAEAD(passphrase, header[12:28])
	if err != nil {
		return nil, err
	}
	return aead.Seal(header, header[28:40], plaintext, header), nil
}

func DecryptBackup(data []byte, passphrase string) ([]byte, error) {
	if len(data) < backupHeaderSize+16 || len(data) > MaxBackupSize || !IsEncryptedBackup(data) {
		return nil, errors.New("invalid encrypted backup")
	}
	if !bytes.Equal(data[8:12], []byte{1, 1, 1, 0}) {
		return nil, errors.New("unsupported backup version, cipher, or KDF")
	}
	aead, err := backupAEAD(passphrase, data[12:28])
	if err != nil {
		return nil, err
	}
	plaintext, err := aead.Open(nil, data[28:40], data[40:], data[:40])
	if err != nil {
		return nil, fmt.Errorf("backup authentication failed (wrong passphrase or damaged backup): %w", err)
	}
	return plaintext, nil
}
