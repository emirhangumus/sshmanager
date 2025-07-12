package flag

import (
	"errors"
	"flag"
)

var SSHManagerVersion = "v1.1.0"

// Check function checks the command line flags for specific actions.
// Returns true if process should continue, false if it should exit.
func Check(connectionFilePath string, secretKeyFilePath string, configFilePath string) (bool, error) {
	clean := flag.Bool("clean", false, "Reset all saved SSH connections and key file")
	setConfig := flag.Bool("set", false, "Set a SSHManager configuration.\nUsage: sshmanager -set <config-name> <config-value>\nFor possible config names and values, see the documentation or README.")
	version := flag.Bool("version", false, "Show the version of SSHManager")

	flag.Parse()

	if *clean {
		CleanSSHFile(clean, connectionFilePath, secretKeyFilePath)
		return false, nil
	}
	if *setConfig {
		if len(flag.Args()) < 2 {
			return false, errors.New("not enough arguments for -set flag, expected: sshmanager -set <config-name> <config-value>")
		}
		configName := flag.Arg(0)
		configValue := flag.Arg(1)
		err := SetConfig(configFilePath, configName, configValue)
		if err != nil {
			return false, err
		}
		return false, nil
	}
	if *version {
		println(SSHManagerVersion)
		return false, nil
	}

	return true, nil
}
