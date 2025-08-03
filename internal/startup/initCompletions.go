package startup

import (
	_ "embed"
	"fmt"
	"github.com/emirhangumus/sshmanager/internal/cli/flag/compScripts"
	"os"
	"path/filepath"
	"strings"
)

// ensureShellCompletion installs *one* script that matches the current shell.
// Call this at the very top of main(); it returns quickly if everything exists.
func ensureShellCompletion() {
	switch detectShell() {
	case "bash":
		installBash()
	case "zsh":
		installZsh()
	default:
		// unknown or non-interactive shell → do nothing
	}
}

func detectShell() string {
	// Preferred: interactive shell env vars
	if os.Getenv("ZSH_VERSION") != "" {
		return "zsh"
	}
	if os.Getenv("BASH_VERSION") != "" {
		return "bash"
	}

	// Fallback: login shell path ($SHELL)
	switch base := filepath.Base(os.Getenv("SHELL")); {
	case strings.HasPrefix(base, "zsh"):
		return "zsh"
	case strings.HasPrefix(base, "bash"):
		return "bash"
	default:
		return ""
	}
}

func installBash() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dst := filepath.Join(home, ".local/share/bash-completion/completions/sshmanager")
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		os.MkdirAll(filepath.Dir(dst), 0o755)
		_ = os.WriteFile(dst, []byte(compScripts.BashScript), 0o644)
	}
}

func installZsh() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".zsh/completions")
	dst := filepath.Join(dir, "_sshmanager")

	if _, err := os.Stat(dst); os.IsNotExist(err) {
		os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(dst, []byte(compScripts.ZshScript), 0o644)
		ensureDirOnFpath(dir, filepath.Join(home, ".zshrc"))
	}
}

func ensureDirOnFpath(dir, zshrc string) {
	line := fmt.Sprintf("fpath=(%s $fpath)", dir)

	if data, err := os.ReadFile(zshrc); err == nil && strings.Contains(string(data), line) {
		return // already present
	}
	f, err := os.OpenFile(zshrc, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err == nil {
		defer f.Close()
		_, _ = f.WriteString("\n# added by sshmanager\n" + line + "\n")
	}
}
