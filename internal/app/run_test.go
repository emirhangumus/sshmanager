package app

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emirhangumus/sshmanager/internal/model"
)

func TestVersionStringDefaultsAndTrims(t *testing.T) {
	if got := (BuildInfo{}).VersionString(); got != "dev" {
		t.Fatalf("empty version should default to dev, got %q", got)
	}
	if got := (BuildInfo{Version: " v1.2.3 "}).VersionString(); got != "v1.2.3" {
		t.Fatalf("version should be trimmed, got %q", got)
	}
}

func TestRunWithVersionFlag(t *testing.T) {
	home := t.TempDir()
	setHomeEnv(t, home)

	output := captureStdout(t, func() {
		err := Run([]string{"sshmanager", "-version"}, BuildInfo{Version: "v1.2.3"})
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})

	if !strings.Contains(output, "v1.2.3") {
		t.Fatalf("expected version output, got %q", output)
	}

	assertExists(t, filepath.Join(home, ".sshmanager", "conn"))
	assertExists(t, filepath.Join(home, ".sshmanager", "config.yaml"))
	assertExists(t, filepath.Join(home, ".sshmanager", "secret.key"))
}

func TestRunListSubcommandNoConnections(t *testing.T) {
	home := t.TempDir()
	setHomeEnv(t, home)

	output := captureStdout(t, func() {
		err := Run([]string{"sshmanager", "list"}, BuildInfo{})
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})

	if !strings.Contains(output, "No SSH connections found.") {
		t.Fatalf("expected empty list message, got %q", output)
	}
}

func TestRunImportAndListJSON(t *testing.T) {
	home := t.TempDir()
	setHomeEnv(t, home)

	importPath := filepath.Join(t.TempDir(), "connections.json")
	payload := model.ConnectionFile{
		Version: model.CurrentConnectionFileVersion,
		Connections: []model.SSHConnection{
			{
				Username: "ubuntu",
				Host:     "api.internal",
				AuthMode: model.AuthModePassword,
				Password: "secret",
				Alias:    "prod",
			},
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if err := os.WriteFile(importPath, raw, 0o600); err != nil {
		t.Fatalf("failed to write import fixture: %v", err)
	}

	if err := Run([]string{"sshmanager", "import", "--in", importPath, "--mode", "replace"}, BuildInfo{}); err != nil {
		t.Fatalf("Run(import) returned error: %v", err)
	}

	output := captureStdout(t, func() {
		if err := Run([]string{"sshmanager", "list", "--json"}, BuildInfo{}); err != nil {
			t.Fatalf("Run(list --json) returned error: %v", err)
		}
	})

	var listed []map[string]any
	if err := json.Unmarshal([]byte(output), &listed); err != nil {
		t.Fatalf("failed to decode list --json output: %v\nraw: %s", err, output)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 listed connection, got %d", len(listed))
	}
	if listed[0]["alias"] != "prod" {
		t.Fatalf("unexpected alias: %v", listed[0]["alias"])
	}
}

func TestRunAddEditAndRemoveSubcommands(t *testing.T) {
	home := t.TempDir()
	setHomeEnv(t, home)

	if err := Run([]string{
		"sshmanager",
		"add",
		"--host", "app.internal",
		"--username", "ubuntu",
		"--auth-mode", model.AuthModeAgent,
		"--group", "production",
		"--tag", "api",
		"--alias", "prod",
	}, BuildInfo{}); err != nil {
		t.Fatalf("Run(add) returned error: %v", err)
	}

	if err := Run([]string{
		"sshmanager",
		"edit",
		"--alias", "prod",
		"--new-host", "new.internal",
		"--new-port", "2222",
		"--new-group", "platform",
		"--new-tag", "core",
	}, BuildInfo{}); err != nil {
		t.Fatalf("Run(edit) returned error: %v", err)
	}

	if err := Run([]string{"sshmanager", "rename", "--alias", "prod", "--to", "prod-new"}, BuildInfo{}); err != nil {
		t.Fatalf("Run(rename) returned error: %v", err)
	}

	listAfterEdit := captureStdout(t, func() {
		if err := Run([]string{"sshmanager", "list", "--json"}, BuildInfo{}); err != nil {
			t.Fatalf("Run(list --json) returned error: %v", err)
		}
	})
	var edited []map[string]any
	if err := json.Unmarshal([]byte(listAfterEdit), &edited); err != nil {
		t.Fatalf("failed to decode list output: %v", err)
	}
	if len(edited) != 1 {
		t.Fatalf("expected 1 listed connection, got %d", len(edited))
	}
	if edited[0]["host"] != "new.internal" {
		t.Fatalf("expected edited host new.internal, got %v", edited[0]["host"])
	}
	if edited[0]["port"] != float64(2222) {
		t.Fatalf("expected edited port 2222, got %v", edited[0]["port"])
	}
	if edited[0]["alias"] != "prod-new" {
		t.Fatalf("expected renamed alias prod-new, got %v", edited[0]["alias"])
	}
	if edited[0]["group"] != "platform" {
		t.Fatalf("expected updated group platform, got %v", edited[0]["group"])
	}

	if err := Run([]string{"sshmanager", "remove", "--alias", "prod-new", "--yes"}, BuildInfo{}); err != nil {
		t.Fatalf("Run(remove) returned error: %v", err)
	}

	listAfterRemove := captureStdout(t, func() {
		if err := Run([]string{"sshmanager", "list"}, BuildInfo{}); err != nil {
			t.Fatalf("Run(list) returned error: %v", err)
		}
	})
	if !strings.Contains(listAfterRemove, "No SSH connections found.") {
		t.Fatalf("expected empty list message after remove, got %q", listAfterRemove)
	}
}

func TestRunBackupAndRestoreSubcommands(t *testing.T) {
	home := t.TempDir()
	setHomeEnv(t, home)

	if err := Run([]string{
		"sshmanager",
		"add",
		"--host", "app.internal",
		"--username", "ubuntu",
		"--auth-mode", model.AuthModePassword,
		"--password", "secret",
		"--alias", "prod",
	}, BuildInfo{}); err != nil {
		t.Fatalf("Run(add) returned error: %v", err)
	}

	backupPath := filepath.Join(t.TempDir(), "backup.yaml")
	if err := Run([]string{"sshmanager", "backup", "--out", backupPath, "--format", "yaml"}, BuildInfo{}); err != nil {
		t.Fatalf("Run(backup) returned error: %v", err)
	}

	if err := Run([]string{"sshmanager", "remove", "--alias", "prod", "--yes"}, BuildInfo{}); err != nil {
		t.Fatalf("Run(remove) returned error: %v", err)
	}

	if err := Run([]string{"sshmanager", "restore", "--in", backupPath, "--mode", "replace"}, BuildInfo{}); err != nil {
		t.Fatalf("Run(restore) returned error: %v", err)
	}

	output := captureStdout(t, func() {
		if err := Run([]string{"sshmanager", "list", "--json"}, BuildInfo{}); err != nil {
			t.Fatalf("Run(list --json) returned error: %v", err)
		}
	})

	var listed []map[string]any
	if err := json.Unmarshal([]byte(output), &listed); err != nil {
		t.Fatalf("failed to decode list --json output: %v\nraw: %s", err, output)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 listed connection after restore, got %d", len(listed))
	}
	if listed[0]["alias"] != "prod" {
		t.Fatalf("unexpected alias after restore: %v", listed[0]["alias"])
	}
}

func TestRunDoctorBypassesStartupAndDoesNotCreateDataDir(t *testing.T) {
	home := t.TempDir()
	setHomeEnv(t, home)

	var runErr error
	output := captureStdout(t, func() {
		runErr = Run([]string{"sshmanager", "doctor"}, BuildInfo{})
	})
	if runErr == nil {
		t.Fatal("expected doctor to fail on missing files")
	}
	if !strings.Contains(output, "Doctor status: unhealthy") {
		t.Fatalf("unexpected doctor output: %q", output)
	}

	dataDir := filepath.Join(home, ".sshmanager")
	if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
		t.Fatalf("doctor should not create data directory, stat err: %v", err)
	}
}

func setHomeEnv(t *testing.T, home string) {
	t.Helper()

	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}
	os.Stdout = w

	defer func() {
		os.Stdout = oldStdout
		_ = r.Close()
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close write pipe: %v", err)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read captured stdout: %v", err)
	}
	return string(b)
}

func assertExists(t *testing.T, filePath string) {
	t.Helper()
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("expected file to exist: %s (%v)", filePath, err)
	}
}
