package main

import (
	"fmt"

	"github.com/manifoldco/promptui"
)

func addSSHConnectionPrompt() (string, error) {
	prompt := promptui.Prompt{
		Label: "Enter Host",
	}
	host, err := prompt.Run()
	if err != nil {
		return "", err
	}

	prompt = promptui.Prompt{
		Label: "Enter Username",
	}
	username, err := prompt.Run()
	if err != nil {
		return "", err
	}

	prompt = promptui.Prompt{
		Label: "Enter Password",
		Mask:  '*',
	}
	password, err := prompt.Run()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s@%s\t%s\n", username, host, password), nil
}
