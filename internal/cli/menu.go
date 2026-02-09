package cli

import (
	"fmt"

	"github.com/emirhangumus/sshmanager/internal/cli/commands"
	"github.com/emirhangumus/sshmanager/internal/config"
	prompttext "github.com/emirhangumus/sshmanager/internal/ui/prompt"
	"github.com/manifoldco/promptui"
)

func ShowMainMenu(connectionFilePath, secretKeyFilePath, configFilePath, version string) error {
	cfg, err := config.LoadConfig(configFilePath)
	if err != nil {
		return fmt.Errorf(prompttext.DefaultPromptTexts.ErrorMessages.FailedToLoadConfigX, err)
	}

	options := []string{
		prompttext.DefaultPromptTexts.Exit,
		prompttext.DefaultPromptTexts.ConnectToSSH,
		prompttext.DefaultPromptTexts.AddSSHConnection,
		prompttext.DefaultPromptTexts.EditSSHConnection,
		prompttext.DefaultPromptTexts.RemoveSSHConnection,
	}

	for {
		selector := promptui.Select{Label: "Menu Options | " + version, Items: options}
		_, choice, err := selector.Run()
		if err != nil {
			fmt.Printf(prompttext.DefaultPromptTexts.ErrorMessages.InvalidSelectionX+"\n", err)
			continue
		}

		switch choice {
		case prompttext.DefaultPromptTexts.Exit:
			return nil
		case prompttext.DefaultPromptTexts.ConnectToSSH:
			exitAfterSSH, err := commands.HandleConnect(connectionFilePath, secretKeyFilePath, &cfg)
			if err != nil {
				fmt.Printf(prompttext.DefaultPromptTexts.ErrorMessages.FailedToLoadConnectionsX+"\n", err)
				continue
			}
			if exitAfterSSH {
				return nil
			}
		case prompttext.DefaultPromptTexts.AddSSHConnection:
			if err := commands.HandleAdd(connectionFilePath, secretKeyFilePath); err != nil {
				fmt.Printf(prompttext.DefaultPromptTexts.ErrorMessages.FailedToAddConnectionX+"\n", err)
			}
		case prompttext.DefaultPromptTexts.EditSSHConnection:
			if err := commands.HandleEdit(connectionFilePath, secretKeyFilePath); err != nil {
				fmt.Printf(prompttext.DefaultPromptTexts.ErrorMessages.FailedToStoreUpdatedConnectionsX+"\n", err)
			}
		case prompttext.DefaultPromptTexts.RemoveSSHConnection:
			if err := commands.HandleRemove(connectionFilePath, secretKeyFilePath); err != nil {
				fmt.Printf(prompttext.DefaultPromptTexts.ErrorMessages.FailedToStoreUpdatedConnectionsX+"\n", err)
			}
		}
	}
}
