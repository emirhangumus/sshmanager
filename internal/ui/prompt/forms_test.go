package prompt

import (
	"strings"
	"testing"

	"github.com/emirhangumus/sshmanager/internal/model"
)

func TestValidateHost(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "hostname", input: "example.com"},
		{name: "ipv4", input: "192.168.1.10"},
		{name: "ipv6", input: "::1"},
		{name: "single label", input: "localhost"},
		{name: "empty", input: "", wantErr: true},
		{name: "contains whitespace", input: "bad host", wantErr: true},
		{name: "starts with hyphen", input: "-host.example", wantErr: true},
		{name: "invalid separator", input: "host..example", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validateHost(tc.input)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateAlias(t *testing.T) {
	longAlias := strings.Repeat("a", 65)
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "empty is allowed", input: ""},
		{name: "simple alias", input: "prod-01"},
		{name: "trimmed alias", input: "  web.api  "},
		{name: "contains space", input: "bad alias", wantErr: true},
		{name: "contains symbols", input: "bad*alias", wantErr: true},
		{name: "too long", input: longAlias, wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validateAlias(tc.input)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "empty allowed", input: ""},
		{name: "valid", input: "22"},
		{name: "valid max", input: "65535"},
		{name: "zero invalid", input: "0", wantErr: true},
		{name: "negative invalid", input: "-1", wantErr: true},
		{name: "too large", input: "65536", wantErr: true},
		{name: "not int", input: "abc", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validatePort(tc.input)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateAuthMode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "password", input: "password"},
		{name: "key", input: "key"},
		{name: "agent", input: "agent"},
		{name: "trimmed and case insensitive", input: "  KeY  "},
		{name: "empty", input: "", wantErr: true},
		{name: "unknown", input: "token", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validateAuthMode(tc.input)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateProxyJump(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "empty allowed", input: ""},
		{name: "simple", input: "jump.internal"},
		{name: "with port", input: "jump.internal:2222"},
		{name: "chain", input: "jump-a:2200,jump-b:2201"},
		{name: "has whitespace", input: "jump bad", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validateProxyJump(tc.input)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateForwardList(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "empty allowed", input: ""},
		{name: "single", input: "8080:127.0.0.1:80"},
		{name: "multiple", input: "8080:127.0.0.1:80,9000:127.0.0.1:9000"},
		{name: "invalid", input: "bad-forward", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validateForwardList(tc.input)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateExtraSSHArgs(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "empty allowed", input: ""},
		{name: "standalone", input: "-vv,-C"},
		{name: "option", input: "-o,ServerAliveInterval=30"},
		{name: "blocked option", input: "-o,ProxyCommand=nc", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validateExtraSSHArgs(tc.input)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateGroup(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "empty allowed", input: ""},
		{name: "valid", input: "production"},
		{name: "valid slash", input: "team/core"},
		{name: "invalid", input: "bad group", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validateGroup(tc.input)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateTags(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "empty allowed", input: ""},
		{name: "single tag", input: "prod"},
		{name: "multiple tags", input: "linux,api"},
		{name: "invalid tag", input: "bad tag", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validateTags(tc.input)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNormalizeConnectionTrimsTextFields(t *testing.T) {
	got := normalizeConnection(connectionFixture())

	if got.Username != "user" {
		t.Fatalf("username not trimmed: %q", got.Username)
	}
	if got.Host != "example.com" {
		t.Fatalf("host not trimmed: %q", got.Host)
	}
	if got.Description != "desc" {
		t.Fatalf("description not trimmed: %q", got.Description)
	}
	if got.Alias != "prod" {
		t.Fatalf("alias not trimmed: %q", got.Alias)
	}
	if got.IdentityFile != "/tmp/id_ed25519" {
		t.Fatalf("identityFile not trimmed: %q", got.IdentityFile)
	}
	if got.ProxyJump != "jump.internal:2200" {
		t.Fatalf("proxyJump not trimmed: %q", got.ProxyJump)
	}
	if len(got.LocalForwards) != 1 || got.LocalForwards[0] != "8080:127.0.0.1:80" {
		t.Fatalf("localForwards not normalized: %v", got.LocalForwards)
	}
	if len(got.RemoteForwards) != 1 || got.RemoteForwards[0] != "9000:127.0.0.1:9000" {
		t.Fatalf("remoteForwards not normalized: %v", got.RemoteForwards)
	}
	if len(got.ExtraSSHArgs) != 3 || got.ExtraSSHArgs[0] != "-vv" {
		t.Fatalf("extraSSHArgs not normalized: %v", got.ExtraSSHArgs)
	}
	if got.Group != "production" {
		t.Fatalf("group not trimmed: %q", got.Group)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "linux" || got.Tags[1] != "api" {
		t.Fatalf("tags not normalized: %v", got.Tags)
	}
	if got.AuthMode != model.AuthModeKey {
		t.Fatalf("authMode not normalized: %q", got.AuthMode)
	}
	if got.Password != " pass " {
		t.Fatalf("password should not be trimmed: %q", got.Password)
	}
}

func TestNormalizeAuthSensitiveFields(t *testing.T) {
	passwordMode := normalizeAuthSensitiveFields(model.SSHConnection{
		AuthMode:     model.AuthModePassword,
		Password:     "secret",
		IdentityFile: "/tmp/id",
	})
	if passwordMode.IdentityFile != "" {
		t.Fatalf("identityFile should be cleared in password mode, got %q", passwordMode.IdentityFile)
	}
	if passwordMode.Password == "" {
		t.Fatal("password should be kept in password mode")
	}

	keyMode := normalizeAuthSensitiveFields(model.SSHConnection{
		AuthMode:     model.AuthModeKey,
		Password:     "secret",
		IdentityFile: "/tmp/id",
	})
	if keyMode.Password != "" {
		t.Fatalf("password should be cleared in key mode, got %q", keyMode.Password)
	}
	if keyMode.IdentityFile == "" {
		t.Fatal("identityFile should be kept in key mode")
	}

	agentMode := normalizeAuthSensitiveFields(model.SSHConnection{
		AuthMode:     model.AuthModeAgent,
		Password:     "secret",
		IdentityFile: "/tmp/id",
	})
	if agentMode.Password != "" || agentMode.IdentityFile != "" {
		t.Fatalf("password and identityFile should be cleared in agent mode, got %+v", agentMode)
	}
}

func TestParsePort(t *testing.T) {
	if got := parsePort(""); got != 0 {
		t.Fatalf("parsePort empty = %d, want 0", got)
	}
	if got := parsePort("2222"); got != 2222 {
		t.Fatalf("parsePort valid = %d, want 2222", got)
	}
	if got := parsePort("invalid"); got != 0 {
		t.Fatalf("parsePort invalid = %d, want 0", got)
	}
}

func connectionFixture() model.SSHConnection {
	return model.SSHConnection{
		ID:           "id-1",
		Username:     " user ",
		Host:         " example.com ",
		AuthMode:     " KeY ",
		Password:     " pass ",
		IdentityFile: " /tmp/id_ed25519 ",
		ProxyJump:    " jump.internal:2200 ",
		LocalForwards: []string{
			" 8080:127.0.0.1:80 ",
			" ",
		},
		RemoteForwards: []string{
			" 9000:127.0.0.1:9000 ",
		},
		ExtraSSHArgs: []string{
			" -vv ",
			" -o ",
			" ServerAliveInterval=30 ",
		},
		Group:       " production ",
		Tags:        []string{" linux ", "api", "API", ""},
		Description: " desc ",
		Alias:       " prod ",
	}
}
