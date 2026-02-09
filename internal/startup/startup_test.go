package startup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emirhangumus/sshmanager/internal/config"
	"github.com/emirhangumus/sshmanager/internal/store"
)

func TestSetupCreatesInitialState(t *testing.T) {
	tmpDir := t.TempDir()
	connPath := filepath.Join(tmpDir, ".sshmanager", "conn")
	configPath := filepath.Join(tmpDir, ".sshmanager", "config.yaml")
	keyPath := filepath.Join(tmpDir, ".sshmanager", "secret.key")

	if err := Setup(connPath, configPath, keyPath); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	assertFileExists(t, connPath)
	assertFileExists(t, configPath)
	assertFileExists(t, keyPath)

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg != config.Default() {
		t.Fatalf("expected default config, got %+v", cfg)
	}

	connStore := store.NewConnectionStore(connPath, keyPath)
	connFile, err := connStore.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if strings.TrimSpace(connFile.Version) == "" {
		t.Fatal("expected non-empty connection schema version")
	}
	if len(connFile.Connections) != 0 {
		t.Fatalf("expected no initial connections, got %d", len(connFile.Connections))
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("failed to read key file: %v", err)
	}
	if len(keyData) != 32 {
		t.Fatalf("expected 32-byte key, got %d bytes", len(keyData))
	}
}

func TestSetupDoesNotOverrideExistingConfig(t *testing.T) {
	tmpDir := t.TempDir()
	connPath := filepath.Join(tmpDir, ".sshmanager", "conn")
	configPath := filepath.Join(tmpDir, ".sshmanager", "config.yaml")
	keyPath := filepath.Join(tmpDir, ".sshmanager", "secret.key")

	custom := config.Default()
	custom.Behaviour.ContinueAfterSSHExit = true
	custom.Behaviour.ShowCredentialsOnConnect = true
	if err := config.SaveConfig(configPath, custom); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	if err := Setup(connPath, configPath, keyPath); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	got, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if got != custom {
		t.Fatalf("expected custom config to remain unchanged; got %+v want %+v", got, custom)
	}
}

func TestSetupFailsForInvalidExistingKeySize(t *testing.T) {
	tmpDir := t.TempDir()
	connPath := filepath.Join(tmpDir, ".sshmanager", "conn")
	configPath := filepath.Join(tmpDir, ".sshmanager", "config.yaml")
	keyPath := filepath.Join(tmpDir, ".sshmanager", "secret.key")

	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte("short"), 0o600); err != nil {
		t.Fatalf("failed to write invalid key fixture: %v", err)
	}

	err := Setup(connPath, configPath, keyPath)
	if err == nil {
		t.Fatal("expected Setup to fail for invalid key size, got nil")
	}
	if !strings.Contains(err.Error(), "invalid key") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertFileExists(t *testing.T, filePath string) {
	t.Helper()

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("expected file %s to exist: %v", filePath, err)
	}
	if info.IsDir() {
		t.Fatalf("expected file %s, got directory", filePath)
	}
}
