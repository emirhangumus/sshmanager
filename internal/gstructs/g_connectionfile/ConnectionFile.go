package g_connectionfile

import (
	"fmt"

	"github.com/emirhangumus/sshmanager/internal/encryption"
	"github.com/emirhangumus/sshmanager/internal/gstructs/g_sshconnection"
	"github.com/emirhangumus/sshmanager/internal/storage"
)

type ConnectionFile struct {
	Version     string                          `yaml:"version"`
	Connections []g_sshconnection.SSHConnection `yaml:"connections"`
}

func NewConnectionFile(connectionFilePath string, secretKeyFilePath string) *ConnectionFile {
	c := &ConnectionFile{
		Version:     "1.0",
		Connections: []g_sshconnection.SSHConnection{},
	}
	c.Load(connectionFilePath, secretKeyFilePath)
	return c
}

func (c *ConnectionFile) Load(connectionFilePath string, secretKeyFilePath string) error {
	key, err := encryption.LoadKey(secretKeyFilePath)
	if err != nil {
		return err
	}

	_content, err := storage.DecryptAndReadFile(connectionFilePath, key)
	if err != nil {
		return err
	}

	err = storage.FromYAMLString(_content, &c.Connections)
	if err != nil {
		return err
	}

	return nil
}

func (c ConnectionFile) String() string {
	var result string
	for _, conn := range c.Connections {
		result += conn.String() + "\n"
	}
	return result
}

func (c *ConnectionFile) AddConnection(conn g_sshconnection.SSHConnection) {
	conn.Index = fmt.Sprintf("%d", len(c.Connections)+1) // Set index based on current length
	c.Connections = append(c.Connections, conn)
}

func (c *ConnectionFile) RemoveConnection(index string) {
	for i, conn := range c.Connections {
		if conn.Index == index {
			c.Connections = append(c.Connections[:i], c.Connections[i+1:]...)
			break
		}
	}
}

func (c *ConnectionFile) GetConnection(index string) *g_sshconnection.SSHConnection {
	for _, conn := range c.Connections {
		if conn.Index == index {
			return &conn
		}
	}
	return nil
}

func (c *ConnectionFile) UpdateConnection(index string, updatedConn g_sshconnection.SSHConnection) {
	for i, conn := range c.Connections {
		if conn.Index == index {
			updatedConn.Index = index // Keep the same index
			c.Connections[i] = updatedConn
			break
		}
	}
}

func (c *ConnectionFile) Save(connectionFilePath string, secretKeyFilePath string) error {
	key, err := encryption.LoadKey(secretKeyFilePath)
	if err != nil {
		return err
	}

	contentStr, err := storage.ToYAMLString(c.Connections)
	if err != nil {
		return err
	}

	if err := storage.EncryptAndStoreFile(contentStr, connectionFilePath, key); err != nil {
		return err
	}

	return nil
}

func (c *ConnectionFile) SafeConnectionListString() []string {
	var items []string
	for _, conn := range c.Connections {
		display := fmt.Sprintf("%s. %s@%s - %s", conn.Index, conn.Username, conn.Host, conn.Description)
		if conn.Alias != "" {
			display += fmt.Sprintf(" (%s)", conn.Alias)
		}
		items = append(items, display)
	}
	return items
}

func (c *ConnectionFile) GetConnectionByAlias(alias string) *g_sshconnection.SSHConnection {
	for _, conn := range c.Connections {
		if conn.Alias == alias {
			return &conn
		}
	}
	return nil
}
