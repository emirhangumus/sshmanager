package model

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

var (
	proxyJumpHopPattern = regexp.MustCompile(`^(?:([^@\s,]+)@)?(\[[^\]\s,]+\]|[^:@\s,]+)(?::(\d{1,5}))?$`)
	forwardSpecPattern  = regexp.MustCompile(`^(?:([^:\s]+):)?(\d{1,5}):([^:\s]+|\[[^\]\s]+\]):(\d{1,5})$`)
	sshOptionKeyPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)
	sshUsernamePattern  = regexp.MustCompile(`^[A-Za-z0-9_.+\\-]+$`)
	sshHostPattern      = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
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
		if _, _, _, err := ParseProxyJumpHop(trimmedHop); err != nil {
			return err
		}
	}

	return nil
}

// ParseProxyJumpHop parses a single "[user@]host[:port]" proxy jump hop into
// its components. The port is returned as a string ("" when not specified)
// since callers typically pass it straight back into ssh argv/ProxyCommand
// text rather than needing it as an int.
func ParseProxyJumpHop(hop string) (user, host, port string, err error) {
	trimmedHop := strings.TrimSpace(hop)
	matches := proxyJumpHopPattern.FindStringSubmatch(trimmedHop)
	if matches == nil {
		return "", "", "", fmt.Errorf("invalid proxy jump hop %q", trimmedHop)
	}

	user = matches[1]
	host = matches[2]
	if err := ValidateSSHHost(host); err != nil {
		return "", "", "", err
	}
	if user != "" {
		if err := ValidateSSHTarget(user, host, 0); err != nil {
			return "", "", "", err
		}
	}
	port = matches[3]
	if port != "" {
		p, convErr := strconv.Atoi(port)
		if convErr != nil || p < 1 || p > 65535 {
			return "", "", "", fmt.Errorf("invalid proxy jump port in %q", trimmedHop)
		}
	}

	return user, host, port, nil
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

	if matches[1] != "" && matches[1] != "*" {
		if err := ValidateSSHHost(matches[1]); err != nil {
			return err
		}
	}
	if err := ValidateSSHHost(matches[3]); err != nil {
		return err
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
	key, value, ok := strings.Cut(strings.TrimSpace(option), "=")
	if !ok {
		return fmt.Errorf("ssh option %q must be in key=value format", option)
	}

	key = strings.TrimSpace(key)
	if key == "" || !sshOptionKeyPattern.MatchString(key) {
		return fmt.Errorf("ssh option key %q is invalid", key)
	}

	if strings.ContainsAny(value, "\x00\r\n") {
		return fmt.Errorf("SSH option contains a control character")
	}
	// An allowlist prevents executable hooks and config-file loading escapes.
	allowed := map[string]bool{
		"serveraliveinterval": true, "serveralivecountmax": true, "connecttimeout": true,
		"connectionattempts": true, "compression": true, "loglevel": true,
		"stricthostkeychecking": true, "userknownhostsfile": true, "globalknownhostsfile": true,
		"forwardagent": true, "forwardx11": true, "forwardx11trusted": true,
		"batchmode": true, "identitiesonly": true, "preferredauthentications": true,
		"passwordauthentication": true, "pubkeyauthentication": true, "kbdinteractiveauthentication": true,
		"requesttty": true, "stdinnull": true, "sessiontype": true, "forkafterauthentication": true,
		"tcpkeepalive": true, "addressfamily": true, "exitonforwardfailure": true,
	}
	if !allowed[strings.ToLower(key)] {
		return fmt.Errorf("SSH option %q is not supported in extra args", key)
	}

	return nil
}

// ValidateSSHTarget restricts values OpenSSH may substitute into shell commands.
func ValidateSSHTarget(username, host string, port int) error {
	if username == "" || strings.HasPrefix(username, "-") || !sshUsernamePattern.MatchString(username) {
		return fmt.Errorf("invalid SSH username")
	}
	if err := ValidateSSHHost(host); err != nil {
		return err
	}
	if port < 0 || port > 65535 {
		return fmt.Errorf("invalid SSH port")
	}
	return nil
}

// ValidateSSHHost accepts DNS names, SSH aliases, and IP addresses.
func ValidateSSHHost(host string) error {
	ip := host
	if strings.HasPrefix(host, "[") || strings.HasSuffix(host, "]") {
		if !strings.HasPrefix(host, "[") || !strings.HasSuffix(host, "]") {
			return fmt.Errorf("invalid bracketed SSH host")
		}
		ip = host[1 : len(host)-1]
	}
	if net.ParseIP(ip) != nil {
		return nil
	}
	if host == "" || strings.HasPrefix(host, "-") || !sshHostPattern.MatchString(host) {
		return fmt.Errorf("invalid SSH host")
	}
	return nil
}
