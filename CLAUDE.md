# CLAUDE.md

Guidance for working in this repository.

## Project overview

`sshmanager` is a terminal-based SSH connection manager. It stores connection
profiles (host, user, auth mode, identity file, proxy/forwarding options) in
an encrypted local store under `~/.sshmanager/`, and shells out to the
system `ssh`/`sshpass` binaries to actually connect. There is no external CLI
framework (no Cobra) — command dispatch is a small hand-written switch in
`internal/app/run.go`.

## Directory map

- `cmd/sshmanager` — main entrypoint, build metadata (version/commit/time).
- `internal/app` — `Run()`: top-level argument routing, legacy flag normalization.
- `internal/cli/commands` — one file per subcommand (`add`, `edit`, `remove`,
  `connect`, `list`, `rename`, `export`/`import`, `backup`/`restore`, `doctor`).
- `internal/cli/flags` — flag parsing helpers shared across commands
  (host/alias/port/auth-mode validation, version/help/completion handlers).
- `internal/cli/menu.go` — interactive BubbleTea menu, used when no subcommand
  is given.
- `internal/model` — data structures (`SSHConnection`, `ConnectionFile`,
  `AdvancedSSH`) and their validation functions.
- `internal/store` — `ConnectionStore`: load/save with file-based mutation
  locking and encryption.
- `internal/crypto` — AES-256-GCM encryption, PBKDF2-SHA256 passphrase key
  derivation, key file load/create.
- `internal/storage` — low-level filesystem helpers: atomic writes, secure
  delete (overwrite-then-remove), permission enforcement.
- `internal/config` — `~/.sshmanager/config.yaml` (behaviour settings).
- `internal/startup` — first-run setup/validation.
- `internal/ui/prompt` — interactive prompt/form primitives built on BubbleTea.
- `internal/completion` — shell completion script generation/install.

## Build, test, lint

Use the `Makefile` targets rather than raw `go` commands where one exists:

- `make build` — build to `bin/sshmanager`.
- `make test` — `go test -v ./...`.
- `make test-coverage` — race + coverage, generates `coverage.html`.
- `make lint` — `golangci-lint run` (install via the URL it prints if missing).
- `make fmt` — `go fmt ./...`.
- `make vet` — `go vet ./...`.

CI (`.github/workflows/ci.yml`) runs the race-enabled test suite, enforces a
minimum coverage threshold, runs `go vet`, and runs `golangci-lint` across
Linux/macOS/Windows.

`internal/ui/prompt`'s interactive BubbleTea entry points (`View`, `runTea`,
`InputPrompt`, `SelectPrompt`) are intentionally not unit-tested — they're
thin wrappers over real terminal I/O. Don't chase coverage there; test the
surrounding logic instead.

## Conventions already in use

- Wrap errors with context: `fmt.Errorf("...: %w", err)`.
- Subcommand flag parsing: `flag.NewFlagSet(name, flag.ContinueOnError)` with
  `fs.SetOutput(io.Discard)` and manual error formatting, for consistent
  help/error text across commands.
- Tests are table-driven; prefer extending an existing table over adding a
  new test function when adding a case to a covered function.
- File writes to `~/.sshmanager/*` go through `storage.WriteFileAtomic`
  (temp file + rename) — never a bare `os.WriteFile` for persisted state.

## Security invariants

These properties are deliberate and load-bearing. Preserve them when adding
or modifying code:

- Every file under `~/.sshmanager/` is created with mode `0o600`, and its
  parent directory with `0o700`. Use the existing helpers in
  `internal/storage` and `internal/crypto` rather than calling
  `os.WriteFile`/`os.MkdirAll` directly with different permissions.
- Secrets (passwords, passphrases, derived keys) are only ever passed to
  child processes via a scoped environment variable on the `exec.Cmd` itself
  (see the `SSHPASS=` pattern in `internal/cli/commands/connect.go`) — never
  as a command-line argument, and never written to stdout/logs outside the
  explicit, opt-in `showCredentialsOnConnect` debug path.
- `exec.Command` is always called with an argument slice. Never build a
  shell string and run it via `sh -c`; this is what keeps shell metacharacters
  in hostnames/usernames/paths from being interpreted.
- Any SSH option derived from user-stored connection data (`ProxyJump`,
  `LocalForwards`, `RemoteForwards`, `ExtraSSHArgs`) must pass through the
  corresponding `model.Validate*` function before being placed into the
  `ssh`/`sshpass` argv. This is what blocks dangerous flags like
  `-o ProxyCommand=...` from being smuggled in through connection data.
- Host key verification is intentionally delegated to the system `ssh`
  binary and the user's own `~/.ssh/known_hosts`/`ssh_config` — sshmanager
  does not implement its own host-key checking. `doctor` surfaces this as an
  informational check rather than sshmanager silently overriding it.
