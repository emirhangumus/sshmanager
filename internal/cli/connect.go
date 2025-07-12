package cli

import (
	"fmt"
	"github.com/emirhangumus/sshmanager/internal/cli/flag"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/emirhangumus/sshmanager/internal/prompt"
	"github.com/emirhangumus/sshmanager/internal/storage"
	"github.com/manifoldco/promptui"
)

func HandleConnect(connectionFilePath string, secretKeyFilePath string, config *flag.SSHManagerConfig) {
	connections, err := storage.ReadAllConnections(connectionFilePath, secretKeyFilePath)
	if err != nil {
		fmt.Println(prompt.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return
	}

	items := ConnToStrSlice(connections)
	_prompt := promptui.Select{Label: prompt.DefaultPromptTexts.SelectAnSSHConnection, Items: items}
	_, result, err := _prompt.Run()
	if err != nil || result == prompt.DefaultPromptTexts.BackToMainMenu {
		return
	}

	index := strings.SplitN(result, ".", 2)[0]
	conn, err := GetConnByIndex(index, connections)
	if err != nil {
		fmt.Println(err)
		return
	}

	connect(conn, config)
}

func connect(conn storage.SSHConnection, config *flag.SSHManagerConfig) {
	sshpassPath, err := exec.LookPath("sshpass")
	if err != nil {
		fmt.Println(prompt.DefaultPromptTexts.ErrorMessages.SSHPassNotFound)
		return
	}

	sshTarget := fmt.Sprintf("%s@%s", conn.Username, conn.Host)
	args := []string{"sshpass", "-p", conn.Password, "ssh", sshTarget}

	if config.Behaviour.ContinueAfterSSHExit == true {
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
