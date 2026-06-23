package model

import (
	"errors"
	"strings"
	"testing"
)

func TestGetConnectionByIDReturnsSlicePointer(t *testing.T) {
	file := NewConnectionFile()
	if err := file.AddConnection(SSHConnection{Username: "u", Host: "h", Password: "p"}); err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	id := file.Connections[0].ID
	conn := file.GetConnectionByID(id)
	if conn == nil {
		t.Fatal("expected connection pointer, got nil")
	}

	conn.Host = "changed.example"
	if file.Connections[0].Host != "changed.example" {
		t.Fatal("expected pointer to underlying slice element")
	}
}

func TestAddConnectionRejectsDuplicateAliasCaseInsensitive(t *testing.T) {
	file := NewConnectionFile()
	if err := file.AddConnection(SSHConnection{Username: "u1", Host: "h1", Password: "p1", Alias: "Prod"}); err != nil {
		t.Fatalf("first AddConnection failed: %v", err)
	}

	err := file.AddConnection(SSHConnection{Username: "u2", Host: "h2", Password: "p2", Alias: "  prod  "})
	if err == nil {
		t.Fatal("expected duplicate alias error, got nil")
	}
	if !errors.Is(err, ErrAliasAlreadyExists) {
		t.Fatalf("expected ErrAliasAlreadyExists, got %v", err)
	}
}

func TestUpdateConnectionByIDRejectsDuplicateAliasCaseInsensitive(t *testing.T) {
	file := NewConnectionFile()
	if err := file.AddConnection(SSHConnection{Username: "u1", Host: "h1", Password: "p1", Alias: "alpha"}); err != nil {
		t.Fatalf("first AddConnection failed: %v", err)
	}
	if err := file.AddConnection(SSHConnection{Username: "u2", Host: "h2", Password: "p2", Alias: "beta"}); err != nil {
		t.Fatalf("second AddConnection failed: %v", err)
	}

	targetID := file.Connections[1].ID
	ok, err := file.UpdateConnectionByID(targetID, SSHConnection{
		Username: "u2",
		Host:     "h2",
		Password: "p2",
		Alias:    " ALPHA ",
	})

	if !ok {
		t.Fatal("expected update target to be found")
	}
	if err == nil {
		t.Fatal("expected duplicate alias error, got nil")
	}
	if !errors.Is(err, ErrAliasAlreadyExists) {
		t.Fatalf("expected ErrAliasAlreadyExists, got %v", err)
	}
}

func TestGetConnectionByAliasUsesNormalizedLookup(t *testing.T) {
	file := NewConnectionFile()
	if err := file.AddConnection(SSHConnection{Username: "u1", Host: "h1", Password: "p1", Alias: "Prod"}); err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	conn := file.GetConnectionByAlias("  pRoD ")
	if conn == nil {
		t.Fatal("expected alias lookup to match case-insensitive trimmed alias")
	}
	if conn.Host != "h1" {
		t.Fatalf("unexpected connection returned: got host %q, want %q", conn.Host, "h1")
	}
}

func TestEnsureIDsNormalizesMissingAndDuplicateIDs(t *testing.T) {
	file := ConnectionFile{
		Version: CurrentConnectionFileVersion,
		Connections: []SSHConnection{
			{ID: "dup", Username: "u1", Host: "h1", Password: "p1"},
			{ID: "dup", Username: "u2", Host: "h2", Password: "p2"},
			{Username: "u3", Host: "h3", Password: "p3"},
		},
	}

	changed := file.EnsureIDs()
	if !changed {
		t.Fatal("expected EnsureIDs to mutate IDs")
	}

	seen := map[string]struct{}{}
	for _, conn := range file.Connections {
		if conn.ID == "" {
			t.Fatal("expected non-empty ID")
		}
		if _, exists := seen[conn.ID]; exists {
			t.Fatalf("duplicate ID generated: %s", conn.ID)
		}
		seen[conn.ID] = struct{}{}
	}
}

func TestSelectItemsIncludesGroupAndTags(t *testing.T) {
	file := NewConnectionFile()
	if err := file.AddConnection(SSHConnection{
		Username: "u1",
		Host:     "h1",
		Password: "p1",
		Alias:    "prod",
		Group:    "production",
		Tags:     []string{"linux", "api"},
	}); err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	items := file.SelectItems()
	if len(items) != 1 {
		t.Fatalf("expected one select item, got %d", len(items))
	}
	if got := items[0].Label; !strings.Contains(got, "[group:production]") || !strings.Contains(got, "[tags:linux,api]") {
		t.Fatalf("expected group/tags in label, got %q", got)
	}
}
