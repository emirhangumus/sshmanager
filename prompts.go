package main

import (
	"errors"
	"fmt"

	"github.com/manifoldco/promptui"
)

func validateText(input string) error {
	if len(input) == 0 {
		return errors.New("this field is required")
	}
	return nil
}

func addSSHConnectionPrompt() (string, error) {
	prompt := promptui.Prompt{
		Label:    "Enter Host",
		Validate: validateText,
	}
	host, err := prompt.Run()
	if err != nil {
		return "", err
	}

	prompt = promptui.Prompt{
		Label:    "Enter Username",
		Validate: validateText,
	}
	username, err := prompt.Run()
	if err != nil {
		return "", err
	}

	prompt = promptui.Prompt{
		Label:    "Enter Password",
		Mask:     '*',
		Validate: validateText,
	}
	password, err := prompt.Run()
	if err != nil {
		return "", err
	}

	prompt = promptui.Prompt{
		Label: "Enter Description",
	}
	description, err := prompt.Run()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s@%s\t%s\t%s", username, host, password, description), nil
}
