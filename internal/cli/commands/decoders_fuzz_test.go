package commands

import "testing"

func FuzzBackupDecode(f *testing.F) {
	for _, s := range []string{"", "backupVersion: '1'\nconnectionFile:\n  version: '1.0'\n  connections: []", "{\"backupVersion\":\"1\",\"connectionFile\":{\"version\":\"1.0\",\"connections\":[]}}", "random: data"} {
		f.Add(s)
	}
	f.Fuzz(func(_ *testing.T, s string) { _, _ = decodeBackupSnapshot([]byte(s), "auto", "") })
}
func FuzzImport(f *testing.F) {
	for _, s := range []string{"", "connections: []", "[]", "[{\"host\":\"example.com\",\"username\":\"bob\"}]"} {
		f.Add(s)
	}
	f.Fuzz(func(_ *testing.T, s string) {
		cf, err := decodeImportedConnectionFile([]byte(s), "auto", "")
		if err == nil {
			for _, conn := range cf.Connections {
				_, _ = normalizeImportedConnection(conn)
			}
		}
	})
}
