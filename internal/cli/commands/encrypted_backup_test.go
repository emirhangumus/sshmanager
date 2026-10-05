package commands

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/emirhangumus/sshmanager/v2/internal/config"
	cryptoutil "github.com/emirhangumus/sshmanager/v2/internal/crypto"
	"github.com/emirhangumus/sshmanager/v2/internal/model"
)

func TestEncryptedBackupRestoreAndFailureIsolation(t *testing.T) {
	for _, format := range []string{"yaml", "json"} {
		t.Run(format, func(t *testing.T) {
			source, key := prepareTransferFixture(t, []model.SSHConnection{{Username: "bob", Host: "example.com", Alias: "prod", AuthMode: model.AuthModePassword, Password: "target-secret", ProxyJump: "jump.example.com", ProxyJumpAuthMode: model.AuthModePassword, ProxyJumpPassword: "jump-secret"}})
			cfgPath := filepath.Join(t.TempDir(), "config.yaml")
			if err := config.SaveConfig(cfgPath, config.Default()); err != nil {
				t.Fatal(err)
			}
			backupPath := filepath.Join(t.TempDir(), "backup.sshm")
			secretStdin(t, "a strong portable backup passphrase\n")
			if err := handleBackup(source, key, cfgPath, []string{"--out", backupPath, "--format", format, "--passphrase-stdin"}, io.Discard); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(backupPath)
			if err != nil {
				t.Fatal(err)
			}
			if !cryptoutil.IsEncryptedBackup(raw) || bytes.Contains(raw, []byte("target-secret")) || bytes.Contains(raw, []byte("jump-secret")) {
				t.Fatal("backup leaked credentials")
			}
			if runtime.GOOS != "windows" {
				info, err := os.Stat(backupPath)
				if err != nil || info.Mode().Perm() != 0o600 {
					t.Fatal("backup permissions are unsafe")
				}
			}
			dest, destKey := prepareTransferFixture(t, []model.SSHConnection{{Username: "bob", Host: "old.example.com", Alias: "old", AuthMode: model.AuthModeAgent}})
			destCfg := filepath.Join(t.TempDir(), "config.yaml")
			if err := config.SaveConfig(destCfg, config.Default()); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(dest)
			if err != nil {
				t.Fatal(err)
			}
			beforeCfg, err := os.ReadFile(destCfg)
			if err != nil {
				t.Fatal(err)
			}
			secretStdin(t, "wrong\n")
			args := []string{"--in", backupPath, "--mode", "replace", "--passphrase-stdin"}
			if err := handleRestore(dest, destKey, destCfg, args, io.Discard); err == nil {
				t.Fatal("wrong passphrase accepted")
			}
			after, err := os.ReadFile(dest)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("failed restore changed datastore")
			}
			afterCfg, err := os.ReadFile(destCfg)
			if err != nil || !bytes.Equal(beforeCfg, afterCfg) {
				t.Fatal("failed restore changed config")
			}
			secretStdin(t, "a strong portable backup passphrase\n")
			if err := handleRestore(dest, destKey, destCfg, args, io.Discard); err != nil {
				t.Fatal(err)
			}
			restored := loadTransferConnections(t, dest, destKey)
			conn := restored.GetConnectionByAlias("prod")
			if conn == nil || conn.Password != "target-secret" || conn.ProxyJumpPassword != "jump-secret" {
				t.Fatal("portable restore lost credentials")
			}
		})
	}
}

func TestBackupRequiresSecretUnlessPlaintextExplicit(t *testing.T) {
	conn, key := prepareTransferFixture(t, nil)
	cfg := filepath.Join(t.TempDir(), "config.yaml")
	output := filepath.Join(t.TempDir(), "backup.sshm")
	secretStdin(t, "")
	if err := handleBackup(conn, key, cfg, []string{"--out", output}, io.Discard); err == nil {
		t.Fatal("noninteractive backup silently became plaintext")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("failed backup created output")
	}
}
