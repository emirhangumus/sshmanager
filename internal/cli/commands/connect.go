package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/emirhangumus/sshmanager/internal/config"
	"github.com/emirhangumus/sshmanager/internal/model"
	"github.com/emirhangumus/sshmanager/internal/store"
	prompttext "github.com/emirhangumus/sshmanager/internal/ui/prompt"
	"github.com/manifoldco/promptui"
)

// HandleConnect returns true when caller should exit app after SSH command exits.
func HandleConnect(connectionFilePath, secretKeyFilePath string, cfg *config.SSHManagerConfig) (bool, error) {
	connStore := store.NewConnectionStore(connectionFilePath, secretKeyFilePath)
	connFile, err := connStore.Load()
	if err != nil {
		return false, err
	}
	if len(connFile.Connections) == 0 {
		fmt.Println(prompttext.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return false, nil
	}

	items := connFile.SelectItems()
	selector := promptui.Select{Label: prompttext.DefaultPromptTexts.SelectAnSSHConnection, Items: items}
	idx, _, err := selector.Run()
	if err != nil {
		return false, nil
	}

	selectedID := items[idx].ConnectionID
	conn := connFile.GetConnectionByID(selectedID)
	if conn == nil {
		fmt.Println(prompttext.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return false, nil
	}

	if err := connect(conn); err != nil {
		fmt.Printf(prompttext.DefaultPromptTexts.ErrorMessages.ConnectionToXFailedX+"\n", fmt.Sprintf("%s@%s", conn.Username, conn.Host), err)
		return false, nil
	}

	return !cfg.Behaviour.ContinueAfterSSHExit, nil
}

func connect(conn *model.SSHConnection) error {
	sshpassPath, err := exec.LookPath("sshpass")
	if err != nil {
		return fmt.Errorf(prompttext.DefaultPromptTexts.ErrorMessages.SSHPassNotFound)
	}

	sshTarget := fmt.Sprintf("%s@%s", conn.Username, conn.Host)
	cmd := exec.Command(sshpassPath, "-e", "ssh", sshTarget)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "SSHPASS="+conn.Password)

	return cmd.Run()
}
