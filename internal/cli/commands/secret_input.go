package commands

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"
)

const maxSecretBytes = 64 * 1024

type secretInput struct {
	unsafe *string
	stdin  *bool
	fd     *int
	name   string
}

func registerSecretInput(fs *flag.FlagSet, name string) secretInput {
	return secretInput{
		unsafe: fs.String(name+"-unsafe", "", "UNSAFE: secret in process arguments and shell history"),
		stdin:  fs.Bool(name+"-stdin", false, "Read one secret line from stdin"),
		fd:     fs.Int(name+"-fd", -1, "Read one secret line from an inherited file descriptor"),
		name:   name,
	}
}

func (s secretInput) supplied(fs *flag.FlagSet) bool {
	supplied := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == s.name+"-unsafe" || f.Name == s.name+"-stdin" || f.Name == s.name+"-fd" {
			supplied = true
		}
	})
	return supplied
}

func (s secretInput) read(fs *flag.FlagSet, required, confirm bool) (string, error) {
	count := 0
	unsafeSet, fdSet := false, false
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case s.name + "-unsafe":
			unsafeSet = true
			count++
		case s.name + "-fd":
			fdSet = true
			count++
		}
	})
	if *s.stdin {
		count++
	}
	if count > 1 {
		return "", fmt.Errorf("%s: choose only one secret input source", s.name)
	}
	if fdSet && *s.fd < 0 {
		return "", errors.New("secret file descriptor must be nonnegative")
	}
	if unsafeSet {
		if *s.unsafe == "" {
			return "", errors.New("secret must not be empty")
		}
		return *s.unsafe, nil
	}
	if *s.stdin || fdSet {
		file := os.Stdin
		if fdSet && uintptr(*s.fd) != os.Stdin.Fd() {
			file = os.NewFile(uintptr(*s.fd), "secret-input")
			if file == nil {
				return "", errors.New("invalid secret file descriptor")
			}
			// Secret descriptors are consumed and closed; stdin remains open.
			defer file.Close()
		}
		return readSecretLine(file)
	}
	if !required {
		return "", nil
	}
	if !term.IsTerminal(os.Stdin.Fd()) {
		return "", fmt.Errorf("%s requires a terminal or --%s-stdin/--%s-fd", s.name, s.name, s.name)
	}
	value, err := terminalSecret(s.name + ": ")
	if err != nil {
		return "", err
	}
	if confirm {
		again, err := terminalSecret("Confirm " + s.name + ": ")
		if err != nil {
			return "", err
		}
		if value != again {
			return "", errors.New("passphrases do not match")
		}
	}
	return value, nil
}

func terminalSecret(label string) (string, error) {
	_, _ = fmt.Fprint(os.Stderr, label)
	data, err := term.ReadPassword(os.Stdin.Fd())
	_, _ = fmt.Fprintln(os.Stderr)
	defer clear(data)
	if err != nil {
		return "", fmt.Errorf("read secret: %w", err)
	}
	if len(data) == 0 {
		return "", errors.New("secret must not be empty")
	}
	return string(data), nil
}

// Read byte by byte to avoid consuming the next secret when stdin is shared.
func readSecretLine(r io.Reader) (string, error) {
	data := make([]byte, 0, maxSecretBytes+1)
	newline := false
	defer func() { clear(data) }()
	var one [1]byte
	defer clear(one[:])
	for len(data) <= maxSecretBytes {
		n, err := r.Read(one[:])
		if n > 0 {
			if one[0] == '\n' {
				newline = true
				break
			}
			data = append(data, one[0])
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("read secret: %w", err)
		}
	}
	if len(data) > maxSecretBytes {
		return "", errors.New("secret exceeds size limit")
	}
	value := string(data)
	if newline {
		value = strings.TrimSuffix(value, "\r")
	}
	if value == "" {
		return "", errors.New("secret must not be empty")
	}
	if strings.ContainsRune(value, 0) {
		return "", errors.New("secret cannot contain NUL")
	}
	return value, nil
}
