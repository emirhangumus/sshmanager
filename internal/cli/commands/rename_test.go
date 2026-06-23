package commands

import (
	"strings"
	"testing"

	"github.com/emirhangumus/sshmanager/internal/model"
)

func TestHandleRenameArgsRenamesAlias(t *testing.T) {
	connPath, keyPath := prepareTransferFixture(t, []model.SSHConnection{
		{
			Username: "ubuntu",
			Host:     "old.internal",
			AuthMode: model.AuthModeAgent,
			Alias:    "prod",
		},
	})

	var out strings.Builder
	if err := handleRenameArgs(connPath, keyPath, []string{"--alias", "prod", "--to", "prod-new"}, &out); err != nil {
		t.Fatalf("handleRenameArgs failed: %v", err)
	}
	if !strings.Contains(out.String(), "SSH alias renamed.") {
		t.Fatalf("unexpected output: %q", out.String())
	}

	loaded := loadTransferConnections(t, connPath, keyPath)
	if loaded.GetConnectionByAlias("prod") != nil {
		t.Fatal("expected old alias to be removed")
	}
	if loaded.GetConnectionByAlias("prod-new") == nil {
		t.Fatal("expected new alias to exist")
	}
}

func TestHandleRenameArgsRejectsDuplicateAlias(t *testing.T) {
	connPath, keyPath := prepareTransferFixture(t, []model.SSHConnection{
		{
			Username: "ubuntu",
			Host:     "one.internal",
			AuthMode: model.AuthModeAgent,
			Alias:    "one",
		},
		{
			Username: "ubuntu",
			Host:     "two.internal",
			AuthMode: model.AuthModeAgent,
			Alias:    "two",
		},
	})

	err := handleRenameArgs(connPath, keyPath, []string{"--alias", "one", "--to", "two"}, ioDiscard())
	if err == nil {
		t.Fatal("expected duplicate alias error, got nil")
	}
}
