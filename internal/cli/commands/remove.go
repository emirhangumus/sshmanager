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

func HandleRemove(connectionFilePath, secretKeyFilePath string) error {
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
	selector := promptui.Select{Label: prompttext.DefaultPromptTexts.SelectAConnectionToRemove, Items: items}
	idx, _, err := selector.Run()
	if err != nil {
		if prompttext.IsCancelError(err) {
			fmt.Println(prompttext.DefaultPromptTexts.SuccessMessages.OperationCancelled)
		}
		return nil
	}

	connID := items[idx].ConnectionID
	removed := false
	if err := connStore.Update(func(liveConnFile *model.ConnectionFile) error {
		removed = liveConnFile.RemoveConnectionByID(connID)
		return nil
	}); err != nil {
		return err
	}
	if !removed {
		fmt.Println(prompttext.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return nil
	}

	fmt.Println(prompttext.DefaultPromptTexts.SuccessMessages.SSHConnectionRemoved)
	return nil
}

func HandleRemoveArgs(connectionFilePath, secretKeyFilePath string, args []string) error {
	return handleRemoveArgs(connectionFilePath, secretKeyFilePath, args, os.Stdout)
}

func handleRemoveArgs(connectionFilePath, secretKeyFilePath string, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("remove", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	alias := fs.String("alias", "", "Connection alias")
	id := fs.String("id", "", "Connection ID")
	yes := fs.Bool("yes", false, "Skip confirmation prompt")

	if err := fs.Parse(args); err != nil {
		return err
	}

	selectedAlias, selectedID, err := resolveSelector(*alias, *id, fs.Args(), "remove")
	if err != nil {
		return err
	}

	if selectedAlias == "" && selectedID == "" {
		return HandleRemove(connectionFilePath, secretKeyFilePath)
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

	conn := findConnectionBySelector(&connFile, selectedAlias, selectedID)
	if conn == nil {
		_, _ = fmt.Fprintln(out, notFoundMessage(selectedAlias, selectedID))
		return nil
	}

	if !*yes {
		confirmed, err := confirmRemove(conn)
		if err != nil {
			return err
		}
		if !confirmed {
			_, _ = fmt.Fprintln(out, prompttext.DefaultPromptTexts.SuccessMessages.OperationCancelled)
			return nil
		}
	}

	removed := false
	if err := connStore.Update(func(liveConnFile *model.ConnectionFile) error {
		removed = liveConnFile.RemoveConnectionByID(conn.ID)
		return nil
	}); err != nil {
		return err
	}

	if !removed {
		_, _ = fmt.Fprintln(out, notFoundMessage(selectedAlias, selectedID))
		return nil
	}

	_, _ = fmt.Fprintln(out, prompttext.DefaultPromptTexts.SuccessMessages.SSHConnectionRemoved)
	return nil
}

func confirmRemove(conn *model.SSHConnection) (bool, error) {
	target := fmt.Sprintf("%s@%s", strings.TrimSpace(conn.Username), strings.TrimSpace(conn.Host))
	if alias := strings.TrimSpace(conn.Alias); alias != "" {
		target = fmt.Sprintf("%s (%s)", target, alias)
	}

	p := promptui.Prompt{
		Label: fmt.Sprintf("Remove %s? Type 'yes' to continue", target),
	}
	value, err := p.Run()
	if err != nil {
		if prompttext.IsCancelError(err) {
			return false, nil
		}
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(value), "yes"), nil
}
