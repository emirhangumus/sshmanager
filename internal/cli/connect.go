package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/emirhangumus/sshmanager/internal/storage"
	"github.com/manifoldco/promptui"
)

func HandleConnect(dataPath, keyPath string) {
	connections, err := storage.ReadAllConnections(dataPath, keyPath)
	if err != nil {
		fmt.Println("No SSH connections found.")
		return
	}

	items := ConnToStrSlice(connections)
	prompt := promptui.Select{Label: "Select an SSH connection", Items: items}
	_, result, err := prompt.Run()
	if err != nil || result == "Back to main menu" {
		return
	}

	index := strings.SplitN(result, ".", 2)[0]
	conn, err := GetConnByIndex(index, connections)
	if err != nil {
		fmt.Println(err)
		return
	}

	connect(conn)
}

func connect(conn storage.SSHConnection) {
	sshpassPath, err := exec.LookPath("sshpass")
	if err != nil {
		fmt.Println("Error: sshpass not found. Please install it using your package manager.")
		return
	}

	fmt.Println("host is: " + conn.Host)
	fmt.Println("username is: " + conn.Username)
	fmt.Println("password is: " + conn.Password)

	sshTarget := fmt.Sprintf("%s@%s", conn.Username, conn.Host)
	args := []string{"sshpass", "-p", conn.Password, "ssh", sshTarget}

	if err := syscall.Exec(sshpassPath, args, os.Environ()); err != nil {
		fmt.Printf("Connection to %s failed: %v\n", sshTarget, err)
	}
}
