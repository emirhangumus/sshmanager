package config

type BehaviourConfig struct {
	ContinueAfterSSHExit     bool `yaml:"continueAfterSSHExit"`
	ShowCredentialsOnConnect bool `yaml:"showCredentialsOnConnect"`
}

type SSHManagerConfig struct {
	Behaviour BehaviourConfig `yaml:"behaviour"`
}

func Default() SSHManagerConfig {
	return SSHManagerConfig{
		Behaviour: BehaviourConfig{
			ContinueAfterSSHExit:     false,
			ShowCredentialsOnConnect: false,
		},
	}
}

func (c *SSHManagerConfig) SetDefault() {
	*c = Default()
}
