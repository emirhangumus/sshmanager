package commands

import (
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/emirhangumus/sshmanager/internal/model"
	"github.com/emirhangumus/sshmanager/internal/store"
	prompttext "github.com/emirhangumus/sshmanager/internal/ui/prompt"
	"github.com/manifoldco/promptui"
)

var renameAliasPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func HandleRename(connectionFilePath, secretKeyFilePath string) error {
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
	selector := promptui.Select{Label: "Select a connection to rename alias", Items: items}
	idx, _, err := selector.Run()
	if err != nil {
		if prompttext.IsCancelError(err) {
			fmt.Println(prompttext.DefaultPromptTexts.SuccessMessages.OperationCancelled)
		}
		return nil
	}

	connID := items[idx].ConnectionID
	current := connFile.GetConnectionByID(connID)
	if current == nil {
		fmt.Println(prompttext.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return nil
	}

	prompt := promptui.Prompt{
		Label:    "New Alias",
		Default:  strings.TrimSpace(current.Alias),
		Validate: validateRenameAlias,
	}
	newAlias, err := prompt.Run()
	if err != nil {
		if prompttext.IsCancelError(err) {
			fmt.Println(prompttext.DefaultPromptTexts.SuccessMessages.OperationCancelled)
		}
		return nil
	}

	return renameConnectionAlias(connectionFilePath, secretKeyFilePath, connID, "", newAlias, os.Stdout)
}

func HandleRenameArgs(connectionFilePath, secretKeyFilePath string, args []string) error {
	return handleRenameArgs(connectionFilePath, secretKeyFilePath, args, os.Stdout)
}

func handleRenameArgs(connectionFilePath, secretKeyFilePath string, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("rename", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	alias := fs.String("alias", "", "Current alias")
	id := fs.String("id", "", "Connection ID")
	toAlias := fs.String("to", "", "New alias")

	if err := fs.Parse(args); err != nil {
		return err
	}

	selectedAlias, selectedID, err := resolveSelector(*alias, *id, fs.Args(), "rename")
	if err != nil {
		return err
	}
	if selectedAlias == "" && selectedID == "" {
		return HandleRename(connectionFilePath, secretKeyFilePath)
	}

	return renameConnectionAlias(connectionFilePath, secretKeyFilePath, selectedID, selectedAlias, *toAlias, out)
}

func renameConnectionAlias(connectionFilePath, secretKeyFilePath, selectedID, selectedAlias, toAlias string, out io.Writer) error {
	newAlias := strings.TrimSpace(toAlias)
	if err := validateRenameAlias(newAlias); err != nil {
		return err
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

	target := findConnectionBySelector(&connFile, selectedAlias, selectedID)
	if target == nil {
		_, _ = fmt.Fprintln(out, notFoundMessage(selectedAlias, selectedID))
		return nil
	}

	renamed := false
	if err := connStore.Update(func(liveConnFile *model.ConnectionFile) error {
		current := liveConnFile.GetConnectionByID(target.ID)
		if current == nil {
			return nil
		}
		updated := *current
		updated.Alias = newAlias
		var updateErr error
		renamed, updateErr = liveConnFile.UpdateConnectionByID(target.ID, updated)
		return updateErr
	}); err != nil {
		return err
	}

	if !renamed {
		_, _ = fmt.Fprintln(out, notFoundMessage(selectedAlias, selectedID))
		return nil
	}

	_, _ = fmt.Fprintln(out, prompttext.DefaultPromptTexts.SuccessMessages.SSHConnectionRenamed)
	return nil
}

func validateRenameAlias(input string) error {
	alias := strings.TrimSpace(input)
	if alias == "" {
		return fmt.Errorf("alias is required")
	}
	if len(alias) > 64 {
		return fmt.Errorf("alias must be 64 characters or fewer")
	}
	if !renameAliasPattern.MatchString(alias) {
		return fmt.Errorf("alias may only contain letters, numbers, '.', '_' or '-'")
	}
	return nil
}
