package config

import (
	"path/filepath"
	"testing"

	"github.com/emirhangumus/sshmanager/internal/storage"
)

func TestSetConfigAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	if err := storage.CreateFileIfNotExists(configPath, 0o600); err != nil {
		t.Fatalf("CreateFileIfNotExists failed: %v", err)
	}

	if err := SetConfig(configPath, "behaviour.continueAfterSSHExit", "true"); err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if !cfg.Behaviour.ContinueAfterSSHExit {
		t.Fatal("expected behaviour.continueAfterSSHExit=true")
	}
}

func TestSetConfigRejectsUnknownKey(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	if err := storage.CreateFileIfNotExists(configPath, 0o600); err != nil {
		t.Fatalf("CreateFileIfNotExists failed: %v", err)
	}

	if err := SetConfig(configPath, "unknown.key", "true"); err == nil {
		t.Fatal("expected error for unknown key")
	}
}
