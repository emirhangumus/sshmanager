package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/emirhangumus/sshmanager/v2/internal/model"
)

func TestInvocationRejectsUnsafeTargets(t *testing.T) {
	for _, tc := range []struct{ user, host, jump string }{
		{"bob", "-oProxyCommand=evil", ""},
		{"-oProxyCommand=evil", "example.com", ""},
		{"bob", "foo;touch /tmp/pwned", ""},
		{"foo$(whoami)", "example.com", ""},
		{"bob", "host`id`", ""},
		{"bob", "example.com", "jump;id"},
		{"bob", "example.com", "-oProxyCommand=evil"},
		{"bob", "example.com", "bob$(id)@jump"},
		{"bob", "example.com", "jump%h"},
		{"bob", "example.com", "jump'quote"},
	} {
		conn := &model.SSHConnection{Username: tc.user, Host: tc.host, ProxyJump: tc.jump, AuthMode: model.AuthModeAgent}
		if _, _, _, err := buildConnectInvocation(conn); err == nil {
			t.Fatal("connect accepted unsafe target")
		}
		if _, _, _, err := buildExecInvocation(conn, "uptime"); err == nil {
			t.Fatal("exec accepted unsafe target")
		}
		if _, _, _, err := buildScpInvocation(conn, false, []string{"file"}, "dest"); err == nil {
			t.Fatal("scp accepted unsafe target")
		}
	}
}

func TestInvocationPreservesRemoteCommandAndSCPBoundary(t *testing.T) {
	conn := &model.SSHConnection{Username: "bob", Host: "example.com", Port: 2222, AuthMode: model.AuthModeAgent}
	remote := "echo 'a; b' && uname $(whoami)"
	_, args, _, err := buildExecInvocation(conn, remote)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-p", "2222", "-T", "--", "bob@example.com", remote}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("argv=%q", args)
	}
	_, args, _, err = buildScpInvocation(conn, false, []string{"-oProxyCommand=evil"}, "bob@example.com:/tmp/file")
	if err != nil {
		t.Fatal(err)
	}
	want = []string{"-P", "2222", "--", "-oProxyCommand=evil", "bob@example.com:/tmp/file"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("scp argv=%q", args)
	}
	local := scpArgSpec(scpArg{raw: "unknown:local"}, conn)
	if spec := scpArgSpec(scpArg{conn: conn, remotePath: "-file"}, conn); spec != "bob@example.com:./-file" {
		t.Fatal("remote path can be interpreted as an scp option")
	}
	if local != "./unknown:local" {
		t.Fatal("local path can be interpreted as remote")
	}
}

func TestSSHExecutableHooksRejected(t *testing.T) {
	for _, key := range []string{"ProxyCommand", "LocalCommand", "KnownHostsCommand", "PKCS11Provider", "SecurityKeyProvider", "Include", "Match", "RemoteCommand"} {
		if err := model.ValidateExtraSSHArgs([]string{"-o", key + "=evil"}); err == nil {
			t.Fatalf("accepted %s", key)
		}
	}
	if err := model.ValidateExtraSSHArgs([]string{"-o", "ServerAliveInterval=30\nProxyCommand=evil"}); err == nil {
		t.Fatal("accepted multiline option")
	}
}

func TestSCPRejectsRemoteShellPaths(t *testing.T) {
	cf := model.NewConnectionFile()
	if err := cf.AddConnection(model.SSHConnection{Username: "bob", Host: "example.com", Alias: "prod", AuthMode: model.AuthModeAgent}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/tmp/$(id)", "/tmp/file;id", "/tmp/a b", "/tmp/`id`", "/tmp/a\n"} {
		if _, err := classifyScpArgs(&cf, []string{"file", "prod:" + path}); err == nil {
			t.Fatalf("accepted remote shell path %q", path)
		}
	}
}

func TestProxyCommandQuotesIdentityPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("OpenSSH ProxyCommand shell test uses POSIX tools")
	}
	dir := t.TempDir()
	identity := filepath.Join(dir, "key'$(touch pwned)")
	if err := os.WriteFile(identity, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeFakeExecutable(t, filepath.Join(dir, "ssh"), "#!/bin/sh\nprintf '%s\n' \"$@\"\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	conn := &model.SSHConnection{Username: "bob", Host: "example.com", ProxyJump: "jump@bastion", ProxyJumpIdentityFile: identity}
	args, _, err := buildProxyJumpArgs(conn, conn.ProxyJump)
	if err != nil {
		t.Fatal(err)
	}
	command := strings.TrimPrefix(args[1], "ProxyCommand=")
	command = strings.ReplaceAll(command, "%h:%p", "example.com:22")
	// Exercise the shell boundary OpenSSH uses, with a harmless fake ssh.
	cmd := exec.Command("sh", "-c", command) //nolint:gosec // Generated ProxyCommand is intentionally tested through its real shell boundary.
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{"-i", identity, "-W", "example.com:22", "jump@bastion", ""}, "\n")
	if string(output) != want {
		t.Fatalf("proxy argv=%q", output)
	}
	if _, err := os.Stat(filepath.Join(dir, "pwned")); !os.IsNotExist(err) {
		t.Fatal("identity path executed shell syntax")
	}
}

func TestProcessEnvironmentScopesCredentials(t *testing.T) {
	t.Setenv("SSHPASS", "inherited-target")
	t.Setenv("SSHMANAGER_PROXY_JUMP_SSHPASS", "inherited-jump")
	t.Setenv("SSHMANAGER_MASTER_PASSPHRASE", "master-secret")
	t.Setenv("SSH_AUTH_SOCK", "agent-socket")
	env := scopedProcessEnv([]string{"SSHPASS=current-target"})
	joined := strings.Join(env, "\n")
	for _, secret := range []string{"inherited-target", "inherited-jump", "master-secret"} {
		if strings.Contains(joined, secret) {
			t.Fatal("inherited secret leaked to child")
		}
	}
	if !strings.Contains(joined, "SSHPASS=current-target") || !strings.Contains(joined, "SSH_AUTH_SOCK=agent-socket") {
		t.Fatal("child auth environment lost")
	}
}
