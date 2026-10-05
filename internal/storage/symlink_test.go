package storage

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSymlinkedStateNeverTouchesTarget(t *testing.T) {
	for _, name := range []string{"conn", "secret.key", "config.yaml", "key-storage.yaml", "conn.restore-pending", "conn.lock"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, "target")
			link := filepath.Join(dir, name)
			original := []byte("do not overwrite this file")
			if err := os.WriteFile(target, original, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, link); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			operations := []func() error{
				func() error { return WriteFileAtomic(link, []byte("new"), 0o600) },
				func() error { return CreateFileIfNotExists(link, 0o600) },
				func() error { _, err := ReadFileRegular(link); return err },
				func() error { _, err := IsFileEmpty(link); return err },
				func() error { return SecureDelete(link) },
			}
			for _, operation := range operations {
				if err := operation(); err == nil {
					t.Fatal("symlink accepted")
				}
				got, err := os.ReadFile(target)
				if err != nil || !bytes.Equal(got, original) {
					t.Fatal("symlink target modified")
				}
			}
		})
	}
}

func TestSymlinkedStateDirectoryRejected(t *testing.T) {
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "real")
	link := filepath.Join(dir, "linked")
	if err := os.Mkdir(targetDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(targetDir, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := WriteFileAtomic(filepath.Join(link, "conn"), []byte("secret"), 0o600); err == nil {
		t.Fatal("linked directory accepted")
	}
	if _, err := os.Stat(filepath.Join(targetDir, "conn")); !os.IsNotExist(err) {
		t.Fatal("wrote through linked directory")
	}
}
