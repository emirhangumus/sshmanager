package commands

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/emirhangumus/sshmanager/v2/internal/model"
	"github.com/emirhangumus/sshmanager/v2/internal/store"
	prompttext "github.com/emirhangumus/sshmanager/v2/internal/ui/prompt"
)

func HandleAdd(connectionFilePath, secretKeyFilePath string) error {
	conn, err := prompttext.AddSSHConnectionPrompt()
	if err != nil {
		if prompttext.IsCancelError(err) {
			fmt.Println(prompttext.DefaultPromptTexts.SuccessMessages.OperationCancelled)
			return nil
		}
		return err
	}

	connStore := store.NewConnectionStore(connectionFilePath, secretKeyFilePath)
	if err := connStore.Update(func(connFile *model.ConnectionFile) error {
		return connFile.AddConnection(conn)
	}); err != nil {
		return err
	}

	fmt.Println(prompttext.DefaultPromptTexts.SuccessMessages.SSHConnectionSaved)
	return nil
}

func HandleAddArgs(connectionFilePath, secretKeyFilePath string, args []string) error {
	return handleAddArgs(connectionFilePath, secretKeyFilePath, args, os.Stdout)
}

func handleAddArgs(connectionFilePath, secretKeyFilePath string, args []string, out io.Writer) error {
	if len(args) == 0 {
		return HandleAdd(connectionFilePath, secretKeyFilePath)
	}

	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	host := fs.String("host", "", "Host name or IP")
	username := fs.String("username", "", "SSH username")
	port := fs.Int("port", 0, "SSH port (default 22)")
	authMode := fs.String("auth-mode", "", "Auth mode: password|key|agent")
	password := fs.String("password", "", "SSH password (password mode)")
	identityFile := fs.String("identity-file", "", "Identity file path (key mode)")
	proxyJump := fs.String("proxy-jump", "", "ProxyJump spec ([user@]host[:port][,[user@]host[:port]...])")
	proxyJumpAuthMode := fs.String("proxy-jump-auth-mode", "", "ProxyJump auth mode: password|key|agent (defaults to key/agent, independent of target auth)")
	proxyJumpPassword := fs.String("proxy-jump-password", "", "ProxyJump password (required if proxy-jump-auth-mode is password)")
	proxyJumpIdentityFile := fs.String("proxy-jump-identity-file", "", "ProxyJump identity file path")
	group := fs.String("group", "", "Connection group name")
	alias := fs.String("alias", "", "Connection alias")
	description := fs.String("description", "", "Connection description")
	var localForwards stringListFlag
	var remoteForwards stringListFlag
	var extraSSHArgs stringListFlag
	var tags stringListFlag
	fs.Var(&localForwards, "local-forward", "Local forwarding spec [bind_address:]port:host:hostport (repeatable)")
	fs.Var(&remoteForwards, "remote-forward", "Remote forwarding spec [bind_address:]port:host:hostport (repeatable)")
	fs.Var(&extraSSHArgs, "extra-ssh-arg", "Extra ssh argument token (repeatable, controlled)")
	fs.Var(&tags, "tag", "Connection tag (repeatable)")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("add: unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	conn := model.SSHConnection{
		Host:                  strings.TrimSpace(*host),
		Username:              strings.TrimSpace(*username),
		Port:                  *port,
		AuthMode:              strings.TrimSpace(*authMode),
		Password:              *password,
		IdentityFile:          strings.TrimSpace(*identityFile),
		ProxyJump:             strings.TrimSpace(*proxyJump),
		ProxyJumpAuthMode:     strings.TrimSpace(*proxyJumpAuthMode),
		ProxyJumpPassword:     *proxyJumpPassword,
		ProxyJumpIdentityFile: strings.TrimSpace(*proxyJumpIdentityFile),
		LocalForwards:         localForwards.Values(),
		RemoteForwards:        remoteForwards.Values(),
		ExtraSSHArgs:          extraSSHArgs.Values(),
		Group:                 strings.TrimSpace(*group),
		Tags:                  tags.Values(),
		Alias:                 strings.TrimSpace(*alias),
		Description:           strings.TrimSpace(*description),
	}

	normalized, err := normalizeImportedConnection(conn)
	if err != nil {
		return err
	}

	connStore := store.NewConnectionStore(connectionFilePath, secretKeyFilePath)
	if err := connStore.Update(func(connFile *model.ConnectionFile) error {
		return connFile.AddConnection(normalized)
	}); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(out, prompttext.DefaultPromptTexts.SuccessMessages.SSHConnectionSaved)
	return nil
}
