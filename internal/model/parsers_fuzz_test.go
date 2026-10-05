package model

import "testing"

func FuzzProxyJumpParser(f *testing.F) {
	for _, s := range []string{"", "bob@host:2222", "jump1,jump2", "[::1]:22", "-oProxyCommand=evil", "bob$(id)@host"} {
		f.Add(s)
	}
	f.Fuzz(func(_ *testing.T, s string) { _ = ValidateProxyJump(s); _, _, _, _ = ParseProxyJumpHop(s) })
}
func FuzzForwardParser(f *testing.F) {
	for _, s := range []string{"8080:localhost:80", "9000:[::1]:9001", "", "8080:host;id:22"} {
		f.Add(s)
	}
	f.Fuzz(func(_ *testing.T, s string) { _ = ValidateForwardSpec(s) })
}
func FuzzExtraSSHArgs(f *testing.F) {
	for _, s := range []string{"-v", "-oServerAliveInterval=30", "-oProxyCommand=evil", "", "-oKnownHostsCommand=evil"} {
		f.Add(s)
	}
	f.Fuzz(func(_ *testing.T, s string) {
		_ = ValidateExtraSSHArgs([]string{s})
		_ = ValidateExtraSSHArgs([]string{"-o", s})
	})
}

func FuzzAliasParser(f *testing.F) {
	for _, s := range []string{"prod", "Production-API", "", " foo && bar ", "İstanbul"} {
		f.Add(s)
	}
	f.Fuzz(func(_ *testing.T, alias string) {
		cf := NewConnectionFile()
		_ = cf.AddConnection(SSHConnection{Alias: alias})
		_ = cf.GetConnectionByAlias(alias)
		cf.EnsureIDs()
	})
}
