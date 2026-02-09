package store

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

func toYAMLString(data interface{}) (string, error) {
	dataBytes, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data to YAML: %w", err)
	}
	return string(dataBytes), nil
}

func fromYAMLString(yamlStr string, out interface{}) error {
	if err := yaml.Unmarshal([]byte(yamlStr), out); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}
	return nil
}
