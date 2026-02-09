package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/emirhangumus/sshmanager/internal/storage"
)

func LoadConfig(configFilePath string) (SSHManagerConfig, error) {
	cfg := Default()

	isEmpty, err := storage.IsFileEmpty(configFilePath)
	if err != nil {
		return SSHManagerConfig{}, fmt.Errorf("failed to inspect config file: %w", err)
	}
	if isEmpty {
		return cfg, nil
	}

	if err := storage.ReadYAMLFile(configFilePath, &cfg); err != nil {
		return SSHManagerConfig{}, fmt.Errorf("failed to load configuration: %w", err)
	}
	return cfg, nil
}

func SaveConfig(configFilePath string, cfg SSHManagerConfig) error {
	if err := storage.WriteYAMLFile(configFilePath, cfg, 0o600); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	return nil
}

func SetConfig(configFilePath, configName, configValue string) error {
	cfg, err := LoadConfig(configFilePath)
	if err != nil {
		return err
	}

	switch configName {
	case "behaviour.continueAfterSSHExit":
		v, err := parseBoolValue(configValue)
		if err != nil {
			return err
		}
		cfg.Behaviour.ContinueAfterSSHExit = v
	default:
		return errors.New("unknown configuration name: " + configName)
	}

	return SaveConfig(configFilePath, cfg)
}

func parseBoolValue(v string) (bool, error) {
	parsed, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(v)))
	if err != nil {
		return false, errors.New("invalid value for behaviour.continueAfterSSHExit, expected 'true' or 'false'")
	}
	return parsed, nil
}
