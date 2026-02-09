package model

import "testing"

func TestGetConnectionByIDReturnsSlicePointer(t *testing.T) {
	file := NewConnectionFile()
	file.AddConnection(SSHConnection{Username: "u", Host: "h", Password: "p"})

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
