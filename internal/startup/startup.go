package startup

import (
	"fmt"

	"github.com/emirhangumus/sshmanager/internal/config"
	cryptoutil "github.com/emirhangumus/sshmanager/internal/crypto"
	"github.com/emirhangumus/sshmanager/internal/storage"
	"github.com/emirhangumus/sshmanager/internal/store"
)

func Setup(connectionFilePath, configFilePath, secretKeyFilePath string) error {
	if err := storage.CreateFileIfNotExists(configFilePath, 0o600); err != nil {
		return err
	}

	if err := storage.CreateFileIfNotExists(connectionFilePath, 0o600); err != nil {
		return err
	}

	if _, err := cryptoutil.LoadKey(secretKeyFilePath); err != nil {
		return err
	}

	connStore := store.NewConnectionStore(connectionFilePath, secretKeyFilePath)
	if err := connStore.InitializeIfEmpty(); err != nil {
		return fmt.Errorf("failed to initialize connection store: %w", err)
	}

	isConfigEmpty, err := storage.IsFileEmpty(configFilePath)
	if err != nil {
		return err
	}
	if isConfigEmpty {
		cfg := config.Default()
		if err := config.SaveConfig(configFilePath, cfg); err != nil {
			return err
		}
	}

	return nil
}
