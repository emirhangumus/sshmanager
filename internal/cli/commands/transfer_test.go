package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emirhangumus/sshmanager/internal/model"
	"github.com/emirhangumus/sshmanager/internal/storage"
	"github.com/emirhangumus/sshmanager/internal/store"
	"gopkg.in/yaml.v3"
)

func TestHandleExportWritesJSONFile(t *testing.T) {
	connPath, keyPath := prepareTransferFixture(t, []model.SSHConnection{
		{
			Username: "ubuntu",
			Host:     "app.internal",
			AuthMode: model.AuthModePassword,
			Password: "secret",
			Alias:    "prod",
		},
		{
			Username:     "deploy",
			Host:         "edge.internal",
			Port:         2222,
			AuthMode:     model.AuthModeKey,
			IdentityFile: "/tmp/id_ed25519",
			Alias:        "edge",
		},
	})

	exportPath := filepath.Join(t.TempDir(), "exports", "connections.json")
	var out strings.Builder
	if err := handleExport(connPath, keyPath, []string{"--format", "json", "--out", exportPath}, &out); err != nil {
		t.Fatalf("handleExport failed: %v", err)
	}
	if !strings.Contains(out.String(), "Exported 2 connections") {
		t.Fatalf("unexpected export output: %q", out.String())
	}

	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("failed to read export file: %v", err)
	}

	var parsed model.ConnectionFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to decode exported JSON: %v", err)
	}
	if len(parsed.Connections) != 2 {
		t.Fatalf("expected 2 exported connections, got %d", len(parsed.Connections))
	}
	if parsed.Connections[0].Password == "" {
		t.Fatal("expected exported payload to include password for backup/restore")
	}
}

func TestHandleImportReplaceMode(t *testing.T) {
	connPath, keyPath := prepareTransferFixture(t, []model.SSHConnection{
		{
			Username: "ubuntu",
			Host:     "old.internal",
			AuthMode: model.AuthModePassword,
			Password: "old-pass",
			Alias:    "old",
		},
	})

	importFile := model.ConnectionFile{
		Version: model.CurrentConnectionFileVersion,
		Connections: []model.SSHConnection{
			{
				Username:     "deploy",
				Host:         "new.internal",
				Port:         2222,
				AuthMode:     model.AuthModeKey,
				IdentityFile: "/home/user/.ssh/id_ed25519",
				Alias:        "new",
			},
		},
	}

	importPath := filepath.Join(t.TempDir(), "import.json")
	raw, err := json.Marshal(importFile)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if err := os.WriteFile(importPath, raw, 0o600); err != nil {
		t.Fatalf("failed to write import fixture: %v", err)
	}

	var out strings.Builder
	if err := handleImport(connPath, keyPath, []string{"--in", importPath, "--mode", "replace"}, &out); err != nil {
		t.Fatalf("handleImport failed: %v", err)
	}

	loaded := loadTransferConnections(t, connPath, keyPath)
	if len(loaded.Connections) != 1 {
		t.Fatalf("expected 1 connection after replace, got %d", len(loaded.Connections))
	}
	if loaded.Connections[0].Alias != "new" {
		t.Fatalf("expected replaced connection alias 'new', got %q", loaded.Connections[0].Alias)
	}
	if loaded.GetConnectionByAlias("old") != nil {
		t.Fatal("expected old connection to be removed in replace mode")
	}
}

func TestHandleImportMergeModeUpdatesByAliasAndAddsNew(t *testing.T) {
	connPath, keyPath := prepareTransferFixture(t, []model.SSHConnection{
		{
			Username: "ubuntu",
			Host:     "old.prod.internal",
			AuthMode: model.AuthModePassword,
			Password: "old-pass",
			Alias:    "prod",
		},
	})

	legacyImport := []model.SSHConnection{
		{
			Username: "ubuntu",
			Host:     "new.prod.internal",
			AuthMode: model.AuthModePassword,
			Password: "new-pass",
			Alias:    "PROD", // case-insensitive merge target
		},
		{
			Username:     "deploy",
			Host:         "stage.internal",
			Port:         2200,
			AuthMode:     model.AuthModeKey,
			IdentityFile: "/tmp/id_stage",
			Alias:        "stage",
		},
	}
	importPath := filepath.Join(t.TempDir(), "import.yaml")
	raw, err := yaml.Marshal(legacyImport)
	if err != nil {
		t.Fatalf("yaml.Marshal failed: %v", err)
	}
	if err := os.WriteFile(importPath, raw, 0o600); err != nil {
		t.Fatalf("failed to write import fixture: %v", err)
	}

	if err := handleImport(connPath, keyPath, []string{"--in", importPath, "--mode", "merge"}, ioDiscard()); err != nil {
		t.Fatalf("handleImport failed: %v", err)
	}

	loaded := loadTransferConnections(t, connPath, keyPath)
	if len(loaded.Connections) != 2 {
		t.Fatalf("expected 2 connections after merge, got %d", len(loaded.Connections))
	}

	prod := loaded.GetConnectionByAlias("prod")
	if prod == nil {
		t.Fatal("expected merged prod connection to exist")
	}
	if prod.Host != "new.prod.internal" {
		t.Fatalf("expected prod host update, got %q", prod.Host)
	}
	if prod.Password != "new-pass" {
		t.Fatalf("expected prod password update, got %q", prod.Password)
	}

	stage := loaded.GetConnectionByAlias("stage")
	if stage == nil {
		t.Fatal("expected stage connection to be added")
	}
	if stage.Port != 2200 {
		t.Fatalf("expected stage port 2200, got %d", stage.Port)
	}
}

func TestHandleImportRejectsUnknownMode(t *testing.T) {
	connPath, keyPath := prepareTransferFixture(t, nil)
	importPath := filepath.Join(t.TempDir(), "import.yaml")
	if err := os.WriteFile(importPath, []byte("connections: []\n"), 0o600); err != nil {
		t.Fatalf("failed to write import fixture: %v", err)
	}

	err := handleImport(connPath, keyPath, []string{"--in", importPath, "--mode", "invalid"}, ioDiscard())
	if err == nil {
		t.Fatal("expected error for invalid import mode, got nil")
	}
}

func TestHandleImportRejectsInvalidAdvancedOptions(t *testing.T) {
	connPath, keyPath := prepareTransferFixture(t, nil)

	importFile := model.ConnectionFile{
		Version: model.CurrentConnectionFileVersion,
		Connections: []model.SSHConnection{
			{
				Username:      "ubuntu",
				Host:          "bad.internal",
				AuthMode:      model.AuthModeAgent,
				LocalForwards: []string{"bad-forward"},
			},
		},
	}
	importPath := filepath.Join(t.TempDir(), "invalid-import.json")
	raw, err := json.Marshal(importFile)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if err := os.WriteFile(importPath, raw, 0o600); err != nil {
		t.Fatalf("failed to write import fixture: %v", err)
	}

	err = handleImport(connPath, keyPath, []string{"--in", importPath, "--mode", "replace"}, ioDiscard())
	if err == nil {
		t.Fatal("expected import validation error for advanced options, got nil")
	}
}

func TestHandleImportRejectsInvalidMetadataOptions(t *testing.T) {
	connPath, keyPath := prepareTransferFixture(t, nil)

	importFile := model.ConnectionFile{
		Version: model.CurrentConnectionFileVersion,
		Connections: []model.SSHConnection{
			{
				Username: "ubuntu",
				Host:     "bad.internal",
				AuthMode: model.AuthModeAgent,
				Group:    "bad group",
			},
		},
	}
	importPath := filepath.Join(t.TempDir(), "invalid-metadata-import.json")
	raw, err := json.Marshal(importFile)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if err := os.WriteFile(importPath, raw, 0o600); err != nil {
		t.Fatalf("failed to write import fixture: %v", err)
	}

	err = handleImport(connPath, keyPath, []string{"--in", importPath, "--mode", "replace"}, ioDiscard())
	if err == nil {
		t.Fatal("expected import validation error for metadata options, got nil")
	}
}

func prepareTransferFixture(t *testing.T, conns []model.SSHConnection) (string, string) {
	t.Helper()

	tmpDir := t.TempDir()
	connPath := filepath.Join(tmpDir, "conn")
	keyPath := filepath.Join(tmpDir, "secret.key")

	if err := storage.CreateFileIfNotExists(connPath, 0o600); err != nil {
		t.Fatalf("CreateFileIfNotExists(conn) failed: %v", err)
	}

	connStore := store.NewConnectionStore(connPath, keyPath)
	if err := connStore.InitializeIfEmpty(); err != nil {
		t.Fatalf("InitializeIfEmpty failed: %v", err)
	}

	if len(conns) == 0 {
		return connPath, keyPath
	}

	err := connStore.Update(func(connFile *model.ConnectionFile) error {
		for _, conn := range conns {
			if err := connFile.AddConnection(conn); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to seed connection file: %v", err)
	}

	return connPath, keyPath
}

func loadTransferConnections(t *testing.T, connPath, keyPath string) model.ConnectionFile {
	t.Helper()
	connStore := store.NewConnectionStore(connPath, keyPath)
	loaded, err := connStore.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	return loaded
}

func ioDiscard() *strings.Builder {
	return &strings.Builder{}
}
