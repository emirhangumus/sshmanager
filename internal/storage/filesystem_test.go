package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileAtomicCreatesDirectoriesAndWritesData(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "nested", "state", "config.yaml")
	payload := []byte("hello: world\n")

	if err := WriteFileAtomic(filePath, payload, 0o600); err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("file content mismatch: got %q, want %q", string(got), string(payload))
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("failed to stat output file: %v", err)
	}
	if perms := info.Mode().Perm(); perms != 0o600 {
		t.Fatalf("unexpected file mode: got %o, want %o", perms, 0o600)
	}
}

func TestWriteFileAtomicReplacesExistingFileWithoutTmpLeak(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "conn")
	prefix := "." + filepath.Base(filePath) + ".tmp-"

	if err := os.WriteFile(filePath, []byte("old-content"), 0o600); err != nil {
		t.Fatalf("failed to seed file: %v", err)
	}

	if err := WriteFileAtomic(filePath, []byte("new-content"), 0o600); err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(got) != "new-content" {
		t.Fatalf("file content mismatch: got %q, want %q", string(got), "new-content")
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to list directory: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) {
			t.Fatalf("temporary file leak detected: %s", entry.Name())
		}
	}
}
