package store

import (
	"fmt"
	"strings"

	cryptoutil "github.com/emirhangumus/sshmanager/internal/crypto"
	"github.com/emirhangumus/sshmanager/internal/model"
	"github.com/emirhangumus/sshmanager/internal/storage"
)

type ConnectionStore struct {
	connectionFilePath string
	secretKeyFilePath  string
}

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
		if err := s.Save(connFile); err != nil {
			return model.ConnectionFile{}, err
		}
	}

	return connFile, nil
}

func (s *ConnectionStore) Save(connFile model.ConnectionFile) error {
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
