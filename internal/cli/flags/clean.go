package flags

import (
	"fmt"
	"strings"

	"github.com/emirhangumus/sshmanager/v2/internal/store"
	prompttext "github.com/emirhangumus/sshmanager/v2/internal/ui/prompt"
)

func CleanSSHFiles(connectionFilePath, secretKeyFilePath string) error {
	confirmation, err := prompttext.InputPrompt(
		"Are you sure you want to remove all SSH connections and their file/keyring secrets? This action cannot be undone. Type 'yes' to confirm.",
		"",
		false,
		nil,
	)
	if err != nil || !strings.EqualFold(strings.TrimSpace(confirmation), "yes") {
		fmt.Println(prompttext.DefaultPromptTexts.SuccessMessages.OperationCancelled)
		return nil
	}

	if err := store.NewConnectionStore(connectionFilePath, secretKeyFilePath).Clean(); err != nil {
		return err
	}

	fmt.Println(prompttext.DefaultPromptTexts.SuccessMessages.AllFilesRemoved)
	return nil
}
