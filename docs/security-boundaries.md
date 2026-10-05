# Execution and backup boundaries

SSH Manager invokes system OpenSSH with argument slices. Destination fields are
validated before connect, exec, SCP, and host-key priming. DNS names, IP addresses,
and ordinary OpenSSH aliases are accepted; usernames are limited to ASCII letters,
digits, underscore, dot, plus, backslash, and hyphen, with no leading hyphen.
Custom jump ProxyCommands quote literals and the destination substitution. Jump
identity paths cannot contain percent tokens. Extra SSH options use an allowlist
so executable hooks and alternative config loading cannot enter through saved
options. SCP applies a narrower flag set and rejects remote shell path syntax.
These stricter rules can require changes to existing unusual saved profiles.

The local OpenSSH config, PATH, executables, home directory, and application data
directory are trusted. OpenSSH config can itself execute local commands. Remote
`exec` strings intentionally use the remote shell. Host-key trust remains with
OpenSSH; insecure host-key options are still explicit user choices.

Password CLI inputs use terminal prompts without echo, stdin, or inherited file
descriptors. Explicit `*-unsafe` arguments remain an escape hatch. Execution still
uses child-scoped environment channels for sshpass and dedicated jump passwords.
Inherited sshmanager password and master-passphrase variables are stripped from
child environments; only credentials needed for the selected invocation are added.
The master-passphrase environment interface also remains unchanged. Environment
channels, running processes, terminal output, and Go strings are not secret-storage
boundaries against the same user or root. Mutable input and derived-key buffers
are cleared where practical; this is best-effort lifetime reduction, not guaranteed
memory erasure. Normal configuration cannot enable credential display.

## Portable encrypted backups

Backups are independent of the datastore's OS keyring or file key. Losing the backup
passphrase makes an encrypted backup unrecoverable. Choose a strong, unique
passphrase and store it separately. Exports and `backup --plaintext` remain plaintext.

Backup v1 is a binary envelope:

| Offset | Bytes | Meaning |
| --- | --- | --- |
| 0 | 8 | `SSHMBKUP` magic |
| 8 | 1 | Version: 1 |
| 9 | 1 | Cipher: 1 (AES-256-GCM) |
| 10 | 1 | KDF: 1 (PBKDF2-SHA256, 600,000 iterations, 32-byte key) |
| 11 | 1 | Reserved: zero |
| 12 | 16 | Random salt |
| 28 | 12 | Random GCM nonce |
| 40 | variable | Encrypted YAML/JSON snapshot and 16-byte authentication tag |

The complete 40-byte header is GCM additional authenticated data. Unknown identifiers,
invalid lengths, wrong passphrases, and authentication failures are rejected before
restore mutates state. Restore bounds input at 64 MiB. The plaintext snapshot retains
backup version, creation time, configuration, and connections. Those fields are
inside encryption. Datastore ciphertext uses its own [versioned envelope](storage-format.md).

Legacy plaintext backups and connection exports remain restorable. Authenticated
backups protect confidentiality and integrity, but do not establish who created a
backup or prevent replay of an older valid backup.
