package config

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emirhangumus/sshmanager/v2/internal/storage"
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

func TestLegacyCredentialDisplayIsIgnoredAndRemoved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	legacy := []byte("behaviour:\n  continueAfterSSHExit: true\n  showCredentialsOnConnect: true\n")
	if err := storage.WriteFileAtomic(path, legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil || !cfg.Behaviour.ContinueAfterSSHExit {
		t.Fatalf("legacy config rejected: %v", err)
	}
	if err := SetConfig(path, "behaviour.showCredentialsOnConnect", "true"); err == nil {
		t.Fatal("credential display setting still accepted")
	}
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}
	var raw strings.Builder
	data, err := storage.ReadFileRegular(path)
	if err != nil {
		t.Fatal(err)
	}
	raw.Write(data)
	if strings.Contains(raw.String(), "showCredentialsOnConnect") {
		t.Fatal("removed setting persisted")
	}
	var restored SSHManagerConfig
	if err := json.Unmarshal([]byte(`{"behaviour":{"showCredentialsOnConnect":true,"continueAfterSSHExit":true}}`), &restored); err != nil {
		t.Fatal(err)
	}
	if !restored.Behaviour.ContinueAfterSSHExit {
		t.Fatal("legacy JSON settings lost")
	}
}
