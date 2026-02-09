package flags

import (
	"fmt"
	"strings"

	"github.com/emirhangumus/sshmanager/internal/storage"
	prompttext "github.com/emirhangumus/sshmanager/internal/ui/prompt"
)

func CleanSSHFiles(connectionFilePath, secretKeyFilePath string) error {
	confirmation, err := prompttext.InputPrompt(
		"Are you sure you want to remove all SSH connections and key files? This action cannot be undone. Type 'yes' to confirm.",
		"",
		false,
		nil,
	)
	if err != nil || !strings.EqualFold(strings.TrimSpace(confirmation), "yes") {
		fmt.Println(prompttext.DefaultPromptTexts.SuccessMessages.OperationCancelled)
		return nil
	}

	if err := storage.SecureDelete(connectionFilePath); err != nil {
		return err
	}
	if err := storage.SecureDelete(secretKeyFilePath); err != nil {
		return err
	}

	fmt.Println(prompttext.DefaultPromptTexts.SuccessMessages.AllFilesRemoved)
	return nil
}
