package cli

import (
	"fmt"
	"github.com/emirhangumus/sshmanager/internal/gstructs/connectionfile"
	"github.com/emirhangumus/sshmanager/internal/gstructs/sshconnection"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/emirhangumus/sshmanager/internal/cli/flag"
	"github.com/emirhangumus/sshmanager/internal/prompt"
	"github.com/manifoldco/promptui"
)

func HandleConnect(connectionFilePath string, secretKeyFilePath string, config *flag.SSHManagerConfig) {
	connFile := connectionfile.NewConnectionFile(connectionFilePath, secretKeyFilePath)
	if len(connFile.Connections) == 0 {
		fmt.Println(prompt.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return
	}

	items := connFile.SafeConnectionListString()
	_prompt := promptui.Select{Label: prompt.DefaultPromptTexts.SelectAnSSHConnection, Items: items}
	_, result, err := _prompt.Run()
	if err != nil || result == prompt.DefaultPromptTexts.BackToMainMenu {
		return
	}

	index := strings.SplitN(result, ".", 2)[0]
	conn := connFile.GetConnection(index)
	if conn == nil {
		fmt.Println(prompt.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return
	}

	connect(conn, config)
}

func connect(conn *sshconnection.SSHConnection, config *flag.SSHManagerConfig) {
	sshpassPath, err := exec.LookPath("sshpass")
	if err != nil {
		fmt.Println(prompt.DefaultPromptTexts.ErrorMessages.SSHPassNotFound)
		return
	}

	sshTarget := fmt.Sprintf("%s@%s", conn.Username, conn.Host)
	args := []string{"sshpass", "-p", conn.Password, "ssh", sshTarget}

	if config.Behaviour.ContinueAfterSSHExit {
		// Background process
		bgCmd := exec.Command(sshpassPath, args[1:]...) // skip sshpassPath since it's set via Path
		bgCmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true, // new session
		}
		bgCmd.Stdin = os.Stdin
		bgCmd.Stdout = os.Stdout
		bgCmd.Stderr = os.Stderr

		if err := bgCmd.Start(); err != nil {
			fmt.Println("Failed to start background SSH:", err)
			return
		}

		// Start new foreground SSH session
		fgCmd := exec.Command(sshpassPath, args[1:]...)

		if err := fgCmd.Run(); err != nil {
			fmt.Println("Foreground SSH exited with error:", err)
		}

		// Wait for the background process to finish
		if err := bgCmd.Wait(); err != nil {
			fmt.Printf("Background SSH process exited with error: %v\n", err)
		}
	} else {
		if err := syscall.Exec(sshpassPath, args, os.Environ()); err != nil {
			fmt.Printf(prompt.DefaultPromptTexts.ErrorMessages.ConnectionToXFailedX+"\n", sshTarget, err)
		}
	}
}
