package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	cryptoutil "github.com/emirhangumus/sshmanager/v2/internal/crypto"
	"github.com/emirhangumus/sshmanager/v2/internal/model"
	"github.com/emirhangumus/sshmanager/v2/internal/storage"
	"github.com/gofrs/flock"
	"gopkg.in/yaml.v3"
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

	connFile, legacySchema, err := parseConnectionFileWithMigration(content)
	if err != nil {
		return model.ConnectionFile{}, err
	}

	changed := connFile.EnsureIDs() || legacySchema
	if content != "" {
		data, readErr := storage.ReadFileRegular(s.connectionFilePath)
		if readErr != nil {
			return model.ConnectionFile{}, readErr
		}
		changed = changed || !cryptoutil.IsVersionedStore(data)
	}
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
	if err := os.MkdirAll(lockDir, 0o700); err != nil { //nolint:gosec // Lock directory derives from the explicitly configured local datastore, not a remote profile.
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	if err := storage.ValidateRegularPath(lockPath); err != nil {
		return nil, err
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
	file, _, err := parseConnectionFileWithMigration(content)
	return file, err
}

func parseConnectionFileWithMigration(content string) (model.ConnectionFile, bool, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return model.NewConnectionFile(), false, nil
	}
	if len(content) > cryptoutil.MaxStoreSize {
		return model.ConnectionFile{}, false, fmt.Errorf("connection schema exceeds size limit")
	}
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return model.ConnectionFile{}, false, fmt.Errorf("failed to parse connection file: %w", err)
	}
	if len(document.Content) != 1 {
		return model.ConnectionFile{}, false, fmt.Errorf("invalid connection document")
	}
	root := document.Content[0]
	if root.Kind == yaml.SequenceNode {
		var legacy []model.SSHConnection
		if err := root.Decode(&legacy); err != nil {
			return model.ConnectionFile{}, false, err
		}
		return model.ConnectionFile{Version: model.CurrentConnectionFileVersion, Connections: legacy}, true, nil
	}
	if root.Kind != yaml.MappingNode {
		return model.ConnectionFile{}, false, fmt.Errorf("unsupported connection schema")
	}
	recognized := false
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == "connections" || root.Content[i].Value == "version" {
			recognized = true
		}
	}
	if !recognized {
		return model.ConnectionFile{}, false, fmt.Errorf("missing connection schema fields")
	}
	var schema model.ConnectionFile
	if err := root.Decode(&schema); err != nil {
		return model.ConnectionFile{}, false, err
	}
	legacy := strings.TrimSpace(schema.Version) == ""
	if legacy {
		schema.Version = model.CurrentConnectionFileVersion
	}
	if schema.Version != model.CurrentConnectionFileVersion {
		return model.ConnectionFile{}, false, fmt.Errorf("unsupported connection schema version %q", schema.Version)
	}
	if schema.Connections == nil {
		schema.Connections = []model.SSHConnection{}
	}
	return schema, legacy, nil
}
