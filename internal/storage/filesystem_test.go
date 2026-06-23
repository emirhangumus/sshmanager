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

func TestCreateFileIfNotExistsCreatesFileWithMode(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "nested", "secret.key")

	if err := CreateFileIfNotExists(filePath, 0o600); err != nil {
		t.Fatalf("CreateFileIfNotExists failed: %v", err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("failed to stat created file: %v", err)
	}
	if perms := info.Mode().Perm(); perms != 0o600 {
		t.Fatalf("unexpected file mode: got %o, want %o", perms, 0o600)
	}

	dirInfo, err := os.Stat(filepath.Dir(filePath))
	if err != nil {
		t.Fatalf("failed to stat created directory: %v", err)
	}
	if perms := dirInfo.Mode().Perm(); perms != 0o700 {
		t.Fatalf("unexpected directory mode: got %o, want %o", perms, 0o700)
	}
}

func TestCreateFileIfNotExistsIsIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "secret.key")

	if err := os.WriteFile(filePath, []byte("existing-content"), 0o600); err != nil {
		t.Fatalf("failed to seed file: %v", err)
	}

	if err := CreateFileIfNotExists(filePath, 0o600); err != nil {
		t.Fatalf("CreateFileIfNotExists failed: %v", err)
	}

	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(got) != "existing-content" {
		t.Fatalf("CreateFileIfNotExists overwrote existing content: %q", string(got))
	}
}

func TestIsFileEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	missingPath := filepath.Join(tmpDir, "missing")
	empty, err := IsFileEmpty(missingPath)
	if err != nil {
		t.Fatalf("IsFileEmpty failed for missing file: %v", err)
	}
	if !empty {
		t.Fatal("expected missing file to be reported as empty")
	}

	emptyPath := filepath.Join(tmpDir, "empty")
	if err := os.WriteFile(emptyPath, nil, 0o600); err != nil {
		t.Fatalf("failed to create empty file: %v", err)
	}
	empty, err = IsFileEmpty(emptyPath)
	if err != nil {
		t.Fatalf("IsFileEmpty failed for empty file: %v", err)
	}
	if !empty {
		t.Fatal("expected empty file to be reported as empty")
	}

	nonEmptyPath := filepath.Join(tmpDir, "nonempty")
	if err := os.WriteFile(nonEmptyPath, []byte("data"), 0o600); err != nil {
		t.Fatalf("failed to create non-empty file: %v", err)
	}
	empty, err = IsFileEmpty(nonEmptyPath)
	if err != nil {
		t.Fatalf("IsFileEmpty failed for non-empty file: %v", err)
	}
	if empty {
		t.Fatal("expected non-empty file to be reported as non-empty")
	}
}

func TestSecureDeleteOverwritesAndRemovesFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "secret.key")
	original := []byte("super-secret-key-material")

	if err := os.WriteFile(filePath, original, 0o600); err != nil {
		t.Fatalf("failed to seed file: %v", err)
	}

	if err := SecureDelete(filePath); err != nil {
		t.Fatalf("SecureDelete failed: %v", err)
	}

	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected file to be removed, stat err: %v", err)
	}
}

func TestSecureDeleteOnMissingFileIsNoop(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "does-not-exist")

	if err := SecureDelete(filePath); err != nil {
		t.Fatalf("SecureDelete on missing file should be a no-op, got: %v", err)
	}
}

func TestWriteYAMLFileAndReadYAMLFileRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "config.yaml")

	type sample struct {
		Name  string `yaml:"name"`
		Count int    `yaml:"count"`
	}

	want := sample{Name: "sshmanager", Count: 3}
	if err := WriteYAMLFile(filePath, want, 0o600); err != nil {
		t.Fatalf("WriteYAMLFile failed: %v", err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("failed to stat written file: %v", err)
	}
	if perms := info.Mode().Perm(); perms != 0o600 {
		t.Fatalf("unexpected file mode: got %o, want %o", perms, 0o600)
	}

	var got sample
	if err := ReadYAMLFile(filePath, &got); err != nil {
		t.Fatalf("ReadYAMLFile failed: %v", err)
	}
	if got != want {
		t.Fatalf("round trip mismatch: got %+v, want %+v", got, want)
	}
}

func TestReadYAMLFileMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "missing.yaml")

	var out map[string]string
	if err := ReadYAMLFile(filePath, &out); err == nil {
		t.Fatal("expected error reading missing YAML file, got nil")
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
