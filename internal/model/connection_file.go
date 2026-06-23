package model

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

const CurrentConnectionFileVersion = "1.0"

var ErrAliasAlreadyExists = errors.New("alias already exists")

// ConnectionFile represents persisted SSH connections.
type ConnectionFile struct {
	Version     string          `yaml:"version" json:"version"`
	Connections []SSHConnection `yaml:"connections" json:"connections"`
}

// ConnectionSelectItem is a typed menu item for prompt selection.
type ConnectionSelectItem struct {
	ConnectionID string
	Label        string
}

func (i ConnectionSelectItem) String() string {
	return i.Label
}

func NewConnectionFile() ConnectionFile {
	return ConnectionFile{
		Version:     CurrentConnectionFileVersion,
		Connections: []SSHConnection{},
	}
}

func (c *ConnectionFile) AddConnection(conn SSHConnection) error {
	conn.Alias = strings.TrimSpace(conn.Alias)
	if c.hasAliasConflict(conn.Alias, "") {
		return fmt.Errorf("%w: %s", ErrAliasAlreadyExists, conn.Alias)
	}

	if strings.TrimSpace(conn.ID) == "" || c.hasID(conn.ID) {
		conn.ID = c.generateUniqueID()
	}
	c.Connections = append(c.Connections, conn)
	return nil
}

func (c *ConnectionFile) RemoveConnectionByID(id string) bool {
	for i := range c.Connections {
		if c.Connections[i].ID == id {
			c.Connections = append(c.Connections[:i], c.Connections[i+1:]...)
			return true
		}
	}
	return false
}

func (c *ConnectionFile) GetConnectionByID(id string) *SSHConnection {
	for i := range c.Connections {
		if c.Connections[i].ID == id {
			return &c.Connections[i]
		}
	}
	return nil
}

func (c *ConnectionFile) GetConnectionByAlias(alias string) *SSHConnection {
	needle := normalizeAlias(alias)
	if needle == "" {
		return nil
	}

	for i := range c.Connections {
		if normalizeAlias(c.Connections[i].Alias) == needle {
			return &c.Connections[i]
		}
	}
	return nil
}

func (c *ConnectionFile) UpdateConnectionByID(id string, updated SSHConnection) (bool, error) {
	for i := range c.Connections {
		if c.Connections[i].ID == id {
			updated.Alias = strings.TrimSpace(updated.Alias)
			if c.hasAliasConflict(updated.Alias, id) {
				return true, fmt.Errorf("%w: %s", ErrAliasAlreadyExists, updated.Alias)
			}

			updated.ID = id
			c.Connections[i] = updated
			return true, nil
		}
	}
	return false, nil
}

func (c *ConnectionFile) AllAliases() []string {
	aliases := make([]string, 0, len(c.Connections))
	for _, conn := range c.Connections {
		alias := strings.TrimSpace(conn.Alias)
		if alias != "" {
			aliases = append(aliases, alias)
		}
	}
	return aliases
}

func (c *ConnectionFile) SelectItems() []ConnectionSelectItem {
	items := make([]ConnectionSelectItem, 0, len(c.Connections))
	for i := range c.Connections {
		conn := c.Connections[i]
		display := fmt.Sprintf("%d. %s@%s", i+1, conn.Username, conn.Host)
		if strings.TrimSpace(conn.Description) != "" {
			display += " - " + conn.Description
		}
		if strings.TrimSpace(conn.Alias) != "" {
			display += fmt.Sprintf(" (%s)", conn.Alias)
		}
		if strings.TrimSpace(conn.Group) != "" {
			display += fmt.Sprintf(" [group:%s]", strings.TrimSpace(conn.Group))
		}
		if tags := NormalizeTags(conn.Tags); len(tags) > 0 {
			display += fmt.Sprintf(" [tags:%s]", strings.Join(tags, ","))
		}
		items = append(items, ConnectionSelectItem{
			ConnectionID: conn.ID,
			Label:        display,
		})
	}
	return items
}

// EnsureIDs normalizes the in-memory list and returns true if mutations occurred.
func (c *ConnectionFile) EnsureIDs() bool {
	changed := false
	seen := make(map[string]struct{}, len(c.Connections))
	for i := range c.Connections {
		id := strings.TrimSpace(c.Connections[i].ID)
		if id == "" {
			c.Connections[i].ID = c.generateUniqueIDWithSeen(seen)
			changed = true
			continue
		}
		if _, exists := seen[id]; exists {
			c.Connections[i].ID = c.generateUniqueIDWithSeen(seen)
			changed = true
			continue
		}
		seen[id] = struct{}{}
	}
	return changed
}

func (c *ConnectionFile) hasID(id string) bool {
	for i := range c.Connections {
		if c.Connections[i].ID == id {
			return true
		}
	}
	return false
}

func (c *ConnectionFile) hasAliasConflict(alias, excludeID string) bool {
	needle := normalizeAlias(alias)
	if needle == "" {
		return false
	}

	for i := range c.Connections {
		conn := c.Connections[i]
		if conn.ID == excludeID {
			continue
		}
		if normalizeAlias(conn.Alias) == needle {
			return true
		}
	}

	return false
}

func (c *ConnectionFile) generateUniqueID() string {
	for {
		id := newConnectionID()
		if !c.hasID(id) {
			return id
		}
	}
}

func (c *ConnectionFile) generateUniqueIDWithSeen(seen map[string]struct{}) string {
	for {
		id := newConnectionID()
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		return id
	}
}

func newConnectionID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("id-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func normalizeAlias(alias string) string {
	return strings.ToLower(strings.TrimSpace(alias))
}
