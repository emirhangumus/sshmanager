package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emirhangumus/sshmanager/v2/internal/model"
)

func writeTestIdentityFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(path, []byte("test-key"), 0o600); err != nil {
		t.Fatalf("failed to write test identity file: %v", err)
	}
	return path
}

func TestBuildConnectInvocationPasswordMode(t *testing.T) {
	conn := &model.SSHConnection{
		Username: "ubuntu",
		Host:     "example.com",
		Password: "secret",
		AuthMode: model.AuthModePassword,
	}

	bin, args, env, err := buildConnectInvocation(conn)
	if err != nil {
		t.Fatalf("buildConnectInvocation failed: %v", err)
	}
	if bin != "sshpass" {
		t.Fatalf("unexpected binary: %q", bin)
	}
	wantArgs := []string{"-e", "ssh", "-p", "22", "ubuntu@example.com"}
	assertStringSliceEqual(t, args, wantArgs)
	assertStringSliceEqual(t, env, []string{"SSHPASS=secret"})
}

func TestBuildConnectInvocationKeyMode(t *testing.T) {
	identityFile := writeTestIdentityFile(t)
	conn := &model.SSHConnection{
		Username:     "ubuntu",
		Host:         "example.com",
		Port:         2222,
		AuthMode:     model.AuthModeKey,
		IdentityFile: identityFile,
	}

	bin, args, env, err := buildConnectInvocation(conn)
	if err != nil {
		t.Fatalf("buildConnectInvocation failed: %v", err)
	}
	if bin != "ssh" {
		t.Fatalf("unexpected binary: %q", bin)
	}
	wantArgs := []string{"-p", "2222", "-i", identityFile, "ubuntu@example.com"}
	assertStringSliceEqual(t, args, wantArgs)
	if len(env) != 0 {
		t.Fatalf("expected no extra env for key mode, got %v", env)
	}
}

func TestBuildConnectInvocationWithAdvancedOptions(t *testing.T) {
	identityFile := writeTestIdentityFile(t)
	conn := &model.SSHConnection{
		Username:       "ubuntu",
		Host:           "example.com",
		AuthMode:       model.AuthModeKey,
		IdentityFile:   identityFile,
		ProxyJump:      "jump.internal:2222",
		LocalForwards:  []string{"8080:127.0.0.1:80"},
		RemoteForwards: []string{"9000:127.0.0.1:9000"},
		ExtraSSHArgs:   []string{"-vv", "-o", "ServerAliveInterval=30"},
	}

	bin, args, env, err := buildConnectInvocation(conn)
	if err != nil {
		t.Fatalf("buildConnectInvocation failed: %v", err)
	}
	if bin != "ssh" {
		t.Fatalf("unexpected binary: %q", bin)
	}
	wantArgs := []string{
		"-p", "22",
		"-i", identityFile,
		"-J", "jump.internal:2222",
		"-L", "8080:127.0.0.1:80",
		"-R", "9000:127.0.0.1:9000",
		"-vv",
		"-o", "ServerAliveInterval=30",
		"ubuntu@example.com",
	}
	assertStringSliceEqual(t, args, wantArgs)
	if len(env) != 0 {
		t.Fatalf("expected no extra env for key mode, got %v", env)
	}
}

func TestBuildConnectInvocationAgentMode(t *testing.T) {
	conn := &model.SSHConnection{
		Username: "ubuntu",
		Host:     "example.com",
		AuthMode: model.AuthModeAgent,
	}

	bin, args, env, err := buildConnectInvocation(conn)
	if err != nil {
		t.Fatalf("buildConnectInvocation failed: %v", err)
	}
	if bin != "ssh" {
		t.Fatalf("unexpected binary: %q", bin)
	}
	wantArgs := []string{"-p", "22", "ubuntu@example.com"}
	assertStringSliceEqual(t, args, wantArgs)
	if len(env) != 0 {
		t.Fatalf("expected no extra env for agent mode, got %v", env)
	}
}

func TestBuildConnectInvocationLegacyFallback(t *testing.T) {
	conn := &model.SSHConnection{
		Username: "ubuntu",
		Host:     "example.com",
		Password: "secret",
	}

	bin, args, _, err := buildConnectInvocation(conn)
	if err != nil {
		t.Fatalf("buildConnectInvocation failed: %v", err)
	}
	if bin != "sshpass" {
		t.Fatalf("expected legacy password fallback to sshpass, got %q", bin)
	}
	wantArgs := []string{"-e", "ssh", "-p", "22", "ubuntu@example.com"}
	assertStringSliceEqual(t, args, wantArgs)
}

func TestBuildConnectInvocationRejectsMissingIdentityForKeyMode(t *testing.T) {
	conn := &model.SSHConnection{
		Username: "ubuntu",
		Host:     "example.com",
		AuthMode: model.AuthModeKey,
	}

	if _, _, _, err := buildConnectInvocation(conn); err == nil {
		t.Fatal("expected error for missing identity file in key mode, got nil")
	}
}

func TestBuildConnectInvocationRejectsNonexistentIdentityFile(t *testing.T) {
	conn := &model.SSHConnection{
		Username:     "ubuntu",
		Host:         "example.com",
		AuthMode:     model.AuthModeKey,
		IdentityFile: filepath.Join(t.TempDir(), "does-not-exist"),
	}

	if _, _, _, err := buildConnectInvocation(conn); err == nil {
		t.Fatal("expected error for nonexistent identity file, got nil")
	}
}

func TestBuildConnectInvocationRejectsDirectoryIdentityFile(t *testing.T) {
	conn := &model.SSHConnection{
		Username:     "ubuntu",
		Host:         "example.com",
		AuthMode:     model.AuthModeKey,
		IdentityFile: t.TempDir(),
	}

	if _, _, _, err := buildConnectInvocation(conn); err == nil {
		t.Fatal("expected error for directory identity file, got nil")
	}
}

func TestBuildConnectInvocationRejectsInvalidAdvancedOptions(t *testing.T) {
	tests := []struct {
		name string
		conn *model.SSHConnection
	}{
		{
			name: "invalid proxy jump",
			conn: &model.SSHConnection{
				Username:  "ubuntu",
				Host:      "example.com",
				AuthMode:  model.AuthModeAgent,
				ProxyJump: "bad jump",
			},
		},
		{
			name: "invalid forward",
			conn: &model.SSHConnection{
				Username:      "ubuntu",
				Host:          "example.com",
				AuthMode:      model.AuthModeAgent,
				LocalForwards: []string{"bad"},
			},
		},
		{
			name: "unsupported extra arg",
			conn: &model.SSHConnection{
				Username:     "ubuntu",
				Host:         "example.com",
				AuthMode:     model.AuthModeAgent,
				ExtraSSHArgs: []string{"-L"},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if _, _, _, err := buildConnectInvocation(tc.conn); err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}

func TestBuildConnectInvocationProxyJumpPassword(t *testing.T) {
	conn := &model.SSHConnection{
		Username:          "ubuntu",
		Host:              "internal.example.com",
		AuthMode:          model.AuthModePassword,
		Password:          "target-secret",
		ProxyJump:         "jumpuser@bastion.example.com:2222",
		ProxyJumpAuthMode: model.AuthModePassword,
		ProxyJumpPassword: "jump-secret",
	}

	bin, args, env, err := buildConnectInvocation(conn)
	if err != nil {
		t.Fatalf("buildConnectInvocation failed: %v", err)
	}
	if bin != "sshpass" {
		t.Fatalf("unexpected binary: %q", bin)
	}

	wantProxyCommand := `SSHPASS="$SSHMANAGER_PROXY_JUMP_SSHPASS" sshpass -e ssh -p '2222' -W %h:%p 'jumpuser@bastion.example.com'`
	wantArgs := []string{
		"-e", "ssh", "-p", "22",
		"-o", "ProxyCommand=" + wantProxyCommand,
		"ubuntu@internal.example.com",
	}
	assertStringSliceEqual(t, args, wantArgs)
	assertStringSliceEqual(t, env, []string{
		"SSHPASS=target-secret",
		"SSHMANAGER_PROXY_JUMP_SSHPASS=jump-secret",
	})

	// Regression guard: the jump password must never appear in argv, only
	// referenced by env var name, since argv is visible via `ps`.
	for _, arg := range args {
		if strings.Contains(arg, "jump-secret") {
			t.Fatalf("jump password leaked into argv: %q", arg)
		}
	}
}

func TestBuildConnectInvocationProxyJumpIdentityFileOnly(t *testing.T) {
	targetIdentity := writeTestIdentityFile(t)
	jumpIdentity := writeTestIdentityFile(t)
	conn := &model.SSHConnection{
		Username:              "ubuntu",
		Host:                  "internal.example.com",
		AuthMode:              model.AuthModeKey,
		IdentityFile:          targetIdentity,
		ProxyJump:             "jumpuser@bastion.example.com",
		ProxyJumpIdentityFile: jumpIdentity,
	}

	bin, args, env, err := buildConnectInvocation(conn)
	if err != nil {
		t.Fatalf("buildConnectInvocation failed: %v", err)
	}
	if bin != "ssh" {
		t.Fatalf("unexpected binary: %q", bin)
	}
	if len(env) != 0 {
		t.Fatalf("expected no env for identity-only proxy jump, got %v", env)
	}

	wantProxyCommand := "ssh -i '" + jumpIdentity + "' -W %h:%p 'jumpuser@bastion.example.com'"
	wantArgs := []string{
		"-p", "22", "-i", targetIdentity,
		"-o", "ProxyCommand=" + wantProxyCommand,
		"ubuntu@internal.example.com",
	}
	assertStringSliceEqual(t, args, wantArgs)
}

func TestBuildConnectInvocationRejectsMultiHopProxyJumpPassword(t *testing.T) {
	conn := &model.SSHConnection{
		Username:          "ubuntu",
		Host:              "internal.example.com",
		AuthMode:          model.AuthModeAgent,
		ProxyJump:         "jump1,jump2",
		ProxyJumpAuthMode: model.AuthModePassword,
		ProxyJumpPassword: "jump-secret",
	}

	if _, _, _, err := buildConnectInvocation(conn); err == nil {
		t.Fatal("expected error for multi-hop proxy jump with dedicated password, got nil")
	}
}

func TestBuildConnectInvocationRejectsMissingProxyJumpPassword(t *testing.T) {
	conn := &model.SSHConnection{
		Username:          "ubuntu",
		Host:              "internal.example.com",
		AuthMode:          model.AuthModeAgent,
		ProxyJump:         "bastion.example.com",
		ProxyJumpAuthMode: model.AuthModePassword,
	}

	if _, _, _, err := buildConnectInvocation(conn); err == nil {
		t.Fatal("expected error for missing proxy jump password, got nil")
	}
}

func assertStringSliceEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("slice length mismatch: got=%d want=%d (%v vs %v)", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("slice mismatch at %d: got=%q want=%q", i, got[i], want[i])
		}
	}
}
