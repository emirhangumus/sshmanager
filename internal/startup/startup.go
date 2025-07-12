package startup

import (
	"github.com/emirhangumus/sshmanager/internal/cli/flag"
	"github.com/emirhangumus/sshmanager/internal/storage"
)

func Startup(configFilePath string) error {

	// ----------- Create config file if it does not exist ----------- //
	if err := storage.CreateFileIfNotExists(configFilePath); err != nil {
		return err
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
