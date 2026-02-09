package config

type BehaviourConfig struct {
	ContinueAfterSSHExit bool `yaml:"continueAfterSSHExit"`
}

type SSHManagerConfig struct {
	Behaviour BehaviourConfig `yaml:"behaviour"`
}

func Default() SSHManagerConfig {
	return SSHManagerConfig{
		Behaviour: BehaviourConfig{
			ContinueAfterSSHExit: false,
		},
	}
}

func (c *SSHManagerConfig) SetDefault() {
	*c = Default()
}
