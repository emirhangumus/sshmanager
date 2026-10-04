package config

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/emirhangumus/sshmanager/internal/storage"
	"gopkg.in/yaml.v3"
)

func TestSecurityDefaultsAndValidation(t *testing.T) {
	for _, format := range []string{"yaml", "json"} {
		for _, mode := range []string{"omitted", "file", "keyring", "", "unknown"} {
			t.Run(format+"/"+mode, func(t *testing.T) {
				var cfg SSHManagerConfig
				var data []byte
				if format == "json" {
					data = []byte(`{"behaviour":{"continueAfterSSHExit":true}}`)
					if mode != "omitted" {
						data, _ = json.Marshal(map[string]any{"security": map[string]string{"keyStorage": mode}})
					}
					if err := json.Unmarshal(data, &cfg); err != nil {
						t.Fatal(err)
					}
				} else {
					data = []byte("behaviour:\n  continueAfterSSHExit: true\n")
					if mode != "omitted" {
						data, _ = yaml.Marshal(map[string]any{"security": map[string]string{"keyStorage": mode}})
					}
					if err := yaml.Unmarshal(data, &cfg); err != nil {
						t.Fatal(err)
					}
				}
				if mode == "omitted" {
					if cfg.Security.KeyStorage != "keyring" {
						t.Fatal("missing setting did not default to keyring")
					}
				}
				if (mode == "" || mode == "unknown") != (Validate(cfg) != nil) {
					t.Fatalf("unexpected validation for %q", mode)
				}
			})
		}
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := storage.WriteFileAtomic(path, []byte("security:\n  keyStorage: invalid\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("invalid config loaded")
	}
}
