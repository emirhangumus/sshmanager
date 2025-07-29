package cli

import (
	"fmt"

	"github.com/emirhangumus/sshmanager/internal/cli/flag"

	"github.com/emirhangumus/sshmanager/internal/prompt"
	"github.com/manifoldco/promptui"
)

var options = []string{
	"Exit",
	"Connect to SSH",
	"Add SSH Connection",
	"Edit SSH Connection",
	"Remove SSH Connection",
}

func ShowMainMenu(connectionFilePath string, secretKeyFilePath string, configFilePath string) {

	config, err := flag.LoadConfig(configFilePath)
	if err != nil {
		fmt.Println(prompt.DefaultPromptTexts.ErrorMessages.FailedToLoadConfigX, err)
		return
	}

	for {
		_prompt := promptui.Select{
			Label: "Menu Options | " + flag.SSHManagerVersion,
			Items: options,
		}

		_, choice, err := _prompt.Run()
		if err != nil {
			fmt.Println(prompt.DefaultPromptTexts.ErrorMessages.InvalidSelectionX, err)
			continue
		}

		switch choice {
		case prompt.DefaultPromptTexts.Exit:
			return
		case prompt.DefaultPromptTexts.ConnectToSSH:
			HandleConnect(connectionFilePath, secretKeyFilePath, &config)
		case prompt.DefaultPromptTexts.AddSSHConnection:
			if err := HandleAdd(connectionFilePath, secretKeyFilePath); err != nil {
				fmt.Println(prompt.DefaultPromptTexts.ErrorMessages.FailedToAddConnectionX, err)
			}
		case prompt.DefaultPromptTexts.EditSSHConnection:
			HandleEdit(connectionFilePath, secretKeyFilePath)
		case prompt.DefaultPromptTexts.RemoveSSHConnection:
			if err := HandleRemove(connectionFilePath, secretKeyFilePath); err != nil {
				fmt.Println(prompt.DefaultPromptTexts.ErrorMessages.FailedToLoadConfigX, err) // change to appropriate error message
			}
		}
	}
}
