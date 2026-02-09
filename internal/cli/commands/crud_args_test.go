package commands

import (
	"strings"
	"testing"

	"github.com/emirhangumus/sshmanager/internal/model"
)

func TestHandleAddArgsAddsConnection(t *testing.T) {
	connPath, keyPath := prepareTransferFixture(t, nil)

	var out strings.Builder
	err := handleAddArgs(connPath, keyPath, []string{
		"--host", "prod.internal",
		"--username", "ubuntu",
		"--auth-mode", model.AuthModeAgent,
		"--alias", "prod",
		"--description", "production",
		"--proxy-jump", "jump.internal:2222",
		"--local-forward", "8080:127.0.0.1:80",
		"--remote-forward", "9000:127.0.0.1:9000",
		"--extra-ssh-arg", "-vv",
		"--extra-ssh-arg", "-o",
		"--extra-ssh-arg", "ServerAliveInterval=30",
		"--group", "production",
		"--tag", "linux",
		"--tag", "api",
	}, &out)
	if err != nil {
		t.Fatalf("handleAddArgs failed: %v", err)
	}
	if !strings.Contains(out.String(), "SSH connection saved.") {
		t.Fatalf("unexpected output: %q", out.String())
	}

	loaded := loadTransferConnections(t, connPath, keyPath)
	if len(loaded.Connections) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(loaded.Connections))
	}
	conn := loaded.GetConnectionByAlias("prod")
	if conn == nil {
		t.Fatal("expected saved connection with alias prod")
	}
	if conn.Host != "prod.internal" {
		t.Fatalf("unexpected host: %q", conn.Host)
	}
	if conn.EffectiveAuthMode() != model.AuthModeAgent {
		t.Fatalf("unexpected auth mode: %q", conn.EffectiveAuthMode())
	}
	if conn.ProxyJump != "jump.internal:2222" {
		t.Fatalf("unexpected proxy jump: %q", conn.ProxyJump)
	}
	if len(conn.LocalForwards) != 1 || conn.LocalForwards[0] != "8080:127.0.0.1:80" {
		t.Fatalf("unexpected local forwards: %v", conn.LocalForwards)
	}
	if len(conn.RemoteForwards) != 1 || conn.RemoteForwards[0] != "9000:127.0.0.1:9000" {
		t.Fatalf("unexpected remote forwards: %v", conn.RemoteForwards)
	}
	if len(conn.ExtraSSHArgs) != 3 {
		t.Fatalf("unexpected extra ssh args: %v", conn.ExtraSSHArgs)
	}
	if conn.Group != "production" {
		t.Fatalf("unexpected group: %q", conn.Group)
	}
	if len(conn.Tags) != 2 || conn.Tags[0] != "linux" || conn.Tags[1] != "api" {
		t.Fatalf("unexpected tags: %v", conn.Tags)
	}
}

func TestHandleAddArgsRejectsInvalidInput(t *testing.T) {
	connPath, keyPath := prepareTransferFixture(t, nil)

	err := handleAddArgs(connPath, keyPath, []string{
		"--host", "prod.internal",
		"--auth-mode", model.AuthModePassword,
	}, ioDiscard())
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestHandleEditArgsUpdatesConnection(t *testing.T) {
	connPath, keyPath := prepareTransferFixture(t, []model.SSHConnection{
		{
			Username: "ubuntu",
			Host:     "old.internal",
			AuthMode: model.AuthModePassword,
			Password: "secret",
			Alias:    "prod",
		},
	})

	var out strings.Builder
	err := handleEditArgs(connPath, keyPath, []string{
		"--alias", "prod",
		"--new-host", "new.internal",
		"--new-port", "2222",
		"--new-auth-mode", model.AuthModeKey,
		"--new-identity-file", "/tmp/id_ed25519",
		"--new-proxy-jump", "jump.internal:2200",
		"--new-local-forward", "8080:127.0.0.1:80",
		"--new-remote-forward", "9000:127.0.0.1:9000",
		"--new-extra-ssh-arg", "-vv",
		"--new-extra-ssh-arg", "-o",
		"--new-extra-ssh-arg", "ServerAliveInterval=30",
		"--new-group", "production",
		"--new-tag", "linux",
		"--new-tag", "api",
		"--new-description", "updated",
	}, &out)
	if err != nil {
		t.Fatalf("handleEditArgs failed: %v", err)
	}
	if !strings.Contains(out.String(), "SSH connection updated.") {
		t.Fatalf("unexpected output: %q", out.String())
	}

	loaded := loadTransferConnections(t, connPath, keyPath)
	updated := loaded.GetConnectionByAlias("prod")
	if updated == nil {
		t.Fatal("expected updated connection")
	}
	if updated.Host != "new.internal" {
		t.Fatalf("expected host update, got %q", updated.Host)
	}
	if updated.Port != 2222 {
		t.Fatalf("expected port 2222, got %d", updated.Port)
	}
	if updated.AuthMode != model.AuthModeKey {
		t.Fatalf("expected key auth mode, got %q", updated.AuthMode)
	}
	if updated.Password != "" {
		t.Fatalf("expected password to be cleared for key mode, got %q", updated.Password)
	}
	if updated.IdentityFile != "/tmp/id_ed25519" {
		t.Fatalf("unexpected identity file: %q", updated.IdentityFile)
	}
	if updated.ProxyJump != "jump.internal:2200" {
		t.Fatalf("unexpected proxy jump: %q", updated.ProxyJump)
	}
	if len(updated.LocalForwards) != 1 || updated.LocalForwards[0] != "8080:127.0.0.1:80" {
		t.Fatalf("unexpected local forwards: %v", updated.LocalForwards)
	}
	if len(updated.RemoteForwards) != 1 || updated.RemoteForwards[0] != "9000:127.0.0.1:9000" {
		t.Fatalf("unexpected remote forwards: %v", updated.RemoteForwards)
	}
	if len(updated.ExtraSSHArgs) != 3 {
		t.Fatalf("unexpected extra ssh args: %v", updated.ExtraSSHArgs)
	}
	if updated.Group != "production" {
		t.Fatalf("unexpected group: %q", updated.Group)
	}
	if len(updated.Tags) != 2 || updated.Tags[0] != "linux" || updated.Tags[1] != "api" {
		t.Fatalf("unexpected tags: %v", updated.Tags)
	}
}

func TestHandleRemoveArgsRemovesByAliasWithYes(t *testing.T) {
	connPath, keyPath := prepareTransferFixture(t, []model.SSHConnection{
		{
			Username: "ubuntu",
			Host:     "old.internal",
			AuthMode: model.AuthModeAgent,
			Alias:    "prod",
		},
	})

	var out strings.Builder
	err := handleRemoveArgs(connPath, keyPath, []string{"--alias", "prod", "--yes"}, &out)
	if err != nil {
		t.Fatalf("handleRemoveArgs failed: %v", err)
	}
	if !strings.Contains(out.String(), "SSH connection removed.") {
		t.Fatalf("unexpected output: %q", out.String())
	}

	loaded := loadTransferConnections(t, connPath, keyPath)
	if loaded.GetConnectionByAlias("prod") != nil {
		t.Fatal("expected prod connection to be removed")
	}
}
