package store

import (
	"bytes"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/emirhangumus/sshmanager/v2/internal/config"
	cryptoutil "github.com/emirhangumus/sshmanager/v2/internal/crypto"
	"github.com/emirhangumus/sshmanager/v2/internal/model"
	"github.com/emirhangumus/sshmanager/v2/internal/storage"
	"github.com/zalando/go-keyring"
)

type fakeKeyring struct {
	mu                        sync.Mutex
	entries                   map[string]string
	getErr, setErr, deleteErr error
	corrupt                   bool
}

func (f *fakeKeyring) Get(service, user string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return "", f.getErr
	}
	v, ok := f.entries[service+":"+user]
	if !ok {
		return "", keyring.ErrNotFound
	}
	return v, nil
}
func (f *fakeKeyring) Set(service, user, value string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.setErr != nil {
		return f.setErr
	}
	if f.entries == nil {
		f.entries = make(map[string]string)
	}
	if f.corrupt {
		value = base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{9}, 32))
	}
	f.entries[service+":"+user] = value
	return nil
}
func (f *fakeKeyring) Delete(service, user string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	delete(f.entries, service+":"+user)
	return nil
}

func fixtureStore(t *testing.T, mode string) (*ConnectionStore, *fakeKeyring) {
	t.Helper()
	dir := t.TempDir()
	fake := &fakeKeyring{}
	s := NewConnectionStore(filepath.Join(dir, "conn"), filepath.Join(dir, "secret.key"), WithKeyring(fake))
	if mode != "" {
		must(t, s.SetConfiguration("security.keyStorage", mode))
	}
	must(t, s.InitializeIfEmpty())
	must(t, s.Update(func(cf *model.ConnectionFile) error {
		return cf.AddConnection(model.SSHConnection{Username: "user", Host: "host", Alias: "prod", Password: "target-secret", ProxyJumpPassword: "jump-secret"})
	}))
	return s, fake
}
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func assertCredentials(t *testing.T, s *ConnectionStore) {
	t.Helper()
	cf, err := s.Load()
	must(t, err)
	if len(cf.Connections) != 1 || cf.Connections[0].Password != "target-secret" || cf.Connections[0].ProxyJumpPassword != "jump-secret" || cf.Connections[0].Alias != "prod" || cf.Connections[0].ID == "" {
		t.Fatalf("credentials changed: %+v", cf)
	}
}

func TestKeyStorageDefaultAndRepeatedSwitches(t *testing.T) {
	s, fake := fixtureStore(t, "")
	before, err := os.ReadFile(s.connectionFilePath)
	must(t, err)
	if _, err := os.Stat(s.secretKeyFilePath); !os.IsNotExist(err) {
		t.Fatal("default wrote a local key")
	}
	for _, mode := range []string{"file", "keyring", "file", "file", "keyring"} {
		must(t, s.SetConfiguration("security.keyStorage", mode))
		assertCredentials(t, s)
		after, err := os.ReadFile(s.connectionFilePath)
		must(t, err)
		if !bytes.Equal(before, after) {
			t.Fatal("a mode switch rewrote ciphertext")
		}
		st, err := s.readState()
		must(t, err)
		if st.Active != mode || st.Pending != nil {
			t.Fatalf("unexpected state: %+v", st)
		}
		cfg, err := config.LoadConfig(s.configPath())
		must(t, err)
		if cfg.Security.KeyStorage != mode {
			t.Fatal("config did not commit")
		}
		if mode == "file" && len(fake.entries) != 0 {
			t.Fatal("obsolete keyring entry retained")
		}
	}
}

func TestPassphraseProtectionRestored(t *testing.T) {
	t.Setenv("SSHMANAGER_MASTER_PASSPHRASE", "original passphrase")
	s, _ := fixtureStore(t, "file")
	metadata, err := os.ReadFile(s.secretKeyFilePath)
	must(t, err)
	must(t, s.SetConfiguration("security.keyStorage", "keyring"))
	t.Setenv("SSHMANAGER_MASTER_PASSPHRASE", "")
	assertCredentials(t, s)
	if err := s.SetConfiguration("security.keyStorage", "file"); err == nil {
		t.Fatal("missing passphrase accepted")
	}
	if _, err := os.Stat(s.secretKeyFilePath); !os.IsNotExist(err) {
		t.Fatal("failed switch wrote a key file")
	}
	t.Setenv("SSHMANAGER_MASTER_PASSPHRASE", "wrong")
	if err := s.SetConfiguration("security.keyStorage", "file"); err == nil {
		t.Fatal("wrong passphrase accepted")
	}
	t.Setenv("SSHMANAGER_MASTER_PASSPHRASE", "original passphrase")
	must(t, s.SetConfiguration("security.keyStorage", "file"))
	restored, err := os.ReadFile(s.secretKeyFilePath)
	must(t, err)
	if !bytes.Equal(metadata, restored) {
		t.Fatal("original passphrase metadata changed")
	}
	assertCredentials(t, s)
}

func TestKeyringFailureRetainsSourceAndCanCancel(t *testing.T) {
	for _, failure := range []string{"get", "set", "readback"} {
		t.Run(failure, func(t *testing.T) {
			s, fake := fixtureStore(t, "file")
			before, err := os.ReadFile(s.connectionFilePath)
			must(t, err)
			switch failure {
			case "get":
				fake.getErr = errors.New("locked")
			case "set":
				fake.setErr = errors.New("write denied")
			case "readback":
				fake.corrupt = true
			}
			if err := s.SetConfiguration("security.keyStorage", "keyring"); err == nil {
				t.Fatal("failure ignored")
			}
			key, err := cryptoutil.LoadExistingKey(s.secretKeyFilePath)
			must(t, err)
			_, err = cryptoutil.DecryptData(before, key)
			must(t, err)
			cfg, err := config.LoadConfig(s.configPath())
			must(t, err)
			if cfg.Security.KeyStorage != "file" {
				t.Fatal("failed switch committed config")
			}
			must(t, s.SetConfiguration("security.keyStorage", "file"))
			assertCredentials(t, s)
		})
	}
}

func TestFreshUnavailableKeyringCanSelectFile(t *testing.T) {
	dir := t.TempDir()
	fake := &fakeKeyring{getErr: errors.New("no DBus")}
	s := NewConnectionStore(filepath.Join(dir, "conn"), filepath.Join(dir, "secret.key"), WithKeyring(fake))
	if err := s.InitializeIfEmpty(); err == nil {
		t.Fatal("unavailable keyring ignored")
	}
	must(t, s.SetConfiguration("security.keyStorage", "file"))
	must(t, s.InitializeIfEmpty())
	if _, err := cryptoutil.LoadExistingKey(s.secretKeyFilePath); err != nil {
		t.Fatal(err)
	}
}

func TestMissingDBusSocketIsNotAMissingSecret(t *testing.T) {
	dir := t.TempDir()
	fake := &fakeKeyring{getErr: &os.PathError{Op: "dial", Path: "/missing/dbus", Err: os.ErrNotExist}}
	s := NewConnectionStore(filepath.Join(dir, "conn"), filepath.Join(dir, "secret.key"), WithKeyring(fake))
	if err := s.InitializeIfEmpty(); err == nil {
		t.Fatal("missing socket accepted")
	}
	st, err := s.readState()
	must(t, err)
	if st.KeyringOwned {
		t.Fatal("transport failure was treated as missing secret and triggered key creation")
	}
	must(t, s.SetConfiguration("security.keyStorage", "file"))
	must(t, s.InitializeIfEmpty())
	st, err = s.readState()
	must(t, err)
	if st.Pending != nil || st.KeyringOwned {
		t.Fatal("headless selection retained spurious keyring cleanup")
	}
	must(t, s.SetConfiguration("security.keyStorage", "file"))
	must(t, s.Clean())
}

func TestStorageSwitchReportsStagesAndFailure(t *testing.T) {
	s, fake := fixtureStore(t, "file")
	var stages []string
	s.progress = func(stage string) { stages = append(stages, stage) }
	must(t, s.SetConfiguration("security.keyStorage", "keyring"))
	output := strings.Join(stages, "\n")
	for _, expected := range []string{"Waiting for the storage lock", "Accessing OS keyring", "Saving the encryption key to OS keyring", "Verifying that the destination key", "Committing the storage mode", "Removing the obsolete file key", "Key storage is now configured to use keyring"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("missing progress stage %q in %s", expected, output)
		}
	}
	stages = nil
	fake.getErr = errors.New("locked keyring")
	if err := s.SetConfiguration("security.keyStorage", "file"); err == nil {
		t.Fatal("expected keyring failure")
	}
	output = strings.Join(stages, "\n")
	if !strings.Contains(output, "Storage change failed:") || strings.Contains(output, "now configured") {
		t.Fatalf("incorrect failure progress: %s", output)
	}
}

func TestCleanupFailureDoesNotBlockCredentials(t *testing.T) {
	s, fake := fixtureStore(t, "keyring")
	fake.deleteErr = errors.New("locked during cleanup")
	must(t, s.SetConfiguration("security.keyStorage", "file"))
	assertCredentials(t, s)
	st, err := s.readState()
	must(t, err)
	if st.Active != "file" || st.Pending == nil || st.Pending.Phase != "cleanup" {
		t.Fatal("cleanup not journaled")
	}
	must(t, s.SetConfiguration("security.keyStorage", "file"))
	fake.deleteErr = nil
	assertCredentials(t, NewConnectionStore(s.connectionFilePath, s.secretKeyFilePath, WithKeyring(fake)))
	st, err = s.readState()
	must(t, err)
	if st.Pending != nil || len(fake.entries) != 0 {
		t.Fatal("cleanup retry failed")
	}
}

func TestMigrationRestartAtEachDurablePhase(t *testing.T) {
	for _, from := range []string{"file", "keyring"} {
		for _, phase := range []string{"copy", "commit", "cleanup"} {
			t.Run(from+"/"+phase, func(t *testing.T) {
				s, fake := fixtureStore(t, from)
				to := "keyring"
				if from == "keyring" {
					to = "file"
				}
				st, err := s.readState()
				must(t, err)
				cfg, err := config.LoadConfig(s.configPath())
				must(t, err)
				cfg.Security.KeyStorage = to
				st.Pending = &keyTransaction{From: from, To: to, Phase: phase, Config: cfg}
				if phase != "copy" {
					key, err := s.existingKey(st, from)
					must(t, err)
					must(t, s.copyKey(st, key, to))
					st.KeyringOwned = true
				}
				if phase == "cleanup" {
					st.Active = to
					must(t, config.SaveConfig(s.configPath(), cfg))
				}
				must(t, s.writeState(st))
				reopened := NewConnectionStore(s.connectionFilePath, s.secretKeyFilePath, WithKeyring(fake))
				assertCredentials(t, reopened)
				st, err = s.readState()
				must(t, err)
				if st.Active != to || st.Pending != nil {
					t.Fatal("restart did not complete migration")
				}
			})
		}
	}
}

func TestManualConfigEditMigrates(t *testing.T) {
	s, _ := fixtureStore(t, "file")
	cfg := config.Default()
	must(t, config.SaveConfig(s.configPath(), cfg))
	assertCredentials(t, s)
	if _, err := os.Stat(s.secretKeyFilePath); !os.IsNotExist(err) {
		t.Fatal("legacy key not removed")
	}
}

func TestMissingOrConflictingKeysNeverOverwriteData(t *testing.T) {
	for _, failure := range []string{"missing-file", "missing-keyring", "missing-state", "corrupt-state", "conflict", "corrupt-ciphertext"} {
		t.Run(failure, func(t *testing.T) {
			mode := "keyring"
			if failure == "missing-file" || failure == "conflict" {
				mode = "file"
			}
			s, fake := fixtureStore(t, mode)
			st, err := s.readState()
			must(t, err)
			switch failure {
			case "missing-file":
				must(t, os.Remove(s.secretKeyFilePath))
			case "missing-keyring":
				must(t, fake.Delete(keyringService, account(st)))
			case "missing-state":
				must(t, os.Remove(s.statePath()))
			case "corrupt-state":
				must(t, storage.WriteFileAtomic(s.statePath(), []byte("broken"), 0o600))
			case "conflict":
				must(t, fake.Set(keyringService, account(st), base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))))
			case "corrupt-ciphertext":
				must(t, storage.WriteFileAtomic(s.connectionFilePath, []byte("invalid ciphertext"), 0o600))
			}
			before, err := os.ReadFile(s.connectionFilePath)
			must(t, err)
			if failure == "conflict" {
				err = s.SetConfiguration("security.keyStorage", "keyring")
			} else {
				err = s.Save(model.NewConnectionFile())
			}
			if err == nil {
				t.Fatal("unsafe operation succeeded")
			}
			after, err := os.ReadFile(s.connectionFilePath)
			must(t, err)
			if !bytes.Equal(before, after) {
				t.Fatal("ciphertext replaced on failure")
			}
		})
	}
}

func TestRestoreRestartAndUnavailableDestination(t *testing.T) {
	s, fake := fixtureStore(t, "file")
	before, err := os.ReadFile(s.connectionFilePath)
	must(t, err)
	fake.setErr = errors.New("keyring write denied")
	cfg := config.Default()
	cfg.Behaviour.ContinueAfterSSHExit = true
	err = s.Restore(func(cf *model.ConnectionFile) error { cf.Connections[0].Password = "restored-secret"; return nil }, &cfg)
	if err == nil {
		t.Fatal("failed destination accepted")
	}
	after, err := os.ReadFile(s.connectionFilePath)
	must(t, err)
	if !bytes.Equal(before, after) {
		t.Fatal("failed restore changed existing data")
	}
	fake.setErr = nil
	reopened := NewConnectionStore(s.connectionFilePath, s.secretKeyFilePath, WithKeyring(fake))
	cf, err := reopened.Load()
	must(t, err)
	if cf.Connections[0].Password != "restored-secret" {
		t.Fatal("pending restore not recovered")
	}
	got, err := config.LoadConfig(s.configPath())
	must(t, err)
	if got != cfg {
		t.Fatal("restored configuration not committed")
	}
	if _, err := os.Stat(s.restorePath()); !os.IsNotExist(err) {
		t.Fatal("restore ciphertext not cleaned")
	}
}

func TestInspectDoesNotCreateOrMigrate(t *testing.T) {
	s, fake := fixtureStore(t, "file")
	must(t, config.SaveConfig(s.configPath(), config.Default()))
	before, err := os.ReadFile(s.statePath())
	must(t, err)
	_, backend, pending, err := s.Inspect()
	must(t, err)
	if backend != "file" || !pending || len(fake.entries) != 0 {
		t.Fatal("inspect changed storage")
	}
	after, err := os.ReadFile(s.statePath())
	must(t, err)
	if !bytes.Equal(before, after) {
		t.Fatal("inspect rewrote state")
	}
}

func TestCleanRetryAndStoreIsolation(t *testing.T) {
	s, fake := fixtureStore(t, "keyring")
	other := NewConnectionStore(filepath.Join(t.TempDir(), "conn"), filepath.Join(t.TempDir(), "key"), WithKeyring(fake))
	must(t, other.InitializeIfEmpty())
	otherState, err := other.readState()
	must(t, err)
	fake.deleteErr = errors.New("locked")
	if err := s.Clean(); err == nil {
		t.Fatal("cleanup failure ignored")
	}
	if err := s.InitializeIfEmpty(); err == nil {
		t.Fatal("deletion-pending store recreated credentials")
	}
	fake.deleteErr = nil
	must(t, s.Clean())
	if _, err := fake.Get(keyringService, account(otherState)); err != nil {
		t.Fatal("another store's key deleted")
	}
	for _, path := range []string{s.connectionFilePath, s.secretKeyFilePath, s.statePath()} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("not deleted: %s", path)
		}
	}
}

func TestFileOnlyCleanDoesNotNeedKeyring(t *testing.T) {
	s, fake := fixtureStore(t, "file")
	fake.deleteErr = errors.New("no desktop keyring")
	must(t, s.Clean())
}

func TestConcurrentMigrationAndMutations(t *testing.T) {
	s, _ := fixtureStore(t, "file")
	var wg sync.WaitGroup
	errs := make(chan error, 12)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- s.Update(func(cf *model.ConnectionFile) error { cf.Connections[0].Port++; return nil })
		}()
	}
	wg.Add(1)
	go func() { defer wg.Done(); errs <- s.SetConfiguration("security.keyStorage", "keyring") }()
	wg.Wait()
	close(errs)
	for err := range errs {
		must(t, err)
	}
	cf, err := s.Load()
	must(t, err)
	if cf.Connections[0].Port != 10 {
		t.Fatalf("lost updates: port=%d", cf.Connections[0].Port)
	}
	assertCredentials(t, s)
}

func TestLegacyPassphraseAutomaticallyMigrates(t *testing.T) {
	t.Setenv("SSHMANAGER_MASTER_PASSPHRASE", "legacy passphrase")
	dir := t.TempDir()
	fake := &fakeKeyring{}
	s := NewConnectionStore(filepath.Join(dir, "conn"), filepath.Join(dir, "secret.key"), WithKeyring(fake))
	key, err := cryptoutil.LoadKey(s.secretKeyFilePath)
	must(t, err)
	metadata, err := os.ReadFile(s.secretKeyFilePath)
	must(t, err)
	must(t, encryptAndStoreFile("- username: user\n  host: host\n  alias: prod\n  password: target-secret\n  proxyJumpPassword: jump-secret\n", s.connectionFilePath, key))
	assertCredentials(t, s)
	st, err := s.readState()
	must(t, err)
	if st.Active != "keyring" || !bytes.Equal(st.Passphrase, metadata) {
		t.Fatal("legacy passphrase metadata lost")
	}
	must(t, s.SetConfiguration("security.keyStorage", "file"))
	assertCredentials(t, s)
}

func TestPersistenceFailuresKeepSourceUntilRestart(t *testing.T) {
	for _, failure := range []string{"journal", "config"} {
		t.Run(failure, func(t *testing.T) {
			s, fake := fixtureStore(t, "file")
			st, err := s.readState()
			must(t, err)
			cfg := config.Default()
			st.Pending = &keyTransaction{From: "file", To: "keyring", Phase: "copy", Config: cfg}
			must(t, s.writeState(st))
			if failure == "journal" {
				must(t, os.Rename(s.statePath(), s.statePath()+".saved"))
				must(t, os.Mkdir(s.statePath(), 0o700))
			} else {
				s.configFilePath = filepath.Join(t.TempDir(), "directory")
				must(t, os.Mkdir(s.configFilePath, 0o700))
			}
			if err := s.resume(&st); err == nil {
				t.Fatal("persistence failure ignored")
			}
			key, err := cryptoutil.LoadExistingKey(s.secretKeyFilePath)
			must(t, err)
			must(t, s.verifyKey(key))
			if failure == "journal" {
				must(t, os.Remove(s.statePath()))
				must(t, os.Rename(s.statePath()+".saved", s.statePath()))
			}
			reopened := NewConnectionStore(s.connectionFilePath, s.secretKeyFilePath, WithKeyring(fake))
			assertCredentials(t, reopened)
			st, err = reopened.readState()
			must(t, err)
			if st.Active != "keyring" || st.Pending != nil {
				t.Fatal("restart did not recover failed persistence")
			}
		})
	}
}

func TestCancellingUncommittedRestorePreservesOriginalData(t *testing.T) {
	s, fake := fixtureStore(t, "file")
	fake.setErr = errors.New("no keyring")
	cfg := config.Default()
	if err := s.Restore(func(cf *model.ConnectionFile) error { cf.Connections[0].Password = "replacement"; return nil }, &cfg); err == nil {
		t.Fatal("restore unexpectedly succeeded")
	}
	must(t, s.SetConfiguration("security.keyStorage", "file"))
	assertCredentials(t, s)
	if _, err := os.Stat(s.restorePath()); !os.IsNotExist(err) {
		t.Fatal("cancelled restore retained staging data")
	}
}
