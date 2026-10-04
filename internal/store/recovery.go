package store

import (
	"errors"
	"os"

	"github.com/emirhangumus/sshmanager/internal/config"
	cryptoutil "github.com/emirhangumus/sshmanager/internal/crypto"
	"github.com/emirhangumus/sshmanager/internal/model"
	"github.com/emirhangumus/sshmanager/internal/storage"
	"github.com/zalando/go-keyring"
)

// Restore journals ciphertext and configuration as one recoverable operation.
func (s *ConnectionStore) Restore(mutator func(*model.ConnectionFile) error, cfg *config.SSHManagerConfig) error {
	if cfg != nil {
		if err := config.Validate(*cfg); err != nil {
			return err
		}
	}
	unlock, err := s.acquireMutationLock()
	if err != nil {
		return err
	}
	defer unlock()
	current, err := s.loadWithoutLock()
	if err != nil {
		return err
	}
	if err := mutator(&current); err != nil {
		return err
	}
	current.EnsureIDs()
	if cfg == nil {
		return s.saveWithoutLock(current)
	}
	key, err := s.keyWithoutLock()
	if err != nil {
		return err
	}
	st, err := s.readState()
	if err != nil {
		return err
	}
	if st.Pending != nil {
		return errors.New("complete pending cleanup before restoring configuration")
	}
	content, err := toYAMLString(current)
	if err != nil {
		return err
	}
	ciphertext, err := cryptoutil.EncryptData(content, key)
	if err != nil {
		return err
	}
	// Destination preparation is journaled before touching either backend.
	if err := storage.WriteFileAtomic(s.restorePath(), ciphertext, 0o600); err != nil {
		return err
	}
	st.Pending = &keyTransaction{From: st.Active, To: cfg.Security.KeyStorage, Phase: "copy", Config: *cfg, Restore: true}
	if err := s.writeState(st); err != nil {
		return err
	}
	return s.resume(&st)
}

// Inspect decrypts without creating keys, reconciling config, or rewriting IDs.
func (s *ConnectionStore) Inspect() (model.ConnectionFile, string, bool, error) {
	unlock, err := s.acquireMutationLock()
	if err != nil {
		return model.ConnectionFile{}, "", false, err
	}
	defer unlock()
	cfg, err := config.LoadConfig(s.configPath())
	if err != nil {
		return model.ConnectionFile{}, "", false, err
	}
	st, err := s.readState()
	if errors.Is(err, os.ErrNotExist) {
		key, err := cryptoutil.LoadExistingKey(s.secretKeyFilePath)
		if err != nil {
			return model.ConnectionFile{}, "file", cfg.Security.KeyStorage != "file", err
		}
		content, err := decryptAndReadFile(s.connectionFilePath, key)
		if err != nil {
			return model.ConnectionFile{}, "file", cfg.Security.KeyStorage != "file", err
		}
		file, err := parseConnectionFile(content)
		return file, "file", cfg.Security.KeyStorage != "file", err
	}
	if err != nil {
		return model.ConnectionFile{}, "", false, err
	}
	key, err := s.existingKey(st, st.Active)
	if err != nil {
		return model.ConnectionFile{}, st.Active, st.Pending != nil, err
	}
	content, err := decryptAndReadFile(s.connectionFilePath, key)
	if err != nil {
		return model.ConnectionFile{}, st.Active, st.Pending != nil, err
	}
	file, err := parseConnectionFile(content)
	return file, st.Active, st.Pending != nil || st.Active != cfg.Security.KeyStorage, err
}

// Clean removes data before its keys and keeps identifiers until cleanup succeeds.
func (s *ConnectionStore) Clean() error {
	unlock, err := s.acquireMutationLock()
	if err != nil {
		return err
	}
	defer unlock()
	st, err := s.readState()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if st.StoreID != "" {
		st.Deleting = true
		if err := s.writeState(st); err != nil {
			return err
		}
	}
	if err := storage.SecureDelete(s.connectionFilePath); err != nil {
		return err
	}
	if err := storage.SecureDelete(s.restorePath()); err != nil {
		return err
	}
	if st.StoreID != "" && st.KeyringOwned {
		if err := s.keyring.Delete(keyringService, account(st)); err != nil && !errors.Is(err, keyring.ErrNotFound) {
			return err
		}
	}
	if err := storage.SecureDelete(s.secretKeyFilePath); err != nil {
		return err
	}
	return storage.SecureDelete(s.statePath())
}
