# SSH Manager

SSH Manager is a terminal application for storing and connecting to SSH hosts from an interactive menu or alias command.

## Demo

![Demo](demo.gif)

## Features

- AES-GCM encrypted storage for saved connections
- OS keyring protection by default, with configurable local key storage and recoverable migration
- Atomic file writes for connection/config persistence
- Lock-protected connection mutations to reduce concurrent write races
- Add, edit, remove, and connect from an interactive menu
- Direct alias connection (`sshmanager myserver`)
- Scriptable subcommands: `add`, `edit`, `remove`, `connect`, `exec`, `scp`, `list`, `export`, `import`, `backup`, `restore`, `doctor`, `clean`, `set`, `version`, `complete`, `completion`
- File transfer to/from saved hosts using alias syntax (`sshmanager scp file.txt myserver:/path`)
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
sshmanager add --host internal.example.com --username targetuser --auth-mode password --password TARGET_PASSWORD --proxy-jump jumpuser@bastion.example.com --proxy-jump-auth-mode password --proxy-jump-password JUMP_PASSWORD --alias internal-server
```

When the jump host and the target host need different credentials (e.g. both
require password auth), set `--proxy-jump-auth-mode`/`--proxy-jump-password`/
`--proxy-jump-identity-file` independently of the target host's own auth
fields. If left unset, ProxyJump behaves as before (passed through natively
to `ssh -J`, relying on the system's own key/agent/ssh_config for the hop).
A dedicated jump password or identity file is only supported for a single
ProxyJump hop.

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

- Run a command or stream a local script to a saved host without a TTY:

```bash
sshmanager exec --alias prod -- 'uname -a'
sshmanager exec --id <connection-id> -- '/path/to/remote-script.sh'
sshmanager exec --alias prod --script ./deploy.sh
sshmanager exec --alias prod --script ./check.sh --shell bash
```

The command form takes one shell command string after `--`; the remote shell
interprets it. The script form streams a local file to `sh -s` by default or
`bash -s` when selected, so the chosen interpreter must exist on the remote
host. Output streams live and the process returns the remote command's exit
status when available. Script arguments are not supported. Saved port
forwards are ignored for one-shot execution.

- Copy files to/from a saved host (`scp` using alias syntax):

```bash
sshmanager scp ./file.txt myserver:/home/user1
sshmanager scp myserver:/var/log/app.log ./logs/
sshmanager scp -r ./dist myserver:/srv/www
```

`scp` resolves `myserver` the same way `connect` does (same auth mode,
identity file, ProxyJump, and extra SSH args), then rewrites it to
`user@host:path` before shelling out to `scp`/`sshpass -e scp`. Exactly one
sshmanager alias may appear among the source(s)/destination — the rest are
treated as plain local paths, so ordinary multi-file/local-to-local usage
still works. Transferring directly between two different sshmanager
aliases in one invocation is not supported.

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
| `security.keyStorage` | `keyring` | `keyring` or `file` | Where the encryption key is stored. Switching migrates the existing key and preserves saved passwords. |

```yaml
security:
  keyStorage: keyring
```

Switch storage modes with:

```bash
sshmanager set security.keyStorage file
sshmanager set security.keyStorage keyring
```

The CLI reports the current migration stage on stderr, with elapsed-time updates
every two seconds while waiting (for example, for OS keyring approval). It reports
completion, failure, or pending cleanup explicitly.

The default also applies to older configurations that omit this setting. Existing
file keys migrate on the next credential operation. Passwords (including ProxyJump
passwords) stay inside the encrypted `conn` file; a mode switch moves the same key,
verifies it can decrypt your data, and removes the obsolete source only after commit.
Direct config edits and restoring a config use the same migration checks.

### OS keyring requirements and recovery

The keyring integration uses [`github.com/zalando/go-keyring`](https://github.com/zalando/go-keyring):

- macOS: the user Keychain, accessed through `/usr/bin/security`.
- Linux/BSD: a session D-Bus Secret Service, such as GNOME Keyring, with a `login` collection.
- Windows: Windows Credential Manager.

Unlock the OS keyring when prompted. For headless systems without a keyring, select
`file` explicitly before saving connections. There is no automatic security downgrade.
A failed automatic migration from a legacy file store can be cancelled with
`sshmanager set security.keyStorage file`. A keyring-only installation must regain
access to its original keyring key before moving to file storage.

Interrupted migrations and restores resume on the next credential operation. Failed
source cleanup is reported on stderr and retried while the committed backend remains
usable. `doctor` reports the active backend and pending migration without changing
keys or connection data. If a restore fails before its commit checkpoint, selecting
the original storage mode cancels the pending restore and preserves the original data.
After the checkpoint, recovery completes the committed restore.

Do not delete `key-storage.yaml`: it identifies this installation's keyring entry and
contains recovery state. A missing key for existing encrypted data is an error; the
application will not create a replacement. Restore the original key and state, or
restore a recovery backup into a fresh installation. Backups/exports retain their
existing plaintext format and must be protected accordingly. They do not include
the encryption key or installation-specific keyring/recovery state.

Before using an older SSH Manager binary, switch to `file` and stop all running
instances. Remove the persistent `conn.lock` file once no process uses it; older
versions used its presence as a lock, whereas this version uses OS locking.

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
- `conn.lock` (persistent OS lock file; its presence does not mean a lock is held)
- `secret.key` (file mode only: raw AES-256 key bytes or passphrase metadata, file mode `0600`)
- `config.yaml` (configuration)
- `key-storage.yaml` (store identifier, key fingerprint, active backend, nonsecret passphrase metadata, and migration journal; mode `0600`)
- `conn.restore-pending` (encrypted staging file during recoverable restore; removed after commit)

### Migrating from older connection files

Older versions of SSH Manager stored connections as a plain list without a
stable `id` per entry. SSH Manager detects this legacy format automatically
on load, assigns each connection a new `id`, and rewrites `conn` in the
current schema on the next save — no manual migration step is required, and
existing aliases/fields are preserved.

## Optional Master Passphrase

For new installations using `security.keyStorage: file`, you can enable
passphrase-derived encryption keys by setting:

```bash
export SSHMANAGER_MASTER_PASSPHRASE='your-strong-passphrase'
```

Behavior:

- If `secret.key` does not exist and the env var is set, SSH Manager stores passphrase KDF metadata in `secret.key` and derives the encryption key from your passphrase.
- If `secret.key` was created in passphrase mode, the same env var must be set on later runs.
- If the env var is not set, SSH Manager uses legacy raw key-file mode.

In keyring mode, the OS keyring protects the stored encryption key; this environment
variable does not initialize passphrase encryption. Migrating an existing passphrase
file requires its original passphrase. The nonsecret salt/KDF metadata is retained
in `key-storage.yaml`, so switching back to file mode requires the matching passphrase
and restores the original protection. Ordinary keyring use does not need that
passphrase. Raw-key installations switch back to a raw key file.

## Development

```bash
make test
make vet
make lint
```

## Security notes

- Connection data is encrypted at rest using AES-GCM.
- Key files are validated and stored with restrictive permissions.
- OS keyring mode avoids storing the raw encryption key alongside encrypted connections.
- Password-mode connections pass passwords to `sshpass` via environment variable (`SSHPASS`) instead of CLI args.
- Key/agent modes use OpenSSH directly (no `sshpass` dependency at runtime).
- Optional master passphrase mode derives encryption keys from `SSHMANAGER_MASTER_PASSPHRASE`.
- State file writes use atomic temp-write + rename flow.
- Connection operations, migration, and cleanup share an OS-backed process lock.
- Secure deletion is best-effort and may not provide full guarantees on all filesystems.
- SSH keys/agent are preferred over password authentication when possible.

## License

Licensed under the [Apache License 2.0](LICENSE).
