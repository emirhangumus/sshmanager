package cli

import (
	"fmt"
	"github.com/emirhangumus/sshmanager/internal/gstructs/connectionfile"

	"github.com/emirhangumus/sshmanager/internal/cli/flag"
	"github.com/emirhangumus/sshmanager/internal/prompt"
)

func FindAndConnect(connectionFilePath string, secretKeyFilePath string, configFilePath string, alias string) {
	config, err := flag.LoadConfig(configFilePath)
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	connFile := connectionfile.NewConnectionFile(connectionFilePath, secretKeyFilePath)
	if len(connFile.Connections) == 0 {
		fmt.Println(prompt.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return
	}

	conn := connFile.GetConnectionByAlias(alias)
	if conn == nil {
		fmt.Println(prompt.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return
	}

	fmt.Printf("Connecting to %s@%s...\n", conn.Username, conn.Host)
	connect(conn, &config)
}
