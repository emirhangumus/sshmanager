# Changelog

All notable changes to this project are documented in this file. Format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Released
versions are tracked via [GitHub releases](https://github.com/emirhangumus/sshmanager/releases)/tags;
this file tracks unreleased work going forward.

## [Unreleased]

### Fixed
- `go.mod` declared `go 1.23.2` while `internal/crypto` used `crypto/pbkdf2`
  (stdlib since Go 1.24), which would fail to build on an exact Go 1.23
  toolchain. Bumped the minimum Go version to 1.24.0.
- `internal/cli/commands/connect.go`: a non-constant format string was
  passed to `fmt.Errorf` for the "sshpass not found" error; switched to
  `errors.New`.

### Added
- `doctor` now checks that `ssh`/`sshpass` are available on `PATH` and
  reports on `~/.ssh/known_hosts` presence, since host key verification is
  delegated entirely to the system SSH configuration.
- `CLAUDE.md`, `CONTRIBUTING.md` documenting architecture, dev workflow, and
  security invariants.
- Expanded test coverage for `internal/storage` (atomic writes, secure
  delete, YAML round-trips).

### Changed
- `connect`'s `showCredentialsOnConnect` debug path now prints an explicit
  warning before printing the username/password.
- Key-mode connections now validate the identity file exists and isn't a
  directory before invoking `ssh`, instead of surfacing a native ssh error.
- `.golangci.yml` now also enables `errcheck`, `gosec`, `revive`, and `gofmt`.
- CI minimum coverage threshold raised from 35% to 50%.
