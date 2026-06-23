package cryptoutil

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte{1}, 32)
	plain := "sshmanager-test-payload"

	encrypted, err := EncryptData(plain, key)
	if err != nil {
		t.Fatalf("EncryptData returned error: %v", err)
	}

	decrypted, err := DecryptData(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptData returned error: %v", err)
	}

	if decrypted != plain {
		t.Fatalf("round trip mismatch: got %q, want %q", decrypted, plain)
	}
}

func TestDecryptDataRejectsShortPayload(t *testing.T) {
	key := bytes.Repeat([]byte{1}, 32)
	if _, err := DecryptData([]byte("short"), key); err == nil {
		t.Fatal("expected error for short payload, got nil")
	}
}

func TestLoadKeyRejectsInvalidSize(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "secret.key")
	if err := os.WriteFile(keyPath, []byte("too-short"), 0o600); err != nil {
		t.Fatalf("failed to write key fixture: %v", err)
	}

	if _, err := LoadKey(keyPath); err == nil {
		t.Fatal("expected key-size validation error, got nil")
	}
}

func TestLoadKeyPassphraseModeRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "secret.key")

	t.Setenv(passphraseEnvVar, "correct horse battery staple")
	key1, err := LoadKey(keyPath)
	if err != nil {
		t.Fatalf("LoadKey(create passphrase mode) failed: %v", err)
	}
	if len(key1) != keySize {
		t.Fatalf("unexpected key length: got %d, want %d", len(key1), keySize)
	}

	raw, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("failed reading key metadata file: %v", err)
	}
	if len(raw) == keySize {
		t.Fatal("expected passphrase metadata file, got legacy raw-key layout")
	}

	var meta passphraseKeyFile
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("invalid passphrase metadata JSON: %v", err)
	}
	if meta.Mode != passphraseKeyFileMode {
		t.Fatalf("unexpected mode: %q", meta.Mode)
	}
	if meta.Iterations <= 0 {
		t.Fatalf("invalid iterations: %d", meta.Iterations)
	}
	if strings.TrimSpace(meta.Salt) == "" {
		t.Fatal("expected non-empty salt in metadata")
	}

	key2, err := LoadKey(keyPath)
	if err != nil {
		t.Fatalf("LoadKey(read passphrase mode) failed: %v", err)
	}
	if !bytes.Equal(key1, key2) {
		t.Fatal("derived keys mismatch across loads")
	}
}

func TestLoadKeyPassphraseModeRequiresEnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "secret.key")

	t.Setenv(passphraseEnvVar, "super-secret")
	if _, err := LoadKey(keyPath); err != nil {
		t.Fatalf("LoadKey(create passphrase mode) failed: %v", err)
	}

	t.Setenv(passphraseEnvVar, "")
	_, err := LoadKey(keyPath)
	if err == nil {
		t.Fatal("expected error when passphrase env var is not set")
	}
	if !strings.Contains(err.Error(), passphraseEnvVar) {
		t.Fatalf("expected error to reference %s, got %v", passphraseEnvVar, err)
	}
}
