package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	cryptoutil "github.com/emirhangumus/sshmanager/internal/crypto"
	"github.com/emirhangumus/sshmanager/internal/model"
	"github.com/emirhangumus/sshmanager/internal/storage"
)

type ConnectionStore struct {
	connectionFilePath string
	secretKeyFilePath  string
}

var (
	connectionLockTimeout       = 5 * time.Second
	connectionLockRetryInterval = 50 * time.Millisecond
	connectionLockStaleAfter    = 2 * time.Minute
)

func NewConnectionStore(connectionFilePath, secretKeyFilePath string) *ConnectionStore {
	return &ConnectionStore{
		connectionFilePath: connectionFilePath,
		secretKeyFilePath:  secretKeyFilePath,
	}
}

func (s *ConnectionStore) InitializeIfEmpty() error {
	isEmpty, err := storage.IsFileEmpty(s.connectionFilePath)
	if err != nil {
		return err
	}
	if !isEmpty {
		return nil
	}
	return s.Save(model.NewConnectionFile())
}

func (s *ConnectionStore) Load() (model.ConnectionFile, error) {
	return s.load(false)
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
	return s.load(true)
}

func (s *ConnectionStore) load(lockHeld bool) (model.ConnectionFile, error) {
	key, err := cryptoutil.LoadKey(s.secretKeyFilePath)
	if err != nil {
		return model.ConnectionFile{}, err
	}

	content, err := decryptAndReadFile(s.connectionFilePath, key)
	if err != nil {
		return model.ConnectionFile{}, err
	}

	connFile, err := parseConnectionFile(content)
	if err != nil {
		return model.ConnectionFile{}, err
	}

	changed := connFile.EnsureIDs()
	if changed {
		if lockHeld {
			if err := s.saveWithoutLock(connFile); err != nil {
				return model.ConnectionFile{}, err
			}
		} else {
			if err := s.Save(connFile); err != nil {
				return model.ConnectionFile{}, err
			}
		}
	}

	return connFile, nil
}

func (s *ConnectionStore) saveWithoutLock(connFile model.ConnectionFile) error {
	if strings.TrimSpace(connFile.Version) == "" {
		connFile.Version = model.CurrentConnectionFileVersion
	}
	connFile.EnsureIDs()

	key, err := cryptoutil.LoadKey(s.secretKeyFilePath)
	if err != nil {
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
	for {
		lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_, _ = fmt.Fprintf(lockFile, "pid=%d\ncreated=%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339Nano))
			_ = lockFile.Close()
			return func() { _ = os.Remove(lockPath) }, nil
		}

		if !os.IsExist(err) {
			return nil, fmt.Errorf("failed to acquire mutation lock: %w", err)
		}

		if s.shouldBreakStaleLock(lockPath) {
			_ = os.Remove(lockPath)
			continue
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out acquiring mutation lock %s", lockPath)
		}
		time.Sleep(connectionLockRetryInterval)
	}
}

func (s *ConnectionStore) shouldBreakStaleLock(lockPath string) bool {
	info, err := os.Stat(lockPath)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) > connectionLockStaleAfter
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
