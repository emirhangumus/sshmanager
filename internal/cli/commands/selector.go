package commands

import (
	"fmt"
	"strings"

	"github.com/emirhangumus/sshmanager/internal/model"
	prompttext "github.com/emirhangumus/sshmanager/internal/ui/prompt"
)

func resolveSelector(aliasFlag, idFlag string, positional []string, command string) (string, string, error) {
	alias := strings.TrimSpace(aliasFlag)
	id := strings.TrimSpace(idFlag)

	if alias != "" && id != "" {
		return "", "", fmt.Errorf("%s: use either --alias or --id, not both", command)
	}

	if len(positional) > 0 {
		if alias != "" || id != "" {
			return "", "", fmt.Errorf("%s: unexpected positional arguments: %s", command, strings.Join(positional, " "))
		}
		if len(positional) > 1 {
			return "", "", fmt.Errorf("%s: unexpected positional arguments: %s", command, strings.Join(positional[1:], " "))
		}
		alias = strings.TrimSpace(positional[0])
	}

	return alias, id, nil
}

func findConnectionBySelector(connFile *model.ConnectionFile, alias, id string) *model.SSHConnection {
	if connFile == nil {
		return nil
	}

	if strings.TrimSpace(id) != "" {
		return connFile.GetConnectionByID(strings.TrimSpace(id))
	}
	if strings.TrimSpace(alias) != "" {
		return connFile.GetConnectionByAlias(strings.TrimSpace(alias))
	}
	return nil
}

func notFoundMessage(alias, id string) string {
	if strings.TrimSpace(alias) != "" {
		return fmt.Sprintf(prompttext.DefaultPromptTexts.ErrorMessages.AliasNotFoundX, strings.TrimSpace(alias))
	}
	if strings.TrimSpace(id) != "" {
		return fmt.Sprintf("No SSH connection found for id: %s", strings.TrimSpace(id))
	}
	return prompttext.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound
}
