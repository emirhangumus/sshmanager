package commands

import (
	"testing"

	"github.com/emirhangumus/sshmanager/internal/model"
)

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
	conn := &model.SSHConnection{
		Username:     "ubuntu",
		Host:         "example.com",
		Port:         2222,
		AuthMode:     model.AuthModeKey,
		IdentityFile: "/tmp/id_ed25519",
	}

	bin, args, env, err := buildConnectInvocation(conn)
	if err != nil {
		t.Fatalf("buildConnectInvocation failed: %v", err)
	}
	if bin != "ssh" {
		t.Fatalf("unexpected binary: %q", bin)
	}
	wantArgs := []string{"-p", "2222", "-i", "/tmp/id_ed25519", "ubuntu@example.com"}
	assertStringSliceEqual(t, args, wantArgs)
	if len(env) != 0 {
		t.Fatalf("expected no extra env for key mode, got %v", env)
	}
}

func TestBuildConnectInvocationWithAdvancedOptions(t *testing.T) {
	conn := &model.SSHConnection{
		Username:       "ubuntu",
		Host:           "example.com",
		AuthMode:       model.AuthModeKey,
		IdentityFile:   "/tmp/id_ed25519",
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
		"-i", "/tmp/id_ed25519",
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
