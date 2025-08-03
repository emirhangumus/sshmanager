package prompt

import (
	"errors"
	"github.com/emirhangumus/sshmanager/internal/gstructs/sshconnection"

	"github.com/manifoldco/promptui"
)

func validateText(input string) error {
	if len(input) == 0 {
		return errors.New("this field is required")
	}
	return nil
}

func AddSSHConnectionPrompt() (sshconnection.SSHConnection, error) {
	prompt := promptui.Prompt{
		Label:    "Enter Host",
		Validate: validateText,
	}
	host, err := prompt.Run()
	if err != nil {
		return sshconnection.SSHConnection{}, err
	}

	prompt = promptui.Prompt{
		Label:    "Enter Username",
		Validate: validateText,
	}
	username, err := prompt.Run()
	if err != nil {
		return sshconnection.SSHConnection{}, err
	}

	prompt = promptui.Prompt{
		Label:    "Enter Password",
		Mask:     '*',
		Validate: validateText,
	}
	password, err := prompt.Run()
	if err != nil {
		return sshconnection.SSHConnection{}, err
	}

	prompt = promptui.Prompt{
		Label: "Enter Description",
	}
	description, err := prompt.Run()
	if err != nil {
		return sshconnection.SSHConnection{}, err
	}

	prompt = promptui.Prompt{
		Label: "Enter Alias",
	}
	alias, err := prompt.Run()
	if err != nil {
		return sshconnection.SSHConnection{}, err
	}

	return sshconnection.SSHConnection{
		Index:       "", // Index will be set later
		Username:    username,
		Host:        host,
		Password:    password,
		Description: description,
		Alias:       alias,
	}, nil
}

func EditSSHConnectionPrompt(conn *sshconnection.SSHConnection) (sshconnection.SSHConnection, error) {
	prompt := promptui.Prompt{
		Label:    "Edit Host",
		Default:  conn.Host,
		Validate: validateText,
	}
	host, err := prompt.Run()
	if err != nil {
		return sshconnection.SSHConnection{}, err
	}

	prompt = promptui.Prompt{
		Label:    "Edit Username",
		Default:  conn.Username,
		Validate: validateText,
	}
	username, err := prompt.Run()
	if err != nil {
		return sshconnection.SSHConnection{}, err
	}

	prompt = promptui.Prompt{
		Label:   "Edit Password",
		Default: conn.Password,
		Mask:    '*',
	}
	password, err := prompt.Run()
	if err != nil {
		return sshconnection.SSHConnection{}, err
	}

	prompt = promptui.Prompt{
		Label:   "Edit Description",
		Default: conn.Description,
	}
	description, err := prompt.Run()
	if err != nil {
		return sshconnection.SSHConnection{}, err
	}

	prompt = promptui.Prompt{
		Label:   "Edit Alias",
		Default: conn.Alias,
	}
	alias, err := prompt.Run()
	if err != nil {
		return sshconnection.SSHConnection{}, err
	}

	return sshconnection.SSHConnection{
		Index:       conn.Index, // Keep the same index
		Username:    username,
		Host:        host,
		Password:    password,
		Description: description,
		Alias:       alias,
	}, nil
}
