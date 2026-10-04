package store

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/emirhangumus/sshmanager/v2/internal/config"
	cryptoutil "github.com/emirhangumus/sshmanager/v2/internal/crypto"
	"github.com/emirhangumus/sshmanager/v2/internal/storage"
	"github.com/zalando/go-keyring"
)

const keyringService = "sshmanager"

// Keyring permits isolated providers in tests and never enumerates other stores' secrets.
type Keyring interface {
	Get(service, account string) (string, error)
	Set(service, account, secret string) error
	Delete(service, account string) error
}

type systemKeyring struct{}

func (systemKeyring) Get(s, a string) (string, error) { return keyring.Get(s, a) }
func (systemKeyring) Set(s, a, v string) error        { return keyring.Set(s, a, v) }
func (systemKeyring) Delete(s, a string) error        { return keyring.Delete(s, a) }

type keyState struct {
	Version      int             `yaml:"version"`
	StoreID      string          `yaml:"storeID"`
	Active       string          `yaml:"activeBackend"`
	Initialized  bool            `yaml:"initialized"`
	KeyDigest    string          `yaml:"keyDigest,omitempty"`
	KeyringOwned bool            `yaml:"keyringOwned,omitempty"`
	Deleting     bool            `yaml:"deleting,omitempty"`
	Passphrase   []byte          `yaml:"passphraseMetadata,omitempty"`
	Pending      *keyTransaction `yaml:"pending,omitempty"`
}

type keyTransaction struct {
	From             string                  `yaml:"from"`
	To               string                  `yaml:"to"`
	Phase            string                  `yaml:"phase"`
	Config           config.SSHManagerConfig `yaml:"config"`
	Restore          bool                    `yaml:"restore,omitempty"`
	DestinationOwned bool                    `yaml:"destinationOwned,omitempty"`
	DiscardRestore   bool                    `yaml:"discardRestore,omitempty"`
}

// WithKeyring injects a provider without changing the process-wide OS keyring.
func WithKeyring(provider Keyring) func(*ConnectionStore) {
	return func(s *ConnectionStore) { s.keyring = provider }
}

// WithConfigPath overrides the config file beside the connection file.
func WithConfigPath(path string) func(*ConnectionStore) {
	return func(s *ConnectionStore) { s.configFilePath = path }
}

// WithProgress reports migration stages before potentially blocking operations.
func WithProgress(report func(string)) func(*ConnectionStore) {
	return func(s *ConnectionStore) { s.progress = report }
}

func (s *ConnectionStore) report(message string) {
	if s.progress != nil {
		s.progress(message)
	}
}

func (s *ConnectionStore) configPath() string {
	if s.configFilePath != "" {
		return s.configFilePath
	}
	return filepath.Join(filepath.Dir(s.connectionFilePath), "config.yaml")
}
func (s *ConnectionStore) statePath() string {
	return filepath.Join(filepath.Dir(s.connectionFilePath), "key-storage.yaml")
}
func (s *ConnectionStore) restorePath() string { return s.connectionFilePath + ".restore-pending" }
func account(st keyState) string               { return "aes-key:" + st.StoreID }
func (s *ConnectionStore) writeState(st keyState) error {
	return storage.WriteYAMLFile(s.statePath(), st, 0o600)
}

func (s *ConnectionStore) readState() (keyState, error) {
	var st keyState
	if err := storage.ReadYAMLFile(s.statePath(), &st); err != nil {
		return st, err
	}
	id, err := hex.DecodeString(st.StoreID)
	if st.Version != 1 || err != nil || len(id) != 16 || (st.Active != "file" && st.Active != "keyring") {
		return st, errors.New("invalid key storage state; restore the original state or a recovery backup")
	}
	if st.Initialized {
		digest, err := hex.DecodeString(st.KeyDigest)
		if err != nil || len(digest) != sha256.Size {
			return st, errors.New("invalid key fingerprint in storage state")
		}
	}
	if len(st.Passphrase) != 0 {
		// Metadata must never contain a raw encryption key.
		if cryptoutil.ValidatePassphraseMetadata(st.Passphrase) != nil {
			return st, errors.New("invalid preserved passphrase metadata")
		}
	}
	if p := st.Pending; p != nil {
		if (p.From != "file" && p.From != "keyring") || (p.To != "file" && p.To != "keyring") || config.Validate(p.Config) != nil || p.Config.Security.KeyStorage != p.To {
			return st, errors.New("invalid migration journal")
		}
		switch p.Phase {
		case "copy", "commit", "cleanup":
		default:
			return st, errors.New("invalid migration phase")
		}
	}
	return st, nil
}

func (s *ConnectionStore) existingKey(st keyState, backend string) ([]byte, error) {
	key, err := s.readBackendKey(st, backend)
	if err != nil {
		return nil, err
	}
	if st.Initialized && keyDigest(key) != st.KeyDigest {
		return nil, errors.New("stored key does not match the original encryption key fingerprint")
	}
	return key, nil
}

func keyDigest(key []byte) string {
	digest := sha256.Sum256(key)
	return hex.EncodeToString(digest[:])
}

// A missing DBus socket can wrap os.ErrNotExist, but is not a missing secret.
func missingKey(backend string, err error) bool {
	if backend == "keyring" {
		return errors.Is(err, keyring.ErrNotFound)
	}
	return errors.Is(err, os.ErrNotExist)
}

func (s *ConnectionStore) readBackendKey(st keyState, backend string) ([]byte, error) {
	if backend == "file" {
		s.report("Reading the local encryption key")
		return cryptoutil.LoadExistingKey(s.secretKeyFilePath)
	}
	s.report("Accessing OS keyring (approve or unlock it if prompted)")
	encoded, err := s.keyring.Get(keyringService, account(st))
	if err != nil {
		return nil, fmt.Errorf("OS keyring: %w; unlock the keyring, or for a fresh/legacy file store run sshmanager set security.keyStorage file (a keyring-only store requires its original key)", err)
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(key) != 32 {
		return nil, errors.New("invalid AES-256 key in OS keyring")
	}
	return key, nil
}

func (s *ConnectionStore) verifyKey(key []byte) error {
	empty, err := storage.IsFileEmpty(s.connectionFilePath)
	if err != nil || empty {
		return err
	}
	_, err = decryptAndReadFile(s.connectionFilePath, key)
	return err
}

func (s *ConnectionStore) bootstrap(cfg config.SSHManagerConfig) (keyState, error) {
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return keyState{}, err
	}
	st := keyState{Version: 1, StoreID: hex.EncodeToString(id), Active: cfg.Security.KeyStorage}
	if data, err := os.ReadFile(s.secretKeyFilePath); err == nil {
		key, err := cryptoutil.DecodeKeyFile(data)
		if err != nil {
			return st, err
		}
		if err := s.verifyKey(key); err != nil {
			return st, err
		}
		st.Active, st.Initialized = "file", true
		st.KeyDigest = keyDigest(key)
		if len(data) != 32 {
			st.Passphrase = data
		}
	} else if !os.IsNotExist(err) {
		return st, err
	} else {
		empty, err := storage.IsFileEmpty(s.connectionFilePath)
		if err != nil {
			return st, err
		}
		if !empty {
			return st, errors.New("encrypted connections exist but key storage state and legacy key are missing; restore the original key/state or a backup")
		}
	}
	return st, s.writeState(st)
}

func (s *ConnectionStore) ensureState(cfg config.SSHManagerConfig) (keyState, error) {
	st, err := s.readState()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return st, err
		}
		st, err = s.bootstrap(cfg)
		if err != nil {
			return st, err
		}
	}
	if st.Deleting {
		return st, errors.New("credential deletion is pending; run sshmanager clean again before accessing this store")
	}
	if !st.Initialized {
		empty, err := storage.IsFileEmpty(s.connectionFilePath)
		if err != nil {
			return st, err
		}
		if !empty {
			return st, errors.New("uninitialized key state with existing ciphertext; refusing to create a replacement key")
		}
		key, err := s.existingKey(st, st.Active)
		if err != nil {
			if !missingKey(st.Active, err) {
				return st, err
			}
			if st.Active == "file" {
				key, err = cryptoutil.LoadKey(s.secretKeyFilePath)
				if err == nil {
					data, readErr := os.ReadFile(s.secretKeyFilePath)
					if readErr != nil {
						return st, readErr
					}
					if len(data) != 32 {
						st.Passphrase = data
					}
				}
			} else {
				st.KeyringOwned = true
				if err := s.writeState(st); err != nil {
					return st, err
				}
				key = make([]byte, 32)
				if _, err = rand.Read(key); err == nil {
					err = s.keyring.Set(keyringService, account(st), base64.StdEncoding.EncodeToString(key))
				}
			}
			if err != nil {
				return st, fmt.Errorf("initialize %s storage: %w; for headless use sshmanager set security.keyStorage file", st.Active, err)
			}
		}
		readback, err := s.existingKey(st, st.Active)
		if err != nil {
			return st, err
		}
		if !bytes.Equal(key, readback) {
			return st, errors.New("new key readback mismatch")
		}
		st.Initialized = true
		st.KeyDigest = keyDigest(key)
		if st.Active == "file" && st.KeyringOwned && st.Pending == nil {
			cleanupCfg := cfg
			cleanupCfg.Security.KeyStorage = "file"
			st.Pending = &keyTransaction{From: "keyring", To: "file", Phase: "cleanup", Config: cleanupCfg}
		}
		if err := s.writeState(st); err != nil {
			return st, err
		}
	}
	return st, nil
}

func (s *ConnectionStore) copyKey(st keyState, key []byte, target string) error {
	existing, err := s.existingKey(st, target)
	if err == nil {
		if !bytes.Equal(existing, key) {
			return errors.New("destination contains a different key; refusing to overwrite it")
		}
		return nil
	}
	if !missingKey(target, err) {
		return err
	}
	if target == "keyring" {
		s.report("Saving the encryption key to OS keyring")
		return s.keyring.Set(keyringService, account(st), base64.StdEncoding.EncodeToString(key))
	}
	data := key
	if len(st.Passphrase) != 0 {
		s.report("Verifying the original master passphrase")
		derived, err := cryptoutil.DecodeKeyFile(st.Passphrase)
		if err != nil {
			return err
		}
		if !bytes.Equal(derived, key) {
			return errors.New("master passphrase does not match the preserved key")
		}
		data = st.Passphrase
	}
	s.report("Saving the local encryption key")
	return storage.WriteFileAtomic(s.secretKeyFilePath, data, 0o600)
}

// resume finishes a journal under the same lock used by connection operations.
func (s *ConnectionStore) resume(st *keyState) error {
	p := st.Pending
	if p == nil {
		return nil
	}
	if p.Phase == "copy" {
		key, err := s.existingKey(*st, p.From)
		if err != nil {
			return err
		}
		s.report("Verifying that the source key decrypts saved connections")
		if err := s.verifyKey(key); err != nil {
			return err
		}
		if !p.DestinationOwned {
			_, err := s.existingKey(*st, p.To)
			if err != nil && !missingKey(p.To, err) {
				return err
			}
			p.DestinationOwned = true
			if p.To == "keyring" {
				st.KeyringOwned = true
			}
			if err := s.writeState(*st); err != nil {
				return err
			}
		}
		if err := s.copyKey(*st, key, p.To); err != nil {
			return err
		}
		readback, err := s.existingKey(*st, p.To)
		if err != nil {
			return err
		}
		if !bytes.Equal(key, readback) {
			return errors.New("destination key readback mismatch")
		}
		s.report("Verifying that the destination key decrypts saved connections")
		if err := s.verifyKey(readback); err != nil {
			return err
		}
		p.Phase = "commit"
		if err := s.writeState(*st); err != nil {
			return err
		}
	}
	if p.Phase == "commit" {
		key, err := s.existingKey(*st, p.To)
		if err != nil {
			return err
		}
		if err := s.verifyKey(key); err != nil {
			return err
		}
		if p.Restore {
			data, err := os.ReadFile(s.restorePath())
			if err != nil {
				return err
			}
			if _, err := cryptoutil.DecryptData(data, key); err != nil {
				return err
			}
			if err := storage.WriteFileAtomic(s.connectionFilePath, data, 0o600); err != nil {
				return err
			}
		}
		s.report("Committing the storage mode and configuration")
		if err := config.SaveConfig(s.configPath(), p.Config); err != nil {
			return err
		}
		st.Active, p.Phase = p.To, "cleanup"
		if err := s.writeState(*st); err != nil {
			return err
		}
	}
	// Verify the surviving key before deleting the source, even after a restart.
	key, err := s.existingKey(*st, st.Active)
	if err != nil {
		return err
	}
	if err := s.verifyKey(key); err != nil {
		return err
	}
	if p.From != p.To {
		s.report("Removing the obsolete " + p.From + " key")
		if p.From == "file" {
			err = storage.SecureDelete(s.secretKeyFilePath)
		} else {
			err = s.keyring.Delete(keyringService, account(*st))
		}
		if err != nil && !errors.Is(err, keyring.ErrNotFound) {
			fmt.Fprintf(os.Stderr, "Storage migration committed; source cleanup pending: %v\n", err)
			return nil
		}
		if p.From == "keyring" {
			st.KeyringOwned = false
		}
	}
	if p.Restore || p.DiscardRestore {
		if err := storage.SecureDelete(s.restorePath()); err != nil {
			fmt.Fprintf(os.Stderr, "Restore committed; staged ciphertext cleanup pending: %v\n", err)
			return nil
		}
	}
	st.Pending = nil
	return s.writeState(*st)
}

func (s *ConnectionStore) keyWithoutLock() ([]byte, error) {
	cfg, err := config.LoadConfig(s.configPath())
	if err != nil {
		return nil, err
	}
	st, err := s.ensureState(cfg)
	if err != nil {
		return nil, err
	}
	if st.Pending != nil {
		phase := st.Pending.Phase
		if err := s.resume(&st); err != nil {
			return nil, err
		}
		if phase != "cleanup" {
			cfg, err = config.LoadConfig(s.configPath())
			if err != nil {
				return nil, err
			}
		}
	}
	if st.Active != cfg.Security.KeyStorage {
		if st.Pending != nil {
			return nil, errors.New("complete pending source cleanup before switching storage again")
		}
		st.Pending = &keyTransaction{From: st.Active, To: cfg.Security.KeyStorage, Phase: "copy", Config: cfg}
		if err := s.writeState(st); err != nil {
			return nil, err
		}
		if err := s.resume(&st); err != nil {
			return nil, err
		}
	}
	return s.existingKey(st, st.Active)
}

// SetConfiguration keeps ordinary settings usable without initializing a keyring.
func (s *ConnectionStore) SetConfiguration(name, value string) (err error) {
	if name == "security.keyStorage" {
		s.report("Waiting for the storage lock")
		defer func() {
			if err != nil {
				s.report("Storage change failed: " + err.Error())
				return
			}
			st, stateErr := s.readState()
			if stateErr == nil && st.Pending != nil {
				s.report("Storage mode is " + st.Active + "; source cleanup is pending")
			} else {
				s.report("Key storage is now configured to use " + value)
			}
		}()
	}
	unlock, err := s.acquireMutationLock()
	if err != nil {
		return err
	}
	defer unlock()
	if name != "security.keyStorage" {
		st, err := s.readState()
		if err == nil && st.Pending != nil {
			if err := s.resume(&st); err != nil {
				return err
			}
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return config.SetConfig(s.configPath(), name, value)
	}
	if value != "file" && value != "keyring" {
		return errors.New("security.keyStorage must be keyring or file")
	}
	s.report("Inspecting current storage and migration state")
	cfg, err := config.LoadConfig(s.configPath())
	if err != nil {
		return err
	}
	cfg.Security.KeyStorage = value
	// A fresh installation can choose file mode even after a failed keyring initialization.
	st, stateErr := s.readState()
	if errors.Is(stateErr, os.ErrNotExist) {
		st, stateErr = s.bootstrap(cfg)
	}
	if stateErr != nil {
		return stateErr
	}
	if st.Deleting {
		return errors.New("credential deletion is pending; run sshmanager clean again")
	}
	if !st.Initialized {
		if st.Active != value {
			// Keep ownership metadata even if a failed initialization is inaccessible.
			key, err := s.existingKey(st, st.Active)
			if err == nil {
				st.Initialized = true
				st.KeyDigest = keyDigest(key)
				if err := s.verifyKey(key); err != nil {
					return err
				}
			} else if missingKey(st.Active, err) || st.Active == "keyring" {
				empty, inspectErr := storage.IsFileEmpty(s.connectionFilePath)
				if inspectErr != nil {
					return inspectErr
				}
				if !empty {
					return errors.New("cannot change uninitialized storage with existing ciphertext")
				}
				st.Active = value
			} else {
				return err
			}
		}
		if err := s.writeState(st); err != nil {
			return err
		}
		if !st.Initialized {
			return config.SaveConfig(s.configPath(), cfg)
		}
	}
	if st.Pending != nil {
		// An uncommitted automatic migration can be cancelled back to its source.
		if st.Pending.Phase == "copy" && st.Pending.From == value {
			key, err := s.existingKey(st, value)
			if err != nil {
				return err
			}
			if err := s.verifyKey(key); err != nil {
				return err
			}
			wasRestore := st.Pending.Restore
			if st.Pending.DestinationOwned {
				// Commit a reversal so failed destination cleanup remains recoverable.
				st.Pending = &keyTransaction{From: st.Pending.To, To: value, Phase: "commit", Config: cfg, DiscardRestore: wasRestore}
				if err := s.writeState(st); err != nil {
					return err
				}
				return s.resume(&st)
			}
			st.Pending = nil
			if err := s.writeState(st); err != nil {
				return err
			}
			if wasRestore {
				if err := storage.SecureDelete(s.restorePath()); err != nil {
					return err
				}
			}
		} else {
			if err := s.resume(&st); err != nil {
				return err
			}
			if st.Pending != nil {
				if st.Active == value {
					return config.SaveConfig(s.configPath(), cfg)
				}
				return errors.New("source cleanup still pending")
			}
			cfg, err = config.LoadConfig(s.configPath())
			if err != nil {
				return err
			}
			cfg.Security.KeyStorage = value
		}
	}
	if st.Active == value {
		return config.SaveConfig(s.configPath(), cfg)
	}
	st.Pending = &keyTransaction{From: st.Active, To: value, Phase: "copy", Config: cfg}
	if err := s.writeState(st); err != nil {
		return err
	}
	return s.resume(&st)
}
