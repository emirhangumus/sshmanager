package store

import (
	"path/filepath"
	"strings"
	"testing"

	cryptoutil "github.com/emirhangumus/sshmanager/internal/crypto"
	"github.com/emirhangumus/sshmanager/internal/storage"
)

func TestLoadMigratesLegacyListAndAddsIDs(t *testing.T) {
	tmpDir := t.TempDir()
	connPath := filepath.Join(tmpDir, "conn")
	keyPath := filepath.Join(tmpDir, "secret.key")

	if err := storage.CreateFileIfNotExists(connPath, 0o600); err != nil {
		t.Fatalf("CreateFileIfNotExists(conn) failed: %v", err)
	}

	key, err := cryptoutil.LoadKey(keyPath)
	if err != nil {
		t.Fatalf("LoadKey failed: %v", err)
	}

	legacy := "- username: u\n  host: h\n  password: p\n"
	if err := encryptAndStoreFile(legacy, connPath, key); err != nil {
		t.Fatalf("encryptAndStoreFile failed: %v", err)
	}

	connStore := NewConnectionStore(connPath, keyPath)
	loaded, err := connStore.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Connections) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(loaded.Connections))
	}
	if loaded.Connections[0].ID == "" {
		t.Fatal("expected generated stable ID on migrated legacy data")
	}
	if strings.TrimSpace(loaded.Version) == "" {
		t.Fatal("expected non-empty schema version")
	}

	content, err := decryptAndReadFile(connPath, key)
	if err != nil {
		t.Fatalf("decryptAndReadFile failed: %v", err)
	}
	if !strings.Contains(content, "version:") {
		t.Fatal("expected migrated schema to include version field")
	}
}
