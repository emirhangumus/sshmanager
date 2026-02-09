package model

import "strings"

const (
	AuthModePassword = "password"
	AuthModeKey      = "key"
	AuthModeAgent    = "agent"
	DefaultSSHPort   = 22
)

// SSHConnection stores credentials and metadata for a remote host.
type SSHConnection struct {
	ID             string   `yaml:"id" json:"id"`
	Username       string   `yaml:"username" json:"username"`
	Host           string   `yaml:"host" json:"host"`
	Port           int      `yaml:"port,omitempty" json:"port,omitempty"`
	AuthMode       string   `yaml:"authMode,omitempty" json:"authMode,omitempty"`
	Password       string   `yaml:"password,omitempty" json:"password,omitempty"`
	IdentityFile   string   `yaml:"identityFile,omitempty" json:"identityFile,omitempty"`
	ProxyJump      string   `yaml:"proxyJump,omitempty" json:"proxyJump,omitempty"`
	LocalForwards  []string `yaml:"localForwards,omitempty" json:"localForwards,omitempty"`
	RemoteForwards []string `yaml:"remoteForwards,omitempty" json:"remoteForwards,omitempty"`
	ExtraSSHArgs   []string `yaml:"extraSSHArgs,omitempty" json:"extraSSHArgs,omitempty"`
	Group          string   `yaml:"group,omitempty" json:"group,omitempty"`
	Tags           []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Description    string   `yaml:"description,omitempty" json:"description,omitempty"`
	Alias          string   `yaml:"alias,omitempty" json:"alias,omitempty"`
}

func (c SSHConnection) EffectivePort() int {
	if c.Port > 0 {
		return c.Port
	}
	return DefaultSSHPort
}

func (c SSHConnection) EffectiveAuthMode() string {
	return ResolveAuthMode(c.AuthMode, c.Password, c.IdentityFile)
}

func NormalizeAuthMode(mode string) string {
	return strings.ToLower(strings.TrimSpace(mode))
}

func IsValidAuthMode(mode string) bool {
	switch NormalizeAuthMode(mode) {
	case AuthModePassword, AuthModeKey, AuthModeAgent:
		return true
	default:
		return false
	}
}

func ResolveAuthMode(mode, password, identityFile string) string {
	norm := NormalizeAuthMode(mode)
	if IsValidAuthMode(norm) {
		return norm
	}

	if strings.TrimSpace(password) != "" {
		return AuthModePassword
	}
	if strings.TrimSpace(identityFile) != "" {
		return AuthModeKey
	}

	return AuthModeAgent
}
