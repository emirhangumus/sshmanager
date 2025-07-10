package cli

import (
	"fmt"

	"github.com/manifoldco/promptui"
)

const version = "v0.2.5"

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
			fmt.Println("Invalid selection:", err)
			continue
		}

		switch choice {
		case "Exit":
			return
		case "Connect to SSH":
			HandleConnect(dataPath, keyPath)
		case "Add SSH Connection":
			if err := HandleAdd(dataPath, keyPath); err != nil {
				fmt.Println("Failed to add:", err)
			}
		case "Remove SSH Connection":
			HandleRemove(dataPath, keyPath)
		}
	}
}
