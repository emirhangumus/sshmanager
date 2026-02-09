package cryptoutil

import (
	"bytes"
	"os"
	"path/filepath"
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
