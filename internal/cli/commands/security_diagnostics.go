package commands

import (
	"strings"

	"github.com/emirhangumus/sshmanager/v2/internal/model"
)

func connectionSecurityChecks(conn model.SSHConnection) []doctorCheck {
	label := conn.Alias
	if label == "" {
		label = conn.ID
	}
	checks := []doctorCheck{}
	add := func(detail string) {
		checks = append(checks, doctorCheck{Name: "connection security", Status: "warn", Detail: label + ": " + detail})
	}
	if strings.EqualFold(conn.Username, "root") && conn.EffectiveAuthMode() == model.AuthModePassword {
		add("root password authentication")
	}
	args := model.NormalizeStringList(conn.ExtraSSHArgs)
	for i := 0; i < len(args); i++ {
		option := ""
		if args[i] == "-A" {
			add("agent forwarding enabled")
		}
		if args[i] == "-o" && i+1 < len(args) {
			i++
			option = args[i]
		} else if strings.HasPrefix(args[i], "-o") {
			option = strings.TrimPrefix(strings.TrimPrefix(args[i], "-o"), "=")
		}
		key, value, ok := strings.Cut(option, "=")
		if !ok {
			continue
		}
		key, value = strings.ToLower(strings.TrimSpace(key)), strings.ToLower(strings.TrimSpace(value))
		switch key {
		case "stricthostkeychecking":
			if value == "no" || value == "off" || value == "false" {
				add("host-key verification disabled")
			}
		case "userknownhostsfile", "globalknownhostsfile":
			if value == "/dev/null" || value == "nul" || value == "none" {
				add("known-hosts file disabled")
			}
		case "forwardagent":
			if value != "no" && value != "false" {
				add("agent forwarding enabled")
			}
		}
	}
	return checks
}
