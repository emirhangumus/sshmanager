package commands

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/emirhangumus/sshmanager/internal/model"
	"github.com/emirhangumus/sshmanager/internal/store"
	prompttext "github.com/emirhangumus/sshmanager/internal/ui/prompt"
	"github.com/manifoldco/promptui"
)

func HandleEdit(connectionFilePath, secretKeyFilePath string) error {
	connStore := store.NewConnectionStore(connectionFilePath, secretKeyFilePath)
	connFile, err := connStore.Load()
	if err != nil {
		return err
	}
	if len(connFile.Connections) == 0 {
		fmt.Println(prompttext.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return nil
	}

	items := connFile.SelectItems()
	selector := promptui.Select{Label: prompttext.DefaultPromptTexts.SelectAConnectionToEdit, Items: items}
	idx, _, err := selector.Run()
	if err != nil {
		if prompttext.IsCancelError(err) {
			fmt.Println(prompttext.DefaultPromptTexts.SuccessMessages.OperationCancelled)
		}
		return nil
	}

	connID := items[idx].ConnectionID
	conn := connFile.GetConnectionByID(connID)
	if conn == nil {
		fmt.Println(prompttext.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return nil
	}

	updatedConn, err := prompttext.EditSSHConnectionPrompt(conn)
	if err != nil {
		if prompttext.IsCancelError(err) {
			fmt.Println(prompttext.DefaultPromptTexts.SuccessMessages.OperationCancelled)
			return nil
		}
		return err
	}

	updated := false
	if err := connStore.Update(func(liveConnFile *model.ConnectionFile) error {
		var updateErr error
		updated, updateErr = liveConnFile.UpdateConnectionByID(connID, updatedConn)
		return updateErr
	}); err != nil {
		return err
	}
	if !updated {
		fmt.Println(prompttext.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return nil
	}

	fmt.Println(prompttext.DefaultPromptTexts.SuccessMessages.SSHConnectionUpdated)
	return nil
}

func HandleEditArgs(connectionFilePath, secretKeyFilePath string, args []string) error {
	return handleEditArgs(connectionFilePath, secretKeyFilePath, args, os.Stdout)
}

func handleEditArgs(connectionFilePath, secretKeyFilePath string, args []string, out io.Writer) error {
	if len(args) == 0 {
		return HandleEdit(connectionFilePath, secretKeyFilePath)
	}

	fs := flag.NewFlagSet("edit", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	alias := fs.String("alias", "", "Connection alias to edit")
	id := fs.String("id", "", "Connection ID to edit")

	newHost := fs.String("new-host", "", "New host")
	newUsername := fs.String("new-username", "", "New username")
	newPort := fs.Int("new-port", -1, "New SSH port (use 22 for default)")
	newAuthMode := fs.String("new-auth-mode", "", "New auth mode: password|key|agent")
	newPassword := fs.String("new-password", "", "New password")
	newIdentityFile := fs.String("new-identity-file", "", "New identity file path")
	newProxyJump := fs.String("new-proxy-jump", "", "New ProxyJump spec")
	newGroup := fs.String("new-group", "", "New connection group")
	newAlias := fs.String("new-alias", "", "New alias")
	newDescription := fs.String("new-description", "", "New description")
	clearAlias := fs.Bool("clear-alias", false, "Clear alias")
	clearDescription := fs.Bool("clear-description", false, "Clear description")
	clearProxyJump := fs.Bool("clear-proxy-jump", false, "Clear proxy jump")
	clearGroup := fs.Bool("clear-group", false, "Clear group")
	clearLocalForwards := fs.Bool("clear-local-forwards", false, "Clear local forward specs")
	clearRemoteForwards := fs.Bool("clear-remote-forwards", false, "Clear remote forward specs")
	clearExtraSSHArgs := fs.Bool("clear-extra-ssh-args", false, "Clear extra ssh args")
	clearTags := fs.Bool("clear-tags", false, "Clear tags")
	var newLocalForwards stringListFlag
	var newRemoteForwards stringListFlag
	var newExtraSSHArgs stringListFlag
	var newTags stringListFlag
	fs.Var(&newLocalForwards, "new-local-forward", "Replace local forward list with provided values (repeatable)")
	fs.Var(&newRemoteForwards, "new-remote-forward", "Replace remote forward list with provided values (repeatable)")
	fs.Var(&newExtraSSHArgs, "new-extra-ssh-arg", "Replace extra ssh args with provided values (repeatable)")
	fs.Var(&newTags, "new-tag", "Replace tags with provided values (repeatable)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	selectedAlias, selectedID, err := resolveSelector(*alias, *id, fs.Args(), "edit")
	if err != nil {
		return err
	}
	if selectedAlias == "" && selectedID == "" {
		return fmt.Errorf("edit: missing target, set --alias or --id")
	}
	if *clearAlias && strings.TrimSpace(*newAlias) != "" {
		return fmt.Errorf("edit: use either --new-alias or --clear-alias, not both")
	}
	if *clearDescription && strings.TrimSpace(*newDescription) != "" {
		return fmt.Errorf("edit: use either --new-description or --clear-description, not both")
	}
	if *clearProxyJump && strings.TrimSpace(*newProxyJump) != "" {
		return fmt.Errorf("edit: use either --new-proxy-jump or --clear-proxy-jump, not both")
	}
	if *clearGroup && strings.TrimSpace(*newGroup) != "" {
		return fmt.Errorf("edit: use either --new-group or --clear-group, not both")
	}
	if *clearLocalForwards && len(newLocalForwards) > 0 {
		return fmt.Errorf("edit: use either --new-local-forward or --clear-local-forwards, not both")
	}
	if *clearRemoteForwards && len(newRemoteForwards) > 0 {
		return fmt.Errorf("edit: use either --new-remote-forward or --clear-remote-forwards, not both")
	}
	if *clearExtraSSHArgs && len(newExtraSSHArgs) > 0 {
		return fmt.Errorf("edit: use either --new-extra-ssh-arg or --clear-extra-ssh-args, not both")
	}
	if *clearTags && len(newTags) > 0 {
		return fmt.Errorf("edit: use either --new-tag or --clear-tags, not both")
	}

	hasUpdate := strings.TrimSpace(*newHost) != "" ||
		strings.TrimSpace(*newUsername) != "" ||
		*newPort >= 0 ||
		strings.TrimSpace(*newAuthMode) != "" ||
		strings.TrimSpace(*newPassword) != "" ||
		strings.TrimSpace(*newIdentityFile) != "" ||
		strings.TrimSpace(*newProxyJump) != "" ||
		strings.TrimSpace(*newGroup) != "" ||
		len(newLocalForwards) > 0 ||
		len(newRemoteForwards) > 0 ||
		len(newExtraSSHArgs) > 0 ||
		len(newTags) > 0 ||
		strings.TrimSpace(*newAlias) != "" ||
		strings.TrimSpace(*newDescription) != "" ||
		*clearAlias ||
		*clearDescription ||
		*clearProxyJump ||
		*clearGroup ||
		*clearLocalForwards ||
		*clearRemoteForwards ||
		*clearExtraSSHArgs ||
		*clearTags
	if !hasUpdate {
		return fmt.Errorf("edit: no update fields provided")
	}

	connStore := store.NewConnectionStore(connectionFilePath, secretKeyFilePath)
	connFile, err := connStore.Load()
	if err != nil {
		return err
	}
	if len(connFile.Connections) == 0 {
		_, _ = fmt.Fprintln(out, prompttext.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return nil
	}

	current := findConnectionBySelector(&connFile, selectedAlias, selectedID)
	if current == nil {
		_, _ = fmt.Fprintln(out, notFoundMessage(selectedAlias, selectedID))
		return nil
	}

	updated := *current
	if v := strings.TrimSpace(*newHost); v != "" {
		updated.Host = v
	}
	if v := strings.TrimSpace(*newUsername); v != "" {
		updated.Username = v
	}
	if *newPort >= 0 {
		updated.Port = *newPort
	}
	if v := strings.TrimSpace(*newAuthMode); v != "" {
		updated.AuthMode = v
	}
	if v := strings.TrimSpace(*newPassword); v != "" {
		updated.Password = v
	}
	if v := strings.TrimSpace(*newIdentityFile); v != "" {
		updated.IdentityFile = v
	}
	if *clearProxyJump {
		updated.ProxyJump = ""
	} else if v := strings.TrimSpace(*newProxyJump); v != "" {
		updated.ProxyJump = v
	}
	if *clearGroup {
		updated.Group = ""
	} else if v := strings.TrimSpace(*newGroup); v != "" {
		updated.Group = v
	}
	if *clearLocalForwards {
		updated.LocalForwards = nil
	} else if len(newLocalForwards) > 0 {
		updated.LocalForwards = newLocalForwards.Values()
	}
	if *clearRemoteForwards {
		updated.RemoteForwards = nil
	} else if len(newRemoteForwards) > 0 {
		updated.RemoteForwards = newRemoteForwards.Values()
	}
	if *clearExtraSSHArgs {
		updated.ExtraSSHArgs = nil
	} else if len(newExtraSSHArgs) > 0 {
		updated.ExtraSSHArgs = newExtraSSHArgs.Values()
	}
	if *clearTags {
		updated.Tags = nil
	} else if len(newTags) > 0 {
		updated.Tags = newTags.Values()
	}
	if *clearAlias {
		updated.Alias = ""
	} else if v := strings.TrimSpace(*newAlias); v != "" {
		updated.Alias = v
	}
	if *clearDescription {
		updated.Description = ""
	} else if v := strings.TrimSpace(*newDescription); v != "" {
		updated.Description = v
	}

	normalized, err := normalizeImportedConnection(updated)
	if err != nil {
		return err
	}

	wasUpdated := false
	if err := connStore.Update(func(liveConnFile *model.ConnectionFile) error {
		var updateErr error
		wasUpdated, updateErr = liveConnFile.UpdateConnectionByID(current.ID, normalized)
		return updateErr
	}); err != nil {
		return err
	}
	if !wasUpdated {
		_, _ = fmt.Fprintln(out, notFoundMessage(selectedAlias, selectedID))
		return nil
	}

	_, _ = fmt.Fprintln(out, prompttext.DefaultPromptTexts.SuccessMessages.SSHConnectionUpdated)
	return nil
}
