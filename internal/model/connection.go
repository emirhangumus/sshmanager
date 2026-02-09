package model

// SSHConnection stores credentials and metadata for a remote host.
type SSHConnection struct {
	ID          string `yaml:"id"`
	Username    string `yaml:"username"`
	Host        string `yaml:"host"`
	Password    string `yaml:"password"`
	Description string `yaml:"description,omitempty"`
	Alias       string `yaml:"alias,omitempty"`
}
