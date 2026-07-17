package commands

import (
	"strings"
	"testing"

	"github.com/emirhangumus/sshmanager/internal/model"
)

func TestBuildScpInvocationPasswordMode(t *testing.T) {
	conn := &model.SSHConnection{
		Username: "ubuntu",
		Host:     "example.com",
		Password: "secret",
		AuthMode: model.AuthModePassword,
	}

	bin, args, env, err := buildScpInvocation(conn, false, []string{"./file.txt"}, "ubuntu@example.com:/tmp/")
	if err != nil {
		t.Fatalf("buildScpInvocation failed: %v", err)
	}
	if bin != "sshpass" {
		t.Fatalf("unexpected binary: %q", bin)
	}
	wantArgs := []string{"-e", "scp", "-P", "22", "./file.txt", "ubuntu@example.com:/tmp/"}
	assertStringSliceEqual(t, args, wantArgs)
	assertStringSliceEqual(t, env, []string{"SSHPASS=secret"})
}

func TestBuildScpInvocationKeyModeRecursive(t *testing.T) {
	identityFile := writeTestIdentityFile(t)
	conn := &model.SSHConnection{
		Username:     "ubuntu",
		Host:         "example.com",
		Port:         2222,
		AuthMode:     model.AuthModeKey,
		IdentityFile: identityFile,
	}

	bin, args, env, err := buildScpInvocation(conn, true, []string{"./dist"}, "ubuntu@example.com:/srv/www")
	if err != nil {
		t.Fatalf("buildScpInvocation failed: %v", err)
	}
	if bin != "scp" {
		t.Fatalf("unexpected binary: %q", bin)
	}
	wantArgs := []string{"-P", "2222", "-i", identityFile, "-r", "./dist", "ubuntu@example.com:/srv/www"}
	assertStringSliceEqual(t, args, wantArgs)
	if len(env) != 0 {
		t.Fatalf("expected no extra env for key mode, got %v", env)
	}
}

func TestBuildScpInvocationAgentMode(t *testing.T) {
	conn := &model.SSHConnection{
		Username: "ubuntu",
		Host:     "example.com",
		AuthMode: model.AuthModeAgent,
	}

	bin, args, env, err := buildScpInvocation(conn, false, []string{"ubuntu@example.com:/tmp/log.txt"}, "./log.txt")
	if err != nil {
		t.Fatalf("buildScpInvocation failed: %v", err)
	}
	if bin != "scp" {
		t.Fatalf("unexpected binary: %q", bin)
	}
	wantArgs := []string{"-P", "22", "ubuntu@example.com:/tmp/log.txt", "./log.txt"}
	assertStringSliceEqual(t, args, wantArgs)
	if len(env) != 0 {
		t.Fatalf("expected no extra env for agent mode, got %v", env)
	}
}

func TestBuildScpInvocationWithProxyJumpAndExtraArgs(t *testing.T) {
	identityFile := writeTestIdentityFile(t)
	conn := &model.SSHConnection{
		Username:     "ubuntu",
		Host:         "example.com",
		AuthMode:     model.AuthModeKey,
		IdentityFile: identityFile,
		ProxyJump:    "jump.internal:2222",
		// LocalForwards/RemoteForwards must be ignored for scp.
		LocalForwards:  []string{"8080:127.0.0.1:80"},
		RemoteForwards: []string{"9000:127.0.0.1:9000"},
		ExtraSSHArgs:   []string{"-vv", "-o", "ServerAliveInterval=30"},
	}

	bin, args, _, err := buildScpInvocation(conn, false, []string{"./file.txt"}, "ubuntu@example.com:/tmp/")
	if err != nil {
		t.Fatalf("buildScpInvocation failed: %v", err)
	}
	if bin != "scp" {
		t.Fatalf("unexpected binary: %q", bin)
	}
	wantArgs := []string{
		"-P", "22",
		"-i", identityFile,
		"-J", "jump.internal:2222",
		"-vv",
		"-o", "ServerAliveInterval=30",
		"./file.txt", "ubuntu@example.com:/tmp/",
	}
	assertStringSliceEqual(t, args, wantArgs)
}

func TestBuildScpInvocationMissingPassword(t *testing.T) {
	conn := &model.SSHConnection{
		Username: "ubuntu",
		Host:     "example.com",
		AuthMode: model.AuthModePassword,
	}

	if _, _, _, err := buildScpInvocation(conn, false, []string{"./file.txt"}, "ubuntu@example.com:/tmp/"); err == nil {
		t.Fatal("expected error for missing password")
	}
}

func TestBuildScpInvocationInvalidExtraArgs(t *testing.T) {
	conn := &model.SSHConnection{
		Username:     "ubuntu",
		Host:         "example.com",
		AuthMode:     model.AuthModeAgent,
		ExtraSSHArgs: []string{"-o", "ProxyCommand=evil"},
	}

	if _, _, _, err := buildScpInvocation(conn, false, []string{"./file.txt"}, "ubuntu@example.com:/tmp/"); err == nil {
		t.Fatal("expected error for blocked ssh option in extra args")
	}
}

func TestClassifyScpArgsAndSingleRemoteConnection(t *testing.T) {
	connFile := &model.ConnectionFile{
		Connections: []model.SSHConnection{
			{ID: "id-1", Alias: "myserver", Username: "ubuntu", Host: "example.com"},
			{ID: "id-2", Alias: "other", Username: "root", Host: "other.example.com"},
		},
	}

	args, err := classifyScpArgs(connFile, []string{"./file.txt", "myserver:/tmp/"})
	if err != nil {
		t.Fatalf("classifyScpArgs failed: %v", err)
	}
	if args[0].isRemote() {
		t.Fatal("expected first arg to be local")
	}
	if !args[1].isRemote() || args[1].conn.Alias != "myserver" || args[1].remotePath != "/tmp/" {
		t.Fatalf("unexpected classification for second arg: %+v", args[1])
	}

	conn, err := singleRemoteConnection(args)
	if err != nil {
		t.Fatalf("singleRemoteConnection failed: %v", err)
	}
	if conn.Alias != "myserver" {
		t.Fatalf("unexpected resolved connection: %+v", conn)
	}
}

func TestClassifyScpArgsLocalPathWithColonIsNotTreatedAsRemote(t *testing.T) {
	connFile := &model.ConnectionFile{
		Connections: []model.SSHConnection{
			{ID: "id-1", Alias: "myserver", Username: "ubuntu", Host: "example.com"},
		},
	}

	args, err := classifyScpArgs(connFile, []string{"C:/Users/test/file.txt"})
	if err != nil {
		t.Fatalf("classifyScpArgs failed: %v", err)
	}
	if args[0].isRemote() {
		t.Fatalf("expected local classification for non-alias colon path, got %+v", args[0])
	}
}

func TestSingleRemoteConnectionZeroAliases(t *testing.T) {
	args := []scpArg{{raw: "./a.txt"}, {raw: "./b.txt"}}
	if _, err := singleRemoteConnection(args); err == nil {
		t.Fatal("expected error when no alias is present")
	}
}

func TestSingleRemoteConnectionMultipleAliases(t *testing.T) {
	connA := &model.SSHConnection{ID: "id-1", Alias: "a"}
	connB := &model.SSHConnection{ID: "id-2", Alias: "b"}
	args := []scpArg{
		{raw: "a:/tmp/x", conn: connA, remotePath: "/tmp/x"},
		{raw: "b:/tmp/y", conn: connB, remotePath: "/tmp/y"},
	}
	if _, err := singleRemoteConnection(args); err == nil {
		t.Fatal("expected error when multiple distinct aliases are present")
	}
}

func TestHandleScpArgsEndToEnd(t *testing.T) {
	connPath, keyPath := prepareListFixture(t, []model.SSHConnection{
		{Alias: "myserver", Username: "ubuntu", Host: "example.com", AuthMode: model.AuthModeAgent},
	})

	// exec.LookPath will find "scp" in most test environments, but we only
	// need to verify argument classification/build succeeds up to the point
	// of attempting to run the command; a nonexistent alias/arg-count error
	// is what we assert on here without actually invoking scp.
	err := handleScpArgs(connPath, keyPath, []string{"./file.txt"}, &strings.Builder{})
	if err == nil {
		t.Fatal("expected usage error for fewer than 2 positional args")
	}

	err = handleScpArgs(connPath, keyPath, []string{"./file.txt", "unknownalias:/tmp/"}, &strings.Builder{})
	if err == nil {
		t.Fatal("expected error when no known alias is referenced")
	}
}
