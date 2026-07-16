package commands

import (
	"os/exec"
	"testing"
)

func requireSSHKeygen(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available")
	}
}

func TestHostKeyKnownFormatsPortedHostPattern(t *testing.T) {
	requireSSHKeygen(t)

	// Neither host has ever been seen, so both lookups should report
	// "unknown" without erroring, regardless of the port used to build the
	// lookup pattern.
	if hostKeyKnown("sshmanager-test-host.invalid", 22) {
		t.Fatal("expected unseen host on default port to be unknown")
	}
	if hostKeyKnown("sshmanager-test-host.invalid", 2222) {
		t.Fatal("expected unseen host on non-default port to be unknown")
	}
}

func TestEnsureHostKeyAcceptedNoopsWhenSSHUnavailable(t *testing.T) {
	// With no ssh binary reachable, ensureHostKeyAccepted must fall through
	// quietly rather than error, leaving the caller's own connection attempt
	// to surface whatever the real problem is.
	t.Setenv("PATH", "")
	if err := ensureHostKeyAccepted("sshmanager-test-host.invalid", 22); err != nil {
		t.Fatalf("expected nil error when ssh is unavailable, got %v", err)
	}
}
