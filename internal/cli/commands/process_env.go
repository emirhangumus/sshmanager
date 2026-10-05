package commands

import (
	"os"
	"strings"
)

// scopedProcessEnv prevents inherited sshmanager secrets leaking to unrelated
// children while retaining OpenSSH's HOME, agent, locale, and other settings.
func scopedProcessEnv(additions []string) []string {
	env := make([]string, 0, len(os.Environ())+len(additions))
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		switch strings.ToUpper(key) {
		case "SSHPASS", "SSHMANAGER_PROXY_JUMP_SSHPASS", "SSHMANAGER_MASTER_PASSPHRASE":
			continue
		}
		env = append(env, entry)
	}
	return append(env, additions...)
}
