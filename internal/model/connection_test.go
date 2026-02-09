package model

import "testing"

func TestEffectivePort(t *testing.T) {
	conn := SSHConnection{}
	if got := conn.EffectivePort(); got != DefaultSSHPort {
		t.Fatalf("default port mismatch: got %d, want %d", got, DefaultSSHPort)
	}

	conn.Port = 2200
	if got := conn.EffectivePort(); got != 2200 {
		t.Fatalf("custom port mismatch: got %d, want %d", got, 2200)
	}
}

func TestResolveAuthMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		password string
		identity string
		want     string
	}{
		{name: "explicit password", mode: "password", want: AuthModePassword},
		{name: "explicit key mixed case", mode: " KeY ", want: AuthModeKey},
		{name: "explicit agent", mode: "agent", want: AuthModeAgent},
		{name: "fallback to password for legacy records", password: "secret", want: AuthModePassword},
		{name: "fallback to key when identity exists", identity: "/tmp/id", want: AuthModeKey},
		{name: "fallback to agent", want: AuthModeAgent},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveAuthMode(tc.mode, tc.password, tc.identity)
			if got != tc.want {
				t.Fatalf("ResolveAuthMode=%q, want %q", got, tc.want)
			}
		})
	}
}
