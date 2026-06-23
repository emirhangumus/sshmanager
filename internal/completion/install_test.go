package completion

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emirhangumus/sshmanager/internal/completion/scripts"
)

func TestScriptReturnsKnownShellScripts(t *testing.T) {
	bash, err := Script("bash")
	if err != nil {
		t.Fatalf("Script(bash) returned error: %v", err)
	}
	if bash != scripts.BashScript {
		t.Fatal("unexpected bash script content")
	}

	zsh, err := Script("zsh")
	if err != nil {
		t.Fatalf("Script(zsh) returned error: %v", err)
	}
	if zsh != scripts.ZshScript {
		t.Fatal("unexpected zsh script content")
	}
}

func TestScriptRejectsUnknownShell(t *testing.T) {
	if _, err := Script("fish"); err == nil {
		t.Fatal("expected error for unknown shell, got nil")
	}
}

func TestInstallBashWritesScriptAndIsIdempotent(t *testing.T) {
	home := t.TempDir()
	setHomeEnv(t, home)

	firstPath, err := Install("bash")
	if err != nil {
		t.Fatalf("Install(bash) returned error: %v", err)
	}
	secondPath, err := Install("bash")
	if err != nil {
		t.Fatalf("second Install(bash) returned error: %v", err)
	}
	if firstPath != secondPath {
		t.Fatalf("expected same install path, got %q and %q", firstPath, secondPath)
	}

	content, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatalf("failed to read installed bash script: %v", err)
	}
	if string(content) != scripts.BashScript {
		t.Fatal("installed bash script content mismatch")
	}

	bashrcPath := filepath.Join(home, ".bashrc")
	bashrcContent, err := os.ReadFile(bashrcPath)
	if err != nil {
		t.Fatalf("failed to read .bashrc: %v", err)
	}
	if strings.Count(string(bashrcContent), "# added by sshmanager") != 1 {
		t.Fatalf("expected single sshmanager marker in .bashrc, got:\n%s", string(bashrcContent))
	}
}

func TestInstallZshWritesScript(t *testing.T) {
	home := t.TempDir()
	setHomeEnv(t, home)

	dst, err := Install("zsh")
	if err != nil {
		t.Fatalf("Install(zsh) returned error: %v", err)
	}

	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read installed zsh script: %v", err)
	}
	if string(content) != scripts.ZshScript {
		t.Fatal("installed zsh script content mismatch")
	}
}

func setHomeEnv(t *testing.T, home string) {
	t.Helper()

	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}
