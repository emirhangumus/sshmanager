package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/emirhangumus/sshmanager/internal/config"
	"github.com/emirhangumus/sshmanager/internal/model"
	"github.com/emirhangumus/sshmanager/internal/storage"
	"github.com/emirhangumus/sshmanager/internal/store"
	"gopkg.in/yaml.v3"
)

const backupSchemaVersion = "1"

type backupSnapshot struct {
	BackupVersion  string                   `yaml:"backupVersion" json:"backupVersion"`
	CreatedAt      string                   `yaml:"createdAt" json:"createdAt"`
	Config         *config.SSHManagerConfig `yaml:"config,omitempty" json:"config,omitempty"`
	ConnectionFile model.ConnectionFile     `yaml:"connectionFile" json:"connectionFile"`
}

func HandleBackup(connectionFilePath, secretKeyFilePath, configFilePath string, args []string) error {
	return handleBackup(connectionFilePath, secretKeyFilePath, configFilePath, args, os.Stdout)
}

func handleBackup(connectionFilePath, secretKeyFilePath, configFilePath string, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("backup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	outPath := fs.String("out", "", "Backup output path")
	format := fs.String("format", "yaml", "Backup format: yaml|json")
	includeConfig := fs.Bool("include-config", true, "Include config in backup")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected arguments for backup: %s", strings.Join(fs.Args(), " "))
	}

	target := strings.TrimSpace(*outPath)
	if target == "" {
		return errors.New("missing required --out path")
	}

	connStore := store.NewConnectionStore(connectionFilePath, secretKeyFilePath, store.WithConfigPath(configFilePath))
	connFile, err := connStore.Load()
	if err != nil {
		return err
	}

	snapshot := backupSnapshot{
		BackupVersion:  backupSchemaVersion,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		ConnectionFile: connFile,
	}

	if *includeConfig {
		cfg, err := config.LoadConfig(configFilePath)
		if err != nil {
			return err
		}
		snapshot.Config = &cfg
	}

	encoded, normalizedFormat, err := marshalBackupSnapshot(snapshot, *format)
	if err != nil {
		return err
	}
	if err := storage.WriteFileAtomic(target, encoded, 0o600); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	_, _ = fmt.Fprintf(out, "Backup saved to %s (%s), %d connections\n", target, normalizedFormat, len(snapshot.ConnectionFile.Connections))
	return nil
}

func HandleRestore(connectionFilePath, secretKeyFilePath, configFilePath string, args []string) error {
	return handleRestore(connectionFilePath, secretKeyFilePath, configFilePath, args, os.Stdout)
}

func handleRestore(connectionFilePath, secretKeyFilePath, configFilePath string, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("restore", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	inPath := fs.String("in", "", "Backup input path")
	format := fs.String("format", "auto", "Backup format: auto|yaml|json")
	mode := fs.String("mode", importModeMerge, "Restore mode: merge|replace")
	withConfig := fs.Bool("with-config", true, "Restore config if available in backup")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected arguments for restore: %s", strings.Join(fs.Args(), " "))
	}

	source := strings.TrimSpace(*inPath)
	if source == "" {
		return errors.New("missing required --in path")
	}

	payload, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("failed to read restore file: %w", err)
	}

	snapshot, err := decodeBackupSnapshot(payload, *format, source)
	if err != nil {
		return err
	}

	connStore := store.NewConnectionStore(connectionFilePath, secretKeyFilePath, store.WithConfigPath(configFilePath))
	modeNorm := strings.ToLower(strings.TrimSpace(*mode))
	var restoredConfig *config.SSHManagerConfig
	if *withConfig {
		restoredConfig = snapshot.Config
	}
	switch modeNorm {
	case importModeMerge:
		err = connStore.Restore(func(connFile *model.ConnectionFile) error {
			return mergeImportedConnections(connFile, snapshot.ConnectionFile.Connections)
		}, restoredConfig)
	case importModeReplace:
		err = connStore.Restore(func(connFile *model.ConnectionFile) error {
			replacement, buildErr := buildConnectionFile(snapshot.ConnectionFile.Connections)
			if buildErr != nil {
				return buildErr
			}
			*connFile = replacement
			return nil
		}, restoredConfig)
	default:
		return fmt.Errorf("unknown restore mode %q (use merge or replace)", modeNorm)
	}
	if err != nil {
		return err
	}

	configRestored := restoredConfig != nil

	_, _ = fmt.Fprintf(out, "Restore completed from %s using %s mode (%d connections, config_restored=%t)\n",
		source,
		modeNorm,
		len(snapshot.ConnectionFile.Connections),
		configRestored,
	)
	return nil
}

func marshalBackupSnapshot(snapshot backupSnapshot, format string) ([]byte, string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "yaml", "yml":
		b, err := yaml.Marshal(snapshot)
		if err != nil {
			return nil, "", fmt.Errorf("failed to marshal backup YAML: %w", err)
		}
		return b, "yaml", nil
	case "json":
		b, err := json.MarshalIndent(snapshot, "", "  ")
		if err != nil {
			return nil, "", fmt.Errorf("failed to marshal backup JSON: %w", err)
		}
		return append(b, '\n'), "json", nil
	default:
		return nil, "", fmt.Errorf("unknown backup format %q (use yaml or json)", format)
	}
}

func decodeBackupSnapshot(data []byte, formatHint, inPath string) (backupSnapshot, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return backupSnapshot{}, errors.New("restore file is empty")
	}

	format := normalizeImportFormat(formatHint, inPath)
	switch format {
	case "yaml":
		return decodeBackupSnapshotYAML(data, formatHint, inPath)
	case "json":
		return decodeBackupSnapshotJSON(data, formatHint, inPath)
	case "auto":
		if snapshot, err := decodeBackupSnapshotJSON(data, formatHint, inPath); err == nil {
			return snapshot, nil
		}
		if snapshot, err := decodeBackupSnapshotYAML(data, formatHint, inPath); err == nil {
			return snapshot, nil
		}
		return backupSnapshot{}, errors.New("failed to decode restore file")
	default:
		return backupSnapshot{}, fmt.Errorf("unknown restore format %q (use auto, yaml, or json)", formatHint)
	}
}

func decodeBackupSnapshotJSON(data []byte, formatHint, inPath string) (backupSnapshot, error) {
	var snapshot backupSnapshot
	if err := json.Unmarshal(bytes.TrimSpace(data), &snapshot); err == nil && isBackupSnapshot(snapshot) {
		return normalizeBackupSnapshot(snapshot), nil
	}

	connFile, err := decodeImportedConnectionFile(data, formatHint, inPath)
	if err != nil {
		return backupSnapshot{}, err
	}
	return backupSnapshot{
		BackupVersion:  backupSchemaVersion,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		ConnectionFile: connFile,
	}, nil
}

func decodeBackupSnapshotYAML(data []byte, formatHint, inPath string) (backupSnapshot, error) {
	var snapshot backupSnapshot
	if err := yaml.Unmarshal(data, &snapshot); err == nil && isBackupSnapshot(snapshot) {
		return normalizeBackupSnapshot(snapshot), nil
	}

	connFile, err := decodeImportedConnectionFile(data, formatHint, inPath)
	if err != nil {
		return backupSnapshot{}, err
	}
	return backupSnapshot{
		BackupVersion:  backupSchemaVersion,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		ConnectionFile: connFile,
	}, nil
}

func isBackupSnapshot(snapshot backupSnapshot) bool {
	return strings.TrimSpace(snapshot.BackupVersion) != "" ||
		strings.TrimSpace(snapshot.CreatedAt) != "" ||
		strings.TrimSpace(snapshot.ConnectionFile.Version) != "" ||
		snapshot.Config != nil
}

func normalizeBackupSnapshot(snapshot backupSnapshot) backupSnapshot {
	if strings.TrimSpace(snapshot.BackupVersion) == "" {
		snapshot.BackupVersion = backupSchemaVersion
	}
	if strings.TrimSpace(snapshot.CreatedAt) == "" {
		snapshot.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if strings.TrimSpace(snapshot.ConnectionFile.Version) == "" {
		snapshot.ConnectionFile.Version = model.CurrentConnectionFileVersion
	}
	if snapshot.ConnectionFile.Connections == nil {
		snapshot.ConnectionFile.Connections = []model.SSHConnection{}
	}
	return snapshot
}

type doctorCheck struct {
	Name   string `json:"name" yaml:"name"`
	Status string `json:"status" yaml:"status"` // ok|warn|error
	Detail string `json:"detail" yaml:"detail"`
}

type doctorReport struct {
	Healthy bool          `json:"healthy" yaml:"healthy"`
	Checks  []doctorCheck `json:"checks" yaml:"checks"`
}

func HandleDoctor(connectionFilePath, secretKeyFilePath, configFilePath string, args []string) error {
	return handleDoctor(connectionFilePath, secretKeyFilePath, configFilePath, args, os.Stdout)
}

func handleDoctor(connectionFilePath, secretKeyFilePath, configFilePath string, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonOutput := fs.Bool("json", false, "Output machine-readable JSON report")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected arguments for doctor: %s", strings.Join(fs.Args(), " "))
	}

	report := doctorReport{
		Healthy: true,
		Checks:  make([]doctorCheck, 0, 15),
	}

	addCheck := func(name, status, detail string) {
		report.Checks = append(report.Checks, doctorCheck{
			Name:   name,
			Status: status,
			Detail: detail,
		})
		if status == "error" {
			report.Healthy = false
		}
	}

	checkFile := func(name, path string) bool {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				addCheck(name, "error", fmt.Sprintf("missing: %s", path))
				return false
			}
			addCheck(name, "error", fmt.Sprintf("failed to stat %s: %v", path, err))
			return false
		}
		if info.IsDir() {
			addCheck(name, "error", fmt.Sprintf("expected file but found directory: %s", path))
			return false
		}
		perms := info.Mode().Perm()
		if perms != 0o600 {
			addCheck(name, "warn", fmt.Sprintf("file mode is %o (recommended 600): %s", perms, path))
			return true
		}
		addCheck(name, "ok", fmt.Sprintf("exists with mode 600: %s", path))
		return true
	}

	configExists := checkFile("config file", configFilePath)
	connectionExists := checkFile("connection file", connectionFilePath)

	lockPath := connectionFilePath + ".lock"
	if _, err := os.Stat(lockPath); err == nil {
		addCheck("connection lock file", "ok", fmt.Sprintf("persistent OS lock file: %s (existence does not indicate a held lock)", lockPath))
	} else if os.IsNotExist(err) {
		addCheck("connection lock file", "ok", "no active lock file")
	} else {
		addCheck("connection lock file", "warn", fmt.Sprintf("failed to inspect lock file: %v", err))
	}

	if _, err := exec.LookPath("ssh"); err != nil {
		addCheck("ssh binary", "error", "ssh not found in PATH; connections cannot be made")
	} else {
		addCheck("ssh binary", "ok", "ssh found in PATH")
	}

	if _, err := exec.LookPath("sshpass"); err != nil {
		addCheck("sshpass binary", "warn", "sshpass not found in PATH; password-auth connections will fail")
	} else {
		addCheck("sshpass binary", "ok", "sshpass found in PATH")
	}

	if homeDir, err := os.UserHomeDir(); err != nil {
		addCheck("known_hosts", "warn", fmt.Sprintf("could not determine home directory: %v", err))
	} else {
		knownHostsPath := filepath.Join(homeDir, ".ssh", "known_hosts")
		info, statErr := os.Stat(knownHostsPath)
		switch {
		case statErr != nil && os.IsNotExist(statErr):
			addCheck("known_hosts", "warn", "no ~/.ssh/known_hosts found; host key verification is delegated to ssh and has not yet trusted any host")
		case statErr != nil:
			addCheck("known_hosts", "warn", fmt.Sprintf("failed to inspect %s: %v", knownHostsPath, statErr))
		case info.Size() == 0:
			addCheck("known_hosts", "warn", fmt.Sprintf("%s is empty; host key verification is delegated to ssh and has not yet trusted any host", knownHostsPath))
		default:
			addCheck("known_hosts", "ok", fmt.Sprintf("host key verification is delegated to ssh via %s", knownHostsPath))
		}
	}

	if !configExists {
		addCheck("config parse", "error", "skipped: config file is missing")
	} else {
		cfg, cfgErr := config.LoadConfig(configFilePath)
		if cfgErr != nil {
			addCheck("config parse", "error", cfgErr.Error())
		} else {
			addCheck("config parse", "ok", fmt.Sprintf("config loaded (continueAfterSSHExit=%t)", cfg.Behaviour.ContinueAfterSSHExit))
		}
	}

	if !connectionExists {
		addCheck("connection data load", "error", "skipped: required connection/key file is missing")
	} else {
		connStore := store.NewConnectionStore(connectionFilePath, secretKeyFilePath, store.WithConfigPath(configFilePath))
		connFile, backend, pending, connErr := connStore.Inspect()
		if backend == "file" {
			checkFile("secret key file", secretKeyFilePath)
		}
		addCheck("key storage", "ok", fmt.Sprintf("active backend: %s", backend))
		if pending {
			addCheck("storage migration", "warn", "migration or cleanup pending; normal credential operations resume it")
		}
		if connErr != nil {
			addCheck("connection data load", "error", connErr.Error())
		} else {
			addCheck("connection data load", "ok", fmt.Sprintf("loaded %d connections", len(connFile.Connections)))

			aliasSeen := map[string]struct{}{}
			invalidCount := 0
			for _, conn := range connFile.Connections {
				if _, err := normalizeImportedConnection(conn); err != nil {
					invalidCount++
				}

				alias := strings.ToLower(strings.TrimSpace(conn.Alias))
				if alias == "" {
					continue
				}
				if _, exists := aliasSeen[alias]; exists {
					invalidCount++
					continue
				}
				aliasSeen[alias] = struct{}{}
			}
			if invalidCount > 0 {
				addCheck("connection schema validation", "error", fmt.Sprintf("found %d invalid connection entries", invalidCount))
			} else {
				addCheck("connection schema validation", "ok", "all connections passed validation")
			}
		}
	}

	if *jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return err
		}
	} else {
		for _, check := range report.Checks {
			_, _ = fmt.Fprintf(out, "[%s] %s: %s\n", strings.ToUpper(check.Status), check.Name, check.Detail)
		}
		if report.Healthy {
			_, _ = fmt.Fprintln(out, "Doctor status: healthy")
		} else {
			_, _ = fmt.Fprintln(out, "Doctor status: unhealthy")
		}
	}

	if !report.Healthy {
		return errors.New("doctor found one or more issues")
	}
	return nil
}
