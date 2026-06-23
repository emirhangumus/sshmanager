package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emirhangumus/sshmanager/internal/model"
	"github.com/emirhangumus/sshmanager/internal/storage"
)

func TestUpdateSerializesConcurrentMutations(t *testing.T) {
	tmpDir := t.TempDir()
	connPath := filepath.Join(tmpDir, "conn")
	keyPath := filepath.Join(tmpDir, "secret.key")

	if err := storage.CreateFileIfNotExists(connPath, 0o600); err != nil {
		t.Fatalf("CreateFileIfNotExists(conn) failed: %v", err)
	}

	connStore := NewConnectionStore(connPath, keyPath)
	if err := connStore.InitializeIfEmpty(); err != nil {
		t.Fatalf("InitializeIfEmpty failed: %v", err)
	}

	const workers = 20
	var wg sync.WaitGroup
	errs := make(chan error, workers)

	for i := 0; i < workers; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := connStore.Update(func(connFile *model.ConnectionFile) error {
				// Increase contention to validate lock-protected update behavior.
				time.Sleep(5 * time.Millisecond)
				return connFile.AddConnection(model.SSHConnection{
					Username: "user",
					Host:     fmt.Sprintf("host-%d.example", i),
					Password: "pass",
					Alias:    fmt.Sprintf("alias-%d", i),
				})
			})
			errs <- err
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("Update returned error: %v", err)
		}
	}

	loaded, err := connStore.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded.Connections) != workers {
		t.Fatalf("expected %d connections after concurrent updates, got %d", workers, len(loaded.Connections))
	}
}

func TestSaveTimesOutWhenLockIsHeld(t *testing.T) {
	tmpDir := t.TempDir()
	connPath := filepath.Join(tmpDir, "conn")
	keyPath := filepath.Join(tmpDir, "secret.key")
	lockPath := connPath + ".lock"

	if err := storage.CreateFileIfNotExists(connPath, 0o600); err != nil {
		t.Fatalf("CreateFileIfNotExists(conn) failed: %v", err)
	}
	if err := os.WriteFile(lockPath, []byte("held"), 0o600); err != nil {
		t.Fatalf("failed to create lock fixture: %v", err)
	}

	oldTimeout := connectionLockTimeout
	oldRetry := connectionLockRetryInterval
	oldStaleAfter := connectionLockStaleAfter
	connectionLockTimeout = 60 * time.Millisecond
	connectionLockRetryInterval = 10 * time.Millisecond
	connectionLockStaleAfter = time.Hour
	defer func() {
		connectionLockTimeout = oldTimeout
		connectionLockRetryInterval = oldRetry
		connectionLockStaleAfter = oldStaleAfter
	}()

	connStore := NewConnectionStore(connPath, keyPath)
	err := connStore.Save(model.NewConnectionFile())
	if err == nil {
		t.Fatal("expected timeout error while lock is held, got nil")
	}
	if !strings.Contains(err.Error(), "timed out acquiring mutation lock") {
		t.Fatalf("unexpected error: %v", err)
	}
}
