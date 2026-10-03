package commands

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/emirhangumus/sshmanager/internal/model"
)

func TestParseExecArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    execOptions
		wantErr bool
	}{
		{"command by alias", []string{"--alias", "prod", "--", "printf '%s' ok"}, execOptions{alias: "prod", command: "printf '%s' ok", shell: "sh"}, false},
		{"script by id", []string{"--id", "abc", "--script", "./run.sh", "--shell", "bash"}, execOptions{id: "abc", script: "./run.sh", shell: "bash"}, false},
		{"missing selector", []string{"--", "true"}, execOptions{}, true},
		{"conflicting selectors", []string{"--alias", "prod", "--id", "abc", "--", "true"}, execOptions{}, true},
		{"missing separator", []string{"--alias", "prod", "true"}, execOptions{}, true},
		{"multiple command strings", []string{"--alias", "prod", "--", "echo", "hello"}, execOptions{}, true},
		{"blank command", []string{"--alias", "prod", "--", "  "}, execOptions{}, true},
		{"command and script", []string{"--alias", "prod", "--script", "./run.sh", "--", "true"}, execOptions{}, true},
		{"shell without script", []string{"--alias", "prod", "--shell", "sh", "--", "true"}, execOptions{}, true},
		{"unsupported shell", []string{"--alias", "prod", "--script", "./run.sh", "--shell", "zsh"}, execOptions{}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseExecArgs(tc.args)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseExecArgs error = %v, want error %v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Fatalf("parseExecArgs = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestBuildExecInvocation(t *testing.T) {
	identity := writeTestIdentityFile(t)
	tests := []struct {
		name     string
		conn     model.SSHConnection
		wantBin  string
		wantArgs []string
		wantEnv  []string
	}{
		{
			name: "agent with proxy and ignored forwards",
			conn: model.SSHConnection{Username: "ubuntu", Host: "example.com", AuthMode: model.AuthModeAgent,
				ProxyJump: "jump.internal:2222", LocalForwards: []string{"8080:127.0.0.1:80"},
				RemoteForwards: []string{"9000:127.0.0.1:9000"}, ExtraSSHArgs: []string{"-vv"}},
			wantBin: "ssh", wantArgs: []string{"-p", "22", "-J", "jump.internal:2222", "-vv", "-T", "ubuntu@example.com", "echo ok"},
		},
		{
			name:    "key",
			conn:    model.SSHConnection{Username: "ubuntu", Host: "example.com", Port: 2222, AuthMode: model.AuthModeKey, IdentityFile: identity},
			wantBin: "ssh", wantArgs: []string{"-p", "2222", "-i", identity, "-T", "ubuntu@example.com", "echo ok"},
		},
		{
			name:    "password",
			conn:    model.SSHConnection{Username: "ubuntu", Host: "example.com", AuthMode: model.AuthModePassword, Password: "secret"},
			wantBin: "sshpass", wantArgs: []string{"-e", "ssh", "-p", "22", "-T", "ubuntu@example.com", "echo ok"},
			wantEnv: []string{"SSHPASS=secret"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bin, args, env, err := buildExecInvocation(&tc.conn, "echo ok")
			if err != nil {
				t.Fatal(err)
			}
			if bin != tc.wantBin || !reflect.DeepEqual(args, tc.wantArgs) || !reflect.DeepEqual(env, tc.wantEnv) {
				t.Fatalf("got (%q, %q, %q), want (%q, %q, %q)", bin, args, env, tc.wantBin, tc.wantArgs, tc.wantEnv)
			}
		})
	}
}

func TestBuildExecInvocationRejectsConflictingSSHOptions(t *testing.T) {
	for _, extra := range [][]string{{"-N"}, {"-n"}, {"-t"}, {"-o", "RequestTTY=force"}, {"-oSessionType=none"}, {"-o", "StdinNull=yes"}} {
		conn := &model.SSHConnection{Username: "ubuntu", Host: "example.com", AuthMode: model.AuthModeAgent, ExtraSSHArgs: extra}
		if _, _, _, err := buildExecInvocation(conn, "true"); err == nil {
			t.Fatalf("expected error for extra SSH args %q", extra)
		}
	}
}

func TestBuildExecInvocationAllowsCompatibleSSHOptions(t *testing.T) {
	conn := &model.SSHConnection{
		Username: "ubuntu", Host: "example.com", AuthMode: model.AuthModeAgent,
		ExtraSSHArgs: []string{"-o", "RequestTTY=no", "-oStdinNull=no", "-o", "SessionType=default"},
	}
	if _, _, _, err := buildExecInvocation(conn, "true"); err != nil {
		t.Fatalf("compatible SSH options were rejected: %v", err)
	}
}

func TestRunExecStreamsAndPreservesExitStatus(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test fake SSH uses a POSIX shell")
	}
	path := filepath.Join(t.TempDir(), "ssh")
	fake := "#!/bin/sh\nprintf 'args:%s\\n' \"$*\"\nprintf 'input:'\ncat\nprintf 'remote warning\\n' >&2\nexit 37\n"
	writeFakeExecutable(t, path, fake)
	t.Setenv("PATH", filepath.Dir(path)+string(os.PathListSeparator)+os.Getenv("PATH"))
	var stdout, stderr strings.Builder
	err := runExec("ssh", []string{"-T", "ubuntu@example.com", "echo ok"}, nil,
		&model.SSHConnection{Host: "example.com"}, strings.NewReader("script body\n"), &stdout, &stderr)
	var exitErr *ExecExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 37 || exitErr.Detail != "" {
		t.Fatalf("runExec error = %v, want bare exit status 37", err)
	}
	if got := stdout.String(); got != "args:-T ubuntu@example.com echo ok\ninput:script body\n" {
		t.Fatalf("stdout = %q", got)
	}
	if got := stderr.String(); got != "remote warning\n" {
		t.Fatalf("stderr = %q", got)
	}
}

func TestRunExecPasswordUsesEnvironmentAndStreamsStdin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test fake SSH tools use a POSIX shell")
	}
	binDir := t.TempDir()
	for name, content := range map[string]string{
		"ssh-keygen": "#!/bin/sh\nexit 0\n",
		"sshpass":    "#!/bin/sh\nprintf 'password:%s\\nargs:%s\\n' \"$SSHPASS\" \"$*\"\ncat\n",
	} {
		writeFakeExecutable(t, filepath.Join(binDir, name), content)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	conn := &model.SSHConnection{Username: "ubuntu", Host: "example.com", Password: "secret", AuthMode: model.AuthModePassword}
	bin, args, env, err := buildExecInvocation(conn, "sh -s")
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr strings.Builder
	if err := runExec(bin, args, env, conn, strings.NewReader("echo script\n"), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "password:secret\nargs:-e ssh -p 22 -T ubuntu@example.com sh -s\necho script\n" {
		t.Fatalf("stdout = %q", got)
	}
	if strings.Contains(strings.Join(args, " "), "secret") {
		t.Fatal("password appeared in SSH arguments")
	}
}

func TestHandleExecArgsCommandAndScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test fake SSH uses a POSIX shell")
	}
	connPath, keyPath := prepareTransferFixture(t, []model.SSHConnection{{
		Username: "ubuntu", Host: "example.com", AuthMode: model.AuthModeAgent, Alias: "prod",
	}})
	connFile := loadTransferConnections(t, connPath, keyPath)
	conn := connFile.GetConnectionByAlias("prod")
	if conn == nil {
		t.Fatal("missing fixture connection")
	}
	binDir := t.TempDir()
	sshPath := filepath.Join(binDir, "ssh")
	capturePath := filepath.Join(t.TempDir(), "capture")
	fake := "#!/bin/sh\nprintf '%s\\n' \"$*\" > \"$SSHMANAGER_TEST_CAPTURE\"\ncase \"$*\" in *' -s') cat >> \"$SSHMANAGER_TEST_CAPTURE\" ;; esac\n"
	writeFakeExecutable(t, sshPath, fake)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SSHMANAGER_TEST_CAPTURE", capturePath)

	if err := HandleExecArgs(connPath, keyPath, []string{"--alias", "prod", "--", "echo 'hello world'"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "-p 22 -T ubuntu@example.com echo 'hello world'\n" {
		t.Fatalf("command invocation = %q", got)
	}

	scriptPath := filepath.Join(t.TempDir(), "check.sh")
	if err := os.WriteFile(scriptPath, []byte("echo scripted\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := HandleExecArgs(connPath, keyPath, []string{"--id", conn.ID, "--script", scriptPath, "--shell", "bash"}); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(capturePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "-p 22 -T ubuntu@example.com bash -s\necho scripted\n" {
		t.Fatalf("script invocation = %q", got)
	}
}

func TestHandleExecArgsRejectsInvalidScript(t *testing.T) {
	connPath, keyPath := prepareTransferFixture(t, []model.SSHConnection{{
		Username: "ubuntu", Host: "example.com", AuthMode: model.AuthModeAgent, Alias: "prod",
	}})
	for _, path := range []string{filepath.Join(t.TempDir(), "missing.sh"), t.TempDir()} {
		if err := HandleExecArgs(connPath, keyPath, []string{"--alias", "prod", "--script", path}); err == nil {
			t.Fatalf("expected script validation error for %q", path)
		}
	}
}

func writeFakeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
}
