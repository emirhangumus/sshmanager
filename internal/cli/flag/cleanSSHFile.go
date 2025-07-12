package flag

import (
	"fmt"
	"github.com/emirhangumus/sshmanager/internal/storage"
	"github.com/manifoldco/promptui"
)

func CleanSSHFile(value *bool, connectionFilePath string, secretKeyFilePath string) {
	if *value {
		_prompt := promptui.Prompt{Label: "Are you sure you want to remove all SSH connections and key files? This action cannot be undone. Type 'yes' to confirm."}
		confirmation, err := _prompt.Run()
		if err != nil || confirmation != "yes" {
			fmt.Println("Operation cancelled.")
			return
		}

		// Proceed with secure deletion
		storage.SecureDelete(connectionFilePath)
		storage.SecureDelete(secretKeyFilePath)
		fmt.Println("All SSH connections and key files removed.")
		return
	}
}
