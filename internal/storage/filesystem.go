package storage

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func CreateFileIfNotExists(filePath string, fileMode os.FileMode) error {
	if _, err := os.Stat(filePath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, fileMode)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer f.Close()
	return nil
}

// IsFileEmpty checks if the specified file is empty.
func IsFileEmpty(filePath string) (bool, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}
	return fileInfo.Size() == 0, nil
}

// SecureDelete best-effort overwrites file bytes before removal.
func SecureDelete(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0o600)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	info, err := os.Stat(path)
	if err == nil {
		size := info.Size()
		randomData := make([]byte, size)
		if _, err := rand.Read(randomData); err == nil {
			if _, err := f.Write(randomData); err != nil {
				return err
			}
		}
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func ReadYAMLFile(filePath string, out interface{}) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	if err := yaml.Unmarshal(data, out); err != nil {
		return fmt.Errorf("failed to unmarshal YAML from file %s: %w", filePath, err)
	}
	return nil
}

func WriteYAMLFile(filePath string, data interface{}, fileMode os.FileMode) error {
	dataBytes, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data to YAML: %w", err)
	}

	if err := os.WriteFile(filePath, dataBytes, fileMode); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}
	return nil
}
