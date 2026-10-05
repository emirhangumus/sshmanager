# Architecture

SSH Manager is a local application over system OpenSSH. The CLI/TUI selects and
edits profiles; the connection store owns encryption, locking, key selection,
migration, and recovery. The application does not implement the SSH protocol.

| Package | Responsibility |
| --- | --- |
| `cmd/sshmanager` | Process entrypoint and build metadata |
| `internal/app` | Command routing and application exit behavior |
| `internal/cli/commands` | Selectors, CLI workflows, SSH/SCP argv and execution |
| `internal/cli`, `internal/ui` | Interactive menu, forms, progress |
| `internal/model` | Connections, IDs, aliases, metadata and SSH-option validation |
| `internal/store` | Locked transactions, keyring/file storage, recoverable migration/restore |
| `internal/crypto` | AES-GCM envelopes, passphrase derivation, portable backups |
| `internal/storage` | Regular-file checks, bounded reads, atomic writes and best-effort cleanup |
| `internal/config`, `internal/startup` | Config defaults/validation and first-run setup |
| `internal/completion` | Shell completion generation and installation |

A mutation flows from the CLI/TUI to `ConnectionStore.Update`, which locks, loads
and authenticates the store, applies a model mutation, then writes new ciphertext
atomically. `Save` writes a complete snapshot: callers needing concurrent updates
must use `Update` so they do not overwrite another writer's intervening changes.

Connect/exec/SCP resolve a saved profile and construct a validated argument vector.
OpenSSH handles SSH configuration, agents, keys, certificates, algorithms and
host-key trust. Exec appends a single intentionally remote-shell-interpreted command;
script mode streams the file over stdin. Dedicated jump credentials use a quoted
ProxyCommand. The environment helper strips inherited application secrets and adds
only credentials required for the selected invocation.

The current code keeps SSH execution in the command package. A separate resolver,
argv and executor package plus abstract key providers remain future refactors;
this document describes the implemented architecture rather than the target layout.

See [storage/recovery](storage-format.md) and [security policy](../SECURITY.md).
