package commands

import (
	"bytes"
	"github.com/emirhangumus/sshmanager/v2/internal/config"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emirhangumus/sshmanager/v2/internal/model"
)

func TestConnectionSecurityWarnings(t *testing.T) {
	conn := model.SSHConnection{Alias: "prod", Username: "root", Host: "example.com", AuthMode: model.AuthModePassword, Password: "dont-display", ExtraSSHArgs: []string{"-o", "StrictHostKeyChecking=no", "-oUserKnownHostsFile=/dev/null", "-o", "ForwardAgent=yes"}}
	checks := connectionSecurityChecks(conn)
	if len(checks) != 4 {
		t.Fatalf("expected four security warnings, got %d", len(checks))
	}
	for _, check := range checks {
		if check.Status != "warn" || !strings.HasPrefix(check.Detail, "prod: ") || strings.Contains(check.Detail, conn.Password) {
			t.Fatal("unsafe diagnostics")
		}
	}
	conn.Username = "bob"
	conn.AuthMode = model.AuthModeAgent
	conn.ExtraSSHArgs = []string{"-o", "StrictHostKeyChecking=accept-new", "-oForwardAgent=no"}
	if len(connectionSecurityChecks(conn)) != 0 {
		t.Fatal("safe options warned")
	}
}

func TestConnectDryRunRedactsCredentials(t *testing.T) {
	conn := model.SSHConnection{Alias: "prod", Username: "bob", Host: "example.com", Port: 2222, AuthMode: model.AuthModePassword, Password: "target-secret", ProxyJump: "jump@bastion", ProxyJumpAuthMode: model.AuthModePassword, ProxyJumpPassword: "jump-secret"}
	var out strings.Builder
	if err := printConnectInvocation(&conn, &out); err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{conn.Password, conn.ProxyJumpPassword, "SSHPASS="} {
		if strings.Contains(out.String(), secret) {
			t.Fatal("dry-run leaked credential data")
		}
	}
	if !strings.Contains(out.String(), "Executable: sshpass") || !strings.Contains(out.String(), "2222") || !strings.Contains(out.String(), "bob@example.com") {
		t.Fatal("dry-run missed invocation")
	}
}

func TestConnectDryRunDoesNotStartSSH(t *testing.T) {
	conn, key := prepareTransferFixture(t, []model.SSHConnection{{Alias: "prod", Username: "bob", Host: "example.com", AuthMode: model.AuthModeAgent}})
	cfg := filepath.Join(filepath.Dir(conn), "config.yaml")
	if err := config.SaveConfig(cfg, config.Default()); err != nil {
		t.Fatal(err)
	}
	// An empty PATH makes any attempted SSH execution fail; dry run needs no binary.
	t.Setenv("PATH", "")
	before, err := os.ReadFile(conn)
	if err != nil {
		t.Fatal(err)
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	os.Stdout = writer
	t.Cleanup(func() { os.Stdout = oldStdout; _ = writer.Close(); _ = reader.Close() })
	runErr := handleConnectArgs(conn, key, cfg, []string{"--alias", "prod", "--dry-run"})
	os.Stdout = oldStdout
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if runErr != nil || !strings.Contains(string(output), "Executable: ssh\n") {
		t.Fatalf("did not produce a dry-run invocation: %v, %s", runErr, output)
	}
	after, err := os.ReadFile(conn)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("dry run changed datastore")
	}
	if err := handleConnectArgs(conn, key, cfg, []string{"--dry-run"}); err == nil {
		t.Fatal("unselected dry run entered an interactive workflow")
	}
}
