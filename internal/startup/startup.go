package startup

import (
	"github.com/emirhangumus/sshmanager/internal/cli/flag"
	"github.com/emirhangumus/sshmanager/internal/encryption"
	"github.com/emirhangumus/sshmanager/internal/storage"
)

func Startup(connectionFilePath string, configFilePath string, secretKeyFilePath string) error {

	// ----------- Create config file if it does not exist ----------- //
	if err := storage.CreateFileIfNotExists(configFilePath); err != nil {
		return err
	}

	// ----------- Create connection file if it does not exist ----------- //
	if err := storage.CreateFileIfNotExists(connectionFilePath); err != nil {
		return err
	}

	key, err := encryption.LoadKey(secretKeyFilePath)
	if err != nil {
		return err
	}

	isEmpty, err := storage.IsFileEmpty(connectionFilePath)
	if err != nil {
		return err
	}

	if isEmpty {
		storage.EncryptAndStoreFile("", connectionFilePath, key) // Ensure the key is valid by encrypting an empty string
	}

	// ----------- If config file is empty, fill with default values ----------- //
	if isEmpty, err := storage.IsFileEmpty(configFilePath); err != nil {
		return err
	} else if isEmpty {
		config := flag.SSHManagerConfig{}
		config.SetDefault()
		if err := storage.WriteYAMLFile(configFilePath, config); err != nil {
			return err
		}
	}

	return nil
}
