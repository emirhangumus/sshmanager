package store

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emirhangumus/sshmanager/internal/model"
	"github.com/emirhangumus/sshmanager/internal/storage"
	"github.com/gofrs/flock"
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
	held := flock.New(lockPath)
	if err := held.Lock(); err != nil {
		t.Fatalf("failed to create lock fixture: %v", err)
	}
	defer func() { _ = held.Close() }()

	oldTimeout := connectionLockTimeout
	oldRetry := connectionLockRetryInterval
	connectionLockTimeout = 60 * time.Millisecond
	connectionLockRetryInterval = 10 * time.Millisecond
	defer func() {
		connectionLockTimeout = oldTimeout
		connectionLockRetryInterval = oldRetry
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

func TestLockHeldByAnotherProcessIsNotStolen(t *testing.T) {
	if path := os.Getenv("SSHMANAGER_TEST_LOCK_PATH"); path != "" {
		held := flock.New(path)
		if err := held.Lock(); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stdout.WriteString("ready\n"); err != nil {
			t.Fatal(err)
		}
		if _, err := io.Copy(io.Discard, os.Stdin); err != nil {
			t.Fatal(err)
		}
		if err := held.Close(); err != nil {
			t.Fatal(err)
		}
		return
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "conn.lock")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestLockHeldByAnotherProcessIsNotStolen$") //nolint:gosec // Launch only this test binary to exercise OS process locking.
	cmd.Env = append(os.Environ(), "SSHMANAGER_TEST_LOCK_PATH="+path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = stdin.Close()
		if err := cmd.Wait(); err != nil {
			t.Errorf("lock helper: %v", err)
		}
	}()
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || line != "ready\n" {
		t.Fatalf("helper not ready: %q, %v", line, err)
	}
	old := time.Now().Add(-24 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	previous := connectionLockTimeout
	connectionLockTimeout = 100 * time.Millisecond
	defer func() { connectionLockTimeout = previous }()
	s := NewConnectionStore(filepath.Join(dir, "conn"), filepath.Join(dir, "secret.key"), WithKeyring(&fakeKeyring{}))
	if err := s.InitializeIfEmpty(); err == nil {
		t.Fatal("a live process's old lock was stolen")
	}
	if _, err := os.Stat(s.statePath()); !os.IsNotExist(err) {
		t.Fatal("state changed while another process held the lock")
	}
}
