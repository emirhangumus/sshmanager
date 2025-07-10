package cli

import (
	"errors"
	"fmt"

	"github.com/emirhangumus/sshmanager/internal/storage"
)

func ConnToStrSlice(conns []storage.SSHConnection) []string {
	items := []string{"Back to main menu"}
	for _, conn := range conns {
		display := fmt.Sprintf("%s. %s@%s - %s", conn.Index, conn.Username, conn.Host, conn.Description)
		items = append(items, display)
	}
	return items
}

func GetConnByIndex(index string, conns []storage.SSHConnection) (storage.SSHConnection, error) {
	for _, conn := range conns {
		if conn.Index == index {
			return conn, nil
		}
	}
	return storage.SSHConnection{}, errors.New("connection not found")
}
