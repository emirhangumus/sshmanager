package flags

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emirhangumus/sshmanager/internal/config"
	"github.com/emirhangumus/sshmanager/internal/model"
	"github.com/emirhangumus/sshmanager/internal/store"
)

func TestCheckNoFlagsContinuesExecution(t *testing.T) {
	continueExecution, err := Check(nil, "", "", "", "v1.2.3")
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if !continueExecution {
		t.Fatal("expected continueExecution=true when no flags are provided")
	}
}

func TestCheckVersionStopsExecution(t *testing.T) {
	continueExecution, err := Check([]string{"-version"}, "", "", "", "v9.9.9")
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if continueExecution {
		t.Fatal("expected continueExecution=false for -version")
	}
}

func TestCheckSetConfigUpdatesValue(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	continueExecution, err := Check(
		[]string{"-set", "behaviour.continueAfterSSHExit", "true"},
		filepath.Join(tmpDir, "conn"),
		filepath.Join(tmpDir, "secret.key"),
		configPath,
		"dev",
	)
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if continueExecution {
		t.Fatal("expected continueExecution=false for -set")
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if !cfg.Behaviour.ContinueAfterSSHExit {
		t.Fatal("expected behaviour.continueAfterSSHExit=true after -set")
	}
}

func TestCheckSetConfigRejectsMissingArgs(t *testing.T) {
	continueExecution, err := Check([]string{"-set", "behaviour.continueAfterSSHExit"}, "", "", "", "dev")
	if err == nil {
		t.Fatal("expected error for missing -set value, got nil")
	}
	if continueExecution {
		t.Fatal("expected continueExecution=false when parsing -set fails")
	}
}

func TestCheckCompletionBashStopsExecution(t *testing.T) {
	continueExecution, err := Check([]string{"-completion", "bash"}, "", "", "", "dev")
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if continueExecution {
		t.Fatal("expected continueExecution=false for -completion bash")
	}
}

func TestCheckCompletionInstallRequiresShellArg(t *testing.T) {
	continueExecution, err := Check([]string{"-completion", "install"}, "", "", "", "dev")
	if err == nil {
		t.Fatal("expected error when install shell is missing")
	}
	if continueExecution {
		t.Fatal("expected continueExecution=false for failing -completion install")
	}
}

func TestCheckCompletePrintsMatchingAliases(t *testing.T) {
	tmpDir := t.TempDir()
	connPath := filepath.Join(tmpDir, "conn")
	keyPath := filepath.Join(tmpDir, "secret.key")
	configPath := filepath.Join(tmpDir, "config.yaml")

	connStore := store.NewConnectionStore(connPath, keyPath)
	if err := connStore.InitializeIfEmpty(); err != nil {
		t.Fatalf("InitializeIfEmpty failed: %v", err)
	}
	err := connStore.Update(func(connFile *model.ConnectionFile) error {
		if err := connFile.AddConnection(model.SSHConnection{
			Username: "u1",
			Host:     "h1",
			Password: "p1",
			Alias:    "prod-api",
		}); err != nil {
			return err
		}
		if err := connFile.AddConnection(model.SSHConnection{
			Username: "u2",
			Host:     "h2",
			Password: "p2",
			Alias:    "stage-api",
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed seeding aliases: %v", err)
	}

	output := captureStdout(t, func() {
		continueExecution, err := Check([]string{"-complete", "prod"}, connPath, keyPath, configPath, "dev")
		if err != nil {
			t.Fatalf("Check returned error: %v", err)
		}
		if continueExecution {
			t.Fatal("expected continueExecution=false for -complete")
		}
	})

	if !strings.Contains(output, "prod-api") {
		t.Fatalf("expected output to contain matching alias, got %q", output)
	}
	if strings.Contains(output, "stage-api") {
		t.Fatalf("expected output to exclude non-matching alias, got %q", output)
	}
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
