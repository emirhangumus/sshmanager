package cli

import (
	"fmt"

	"github.com/emirhangumus/sshmanager/internal/prompts"
	"github.com/manifoldco/promptui"
)

const version = "v1.0.0"

var options = []string{
	"Exit",
	"Connect to SSH",
	"Add SSH Connection",
	"Remove SSH Connection",
}

func ShowMainMenu(dataPath, keyPath string) {
	for {
		prompt := promptui.Select{
			Label: "Menu Options | " + version,
			Items: options,
		}

		_, choice, err := prompt.Run()
		if err != nil {
			fmt.Println(prompts.DefaultPromptTexts.ErrorMessages.InvalidSelectionX, err)
			continue
		}

		switch choice {
		case prompts.DefaultPromptTexts.Exit:
			return
		case prompts.DefaultPromptTexts.ConnectToSSH:
			HandleConnect(dataPath, keyPath)
		case prompts.DefaultPromptTexts.AddSSHConnection:
			if err := HandleAdd(dataPath, keyPath); err != nil {
				fmt.Println(prompts.DefaultPromptTexts.ErrorMessages.FailedToAddConnectionX, err)
			}
		case prompts.DefaultPromptTexts.RemoveSSHConnection:
			HandleRemove(dataPath, keyPath)
		}
	}
}
