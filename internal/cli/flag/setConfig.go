package flag

import (
	"errors"
	"fmt"
	"github.com/emirhangumus/sshmanager/internal/storage"
)

type SSHManagerConfig struct {
	Behaviour struct {
		ContinueAfterSSHExit bool `yaml:"continueAfterSSHExit"`
	}
}

func (config *SSHManagerConfig) SetDefault() {
	config.Behaviour.ContinueAfterSSHExit = false // Default value for continueAfterSSHExit
}

// SetConfig sets a configuration value in the specified configuration file.
// Config file is a yaml file that contains various settings for the SSHManager application.
func SetConfig(configFilePath string, configName string, configValue string) error {
	// Load the existing configuration
	config, err := LoadConfig(configFilePath)
	if err != nil {
		return err
	}

	// Set the specified configuration value
	switch configName {
	case "behaviour.continueAfterSSHExit":
		if configValue == "true" {
			config.Behaviour.ContinueAfterSSHExit = true
		} else if configValue == "false" {
			config.Behaviour.ContinueAfterSSHExit = false
		} else {
			return errors.New("invalid value for behaviour.continueAfterSSHExit, expected 'true' or 'false'")
		}
	default:
		return errors.New("unknown configuration name: " + configName)
	}

	// Save the updated configuration back to the file
	return SaveConfig(configFilePath, config)
}

// LoadConfig loads the configuration from the specified file.
func LoadConfig(configFilePath string) (SSHManagerConfig, error) {
	var config SSHManagerConfig
	err := storage.ReadYAMLFile(configFilePath, &config)
	if err != nil {
		return SSHManagerConfig{}, errors.New("failed to load configuration: " + err.Error())
	}
	return config, nil
}

// SaveConfig saves the configuration to the specified file.
func SaveConfig(configFilePath string, config SSHManagerConfig) error {
	err := storage.WriteYAMLFile(configFilePath, config)
	if err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	return nil
}
