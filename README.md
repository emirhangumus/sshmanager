# SSH Manager

SSH Manager (`sshmanager`) is an open-source SSH connection manager for Linux,
macOS, and Windows, written in Go. It combines a terminal user interface (TUI)
with a scriptable command-line interface (CLI) for saving hosts, connecting by
alias, running remote commands, and transferring files with SCP.

Built on your system's OpenSSH tools, SSH Manager supports SSH keys, SSH agents,
password authentication, jump hosts, and port forwarding. Saved connection data
is encrypted with AES-GCM, and its encryption key is protected by the OS keyring
by default.

## When to use SSH Manager

- You manage several development, staging, or production hosts and want named shortcuts instead of repeatedly typing connection details.
- You want a TUI to browse and maintain saved connections, alongside CLI commands for scripts and automation.
- You need the same saved host settings for SSH sessions, remote command execution, and SCP file transfers.
- You want to organize hosts with groups and tags and keep saved credentials encrypted locally.

## Demo

![SSH Manager TUI demonstration](demo.gif)

## Features

- **TUI and CLI:** add, edit, rename, remove, and select saved hosts from the TUI or use explicit subcommands.
- **Alias connections:** connect with `sshmanager myserver`, or select a connection by its stable ID.
- **SSH authentication:** use password, private-key, or SSH-agent authentication with saved ports and identity-file paths.
- **Jump hosts and tunnels:** configure ProxyJump, separate jump-host credentials, local/remote port forwarding, and supported extra SSH arguments.
- **Remote execution and SCP:** run commands, stream local scripts, and copy files using saved connection settings.
- **Host organization:** group and tag hosts, add descriptions, and filter connection lists.
- **Automation:** JSON output, selectable list fields, remote exit-status propagation, and Bash/Zsh completion.
- **Encrypted storage:** AES-GCM connection encryption, OS keyring protection by default, optional file-key storage, and recoverable switching between modes.
- **Recovery and diagnostics:** YAML/JSON import and export, configuration-aware backups and restores, `doctor` checks, and confirmed cleanup.

## Requirements

- OpenSSH client (`ssh`)
- OpenSSH `scp` for file transfers
- `sshpass` (required only for `password` auth mode)
- An accessible OS keyring for the default storage mode; headless systems can explicitly select [file storage](#configuration)

Building from source also requires Go **1.24+**. The Makefile workflow requires
`make`; installation does not require `sshpass`. It is only required at runtime
for password authentication.

Example (Debian/Ubuntu):

```bash
sudo apt install openssh-client sshpass
```

## Installation

### Install with Go

```bash
go install github.com/emirhangumus/sshmanager/v2/cmd/sshmanager@latest
```

### Build from a checkout

```bash
git clone https://github.com/emirhangumus/sshmanager.git
cd sshmanager
make build
make install
```

This installs `sshmanager` into `~/.local/bin`. Ensure that directory is on your
`PATH`.

Alternatively, install with Go directly:

```bash
go install ./cmd/sshmanager
```

## Quick start

Save a host using your SSH agent, connect by alias, and reuse the connection for
a remote command and file transfer:

```bash
sshmanager add --host app.example.com --username ubuntu --auth-mode agent --alias prod --group production --tag api
sshmanager prod
sshmanager exec --alias prod -- 'uptime'
sshmanager scp ./file.txt prod:/home/ubuntu/
sshmanager list --group production --json
```

Replace the example host and username with your own. For private-key authentication,
use `--auth-mode key --identity-file ~/.ssh/id_ed25519`. For guided connection
creation, run `sshmanager add` without flags.

Launch the TUI to manage and select hosts:

```bash
sshmanager
```

On a headless machine without an OS keyring, choose file storage before adding
your first connection:

```bash
sshmanager set security.keyStorage file
```

Use `sshmanager help` for the command overview. The examples below cover each
workflow in more detail.

## Usage

### Terminal user interface (TUI)

Running without a subcommand opens the TUI, where you can add, edit, rename,
remove, and connect to saved hosts.

```bash
sshmanager
```

### Direct alias connection

```bash
sshmanager myserver
```

### Subcommands

#### List and filter hosts

```bash
sshmanager list
sshmanager list --json
sshmanager list --field alias
sshmanager list --field target
sshmanager list --group production
sshmanager list --group production --tag api
```

#### Add hosts from the CLI

```bash
sshmanager add --host app.internal --username ubuntu --auth-mode agent --alias prod
sshmanager add --host db.internal --username root --auth-mode key --identity-file ~/.ssh/id_ed25519 --alias db
sshmanager add --host app.internal --username ubuntu --auth-mode agent --group production --tag linux --tag api --alias prod
sshmanager add --host app.internal --username ubuntu --auth-mode key --identity-file ~/.ssh/id_ed25519 --proxy-jump bastion.internal:2222 --local-forward 8080:127.0.0.1:80 --remote-forward 9000:127.0.0.1:9000 --extra-ssh-arg -vv --extra-ssh-arg -o --extra-ssh-arg ServerAliveInterval=30
sshmanager add --host internal.example.com --username targetuser --auth-mode password --proxy-jump jumpuser@bastion.example.com --proxy-jump-auth-mode password --alias internal-server
```

When the jump host and the target host need different credentials (e.g. both
require password auth), set `--proxy-jump-auth-mode`/`--proxy-jump-password-stdin`/
`--proxy-jump-identity-file` independently of the target host's own auth
fields. If left unset, ProxyJump behaves as before (passed through natively
to `ssh -J`, relying on the system's own key/agent/ssh_config for the hop).
A dedicated jump password or identity file is only supported for a single
ProxyJump hop.

#### Edit saved connections

Password mode without a supplied secret prompts without echo. For automation:

```bash
printf '%s\n' "$PASSWORD" | sshmanager add --host prod.example.com --username ubuntu --auth-mode password --password-stdin --alias prod
sshmanager edit --alias prod --new-password-fd 3 3< /secure/password-file
```

Input consumes one line, preserves spaces, and strips a trailing CR in CRLF input.
Use separate descriptors when supplying both target and jump secrets; inherited
secret descriptors are closed after reading. No password-value environment variable
is used for these CLI inputs. Password authentication still uses short-lived,
child-scoped `SSHPASS` / jump-password environment channels at execution time;
these do not protect against processes running as the same user or root.

```bash
sshmanager edit --alias prod --new-host new.internal --new-port 2222
sshmanager edit --alias prod --new-group production --new-tag api --new-tag linux
sshmanager edit --id <connection-id> --new-auth-mode key --new-identity-file ~/.ssh/id_ed25519
sshmanager edit --alias prod --new-proxy-jump bastion.internal:2222 --new-local-forward 8080:127.0.0.1:80 --new-remote-forward 9000:127.0.0.1:9000 --new-extra-ssh-arg -vv --new-extra-ssh-arg -o --new-extra-ssh-arg ServerAliveInterval=30
```

#### Rename aliases

```bash
sshmanager rename --alias prod --to prod-new
sshmanager rename --id <connection-id> --to prod-new
```

#### Remove saved connections

```bash
sshmanager remove --alias prod --yes
sshmanager remove --id <connection-id> --yes
```

#### Connect by alias or ID

```bash
sshmanager connect --alias prod
sshmanager connect --alias prod --dry-run
sshmanager connect --id <connection-id>
```

#### Run remote commands and scripts

Run a command or stream a local script to a saved host without a TTY:

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

#### Transfer files with SCP

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

#### Export and import connections

Export decrypted connection data to a plaintext YAML or JSON file:

```bash
sshmanager export --out ./connections.yaml --format yaml
sshmanager export --out ./connections.json --format json
```

Import connection backups:

```bash
sshmanager import --in ./connections.yaml --mode merge
sshmanager import --in ./connections.json --mode replace
```

Import modes:

- `merge`: update existing entries by `id` (then by alias), add missing entries.
- `replace`: replace the entire connection set with imported data.

#### Back up and restore

Create a passphrase-encrypted recovery backup containing connections and optional configuration. The terminal prompts for a passphrase and confirmation. Keep that passphrase separately: the installation keyring cannot recover it.

```bash
sshmanager backup --out ./snapshot.sshm
sshmanager backup --out ./snapshot.sshm --format json --include-config=false
```

Restore from a recovery backup:

```bash
sshmanager restore --in ./snapshot.sshm --mode merge
sshmanager restore --in ./snapshot.sshm --mode replace --with-config=true
```

Automation can use `--passphrase-stdin` or `--passphrase-fd N` on both commands.
Backups use a versioned AES-256-GCM envelope with an authenticated header,
a random salt/nonce, and PBKDF2-SHA256 (600,000 iterations). They do not depend
on the source installation's encryption key and can be restored on another machine.
The encrypted payload retains the selected YAML/JSON format. Restore accepts files
up to 64 MiB and continues to support existing plaintext backups/exports.

Plaintext recovery snapshots require an explicit unsafe choice:

```bash
sshmanager backup --out ./snapshot.yaml --plaintext
```

`export` remains a plaintext interoperability command. Protect those files separately.

Restore modes:

- `merge`: merge restored entries into existing data.
- `replace`: replace the entire connection set with restored data.

#### Diagnose storage and configuration

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
| `behaviour.continueAfterSSHExit` | `false` | boolean | If `true`, return to the TUI after SSH exits. If `false`, exit the app after SSH session ends. |
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
- Linux/BSD: a session D-Bus Secret Service, such as GNOME Keyring or KDE's KSecretD. The integration uses the `login` collection when available, otherwise the default collection.
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
restore a recovery backup into a fresh installation. Backups are passphrase-encrypted by default. Exports and `backup --plaintext`
contain plaintext secrets and must be protected separately. Neither includes
the encryption key or installation-specific keyring/recovery state.

Older binaries cannot read the new datastore envelope, even with file-key storage.
Use a protected plaintext export for interoperability rather than opening a migrated
datastore with an older binary. See [storage compatibility](docs/storage-format.md).

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
current schema atomically during the locked load — no manual migration step is required, and
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

## Frequently asked questions

### Does SSH Manager replace OpenSSH?

SSH Manager manages saved connection details and invokes your system's `ssh` and
`scp` tools. SSH keys and agents continue to work through OpenSSH. Saved connections
live in SSH Manager's own store; it does not automatically import your
`~/.ssh/config` hosts.

### Are individual passwords stored in the OS keyring?

The keyring stores one AES-256 encryption key per SSH Manager data store. Host and
ProxyJump passwords remain inside the encrypted local connection file. Switching
between keyring and file storage moves the same encryption key and verifies the
saved data before committing the change.

### Can I use it from scripts or AI coding tools?

The CLI supports noninteractive host management, `list --json`, field selection,
remote commands through `exec`, and file transfers through `scp`. An AI coding tool
with terminal access can invoke these same commands. SSH Manager does not include
an AI model or require an AI service.

### Can I use it on a headless server?

Yes. Set `security.keyStorage` to `file` before first use if no OS keyring is
available. The CLI works without opening the TUI. SSH-agent or private-key
authentication avoids the runtime `sshpass` dependency.

## Development

```bash
make test
make vet
make lint
```

SSH invocation hardening rejects leading-option destinations, shell syntax in
host/user/jump fields, and executable OpenSSH hooks in extra options. Extra `-o`
options now use an allowlist of connection, authentication, keepalive, logging,
host-key, and session settings; unsupported saved options fail before execution.
SSH/SCP use explicit option separators. SCP local paths containing colons remain
local, and remote paths with whitespace or shell metacharacters are rejected for
compatibility with legacy SCP's remote shell. `exec` command strings are intentionally
interpreted by the remote shell. The user's OpenSSH configuration and executables
remain trusted; see [execution and backup boundaries](docs/security-boundaries.md).

## Security notes

Read the [security policy and threat model](SECURITY.md) and [architecture](docs/architecture.md).
The credential-display configuration has been removed; legacy settings are ignored.
`doctor` warns about disabled host-key checks, agent forwarding, and root password
authentication. `connect --alias prod --dry-run` prints redacted argv without starting SSH.

- Connection data uses a versioned AES-256-GCM envelope with an authenticated header. Legacy ciphertext is migrated atomically on normal load. See [storage format](docs/storage-format.md).
- Backups are passphrase-encrypted by default. Exports and `backup --plaintext` include plaintext passwords.
- Key files are validated and stored with restrictive permissions.
- OS keyring mode avoids storing the raw encryption key alongside encrypted connections.
- Password-mode connections pass passwords to `sshpass` via environment variable (`SSHPASS`) instead of CLI args.
- Password input uses hidden terminal prompts, `--password-stdin`, or `--password-fd N`. Edit uses the `--new-password-*` names; jump credentials use `--proxy-jump-password-*` / `--new-proxy-jump-password-*`. The old argument flags are removed. Explicit `*-unsafe` flags retain argument-based input and expose secrets to process inspection and shell history.
- Key/agent modes use OpenSSH directly (no `sshpass` dependency at runtime).
- Optional master passphrase mode derives encryption keys from `SSHMANAGER_MASTER_PASSPHRASE`.
- State file writes use atomic temp-write + rename flow.
- Connection operations, migration, and cleanup share an OS-backed process lock.
- Secure deletion is best-effort and may not provide full guarantees on all filesystems.
- SSH keys/agent are preferred over password authentication when possible.

## License

Licensed under the [Apache License 2.0](LICENSE).
