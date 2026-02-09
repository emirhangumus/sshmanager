# SSH Manager

SSH Manager is a terminal application for storing and connecting to SSH hosts from an interactive menu or alias command.

## Demo

![Demo](demo.gif)

## Features

- AES-GCM encrypted storage for saved connections
- Atomic file writes for connection/config persistence
- Lock-protected connection mutations to reduce concurrent write races
- Add, edit, remove, and connect from an interactive menu
- Direct alias connection (`sshmanager myserver`)
- Scriptable subcommands: `add`, `edit`, `remove`, `connect`, `list`, `export`, `import`, `backup`, `restore`, `doctor`, `clean`, `set`, `version`, `complete`, `completion`
- Alias rename command (`rename`)
- Grouping/tagging metadata with list filtering (`--group`, `--tag`)
- Multiple SSH auth modes: `password`, `key`, `agent`
- Port and identity-file support per connection
- Advanced SSH options: ProxyJump, local/remote forwarding, controlled extra args
- Configurable post-SSH behavior (`behaviour.continueAfterSSHExit`)
- Shell completion support for Bash and Zsh
- Best-effort secure cleanup (`clean`) for connection and key files

## Requirements

- Go `1.23.2+`
- OpenSSH client (`ssh`)
- `sshpass` (required only for `password` auth mode)

Example (Debian/Ubuntu):

```bash
sudo apt install openssh-client sshpass
```

## Installation

```bash
git clone https://github.com/emirhangumus/sshmanager.git
cd sshmanager
make build
make install
```

Run:

```bash
sshmanager
```

## Usage

### Interactive menu

```bash
sshmanager
```

### Direct alias connection

```bash
sshmanager myserver
```

### Subcommands

- List saved connections:

```bash
sshmanager list
sshmanager list --json
sshmanager list --field alias
sshmanager list --field target
sshmanager list --group production
sshmanager list --group production --tag api
```

- Add a connection non-interactively:

```bash
sshmanager add --host app.internal --username ubuntu --auth-mode agent --alias prod
sshmanager add --host db.internal --username root --auth-mode key --identity-file ~/.ssh/id_ed25519 --alias db
sshmanager add --host app.internal --username ubuntu --auth-mode agent --group production --tag linux --tag api --alias prod
sshmanager add --host app.internal --username ubuntu --auth-mode key --identity-file ~/.ssh/id_ed25519 --proxy-jump bastion.internal:2222 --local-forward 8080:127.0.0.1:80 --remote-forward 9000:127.0.0.1:9000 --extra-ssh-arg -vv --extra-ssh-arg -o --extra-ssh-arg ServerAliveInterval=30
```

- Edit a connection non-interactively:

```bash
sshmanager edit --alias prod --new-host new.internal --new-port 2222
sshmanager edit --alias prod --new-group production --new-tag api --new-tag linux
sshmanager edit --id <connection-id> --new-auth-mode key --new-identity-file ~/.ssh/id_ed25519
sshmanager edit --alias prod --new-proxy-jump bastion.internal:2222 --new-local-forward 8080:127.0.0.1:80 --new-remote-forward 9000:127.0.0.1:9000 --new-extra-ssh-arg -vv --new-extra-ssh-arg -o --new-extra-ssh-arg ServerAliveInterval=30
```

- Rename alias:

```bash
sshmanager rename --alias prod --to prod-new
sshmanager rename --id <connection-id> --to prod-new
```

- Remove a connection non-interactively:

```bash
sshmanager remove --alias prod --yes
sshmanager remove --id <connection-id> --yes
```

- Connect explicitly (subcommand form):

```bash
sshmanager connect --alias prod
sshmanager connect --id <connection-id>
```

- Export encrypted store contents to plaintext backup:

```bash
sshmanager export --out ./connections.yaml --format yaml
sshmanager export --out ./connections.json --format json
```

- Import connection backups:

```bash
sshmanager import --in ./connections.yaml --mode merge
sshmanager import --in ./connections.json --mode replace
```

Import modes:

- `merge`: update existing entries by `id` (then by alias), add missing entries.
- `replace`: replace the entire connection set with imported data.

- Create full recovery backups (connections + optional config):

```bash
sshmanager backup --out ./snapshot.yaml --format yaml
sshmanager backup --out ./snapshot.json --format json --include-config=false
```

- Restore from recovery backups:

```bash
sshmanager restore --in ./snapshot.yaml --mode merge
sshmanager restore --in ./snapshot.json --mode replace --with-config=true
```

Restore modes:

- `merge`: merge restored entries into existing data.
- `replace`: replace the entire connection set with restored data.

- Run diagnostics for file/key/data consistency:

```bash
sshmanager doctor
sshmanager doctor --json
```

List field values:

- `id`, `alias`, `username`, `host`, `port`, `auth-mode`, `identity-file`, `proxy-jump`, `local-forwards`, `remote-forwards`, `extra-ssh-args`, `group`, `tags`, `description`, `target`

### Utility Commands

- Clean data:

```bash
sshmanager clean
```

- Show version:

```bash
sshmanager version
```

- Set config value:

```bash
sshmanager set behaviour.continueAfterSSHExit false
sshmanager set behaviour.showCredentialsOnConnect false
```

- Completion candidates (used by shell completion scripts):

```bash
sshmanager complete [prefix]
```

- Print completion script:

```bash
sshmanager completion bash
sshmanager completion zsh
```

- Install completion script (explicit opt-in):

```bash
sshmanager completion install bash
sshmanager completion install zsh
```

For Bash, reload your shell after installation:

```bash
source ~/.bashrc
```

## Configuration

| Key | Default | Type | Description |
|---|---|---|---|
| `behaviour.continueAfterSSHExit` | `false` | boolean | If `true`, return to menu after SSH exits. If `false`, exit the app after SSH session ends. |
| `behaviour.showCredentialsOnConnect` | `false` | boolean | If `true`, prints username and password before opening SSH connection. |

## Connection Fields

Each saved connection supports:

| Field | Required | Description |
|---|---|---|
| `username` | yes | SSH username |
| `host` | yes | Hostname or IP |
| `port` | no | SSH port (default: `22`) |
| `authMode` | no | `password`, `key`, or `agent` |
| `password` | conditional | Required for `password` mode |
| `identityFile` | conditional | Required for `key` mode |
| `proxyJump` | no | Jump host chain (`[user@]host[:port][,[user@]host[:port]...]`) |
| `localForwards` | no | Local forward specs (`[bind_address:]port:host:hostport`) |
| `remoteForwards` | no | Remote forward specs (`[bind_address:]port:host:hostport`) |
| `extraSSHArgs` | no | Controlled extra SSH args (`-v`, `-C`, `-o key=value`, etc.) |
| `group` | no | Logical grouping value for organization/filtering |
| `tags` | no | Tag list for organization/filtering |
| `description` | no | Free-form description |
| `alias` | no | Shortcut name (unique, case-insensitive) |

## Data files

SSH Manager stores files under:

```text
~/.sshmanager/
```

Files:

- `conn` (encrypted connection file)
- `conn.lock` (temporary lock file during write operations)
- `secret.key` (either raw AES-256 key bytes or passphrase metadata, file mode `0600`)
- `config.yaml` (configuration)

## Optional Master Passphrase

You can enable passphrase-derived encryption keys by setting:

```bash
export SSHMANAGER_MASTER_PASSPHRASE='your-strong-passphrase'
```

Behavior:

- If `secret.key` does not exist and the env var is set, SSH Manager stores passphrase KDF metadata in `secret.key` and derives the encryption key from your passphrase.
- If `secret.key` was created in passphrase mode, the same env var must be set on later runs.
- If the env var is not set, SSH Manager uses legacy raw key-file mode.

## Development

```bash
make test
make vet
make lint
```

## Security notes

- Connection data is encrypted at rest using AES-GCM.
- Key files are validated and stored with restrictive permissions.
- Password-mode connections pass passwords to `sshpass` via environment variable (`SSHPASS`) instead of CLI args.
- Key/agent modes use OpenSSH directly (no `sshpass` dependency at runtime).
- Optional master passphrase mode derives encryption keys from `SSHMANAGER_MASTER_PASSPHRASE`.
- State file writes use atomic temp-write + rename flow.
- Connection mutations are guarded by a lock file to reduce concurrent update races.
- Secure deletion is best-effort and may not provide full guarantees on all filesystems.
- SSH keys/agent are preferred over password authentication when possible.

## License

Licensed under the [Apache License 2.0](LICENSE).
