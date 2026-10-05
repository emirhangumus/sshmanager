package commands

import (
	"bytes"
	"fmt"
	"github.com/emirhangumus/sshmanager/v2/internal/model"
	"io"
	"os"
	"os/exec"
	"strconv"
)

// ensureHostKeyAccepted makes sure the user gets to see and confirm an
// unknown host key before a password-auth connection runs. sshpass takes
// over stdin/stdout to feed the password and deliberately refuses to answer
// ssh's "authenticity of host ... can't be established" prompt itself (it
// exits 6 rather than risk auto-accepting a spoofed key), so the user would
// otherwise never see that prompt at all.
//
// This does not reimplement any of that: it shells out to the real "ssh"
// binary for a cheap, unauthenticated priming connection (no sshpass, no
// stored password) with stdin/stdout/stderr wired to the terminal exactly
// like a normal interactive ssh invocation. If the host key is unknown, the
// user sees ssh's own prompt and answers it themselves; ssh itself writes
// the accepted key to ~/.ssh/known_hosts. Auth is intentionally disabled
// (NumberOfPasswordPrompts=0) so the priming connection fails fast right
// after the host-key stage — we only need the side effect on known_hosts,
// not a real session. The subsequent sshpass connection then proceeds
// exactly as it would have if the host key had already been trusted.
func ensureHostKeyAccepted(host string, port int) error {
	if err := model.ValidateSSHHost(host); err != nil {
		return err
	}
	if hostKeyKnown(host, port) {
		return nil
	}

	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		// No plain ssh on PATH to prime with; let sshpass surface whatever
		// happens on its own.
		return nil
	}

	cmd := exec.Command(sshPath,
		"-p", strconv.Itoa(port),
		"-o", "BatchMode=no",
		"-o", "NumberOfPasswordPrompts=0",
		"--", host, "exit",
	)
	cmd.Env = scopedProcessEnv(nil)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	stderr := &primingStderrFilter{out: os.Stderr}
	cmd.Stderr = stderr
	_ = cmd.Run()
	stderr.flush()

	return nil
}

// primingStderrFilter passes ssh's stderr through line by line, dropping
// only the "Permission denied" line the auth-disabled priming connection is
// always expected to end with. That line reflects the throwaway priming
// attempt failing on purpose, not the real password-based connection that
// follows, so showing it would just confuse the user; every other line —
// including ssh's own host-key prompt and "Warning: Permanently added..."
// confirmation — is passed through unchanged.
type primingStderrFilter struct {
	out io.Writer
	buf []byte
}

func (w *primingStderrFilter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		idx := bytes.IndexByte(w.buf, '\n')
		if idx < 0 {
			break
		}
		line := w.buf[:idx+1]
		w.buf = w.buf[idx+1:]
		if !bytes.Contains(line, []byte("Permission denied (")) {
			if _, err := w.out.Write(line); err != nil {
				return 0, err
			}
		}
	}
	return len(p), nil
}

func (w *primingStderrFilter) flush() {
	if len(w.buf) > 0 && !bytes.Contains(w.buf, []byte("Permission denied (")) {
		_, _ = w.out.Write(w.buf)
	}
	w.buf = nil
}

// hostKeyKnown reports whether host:port already has an entry in the
// user's known_hosts, via the same lookup ssh itself uses.
func hostKeyKnown(host string, port int) bool {
	hostPattern := host
	if port != 22 {
		hostPattern = fmt.Sprintf("[%s]:%d", host, port)
	}
	cmd := exec.Command("ssh-keygen", "-F", hostPattern)
	cmd.Env = scopedProcessEnv(nil)
	return cmd.Run() == nil
}
