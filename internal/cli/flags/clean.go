package flags

import (
	"fmt"

	"github.com/emirhangumus/sshmanager/internal/storage"
	prompttext "github.com/emirhangumus/sshmanager/internal/ui/prompt"
	"github.com/manifoldco/promptui"
)

func CleanSSHFiles(connectionFilePath, secretKeyFilePath string) error {
	p := promptui.Prompt{
		Label: "Are you sure you want to remove all SSH connections and key files? This action cannot be undone. Type 'yes' to confirm.",
	}
	confirmation, err := p.Run()
	if err != nil || confirmation != "yes" {
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
