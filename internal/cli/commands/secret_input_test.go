package commands

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func secretStdin(t *testing.T, value string) {
	t.Helper()
	old := os.Stdin
	file, err := os.CreateTemp(t.TempDir(), "input")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(value); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	os.Stdin = file
	t.Cleanup(func() { os.Stdin = old; _ = file.Close() })
}

func TestSecretInput(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		args        []string
		want        string
		bad         bool
	}{
		{name: "stdin", input: "  secret with spaces  \n", args: []string{"--password-stdin"}, want: "  secret with spaces  "},
		{name: "CRLF", input: "secret\r\n", args: []string{"--password-stdin"}, want: "secret"},
		{name: "EOF", input: "secret", args: []string{"--password-stdin"}, want: "secret"},
		{name: "empty", args: []string{"--password-stdin"}, bad: true},
		{name: "NUL", input: "a\x00b\n", args: []string{"--password-stdin"}, bad: true},
		{name: "oversize", input: strings.Repeat("x", maxSecretBytes+1), args: []string{"--password-stdin"}, bad: true},
		{name: "conflicting", args: []string{"--password-unsafe", "secret", "--password-stdin"}, bad: true},
		{name: "bad fd", args: []string{"--password-fd", "-1"}, bad: true},
		{name: "no terminal", bad: true},
		{name: "unsafe explicit", args: []string{"--password-unsafe", "secret"}, want: "secret"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			secretStdin(t, tc.input)
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			input := registerSecretInput(fs, "password")
			if err := fs.Parse(tc.args); err != nil {
				t.Fatal(err)
			}
			got, err := input.read(fs, true, false)
			if (err != nil) != tc.bad || got != tc.want {
				t.Fatalf("unexpected result, error=%v", err)
			}
		})
	}
}

func TestAddEditSecretChannels(t *testing.T) {
	connPath, keyPath := prepareTransferFixture(t, nil)
	secretStdin(t, " target password \n")
	args := []string{"--host", "example.com", "--username", "bob", "--alias", "prod", "--auth-mode", "password", "--password-stdin"}
	if err := handleAddArgs(connPath, keyPath, args, io.Discard); err != nil {
		t.Fatal(err)
	}
	loaded := loadTransferConnections(t, connPath, keyPath)
	if loaded.Connections[0].Password != " target password " {
		t.Fatal("password whitespace lost")
	}
	file, err := os.CreateTemp(t.TempDir(), "jump-secret")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.WriteString(" new jump password \n"); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	err = handleEditArgs(connPath, keyPath, []string{"--alias", "prod", "--new-proxy-jump", "jump.example.com", "--new-proxy-jump-auth-mode", "password", "--new-proxy-jump-password-fd", fmt.Sprint(file.Fd())}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	loaded = loadTransferConnections(t, connPath, keyPath)
	if loaded.Connections[0].ProxyJumpPassword != " new jump password " {
		t.Fatal("jump password whitespace lost")
	}
	for _, old := range []string{"--password", "--proxy-jump-password"} {
		if err := handleAddArgs(connPath, keyPath, []string{old, "secret"}, io.Discard); err == nil {
			t.Fatal("old argv secret flag accepted")
		}
	}
	for _, old := range []string{"--new-password", "--new-proxy-jump-password"} {
		if err := handleEditArgs(connPath, keyPath, []string{"--alias", "prod", old, "secret"}, io.Discard); err == nil {
			t.Fatal("old edit secret flag accepted")
		}
	}
}
