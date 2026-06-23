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

func TestPrintUsageUsesSubcommandsWithoutDashes(t *testing.T) {
	var out strings.Builder
	PrintUsage(&out)

	text := out.String()
	for _, cmd := range []string{
		"  add [flags]",
		"  edit [flags]",
		"  remove [flags]",
		"  rename [flags]",
		"  connect [flags]",
		"  list [flags]",
		"  export --out <path> [--format yaml|json]",
		"  import --in <path> [--format auto|yaml|json] [--mode merge|replace]",
		"  backup --out <path> [--format yaml|json] [--include-config=true|false]",
		"  restore --in <path> [--format auto|yaml|json] [--mode merge|replace] [--with-config=true|false]",
		"  doctor [--json]",
		"  clean",
		"  set <config-name> <config-value>",
		"  version",
		"  complete [prefix]",
		"  completion <bash|zsh>",
		"  help",
	} {
		if !strings.Contains(text, cmd) {
			t.Fatalf("expected usage to include %q, got %q", cmd, text)
		}
	}
	if strings.Contains(text, "  -clean") {
		t.Fatalf("usage should not include dash-prefixed clean option, got %q", text)
	}
}

func TestHandleVersionWritesVersion(t *testing.T) {
	var out strings.Builder
	HandleVersion("v9.9.9", &out)
	if strings.TrimSpace(out.String()) != "v9.9.9" {
		t.Fatalf("unexpected version output: %q", out.String())
	}
}

func TestHandleSetUpdatesValue(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	if err := HandleSet(configPath, []string{"behaviour.continueAfterSSHExit", "true"}); err != nil {
		t.Fatalf("HandleSet returned error: %v", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if !cfg.Behaviour.ContinueAfterSSHExit {
		t.Fatal("expected behaviour.continueAfterSSHExit=true after set")
	}
}

func TestHandleSetRejectsMissingArgs(t *testing.T) {
	err := HandleSet(filepath.Join(t.TempDir(), "config.yaml"), []string{"behaviour.continueAfterSSHExit"})
	if err == nil {
		t.Fatal("expected error for missing set value, got nil")
	}
}

func TestHandleCompletionBash(t *testing.T) {
	output := captureStdout(t, func() {
		if err := HandleCompletion([]string{"bash"}); err != nil {
			t.Fatalf("HandleCompletion returned error: %v", err)
		}
	})

	if !strings.Contains(output, "_sshmanager") {
		t.Fatalf("expected bash completion function output, got %q", output)
	}
}

func TestHandleCompletionInstallRequiresShellArg(t *testing.T) {
	err := HandleCompletion([]string{"install"})
	if err == nil {
		t.Fatal("expected error when install shell is missing")
	}
}

func TestHandleCompletePrintsMatchingAliases(t *testing.T) {
	tmpDir := t.TempDir()
	connPath := filepath.Join(tmpDir, "conn")
	keyPath := filepath.Join(tmpDir, "secret.key")

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
		if err := HandleComplete(connPath, keyPath, []string{"prod"}); err != nil {
			t.Fatalf("HandleComplete returned error: %v", err)
		}
	})

	if !strings.Contains(output, "prod-api") {
		t.Fatalf("expected output to contain matching alias, got %q", output)
	}
	if strings.Contains(output, "stage-api") {
		t.Fatalf("expected output to exclude non-matching alias, got %q", output)
	}
}

func TestMapLegacyDashCommand(t *testing.T) {
	tests := map[string]string{
		"-clean":      "clean",
		"-complete":   "complete",
		"-completion": "completion",
		"-set":        "set",
		"-version":    "version",
	}
	for in, want := range tests {
		got, ok := MapLegacyDashCommand(in)
		if !ok {
			t.Fatalf("expected %q to map to %q", in, want)
		}
		if got != want {
			t.Fatalf("unexpected mapping for %q: got %q want %q", in, got, want)
		}
	}

	if _, ok := MapLegacyDashCommand("-unknown"); ok {
		t.Fatal("unexpected mapping for unknown legacy option")
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
