package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emirhangumus/sshmanager/internal/config"
	"github.com/emirhangumus/sshmanager/internal/model"
	"github.com/emirhangumus/sshmanager/internal/store"
	"gopkg.in/yaml.v3"
)

func TestHandleBackupJSONWithoutConfig(t *testing.T) {
	connPath, keyPath := prepareTransferFixture(t, []model.SSHConnection{
		{
			Username: "ubuntu",
			Host:     "app.internal",
			AuthMode: model.AuthModePassword,
			Password: "secret",
			Alias:    "prod",
		},
	})

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := config.Default()
	cfg.Behaviour.ContinueAfterSSHExit = true
	if err := config.SaveConfig(cfgPath, cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	backupPath := filepath.Join(t.TempDir(), "backup.json")
	var out strings.Builder
	if err := handleBackup(connPath, keyPath, cfgPath, []string{
		"--out", backupPath,
		"--format", "json",
		"--include-config=false",
	}, &out); err != nil {
		t.Fatalf("handleBackup failed: %v", err)
	}

	raw, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("failed to read backup file: %v", err)
	}

	var snapshot backupSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatalf("failed to decode backup JSON: %v", err)
	}
	if snapshot.Config != nil {
		t.Fatal("expected config to be excluded from backup")
	}
	if len(snapshot.ConnectionFile.Connections) != 1 {
		t.Fatalf("expected 1 backup connection, got %d", len(snapshot.ConnectionFile.Connections))
	}
	if !strings.Contains(out.String(), "(json)") {
		t.Fatalf("unexpected backup output: %q", out.String())
	}
}

func TestHandleRestoreReplaceRestoresConnectionsAndConfig(t *testing.T) {
	connPath, keyPath := prepareTransferFixture(t, []model.SSHConnection{
		{
			Username: "ubuntu",
			Host:     "before.internal",
			AuthMode: model.AuthModePassword,
			Password: "before-pass",
			Alias:    "before",
		},
	})

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	backupCfg := config.Default()
	backupCfg.Behaviour.ContinueAfterSSHExit = true
	backupCfg.Behaviour.ShowCredentialsOnConnect = true
	if err := config.SaveConfig(cfgPath, backupCfg); err != nil {
		t.Fatalf("SaveConfig(backupCfg) failed: %v", err)
	}

	backupPath := filepath.Join(t.TempDir(), "snapshot.yaml")
	if err := handleBackup(connPath, keyPath, cfgPath, []string{"--out", backupPath}, ioDiscard()); err != nil {
		t.Fatalf("handleBackup failed: %v", err)
	}

	connStore := store.NewConnectionStore(connPath, keyPath)
	if err := connStore.Update(func(connFile *model.ConnectionFile) error {
		*connFile = model.NewConnectionFile()
		return connFile.AddConnection(model.SSHConnection{
			Username: "ubuntu",
			Host:     "drift.internal",
			AuthMode: model.AuthModePassword,
			Password: "drift-pass",
			Alias:    "drift",
		})
	}); err != nil {
		t.Fatalf("failed to mutate connection fixture: %v", err)
	}

	driftCfg := config.Default()
	driftCfg.Behaviour.ContinueAfterSSHExit = false
	driftCfg.Behaviour.ShowCredentialsOnConnect = false
	if err := config.SaveConfig(cfgPath, driftCfg); err != nil {
		t.Fatalf("SaveConfig(driftCfg) failed: %v", err)
	}

	var out strings.Builder
	if err := handleRestore(connPath, keyPath, cfgPath, []string{
		"--in", backupPath,
		"--mode", "replace",
		"--with-config=true",
	}, &out); err != nil {
		t.Fatalf("handleRestore failed: %v", err)
	}

	loaded := loadTransferConnections(t, connPath, keyPath)
	if len(loaded.Connections) != 1 {
		t.Fatalf("expected 1 connection after restore replace, got %d", len(loaded.Connections))
	}
	if loaded.GetConnectionByAlias("before") == nil {
		t.Fatal("expected backup connection alias 'before' to be restored")
	}
	if loaded.GetConnectionByAlias("drift") != nil {
		t.Fatal("expected drift connection to be replaced by restore")
	}

	gotCfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if gotCfg.Behaviour != backupCfg.Behaviour {
		t.Fatalf("expected config %+v, got %+v", backupCfg.Behaviour, gotCfg.Behaviour)
	}
	if !strings.Contains(out.String(), "config_restored=true") {
		t.Fatalf("unexpected restore output: %q", out.String())
	}
}

func TestHandleRestoreSupportsLegacyImportPayload(t *testing.T) {
	connPath, keyPath := prepareTransferFixture(t, nil)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := config.SaveConfig(cfgPath, config.Default()); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	legacy := []model.SSHConnection{
		{
			Username: "deploy",
			Host:     "legacy.internal",
			AuthMode: model.AuthModeAgent,
			Alias:    "legacy",
		},
	}
	legacyPath := filepath.Join(t.TempDir(), "legacy.yaml")
	raw, err := yaml.Marshal(legacy)
	if err != nil {
		t.Fatalf("yaml.Marshal failed: %v", err)
	}
	if err := os.WriteFile(legacyPath, raw, 0o600); err != nil {
		t.Fatalf("failed to write legacy fixture: %v", err)
	}

	var out strings.Builder
	if err := handleRestore(connPath, keyPath, cfgPath, []string{"--in", legacyPath, "--mode", "replace"}, &out); err != nil {
		t.Fatalf("handleRestore failed: %v", err)
	}

	loaded := loadTransferConnections(t, connPath, keyPath)
	if len(loaded.Connections) != 1 {
		t.Fatalf("expected 1 restored connection from legacy payload, got %d", len(loaded.Connections))
	}
	if loaded.GetConnectionByAlias("legacy") == nil {
		t.Fatal("expected restored legacy alias to exist")
	}
	if !strings.Contains(out.String(), "config_restored=false") {
		t.Fatalf("unexpected restore output: %q", out.String())
	}
}

func TestHandleDoctorHealthyTextReport(t *testing.T) {
	connPath, keyPath := prepareTransferFixture(t, []model.SSHConnection{
		{
			Username: "ubuntu",
			Host:     "ok.internal",
			AuthMode: model.AuthModeAgent,
			Alias:    "ok",
		},
	})
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := config.SaveConfig(cfgPath, config.Default()); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	var out strings.Builder
	if err := handleDoctor(connPath, keyPath, cfgPath, nil, &out); err != nil {
		t.Fatalf("handleDoctor returned error for healthy fixture: %v", err)
	}
	if !strings.Contains(out.String(), "Doctor status: healthy") {
		t.Fatalf("unexpected doctor output: %q", out.String())
	}
}

func TestHandleDoctorMissingFilesJSONReportWithoutCreatingKey(t *testing.T) {
	tmpDir := t.TempDir()
	connPath := filepath.Join(tmpDir, "conn")
	keyPath := filepath.Join(tmpDir, "secret.key")
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	var out strings.Builder
	err := handleDoctor(connPath, keyPath, cfgPath, []string{"--json"}, &out)
	if err == nil {
		t.Fatal("expected doctor to return error for missing files")
	}

	var report doctorReport
	if decErr := json.Unmarshal([]byte(out.String()), &report); decErr != nil {
		t.Fatalf("failed to decode doctor json output: %v\nraw: %s", decErr, out.String())
	}
	if report.Healthy {
		t.Fatal("expected doctor report to be unhealthy for missing files")
	}
	if _, statErr := os.Stat(keyPath); !os.IsNotExist(statErr) {
		t.Fatalf("doctor should not create secret.key when missing, stat err: %v", statErr)
	}
}
