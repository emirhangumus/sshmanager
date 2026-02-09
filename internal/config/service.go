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
		v, err := parseBoolValue(configName, configValue)
		if err != nil {
			return err
		}
		cfg.Behaviour.ContinueAfterSSHExit = v
	case "behaviour.showCredentialsOnConnect":
		v, err := parseBoolValue(configName, configValue)
		if err != nil {
			return err
		}
		cfg.Behaviour.ShowCredentialsOnConnect = v
	default:
		return errors.New("unknown configuration name: " + configName)
	}

	return SaveConfig(configFilePath, cfg)
}

func parseBoolValue(configName, v string) (bool, error) {
	parsed, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(v)))
	if err != nil {
		return false, fmt.Errorf("invalid value for %s, expected 'true' or 'false'", configName)
	}
	return parsed, nil
}
