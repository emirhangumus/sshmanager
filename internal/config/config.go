package config

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
)

type BehaviourConfig struct {
	ContinueAfterSSHExit     bool `yaml:"continueAfterSSHExit"`
	ShowCredentialsOnConnect bool `yaml:"showCredentialsOnConnect"`
}

type SSHManagerConfig struct {
	Behaviour BehaviourConfig `yaml:"behaviour"`
	Security  SecurityConfig  `yaml:"security"`
}

// SecurityConfig selects protection for the saved connection encryption key.
type SecurityConfig struct {
	KeyStorage string `yaml:"keyStorage"`
}

// UnmarshalYAML decodes with defaults so legacy configs select keyring,
// while explicitly invalid values still fail validation.
func (c *SSHManagerConfig) UnmarshalYAML(node *yaml.Node) error {
	type plain SSHManagerConfig
	value := plain(Default())
	if err := node.Decode(&value); err != nil {
		return err
	}
	*c = SSHManagerConfig(value)
	return nil
}

// UnmarshalJSON applies the same defaults to legacy recovery snapshots.
func (c *SSHManagerConfig) UnmarshalJSON(data []byte) error {
	type plain SSHManagerConfig
	value := plain(Default())
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*c = SSHManagerConfig(value)
	return nil
}

func Default() SSHManagerConfig {
	return SSHManagerConfig{
		Security: SecurityConfig{KeyStorage: "keyring"},
		Behaviour: BehaviourConfig{
			ContinueAfterSSHExit:     false,
			ShowCredentialsOnConnect: false,
		},
	}
}

func (c *SSHManagerConfig) SetDefault() {
	*c = Default()
}
