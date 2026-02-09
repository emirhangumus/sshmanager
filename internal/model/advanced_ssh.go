package model

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	proxyJumpHopPattern = regexp.MustCompile(`^(?:[^@\s,]+@)?(?:\[[^\]\s,]+\]|[^:@\s,]+)(?::(\d{1,5}))?$`)
	forwardSpecPattern  = regexp.MustCompile(`^(?:([^:\s]+):)?(\d{1,5}):([^:\s]+|\[[^\]\s]+\]):(\d{1,5})$`)
	sshOptionKeyPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)
)

var allowedStandaloneExtraSSHArgs = map[string]struct{}{
	"-4":   {},
	"-6":   {},
	"-A":   {},
	"-a":   {},
	"-C":   {},
	"-g":   {},
	"-K":   {},
	"-k":   {},
	"-N":   {},
	"-n":   {},
	"-q":   {},
	"-T":   {},
	"-t":   {},
	"-v":   {},
	"-vv":  {},
	"-vvv": {},
	"-X":   {},
	"-x":   {},
	"-Y":   {},
}

var blockedExtraSSHOptionKeys = map[string]struct{}{
	"identityfile":       {},
	"localcommand":       {},
	"localforward":       {},
	"permitlocalcommand": {},
	"port":               {},
	"proxycommand":       {},
	"proxyjump":          {},
	"remoteforward":      {},
}

func NormalizeStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}

	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func ValidateProxyJump(proxyJump string) error {
	trimmed := strings.TrimSpace(proxyJump)
	if trimmed == "" {
		return nil
	}
	if strings.ContainsAny(trimmed, " \t\r\n") {
		return fmt.Errorf("proxy jump cannot contain whitespace")
	}

	hops := strings.Split(trimmed, ",")
	for _, hop := range hops {
		trimmedHop := strings.TrimSpace(hop)
		if trimmedHop == "" {
			return fmt.Errorf("proxy jump cannot contain empty hops")
		}
		matches := proxyJumpHopPattern.FindStringSubmatch(trimmedHop)
		if matches == nil {
			return fmt.Errorf("invalid proxy jump hop %q", trimmedHop)
		}
		if matches[1] != "" {
			port, err := strconv.Atoi(matches[1])
			if err != nil || port < 1 || port > 65535 {
				return fmt.Errorf("invalid proxy jump port in %q", trimmedHop)
			}
		}
	}

	return nil
}

func ValidateForwardSpecs(specs []string) error {
	for _, raw := range specs {
		if err := ValidateForwardSpec(raw); err != nil {
			return err
		}
	}
	return nil
}

func ValidateForwardSpec(spec string) error {
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" {
		return fmt.Errorf("forward spec cannot be empty")
	}
	if strings.ContainsAny(trimmed, " \t\r\n") {
		return fmt.Errorf("forward spec cannot contain whitespace: %q", trimmed)
	}

	matches := forwardSpecPattern.FindStringSubmatch(trimmed)
	if matches == nil {
		return fmt.Errorf("invalid forward spec %q, expected [bind_address:]port:host:hostport", trimmed)
	}

	localPort, err := strconv.Atoi(matches[2])
	if err != nil || localPort < 1 || localPort > 65535 {
		return fmt.Errorf("invalid local port in forward spec %q", trimmed)
	}

	remotePort, err := strconv.Atoi(matches[4])
	if err != nil || remotePort < 1 || remotePort > 65535 {
		return fmt.Errorf("invalid remote port in forward spec %q", trimmed)
	}

	return nil
}

func ValidateExtraSSHArgs(args []string) error {
	normalized := NormalizeStringList(args)
	for i := 0; i < len(normalized); i++ {
		arg := normalized[i]

		switch {
		case arg == "-o":
			if i+1 >= len(normalized) {
				return fmt.Errorf("extra ssh arg -o requires a key=value token")
			}
			if err := validateSSHOptionToken(normalized[i+1]); err != nil {
				return err
			}
			i++
		case strings.HasPrefix(arg, "-o"):
			option := strings.TrimPrefix(arg, "-o")
			option = strings.TrimPrefix(option, "=")
			if err := validateSSHOptionToken(option); err != nil {
				return err
			}
		default:
			if _, ok := allowedStandaloneExtraSSHArgs[arg]; ok {
				continue
			}
			return fmt.Errorf("unsupported extra ssh argument %q", arg)
		}
	}

	return nil
}

func validateSSHOptionToken(option string) error {
	key, _, ok := strings.Cut(strings.TrimSpace(option), "=")
	if !ok {
		return fmt.Errorf("ssh option %q must be in key=value format", option)
	}

	key = strings.TrimSpace(key)
	if key == "" || !sshOptionKeyPattern.MatchString(key) {
		return fmt.Errorf("ssh option key %q is invalid", key)
	}

	if _, blocked := blockedExtraSSHOptionKeys[strings.ToLower(key)]; blocked {
		return fmt.Errorf("ssh option %q is not allowed in extra args", key)
	}

	return nil
}
