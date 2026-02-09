package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/emirhangumus/sshmanager/internal/model"
	"github.com/emirhangumus/sshmanager/internal/storage"
	"github.com/emirhangumus/sshmanager/internal/store"
	"gopkg.in/yaml.v3"
)

const (
	importModeMerge   = "merge"
	importModeReplace = "replace"
)

func HandleExport(connectionFilePath, secretKeyFilePath string, args []string) error {
	return handleExport(connectionFilePath, secretKeyFilePath, args, os.Stdout)
}

func handleExport(connectionFilePath, secretKeyFilePath string, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := fs.String("format", "yaml", "Export format: yaml|json")
	outPath := fs.String("out", "", "Export file path")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected arguments for export: %s", strings.Join(fs.Args(), " "))
	}

	target := strings.TrimSpace(*outPath)
	if target == "" {
		return errors.New("missing required --out path")
	}

	connStore := store.NewConnectionStore(connectionFilePath, secretKeyFilePath)
	connFile, err := connStore.Load()
	if err != nil {
		return err
	}

	serialized, normalizedFormat, err := marshalConnectionFile(connFile, *format)
	if err != nil {
		return err
	}

	if err := storage.WriteFileAtomic(target, serialized, 0o600); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	_, _ = fmt.Fprintf(out, "Exported %d connections to %s (%s)\n", len(connFile.Connections), target, normalizedFormat)
	return nil
}

func HandleImport(connectionFilePath, secretKeyFilePath string, args []string) error {
	return handleImport(connectionFilePath, secretKeyFilePath, args, os.Stdout)
}

func handleImport(connectionFilePath, secretKeyFilePath string, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	inPath := fs.String("in", "", "Import file path")
	format := fs.String("format", "auto", "Import format: auto|yaml|json")
	mode := fs.String("mode", importModeMerge, "Import mode: merge|replace")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected arguments for import: %s", strings.Join(fs.Args(), " "))
	}

	source := strings.TrimSpace(*inPath)
	if source == "" {
		return errors.New("missing required --in path")
	}

	payload, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	importFile, err := decodeImportedConnectionFile(payload, *format, source)
	if err != nil {
		return err
	}

	connStore := store.NewConnectionStore(connectionFilePath, secretKeyFilePath)
	modeNorm := strings.ToLower(strings.TrimSpace(*mode))
	switch modeNorm {
	case importModeMerge:
		err = connStore.Update(func(connFile *model.ConnectionFile) error {
			return mergeImportedConnections(connFile, importFile.Connections)
		})
	case importModeReplace:
		err = connStore.Update(func(connFile *model.ConnectionFile) error {
			replacement, buildErr := buildConnectionFile(importFile.Connections)
			if buildErr != nil {
				return buildErr
			}
			*connFile = replacement
			return nil
		})
	default:
		return fmt.Errorf("unknown import mode %q (use merge or replace)", modeNorm)
	}
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(out, "Imported %d connections from %s using %s mode\n", len(importFile.Connections), source, modeNorm)
	return nil
}

func marshalConnectionFile(connFile model.ConnectionFile, format string) ([]byte, string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "yaml", "yml", "":
		b, err := yaml.Marshal(connFile)
		if err != nil {
			return nil, "", fmt.Errorf("failed to marshal YAML: %w", err)
		}
		return b, "yaml", nil
	case "json":
		b, err := json.MarshalIndent(connFile, "", "  ")
		if err != nil {
			return nil, "", fmt.Errorf("failed to marshal JSON: %w", err)
		}
		return append(b, '\n'), "json", nil
	default:
		return nil, "", fmt.Errorf("unknown export format %q (use yaml or json)", format)
	}
}

func decodeImportedConnectionFile(data []byte, formatHint, inPath string) (model.ConnectionFile, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return model.ConnectionFile{}, errors.New("import file is empty")
	}

	format := normalizeImportFormat(formatHint, inPath)
	switch format {
	case "yaml":
		return decodeYAMLConnectionFile(data)
	case "json":
		return decodeJSONConnectionFile(data)
	case "auto":
		if parsed, err := decodeJSONConnectionFile(data); err == nil {
			return parsed, nil
		}
		if parsed, err := decodeYAMLConnectionFile(data); err == nil {
			return parsed, nil
		}
		return model.ConnectionFile{}, errors.New("failed to decode import file as JSON or YAML")
	default:
		return model.ConnectionFile{}, fmt.Errorf("unknown import format %q (use auto, yaml, or json)", formatHint)
	}
}

func normalizeImportFormat(formatHint, inPath string) string {
	norm := strings.ToLower(strings.TrimSpace(formatHint))
	if norm != "" && norm != "auto" {
		if norm == "yml" {
			return "yaml"
		}
		return norm
	}

	switch strings.ToLower(filepath.Ext(strings.TrimSpace(inPath))) {
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	default:
		return "auto"
	}
}

func decodeJSONConnectionFile(data []byte) (model.ConnectionFile, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return model.ConnectionFile{}, errors.New("import file is empty")
	}

	if trimmed[0] == '[' {
		var legacy []model.SSHConnection
		if err := json.Unmarshal(trimmed, &legacy); err != nil {
			return model.ConnectionFile{}, fmt.Errorf("failed to decode JSON list: %w", err)
		}
		return model.ConnectionFile{
			Version:     model.CurrentConnectionFileVersion,
			Connections: legacy,
		}, nil
	}

	var parsed model.ConnectionFile
	if err := json.Unmarshal(trimmed, &parsed); err != nil {
		return model.ConnectionFile{}, fmt.Errorf("failed to decode JSON object: %w", err)
	}
	if strings.TrimSpace(parsed.Version) == "" {
		parsed.Version = model.CurrentConnectionFileVersion
	}
	if parsed.Connections == nil {
		parsed.Connections = []model.SSHConnection{}
	}
	return parsed, nil
}

func decodeYAMLConnectionFile(data []byte) (model.ConnectionFile, error) {
	var parsed model.ConnectionFile
	if err := yaml.Unmarshal(data, &parsed); err == nil {
		if strings.TrimSpace(parsed.Version) == "" {
			parsed.Version = model.CurrentConnectionFileVersion
		}
		if parsed.Connections == nil {
			parsed.Connections = []model.SSHConnection{}
		}
		return parsed, nil
	}

	var legacy []model.SSHConnection
	if err := yaml.Unmarshal(data, &legacy); err == nil {
		return model.ConnectionFile{
			Version:     model.CurrentConnectionFileVersion,
			Connections: legacy,
		}, nil
	}

	return model.ConnectionFile{}, errors.New("failed to decode YAML content")
}

func buildConnectionFile(connections []model.SSHConnection) (model.ConnectionFile, error) {
	built := model.NewConnectionFile()
	for _, raw := range connections {
		normalized, err := normalizeImportedConnection(raw)
		if err != nil {
			return model.ConnectionFile{}, err
		}
		if err := built.AddConnection(normalized); err != nil {
			return model.ConnectionFile{}, err
		}
	}
	return built, nil
}

func mergeImportedConnections(target *model.ConnectionFile, incoming []model.SSHConnection) error {
	for _, raw := range incoming {
		normalized, err := normalizeImportedConnection(raw)
		if err != nil {
			return err
		}

		if id := strings.TrimSpace(normalized.ID); id != "" {
			if existing := target.GetConnectionByID(id); existing != nil {
				if _, err := target.UpdateConnectionByID(existing.ID, normalized); err != nil {
					return err
				}
				continue
			}
		}

		if alias := strings.TrimSpace(normalized.Alias); alias != "" {
			if existing := target.GetConnectionByAlias(alias); existing != nil {
				if _, err := target.UpdateConnectionByID(existing.ID, normalized); err != nil {
					return err
				}
				continue
			}
		}

		if err := target.AddConnection(normalized); err != nil {
			return err
		}
	}
	return nil
}

func normalizeImportedConnection(conn model.SSHConnection) (model.SSHConnection, error) {
	conn.Username = strings.TrimSpace(conn.Username)
	conn.Host = strings.TrimSpace(conn.Host)
	conn.ProxyJump = strings.TrimSpace(conn.ProxyJump)
	conn.Group = strings.TrimSpace(conn.Group)
	conn.Description = strings.TrimSpace(conn.Description)
	conn.Alias = strings.TrimSpace(conn.Alias)
	conn.IdentityFile = strings.TrimSpace(conn.IdentityFile)
	conn.LocalForwards = model.NormalizeStringList(conn.LocalForwards)
	conn.RemoteForwards = model.NormalizeStringList(conn.RemoteForwards)
	conn.ExtraSSHArgs = model.NormalizeStringList(conn.ExtraSSHArgs)
	conn.Tags = model.NormalizeTags(conn.Tags)
	conn.AuthMode = model.NormalizeAuthMode(conn.AuthMode)

	if conn.Username == "" {
		return model.SSHConnection{}, errors.New("imported connection has empty username")
	}
	if conn.Host == "" {
		return model.SSHConnection{}, errors.New("imported connection has empty host")
	}
	if conn.Port < 0 || conn.Port > 65535 {
		return model.SSHConnection{}, fmt.Errorf("imported connection has invalid port %d", conn.Port)
	}
	if err := model.ValidateProxyJump(conn.ProxyJump); err != nil {
		return model.SSHConnection{}, fmt.Errorf("imported connection has invalid proxyJump: %w", err)
	}
	if err := model.ValidateForwardSpecs(conn.LocalForwards); err != nil {
		return model.SSHConnection{}, fmt.Errorf("imported connection has invalid localForwards: %w", err)
	}
	if err := model.ValidateForwardSpecs(conn.RemoteForwards); err != nil {
		return model.SSHConnection{}, fmt.Errorf("imported connection has invalid remoteForwards: %w", err)
	}
	if err := model.ValidateExtraSSHArgs(conn.ExtraSSHArgs); err != nil {
		return model.SSHConnection{}, fmt.Errorf("imported connection has invalid extraSSHArgs: %w", err)
	}
	if err := model.ValidateGroup(conn.Group); err != nil {
		return model.SSHConnection{}, fmt.Errorf("imported connection has invalid group: %w", err)
	}
	if err := model.ValidateTags(conn.Tags); err != nil {
		return model.SSHConnection{}, fmt.Errorf("imported connection has invalid tags: %w", err)
	}

	conn.AuthMode = conn.EffectiveAuthMode()
	switch conn.AuthMode {
	case model.AuthModePassword:
		conn.IdentityFile = ""
		if strings.TrimSpace(conn.Password) == "" {
			return model.SSHConnection{}, errors.New("imported password auth connection is missing password")
		}
	case model.AuthModeKey:
		conn.Password = ""
		if conn.IdentityFile == "" {
			return model.SSHConnection{}, errors.New("imported key auth connection is missing identityFile")
		}
	case model.AuthModeAgent:
		conn.Password = ""
		conn.IdentityFile = ""
	default:
		return model.SSHConnection{}, fmt.Errorf("unsupported imported auth mode %q", conn.AuthMode)
	}

	if conn.Port == model.DefaultSSHPort {
		conn.Port = 0
	}

	return conn, nil
}
