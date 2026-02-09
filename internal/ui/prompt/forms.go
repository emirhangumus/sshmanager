package prompt

import (
	"errors"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/emirhangumus/sshmanager/internal/model"
	"github.com/manifoldco/promptui"
)

var (
	hostLabelPattern = regexp.MustCompile(`^[A-Za-z0-9_](?:[A-Za-z0-9_-]{0,61}[A-Za-z0-9_])?$`)
	aliasPattern     = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

func validateText(input string) error {
	if strings.TrimSpace(input) == "" {
		return errors.New("this field is required")
	}
	return nil
}

func validateHost(input string) error {
	host := strings.TrimSpace(input)
	if host == "" {
		return errors.New("host is required")
	}
	if strings.ContainsAny(host, " \t\r\n") {
		return errors.New("host cannot contain whitespace")
	}

	if net.ParseIP(host) != nil {
		return nil
	}

	if len(host) > 253 {
		return errors.New("host is too long")
	}

	labels := strings.Split(host, ".")
	for _, label := range labels {
		if label == "" || !hostLabelPattern.MatchString(label) {
			return errors.New("invalid host format")
		}
	}

	return nil
}

func validateAlias(input string) error {
	alias := strings.TrimSpace(input)
	if alias == "" {
		return nil
	}
	if len(alias) > 64 {
		return errors.New("alias must be 64 characters or fewer")
	}
	if !aliasPattern.MatchString(alias) {
		return errors.New("alias may only contain letters, numbers, '.', '_' or '-'")
	}
	return nil
}

func validatePort(input string) error {
	portRaw := strings.TrimSpace(input)
	if portRaw == "" {
		return nil
	}

	port, err := strconv.Atoi(portRaw)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("port must be an integer between 1 and 65535")
	}
	return nil
}

func validateAuthMode(input string) error {
	mode := model.NormalizeAuthMode(input)
	if mode == "" {
		return errors.New("auth mode is required")
	}
	if !model.IsValidAuthMode(mode) {
		return errors.New("auth mode must be one of: password, key, agent")
	}
	return nil
}

func validateProxyJump(input string) error {
	return model.ValidateProxyJump(strings.TrimSpace(input))
}

func validateForwardList(input string) error {
	return model.ValidateForwardSpecs(parseCommaSeparatedValues(input))
}

func validateExtraSSHArgs(input string) error {
	return model.ValidateExtraSSHArgs(parseCommaSeparatedValues(input))
}

func validateGroup(input string) error {
	return model.ValidateGroup(strings.TrimSpace(input))
}

func validateTags(input string) error {
	return model.ValidateTags(parseCommaSeparatedValues(input))
}

func AddSSHConnectionPrompt() (model.SSHConnection, error) {
	host, err := runHostPrompt(DefaultPromptTexts.EnterHost, "")
	if err != nil {
		return model.SSHConnection{}, err
	}

	username, err := runValidatedPrompt(DefaultPromptTexts.EnterUsername, "")
	if err != nil {
		return model.SSHConnection{}, err
	}

	portRaw, err := runPortPrompt(DefaultPromptTexts.EnterPort, "")
	if err != nil {
		return model.SSHConnection{}, err
	}

	authModeRaw, err := runAuthModePrompt(DefaultPromptTexts.EnterAuthMode, model.AuthModePassword)
	if err != nil {
		return model.SSHConnection{}, err
	}
	authMode := model.NormalizeAuthMode(authModeRaw)

	password, err := runPasswordPrompt(DefaultPromptTexts.EnterPassword, "", authMode == model.AuthModePassword)
	if err != nil {
		return model.SSHConnection{}, err
	}

	identityFile, err := runIdentityFilePrompt(DefaultPromptTexts.EnterIdentityFile, "", authMode == model.AuthModeKey)
	if err != nil {
		return model.SSHConnection{}, err
	}

	proxyJump, err := runProxyJumpPrompt(DefaultPromptTexts.EnterProxyJump, "")
	if err != nil {
		return model.SSHConnection{}, err
	}

	localForwardsRaw, err := runForwardListPrompt(DefaultPromptTexts.EnterLocalForwards, "")
	if err != nil {
		return model.SSHConnection{}, err
	}

	remoteForwardsRaw, err := runForwardListPrompt(DefaultPromptTexts.EnterRemoteForwards, "")
	if err != nil {
		return model.SSHConnection{}, err
	}

	extraSSHArgsRaw, err := runExtraSSHArgsPrompt(DefaultPromptTexts.EnterExtraSSHArgs, "")
	if err != nil {
		return model.SSHConnection{}, err
	}

	group, err := runGroupPrompt(DefaultPromptTexts.EnterGroup, "")
	if err != nil {
		return model.SSHConnection{}, err
	}

	tagsRaw, err := runTagsPrompt(DefaultPromptTexts.EnterTags, "")
	if err != nil {
		return model.SSHConnection{}, err
	}

	description, err := runPlainPrompt(DefaultPromptTexts.EnterDescription, "")
	if err != nil {
		return model.SSHConnection{}, err
	}

	alias, err := runAliasPrompt(DefaultPromptTexts.EnterAlias, "")
	if err != nil {
		return model.SSHConnection{}, err
	}

	conn := normalizeConnection(model.SSHConnection{
		Username:       username,
		Host:           host,
		Port:           parsePort(portRaw),
		AuthMode:       authMode,
		Password:       password,
		IdentityFile:   identityFile,
		ProxyJump:      proxyJump,
		LocalForwards:  parseCommaSeparatedValues(localForwardsRaw),
		RemoteForwards: parseCommaSeparatedValues(remoteForwardsRaw),
		ExtraSSHArgs:   parseCommaSeparatedValues(extraSSHArgsRaw),
		Group:          group,
		Tags:           parseCommaSeparatedValues(tagsRaw),
		Description:    description,
		Alias:          alias,
	})
	conn = normalizeAuthSensitiveFields(conn)
	return conn, nil
}

func EditSSHConnectionPrompt(conn *model.SSHConnection) (model.SSHConnection, error) {
	host, err := runHostPrompt(DefaultPromptTexts.EditHost, conn.Host)
	if err != nil {
		return model.SSHConnection{}, err
	}

	username, err := runValidatedPrompt(DefaultPromptTexts.EditUsername, conn.Username)
	if err != nil {
		return model.SSHConnection{}, err
	}

	portDefault := ""
	if conn.Port > 0 {
		portDefault = strconv.Itoa(conn.Port)
	}
	portRaw, err := runPortPrompt(DefaultPromptTexts.EditPort, portDefault)
	if err != nil {
		return model.SSHConnection{}, err
	}

	authModeRaw, err := runAuthModePrompt(DefaultPromptTexts.EditAuthMode, conn.EffectiveAuthMode())
	if err != nil {
		return model.SSHConnection{}, err
	}
	authMode := model.NormalizeAuthMode(authModeRaw)

	password, err := runPasswordPrompt(DefaultPromptTexts.EditPassword, conn.Password, authMode == model.AuthModePassword)
	if err != nil {
		return model.SSHConnection{}, err
	}

	identityFile, err := runIdentityFilePrompt(DefaultPromptTexts.EditIdentityFile, conn.IdentityFile, authMode == model.AuthModeKey)
	if err != nil {
		return model.SSHConnection{}, err
	}

	proxyJump, err := runProxyJumpPrompt(DefaultPromptTexts.EditProxyJump, conn.ProxyJump)
	if err != nil {
		return model.SSHConnection{}, err
	}

	localForwardsRaw, err := runForwardListPrompt(DefaultPromptTexts.EditLocalForwards, joinListForPrompt(conn.LocalForwards))
	if err != nil {
		return model.SSHConnection{}, err
	}

	remoteForwardsRaw, err := runForwardListPrompt(DefaultPromptTexts.EditRemoteForwards, joinListForPrompt(conn.RemoteForwards))
	if err != nil {
		return model.SSHConnection{}, err
	}

	extraSSHArgsRaw, err := runExtraSSHArgsPrompt(DefaultPromptTexts.EditExtraSSHArgs, joinListForPrompt(conn.ExtraSSHArgs))
	if err != nil {
		return model.SSHConnection{}, err
	}

	group, err := runGroupPrompt(DefaultPromptTexts.EditGroup, conn.Group)
	if err != nil {
		return model.SSHConnection{}, err
	}

	tagsRaw, err := runTagsPrompt(DefaultPromptTexts.EditTags, joinListForPrompt(conn.Tags))
	if err != nil {
		return model.SSHConnection{}, err
	}

	description, err := runPlainPrompt(DefaultPromptTexts.EditDescription, conn.Description)
	if err != nil {
		return model.SSHConnection{}, err
	}

	alias, err := runAliasPrompt(DefaultPromptTexts.EditAlias, conn.Alias)
	if err != nil {
		return model.SSHConnection{}, err
	}

	updated := normalizeConnection(model.SSHConnection{
		ID:             conn.ID,
		Username:       username,
		Host:           host,
		Port:           parsePort(portRaw),
		AuthMode:       authMode,
		Password:       password,
		IdentityFile:   identityFile,
		ProxyJump:      proxyJump,
		LocalForwards:  parseCommaSeparatedValues(localForwardsRaw),
		RemoteForwards: parseCommaSeparatedValues(remoteForwardsRaw),
		ExtraSSHArgs:   parseCommaSeparatedValues(extraSSHArgsRaw),
		Group:          group,
		Tags:           parseCommaSeparatedValues(tagsRaw),
		Description:    description,
		Alias:          alias,
	})
	updated = normalizeAuthSensitiveFields(updated)
	return updated, nil
}

func runValidatedPrompt(label, defaultValue string) (string, error) {
	p := promptui.Prompt{Label: label, Default: defaultValue, Validate: validateText}
	return p.Run()
}

func runHostPrompt(label, defaultValue string) (string, error) {
	p := promptui.Prompt{Label: label, Default: defaultValue, Validate: validateHost}
	return p.Run()
}

func runPortPrompt(label, defaultValue string) (string, error) {
	p := promptui.Prompt{Label: label, Default: defaultValue, Validate: validatePort}
	return p.Run()
}

func runAuthModePrompt(label, defaultValue string) (string, error) {
	p := promptui.Prompt{Label: label, Default: defaultValue, Validate: validateAuthMode}
	return p.Run()
}

func runPasswordPrompt(label, defaultValue string, required bool) (string, error) {
	var validator func(string) error
	if required {
		validator = validateText
	}
	p := promptui.Prompt{Label: label, Default: defaultValue, Mask: '*', Validate: validator}
	return p.Run()
}

func runPlainPrompt(label, defaultValue string) (string, error) {
	p := promptui.Prompt{Label: label, Default: defaultValue}
	return p.Run()
}

func runAliasPrompt(label, defaultValue string) (string, error) {
	p := promptui.Prompt{Label: label, Default: defaultValue, Validate: validateAlias}
	return p.Run()
}

func runIdentityFilePrompt(label, defaultValue string, required bool) (string, error) {
	var validator func(string) error
	if required {
		validator = validateText
	}
	p := promptui.Prompt{Label: label, Default: defaultValue, Validate: validator}
	return p.Run()
}

func runProxyJumpPrompt(label, defaultValue string) (string, error) {
	p := promptui.Prompt{Label: label, Default: defaultValue, Validate: validateProxyJump}
	return p.Run()
}

func runForwardListPrompt(label, defaultValue string) (string, error) {
	p := promptui.Prompt{Label: label, Default: defaultValue, Validate: validateForwardList}
	return p.Run()
}

func runExtraSSHArgsPrompt(label, defaultValue string) (string, error) {
	p := promptui.Prompt{Label: label, Default: defaultValue, Validate: validateExtraSSHArgs}
	return p.Run()
}

func runGroupPrompt(label, defaultValue string) (string, error) {
	p := promptui.Prompt{Label: label, Default: defaultValue, Validate: validateGroup}
	return p.Run()
}

func runTagsPrompt(label, defaultValue string) (string, error) {
	p := promptui.Prompt{Label: label, Default: defaultValue, Validate: validateTags}
	return p.Run()
}

func normalizeConnection(conn model.SSHConnection) model.SSHConnection {
	conn.Username = strings.TrimSpace(conn.Username)
	conn.Host = strings.TrimSpace(conn.Host)
	conn.AuthMode = model.NormalizeAuthMode(conn.AuthMode)
	conn.IdentityFile = strings.TrimSpace(conn.IdentityFile)
	conn.ProxyJump = strings.TrimSpace(conn.ProxyJump)
	conn.LocalForwards = model.NormalizeStringList(conn.LocalForwards)
	conn.RemoteForwards = model.NormalizeStringList(conn.RemoteForwards)
	conn.ExtraSSHArgs = model.NormalizeStringList(conn.ExtraSSHArgs)
	conn.Group = strings.TrimSpace(conn.Group)
	conn.Tags = model.NormalizeTags(conn.Tags)
	conn.Description = strings.TrimSpace(conn.Description)
	conn.Alias = strings.TrimSpace(conn.Alias)
	return conn
}

func normalizeAuthSensitiveFields(conn model.SSHConnection) model.SSHConnection {
	switch conn.AuthMode {
	case model.AuthModePassword:
		conn.IdentityFile = ""
	case model.AuthModeKey:
		conn.Password = ""
	case model.AuthModeAgent:
		conn.Password = ""
		conn.IdentityFile = ""
	}
	return conn
}

func parseCommaSeparatedValues(raw string) []string {
	parts := strings.Split(raw, ",")
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func joinListForPrompt(values []string) string {
	return strings.Join(model.NormalizeStringList(values), ", ")
}

func parsePort(portRaw string) int {
	trimmed := strings.TrimSpace(portRaw)
	if trimmed == "" {
		return 0
	}
	port, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0
	}
	return port
}
