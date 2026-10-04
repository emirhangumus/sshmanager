package commands

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/emirhangumus/sshmanager/v2/internal/model"
	"github.com/emirhangumus/sshmanager/v2/internal/store"
	prompttext "github.com/emirhangumus/sshmanager/v2/internal/ui/prompt"
)

type listOutputItem struct {
	ID             string   `json:"id"`
	Alias          string   `json:"alias,omitempty"`
	Username       string   `json:"username"`
	Host           string   `json:"host"`
	Port           int      `json:"port"`
	AuthMode       string   `json:"authMode"`
	Password       string   `json:"password,omitempty"`
	IdentityFile   string   `json:"identityFile,omitempty"`
	ProxyJump      string   `json:"proxyJump,omitempty"`
	LocalForwards  []string `json:"localForwards,omitempty"`
	RemoteForwards []string `json:"remoteForwards,omitempty"`
	ExtraSSHArgs   []string `json:"extraSSHArgs,omitempty"`
	Group          string   `json:"group,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	Description    string   `json:"description,omitempty"`
}

func HandleList(connectionFilePath, secretKeyFilePath string, args []string) error {
	return handleList(connectionFilePath, secretKeyFilePath, args, os.Stdout)
}

func handleList(connectionFilePath, secretKeyFilePath string, args []string, out io.Writer) error {
	args = expandCombinedShortFlags(args, "py")

	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := fs.Bool("json", false, "Output JSON")
	field := fs.String("field", "", "Output only one field per line (id|alias|username|host|port|auth-mode|password|identity-file|proxy-jump|local-forwards|remote-forwards|extra-ssh-args|group|tags|description|target)")
	groupFilter := fs.String("group", "", "Filter by group")
	showPasswords := fs.Bool("p", false, "Show saved passwords too (requires confirmation unless -y is set)")
	autoYes := fs.Bool("y", false, "Auto-confirm the password reveal prompt")
	var tagFilters stringListFlag
	fs.Var(&tagFilters, "tag", "Filter by tag (repeatable)")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected arguments for list: %s", strings.Join(fs.Args(), " "))
	}
	if *jsonOutput && strings.TrimSpace(*field) != "" {
		return errors.New("--json and --field cannot be used together")
	}

	if *showPasswords && !*autoYes {
		confirmed, err := confirmShowPasswords()
		if err != nil {
			return err
		}
		if !confirmed {
			_, _ = fmt.Fprintln(out, prompttext.DefaultPromptTexts.SuccessMessages.OperationCancelled)
			return nil
		}
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

	items := make([]listOutputItem, 0, len(connFile.Connections))
	for _, conn := range connFile.Connections {
		if !matchesListFilters(conn, *groupFilter, tagFilters.Values()) {
			continue
		}
		item := listOutputItem{
			ID:             conn.ID,
			Alias:          strings.TrimSpace(conn.Alias),
			Username:       conn.Username,
			Host:           conn.Host,
			Port:           conn.EffectivePort(),
			AuthMode:       conn.EffectiveAuthMode(),
			IdentityFile:   strings.TrimSpace(conn.IdentityFile),
			ProxyJump:      strings.TrimSpace(conn.ProxyJump),
			LocalForwards:  model.NormalizeStringList(conn.LocalForwards),
			RemoteForwards: model.NormalizeStringList(conn.RemoteForwards),
			ExtraSSHArgs:   model.NormalizeStringList(conn.ExtraSSHArgs),
			Group:          strings.TrimSpace(conn.Group),
			Tags:           model.NormalizeTags(conn.Tags),
			Description:    conn.Description,
		}
		if *showPasswords {
			item.Password = conn.Password
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		_, _ = fmt.Fprintln(out, prompttext.DefaultPromptTexts.ErrorMessages.NoSSHConnectionsFound)
		return nil
	}

	if *jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(items) // #nosec G117 -- passwords are omitted unless the user explicitly requests -p.
	}

	if strings.TrimSpace(*field) != "" {
		for _, item := range items {
			value, err := listFieldValue(item, *field)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(out, value)
		}
		return nil
	}

	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	if *showPasswords {
		_, _ = fmt.Fprintln(tw, "ALIAS\tUSERNAME\tHOST\tPORT\tAUTH_MODE\tPASSWORD\tGROUP\tTAGS\tDESCRIPTION")
	} else {
		_, _ = fmt.Fprintln(tw, "ALIAS\tUSERNAME\tHOST\tPORT\tAUTH_MODE\tGROUP\tTAGS\tDESCRIPTION")
	}
	for _, item := range items {
		alias := item.Alias
		if alias == "" {
			alias = "-"
		}
		authMode := item.AuthMode
		if authMode == "" {
			authMode = model.AuthModeAgent
		}
		if *showPasswords {
			password := item.Password
			if password == "" {
				password = "-"
			}
			_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\t%s\t%s\t%s\t%s\n",
				alias,
				item.Username,
				item.Host,
				item.Port,
				authMode,
				password,
				item.Group,
				strings.Join(item.Tags, ","),
				item.Description,
			)
			continue
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\t%s\t%s\t%s\n",
			alias,
			item.Username,
			item.Host,
			item.Port,
			authMode,
			item.Group,
			strings.Join(item.Tags, ","),
			item.Description,
		)
	}
	return tw.Flush()
}

func confirmShowPasswords() (bool, error) {
	value, err := prompttext.InputPrompt(
		"Show saved passwords in plaintext? Type 'yes' to continue",
		"",
		false,
		nil,
	)
	if err != nil {
		if prompttext.IsCancelError(err) {
			return false, nil
		}
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(value), "yes"), nil
}

// expandCombinedShortFlags rewrites getopt-style combined short flags (e.g.
// "-py") into their individual flags (e.g. "-p", "-y") so the stdlib flag
// package, which has no notion of combined short flags, can parse them.
func expandCombinedShortFlags(args []string, letters string) []string {
	expanded := make([]string, 0, len(args))
	for _, arg := range args {
		if !isCombinedShortFlag(arg, letters) {
			expanded = append(expanded, arg)
			continue
		}
		for _, ch := range arg[1:] {
			expanded = append(expanded, "-"+string(ch))
		}
	}
	return expanded
}

func isCombinedShortFlag(arg, letters string) bool {
	if len(arg) < 3 || arg[0] != '-' || arg[1] == '-' {
		return false
	}
	for _, ch := range arg[1:] {
		if !strings.ContainsRune(letters, ch) {
			return false
		}
	}
	return true
}

func listFieldValue(item listOutputItem, field string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "id":
		return item.ID, nil
	case "alias":
		return item.Alias, nil
	case "username":
		return item.Username, nil
	case "host":
		return item.Host, nil
	case "port":
		return strconv.Itoa(item.Port), nil
	case "auth-mode", "auth_mode", "authmode":
		return item.AuthMode, nil
	case "password":
		return item.Password, nil
	case "identity-file", "identity_file", "identityfile":
		return item.IdentityFile, nil
	case "proxy-jump", "proxy_jump", "proxyjump":
		return item.ProxyJump, nil
	case "local-forwards", "local_forwards", "localforwards":
		return strings.Join(item.LocalForwards, ","), nil
	case "remote-forwards", "remote_forwards", "remoteforwards":
		return strings.Join(item.RemoteForwards, ","), nil
	case "extra-ssh-args", "extra_ssh_args", "extrasshargs":
		return strings.Join(item.ExtraSSHArgs, ","), nil
	case "group":
		return item.Group, nil
	case "tags":
		return strings.Join(item.Tags, ","), nil
	case "description":
		return item.Description, nil
	case "target":
		return fmt.Sprintf("%s@%s", item.Username, item.Host), nil
	default:
		return "", fmt.Errorf("unknown list field %q", field)
	}
}

func matchesListFilters(conn model.SSHConnection, groupFilter string, tagFilters []string) bool {
	groupNeedle := strings.ToLower(strings.TrimSpace(groupFilter))
	if groupNeedle != "" {
		if strings.ToLower(strings.TrimSpace(conn.Group)) != groupNeedle {
			return false
		}
	}

	needTags := model.NormalizeTags(tagFilters)
	if len(needTags) == 0 {
		return true
	}

	have := make(map[string]struct{}, len(conn.Tags))
	for _, tag := range model.NormalizeTags(conn.Tags) {
		have[strings.ToLower(tag)] = struct{}{}
	}
	for _, tag := range needTags {
		if _, ok := have[strings.ToLower(tag)]; !ok {
			return false
		}
	}
	return true
}
