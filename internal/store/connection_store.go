package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emirhangumus/sshmanager/internal/model"
	"github.com/emirhangumus/sshmanager/internal/storage"
	"github.com/gofrs/flock"
)

type ConnectionStore struct {
	connectionFilePath string
	secretKeyFilePath  string
	keyring            Keyring
	configFilePath     string
	progress           func(string)
}

var (
	connectionLockTimeout       = 5 * time.Second
	connectionLockRetryInterval = 50 * time.Millisecond
)

// NewConnectionStore coordinates encrypted data and its configured key backend.
func NewConnectionStore(connectionFilePath, secretKeyFilePath string, options ...func(*ConnectionStore)) *ConnectionStore {
	s := &ConnectionStore{
		connectionFilePath: connectionFilePath,
		secretKeyFilePath:  secretKeyFilePath,
		keyring:            systemKeyring{},
	}
	for _, option := range options {
		option(s)
	}
	return s
}

func (s *ConnectionStore) InitializeIfEmpty() error {
	unlock, err := s.acquireMutationLock()
	if err != nil {
		return err
	}
	defer unlock()
	if _, err := s.keyWithoutLock(); err != nil {
		return err
	}
	isEmpty, err := storage.IsFileEmpty(s.connectionFilePath)
	if err != nil {
		return err
	}
	if !isEmpty {
		return nil
	}
	return s.saveWithoutLock(model.NewConnectionFile())
}

func (s *ConnectionStore) Load() (model.ConnectionFile, error) {
	unlock, err := s.acquireMutationLock()
	if err != nil {
		return model.ConnectionFile{}, err
	}
	defer unlock()
	return s.loadWithoutLock()
}

func (s *ConnectionStore) Save(connFile model.ConnectionFile) error {
	unlock, err := s.acquireMutationLock()
	if err != nil {
		return err
	}
	defer unlock()

	return s.saveWithoutLock(connFile)
}

// Update executes an in-place mutation under a process lock and persists it.
func (s *ConnectionStore) Update(mutator func(*model.ConnectionFile) error) error {
	unlock, err := s.acquireMutationLock()
	if err != nil {
		return err
	}
	defer unlock()

	connFile, err := s.loadWithoutLock()
	if err != nil {
		return err
	}

	if err := mutator(&connFile); err != nil {
		return err
	}
	return s.saveWithoutLock(connFile)
}

func (s *ConnectionStore) loadWithoutLock() (model.ConnectionFile, error) {
	key, err := s.keyWithoutLock()
	if err != nil {
		return model.ConnectionFile{}, err
	}

	content, err := decryptAndReadFile(s.connectionFilePath, key)
	if os.IsNotExist(err) {
		content, err = "", nil
	}
	if err != nil {
		return model.ConnectionFile{}, err
	}

	connFile, err := parseConnectionFile(content)
	if err != nil {
		return model.ConnectionFile{}, err
	}

	changed := connFile.EnsureIDs()
	if changed {
		if err := s.saveWithoutLock(connFile); err != nil {
			return model.ConnectionFile{}, err
		}
	}

	return connFile, nil
}

func (s *ConnectionStore) saveWithoutLock(connFile model.ConnectionFile) error {
	if strings.TrimSpace(connFile.Version) == "" {
		connFile.Version = model.CurrentConnectionFileVersion
	}
	connFile.EnsureIDs()

	key, err := s.keyWithoutLock()
	if err != nil {
		return err
	}
	if err := s.verifyKey(key); err != nil {
		return err
	}

	contentStr, err := toYAMLString(connFile)
	if err != nil {
		return err
	}

	if err := encryptAndStoreFile(contentStr, s.connectionFilePath, key); err != nil {
		return err
	}
	return nil
}

func (s *ConnectionStore) acquireMutationLock() (func(), error) {
	lockPath := s.connectionFilePath + ".lock"
	lockDir := filepath.Dir(lockPath)
	if err := os.MkdirAll(lockDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	deadline := time.Now().Add(connectionLockTimeout)
	lockFile := flock.New(lockPath, flock.SetPermissions(0o600))
	for {
		locked, err := lockFile.TryLock()
		if err != nil {
			_ = lockFile.Close()
			return nil, fmt.Errorf("failed to acquire mutation lock: %w", err)
		}
		if locked {
			return func() { _ = lockFile.Close() }, nil
		}

		if time.Now().After(deadline) {
			_ = lockFile.Close()
			return nil, fmt.Errorf("timed out acquiring mutation lock %s", lockPath)
		}
		time.Sleep(connectionLockRetryInterval)
	}
}

func parseConnectionFile(content string) (model.ConnectionFile, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return model.NewConnectionFile(), nil
	}

	var schema model.ConnectionFile
	if err := fromYAMLString(content, &schema); err == nil {
		if strings.TrimSpace(schema.Version) == "" {
			schema.Version = model.CurrentConnectionFileVersion
		}
		if schema.Connections == nil {
			schema.Connections = []model.SSHConnection{}
		}
		return schema, nil
	}

	var legacy []model.SSHConnection
	if err := fromYAMLString(content, &legacy); err == nil {
		return model.ConnectionFile{
			Version:     model.CurrentConnectionFileVersion,
			Connections: legacy,
		}, nil
	}

	return model.ConnectionFile{}, fmt.Errorf("failed to parse connection file: unsupported YAML schema")
}
