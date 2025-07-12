package storage

import (
	"fmt"
	"strings"

	"github.com/emirhangumus/sshmanager/internal/encryption"
)

type SSHConnection struct {
	Index       string
	Username    string
	Host        string
	Password    string
	Description string
}

// ReadAllConnections reads and parses all stored SSH connections.
func ReadAllConnections(connectionFilePath string, secretKeyFilePath string) ([]SSHConnection, error) {
	key, err := encryption.LoadKey(secretKeyFilePath)
	if err != nil {
		return nil, err
	}

	content, err := DecryptAndReadFile(connectionFilePath, key)
	if err != nil {
		return nil, err
	}

	var connections []SSHConnection
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		userHost := strings.Split(parts[0], "@")
		conn := SSHConnection{
			Index:       fmt.Sprintf("%d", i+1),
			Username:    userHost[0],
			Host:        userHost[1],
			Password:    parts[1],
			Description: parts[2],
		}
		connections = append(connections, conn)
	}

	return connections, nil
}
