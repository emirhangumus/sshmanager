package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emirhangumus/sshmanager/internal/model"
	"github.com/emirhangumus/sshmanager/internal/storage"
	"github.com/emirhangumus/sshmanager/internal/store"
)

func TestHandleListTextOutput(t *testing.T) {
	connPath, keyPath := prepareListFixture(t, []model.SSHConnection{
		{Username: "ubuntu", Host: "1.2.3.4", Password: "secret", AuthMode: model.AuthModePassword, Alias: "prod", Group: "production", Tags: []string{"linux", "api"}, Description: "main server"},
		{Username: "root", Host: "db.internal"},
	})

	var out strings.Builder
	if err := handleList(connPath, keyPath, nil, &out); err != nil {
		t.Fatalf("handleList failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "ALIAS\tUSERNAME\tHOST\tPORT\tAUTH_MODE\tGROUP\tTAGS\tDESCRIPTION") {
		t.Fatalf("missing header in output: %q", got)
	}
	if !strings.Contains(got, "prod\tubuntu\t1.2.3.4\t22\tpassword\tproduction\tlinux,api\tmain server") {
		t.Fatalf("missing first row in output: %q", got)
	}
	if !strings.Contains(got, "-\troot\tdb.internal\t22\tagent") {
		t.Fatalf("missing second row in output: %q", got)
	}
}

func TestHandleListJSONOutput(t *testing.T) {
	connPath, keyPath := prepareListFixture(t, []model.SSHConnection{
		{
			Username:     "ubuntu",
			Host:         "host.example",
			Port:         2222,
			AuthMode:     model.AuthModeKey,
			IdentityFile: "/home/user/.ssh/id_ed25519",
			ProxyJump:    "jump.internal:2200",
			LocalForwards: []string{
				"8080:127.0.0.1:80",
			},
			RemoteForwards: []string{
				"9000:127.0.0.1:9000",
			},
			ExtraSSHArgs: []string{
				"-vv",
			},
			Group:       "production",
			Tags:        []string{"linux", "api"},
			Alias:       "prod",
			Description: "api",
		},
	})

	var out strings.Builder
	if err := handleList(connPath, keyPath, []string{"--json"}, &out); err != nil {
		t.Fatalf("handleList failed: %v", err)
	}

	var payload []map[string]any
	if err := json.Unmarshal([]byte(out.String()), &payload); err != nil {
		t.Fatalf("failed to decode json output: %v\nraw: %s", err, out.String())
	}
	if len(payload) != 1 {
		t.Fatalf("expected 1 list item, got %d", len(payload))
	}
	if _, ok := payload[0]["password"]; ok {
		t.Fatalf("password must not be present in list JSON: %v", payload[0])
	}
	if payload[0]["alias"] != "prod" {
		t.Fatalf("unexpected alias: %v", payload[0]["alias"])
	}
	if payload[0]["username"] != "ubuntu" {
		t.Fatalf("unexpected username: %v", payload[0]["username"])
	}
	if payload[0]["authMode"] != model.AuthModeKey {
		t.Fatalf("unexpected authMode: %v", payload[0]["authMode"])
	}
	if payload[0]["port"] != float64(2222) {
		t.Fatalf("unexpected port: %v", payload[0]["port"])
	}
	if payload[0]["identityFile"] != "/home/user/.ssh/id_ed25519" {
		t.Fatalf("unexpected identityFile: %v", payload[0]["identityFile"])
	}
	if payload[0]["proxyJump"] != "jump.internal:2200" {
		t.Fatalf("unexpected proxyJump: %v", payload[0]["proxyJump"])
	}
	if payload[0]["group"] != "production" {
		t.Fatalf("unexpected group: %v", payload[0]["group"])
	}
	tags, ok := payload[0]["tags"].([]any)
	if !ok || len(tags) != 2 {
		t.Fatalf("unexpected tags payload: %v", payload[0]["tags"])
	}
}

func TestHandleListEmptyPrintsFriendlyMessage(t *testing.T) {
	tmpDir := t.TempDir()
	connPath := filepath.Join(tmpDir, "conn")
	keyPath := filepath.Join(tmpDir, "secret.key")

	if err := storage.CreateFileIfNotExists(connPath, 0o600); err != nil {
		t.Fatalf("CreateFileIfNotExists(conn) failed: %v", err)
	}

	connStore := store.NewConnectionStore(connPath, keyPath)
	if err := connStore.InitializeIfEmpty(); err != nil {
		t.Fatalf("InitializeIfEmpty failed: %v", err)
	}

	var out strings.Builder
	if err := handleList(connPath, keyPath, nil, &out); err != nil {
		t.Fatalf("handleList failed: %v", err)
	}

	if !strings.Contains(out.String(), "No SSH connections found.") {
		t.Fatalf("expected empty-state message, got %q", out.String())
	}
}

func TestHandleListRejectsUnexpectedArgs(t *testing.T) {
	connPath, keyPath := prepareListFixture(t, []model.SSHConnection{
		{Username: "ubuntu", Host: "1.2.3.4", Password: "secret"},
	})

	var out strings.Builder
	err := handleList(connPath, keyPath, []string{"--json", "extra"}, &out)
	if err == nil {
		t.Fatal("expected error for unexpected positional args, got nil")
	}
}

func TestHandleListFieldOutput(t *testing.T) {
	connPath, keyPath := prepareListFixture(t, []model.SSHConnection{
		{Username: "ubuntu", Host: "1.2.3.4", Password: "secret", AuthMode: model.AuthModePassword, Alias: "prod"},
		{Username: "root", Host: "db.internal", AuthMode: model.AuthModeAgent, Alias: "db"},
	})

	var out strings.Builder
	if err := handleList(connPath, keyPath, []string{"--field", "target"}, &out); err != nil {
		t.Fatalf("handleList failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "ubuntu@1.2.3.4") {
		t.Fatalf("missing first target in output: %q", got)
	}
	if !strings.Contains(got, "root@db.internal") {
		t.Fatalf("missing second target in output: %q", got)
	}
}

func TestHandleListFieldOutputAdvancedFields(t *testing.T) {
	connPath, keyPath := prepareListFixture(t, []model.SSHConnection{
		{
			Username:  "ubuntu",
			Host:      "1.2.3.4",
			AuthMode:  model.AuthModeAgent,
			ProxyJump: "jump.internal:2200",
			LocalForwards: []string{
				"8080:127.0.0.1:80",
			},
			Group: "production",
			Tags:  []string{"linux", "api"},
		},
	})

	var out strings.Builder
	if err := handleList(connPath, keyPath, []string{"--field", "proxy-jump"}, &out); err != nil {
		t.Fatalf("handleList failed: %v", err)
	}
	if !strings.Contains(out.String(), "jump.internal:2200") {
		t.Fatalf("unexpected proxy-jump field output: %q", out.String())
	}

	out.Reset()
	if err := handleList(connPath, keyPath, []string{"--field", "local-forwards"}, &out); err != nil {
		t.Fatalf("handleList failed: %v", err)
	}
	if !strings.Contains(out.String(), "8080:127.0.0.1:80") {
		t.Fatalf("unexpected local-forwards field output: %q", out.String())
	}

	out.Reset()
	if err := handleList(connPath, keyPath, []string{"--field", "group"}, &out); err != nil {
		t.Fatalf("handleList failed: %v", err)
	}
	if !strings.Contains(out.String(), "production") {
		t.Fatalf("unexpected group field output: %q", out.String())
	}

	out.Reset()
	if err := handleList(connPath, keyPath, []string{"--field", "tags"}, &out); err != nil {
		t.Fatalf("handleList failed: %v", err)
	}
	if !strings.Contains(out.String(), "linux,api") {
		t.Fatalf("unexpected tags field output: %q", out.String())
	}
}

func TestHandleListRejectsFieldWithJSON(t *testing.T) {
	connPath, keyPath := prepareListFixture(t, []model.SSHConnection{
		{Username: "ubuntu", Host: "1.2.3.4", Password: "secret", AuthMode: model.AuthModePassword},
	})

	var out strings.Builder
	err := handleList(connPath, keyPath, []string{"--json", "--field", "host"}, &out)
	if err == nil {
		t.Fatal("expected error for --json with --field, got nil")
	}
}

func TestHandleListFiltersByGroupAndTag(t *testing.T) {
	connPath, keyPath := prepareListFixture(t, []model.SSHConnection{
		{Username: "ubuntu", Host: "api.internal", AuthMode: model.AuthModeAgent, Alias: "api", Group: "production", Tags: []string{"linux", "api"}},
		{Username: "ubuntu", Host: "db.internal", AuthMode: model.AuthModeAgent, Alias: "db", Group: "production", Tags: []string{"linux", "db"}},
		{Username: "ubuntu", Host: "stage.internal", AuthMode: model.AuthModeAgent, Alias: "stage", Group: "staging", Tags: []string{"linux", "api"}},
	})

	var out strings.Builder
	if err := handleList(connPath, keyPath, []string{"--group", "production", "--tag", "api"}, &out); err != nil {
		t.Fatalf("handleList failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "api.internal") {
		t.Fatalf("expected filtered output to contain api.internal, got %q", got)
	}
	if strings.Contains(got, "db.internal") || strings.Contains(got, "stage.internal") {
		t.Fatalf("expected non-matching hosts to be filtered out, got %q", got)
	}
}

func prepareListFixture(t *testing.T, conns []model.SSHConnection) (string, string) {
	t.Helper()

	tmpDir := t.TempDir()
	connPath := filepath.Join(tmpDir, "conn")
	keyPath := filepath.Join(tmpDir, "secret.key")

	if err := storage.CreateFileIfNotExists(connPath, 0o600); err != nil {
		t.Fatalf("CreateFileIfNotExists(conn) failed: %v", err)
	}

	connStore := store.NewConnectionStore(connPath, keyPath)
	if err := connStore.InitializeIfEmpty(); err != nil {
		t.Fatalf("InitializeIfEmpty failed: %v", err)
	}

	if len(conns) == 0 {
		return connPath, keyPath
	}

	err := connStore.Update(func(connFile *model.ConnectionFile) error {
		for _, conn := range conns {
			if err := connFile.AddConnection(conn); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to seed connection file: %v", err)
	}

	if _, err := os.Stat(connPath); err != nil {
		t.Fatalf("expected conn file to exist: %v", err)
	}

	return connPath, keyPath
}
