package prompt

import (
	"errors"
	"strings"

	"github.com/emirhangumus/sshmanager/internal/model"
	"github.com/manifoldco/promptui"
)

func validateText(input string) error {
	if strings.TrimSpace(input) == "" {
		return errors.New("this field is required")
	}
	return nil
}

func AddSSHConnectionPrompt() (model.SSHConnection, error) {
	host, err := runValidatedPrompt(DefaultPromptTexts.EnterHost, "")
	if err != nil {
		return model.SSHConnection{}, err
	}

	username, err := runValidatedPrompt(DefaultPromptTexts.EnterUsername, "")
	if err != nil {
		return model.SSHConnection{}, err
	}

	password, err := runPasswordPrompt(DefaultPromptTexts.EnterPassword, "")
	if err != nil {
		return model.SSHConnection{}, err
	}

	description, err := runPlainPrompt(DefaultPromptTexts.EnterDescription, "")
	if err != nil {
		return model.SSHConnection{}, err
	}

	alias, err := runPlainPrompt(DefaultPromptTexts.EnterAlias, "")
	if err != nil {
		return model.SSHConnection{}, err
	}

	return model.SSHConnection{
		Username:    username,
		Host:        host,
		Password:    password,
		Description: description,
		Alias:       alias,
	}, nil
}

func EditSSHConnectionPrompt(conn *model.SSHConnection) (model.SSHConnection, error) {
	host, err := runValidatedPrompt(DefaultPromptTexts.EditHost, conn.Host)
	if err != nil {
		return model.SSHConnection{}, err
	}

	username, err := runValidatedPrompt(DefaultPromptTexts.EditUsername, conn.Username)
	if err != nil {
		return model.SSHConnection{}, err
	}

	password, err := runPasswordPrompt(DefaultPromptTexts.EditPassword, conn.Password)
	if err != nil {
		return model.SSHConnection{}, err
	}

	description, err := runPlainPrompt(DefaultPromptTexts.EditDescription, conn.Description)
	if err != nil {
		return model.SSHConnection{}, err
	}

	alias, err := runPlainPrompt(DefaultPromptTexts.EditAlias, conn.Alias)
	if err != nil {
		return model.SSHConnection{}, err
	}

	return model.SSHConnection{
		ID:          conn.ID,
		Username:    username,
		Host:        host,
		Password:    password,
		Description: description,
		Alias:       alias,
	}, nil
}

func runValidatedPrompt(label, defaultValue string) (string, error) {
	p := promptui.Prompt{Label: label, Default: defaultValue, Validate: validateText}
	return p.Run()
}

func runPasswordPrompt(label, defaultValue string) (string, error) {
	p := promptui.Prompt{Label: label, Default: defaultValue, Mask: '*', Validate: validateText}
	return p.Run()
}

func runPlainPrompt(label, defaultValue string) (string, error) {
	p := promptui.Prompt{Label: label, Default: defaultValue}
	return p.Run()
}
