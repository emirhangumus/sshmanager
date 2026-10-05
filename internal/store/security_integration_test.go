package store

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emirhangumus/sshmanager/v2/internal/config"
	cryptoutil "github.com/emirhangumus/sshmanager/v2/internal/crypto"
	"github.com/emirhangumus/sshmanager/v2/internal/model"
	"github.com/emirhangumus/sshmanager/v2/internal/storage"
)

func TestStatePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX mode bits do not describe Windows ACLs")
	}
	dir := filepath.Join(t.TempDir(), "state")
	conn, key := filepath.Join(dir, "conn"), filepath.Join(dir, "secret.key")
	t.Setenv("SSHMANAGER_MASTER_PASSPHRASE", "")
	if err := config.SaveConfig(filepath.Join(dir, "config.yaml"), config.SSHManagerConfig{Security: config.SecurityConfig{KeyStorage: "file"}}); err != nil {
		t.Fatal(err)
	}
	s := NewConnectionStore(conn, key, WithKeyring(&fakeKeyring{setErr: errors.New("injected keyring failure")}))
	if err := s.InitializeIfEmpty(); err != nil {
		t.Fatal(err)
	}
	targetCfg := config.Default()
	if err := s.Restore(func(_ *model.ConnectionFile) error { return nil }, &targetCfg); err == nil {
		t.Fatal("expected staged restore to stop at injected keyring failure")
	}
	for _, name := range []string{"conn", "secret.key", "config.yaml", "key-storage.yaml", "conn.lock", "conn.restore-pending"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("%s mode=%o", name, info.Mode().Perm())
		}
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatal("state directory not 0700")
	}
}

func TestMultiProcessMutationsPreserveAllUpdates(t *testing.T) {
	if dir := os.Getenv("SSHMANAGER_TEST_MULTIPROCESS_DIR"); dir != "" {
		connectionLockTimeout = 20 * time.Second
		alias := os.Getenv("SSHMANAGER_TEST_MULTIPROCESS_ALIAS")
		s := NewConnectionStore(filepath.Join(dir, "conn"), filepath.Join(dir, "secret.key"))
		if err := s.Update(func(cf *model.ConnectionFile) error {
			return cf.AddConnection(model.SSHConnection{Alias: alias, Host: "example.com", Username: "bob", Password: "original"})
		}); err != nil {
			t.Fatal(err)
		}
		if err := s.Update(func(cf *model.ConnectionFile) error {
			conn := cf.GetConnectionByAlias(alias)
			if conn == nil {
				return fmt.Errorf("lost connection")
			}
			conn.Password = "updated"
			conn.Alias = alias + "-renamed"
			return cf.AddConnection(model.SSHConnection{Alias: alias + "-temp", Host: "example.com", Username: "bob", AuthMode: model.AuthModeAgent})
		}); err != nil {
			t.Fatal(err)
		}
		if err := s.Update(func(cf *model.ConnectionFile) error {
			temp := cf.GetConnectionByAlias(alias + "-temp")
			if temp == nil {
				return fmt.Errorf("lost temporary connection")
			}
			cf.RemoveConnectionByID(temp.ID)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Load(); err != nil {
			t.Fatal(err)
		}
		return
	}
	dir := t.TempDir()
	t.Setenv("SSHMANAGER_MASTER_PASSPHRASE", "")
	if err := config.SaveConfig(filepath.Join(dir, "config.yaml"), config.SSHManagerConfig{Security: config.SecurityConfig{KeyStorage: "file"}}); err != nil {
		t.Fatal(err)
	}
	s := NewConnectionStore(filepath.Join(dir, "conn"), filepath.Join(dir, "secret.key"))
	if err := s.InitializeIfEmpty(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	const workers = 20
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMultiProcessMutationsPreserveAllUpdates$") //nolint:gosec // Run this test binary as separate writers against an isolated datastore.
			cmd.Env = append(os.Environ(), "SSHMANAGER_TEST_MULTIPROCESS_DIR="+dir, fmt.Sprintf("SSHMANAGER_TEST_MULTIPROCESS_ALIAS=worker-%d", i))
			output, err := cmd.CombinedOutput()
			if err != nil {
				errs <- fmt.Errorf("worker %d: %w: %s", i, err, output)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	if t.Failed() {
		return
	}
	cf, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cf.Connections) != workers {
		t.Fatalf("lost updates: %d connections", len(cf.Connections))
	}
	for i := 0; i < workers; i++ {
		conn := cf.GetConnectionByAlias(fmt.Sprintf("worker-%d-renamed", i))
		if conn == nil || conn.Password != "updated" {
			t.Fatal("lost edit/rename")
		}
	}
	raw, err := os.ReadFile(s.connectionFilePath)
	if err != nil {
		t.Fatal(err)
	}
	if !cryptoutil.IsVersionedStore(raw) || bytes.Contains(raw, []byte("updated")) {
		t.Fatal("datastore corrupted or plaintext")
	}
}

func TestSchemaVersionAndLegacyRewrite(t *testing.T) {
	s, _ := fixtureStore(t, "file")
	key, err := cryptoutil.LoadExistingKey(s.secretKeyFilePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, plain := range []string{
		"- id: fixed-id\n  alias: legacy\n  host: example.com\n  username: bob\n  authMode: agent\n",
		"connections:\n- id: fixed-id\n  alias: legacy\n  host: example.com\n  username: bob\n  authMode: agent\n",
	} {
		if err := encryptAndStoreFile(plain, s.connectionFilePath, key); err != nil {
			t.Fatal(err)
		}
		cf, err := s.Load()
		if err != nil {
			t.Fatal(err)
		}
		if len(cf.Connections) != 1 || cf.Connections[0].ID != "fixed-id" {
			t.Fatal("legacy fields lost")
		}
		rewritten, err := decryptAndReadFile(s.connectionFilePath, key)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(rewritten, "version: ") {
			t.Fatal("legacy schema not migrated during load")
		}
	}
	if err := encryptAndStoreFile("version: 999\nconnections: []\n", s.connectionFilePath, key); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(s.connectionFilePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(); err == nil {
		t.Fatal("unknown schema version accepted")
	}
	after, err := os.ReadFile(s.connectionFilePath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("failed schema load changed store")
	}
}

func FuzzConnectionDecode(f *testing.F) {
	for _, seed := range []string{"", "connections: []", "version: 1.0\nconnections: []", "version: 999\nconnections: []", "- host: example.com\n  username: bob", "unexpected: field"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		cf, err := parseConnectionFile(input)
		if err == nil && cf.Version != model.CurrentConnectionFileVersion {
			t.Fatal("accepted unknown schema")
		}
	})
}

func TestLegacyCiphertextMigratesOnlyOnNormalLoad(t *testing.T) {
	s, _ := fixtureStore(t, "file")
	cf, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	plain, err := toYAMLString(cf)
	if err != nil {
		t.Fatal(err)
	}
	key, err := cryptoutil.LoadExistingKey(s.secretKeyFilePath)
	if err != nil {
		t.Fatal(err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	nonce := bytes.Repeat([]byte{7}, 12)
	legacy := gcm.Seal(nonce, nonce, []byte(plain), nil)
	if err := storage.WriteFileAtomic(s.connectionFilePath, legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := s.Inspect(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(s.connectionFilePath)
	if err != nil || !bytes.Equal(before, legacy) {
		t.Fatal("inspect changed legacy ciphertext")
	}
	migrated, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(migrated.Connections) != len(cf.Connections) || migrated.Connections[0].ID != cf.Connections[0].ID || migrated.Connections[0].Password != cf.Connections[0].Password {
		t.Fatal("migration lost IDs or credentials")
	}
	after, err := os.ReadFile(s.connectionFilePath)
	if err != nil || !cryptoutil.IsVersionedStore(after) {
		t.Fatal("load did not commit versioned ciphertext")
	}
}
