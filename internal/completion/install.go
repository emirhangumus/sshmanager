package completion

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/emirhangumus/sshmanager/internal/completion/scripts"
)

func Script(shell string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "bash":
		return scripts.BashScript, nil
	case "zsh":
		return scripts.ZshScript, nil
	default:
		return "", fmt.Errorf("unknown shell %q (want \"bash\" or \"zsh\")", shell)
	}
}

func Install(shell string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}

	shell = strings.ToLower(strings.TrimSpace(shell))
	script, err := Script(shell)
	if err != nil {
		return "", err
	}

	var dst string
	switch shell {
	case "bash":
		dst = filepath.Join(home, ".local", "share", "bash-completion", "completions", "sshmanager")
	case "zsh":
		dst = filepath.Join(home, ".zsh", "completions", "_sshmanager")
	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", fmt.Errorf("failed to create completion directory: %w", err)
	}
	if err := os.WriteFile(dst, []byte(script), 0o644); err != nil {
		return "", fmt.Errorf("failed to write completion file: %w", err)
	}

	if shell == "bash" {
		if err := ensureBashSourceLine(home, dst); err != nil {
			return "", err
		}
	}

	return dst, nil
}

func ensureBashSourceLine(home, completionPath string) error {
	bashrcPath := filepath.Join(home, ".bashrc")
	quotedPath := fmt.Sprintf("%q", completionPath)
	line := fmt.Sprintf("[ -f %s ] && source %s", quotedPath, quotedPath)

	return ensureLineInFile(bashrcPath, line)
}

func ensureLineInFile(filePath, line string) error {
	if data, err := os.ReadFile(filePath); err == nil {
		if strings.Contains(string(data), line) {
			return nil
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", filePath, err)
	}
	defer f.Close()

	if _, err := f.WriteString("\n# added by sshmanager\n" + line + "\n"); err != nil {
		return fmt.Errorf("failed to write %s: %w", filePath, err)
	}
	return nil
}
