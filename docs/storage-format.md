# Storage format and recovery

## Connection ciphertext

New writes use the binary datastore envelope below. The format is separate from
both the YAML payload version and the independent portable backup format.

| Offset | Bytes | Meaning |
| --- | --- | --- |
| 0 | 8 | `SSHMSTOR` magic |
| 8 | 1 | Format version: 1 |
| 9 | 1 | Cipher: 1 (AES-256-GCM with a 32-byte key) |
| 10 | 1 | Payload schema identifier: 1 (connection YAML version `1.0`) |
| 11 | 1 | Reserved: zero |
| 12 | 12 | Random GCM nonce |
| 24 | variable | Ciphertext followed by a 16-byte GCM authentication tag |

All 24 header bytes are authenticated as AEAD additional data. Unsupported
identifiers, truncated envelopes, altered metadata/ciphertext, and wrong keys fail
before plaintext is returned. Encrypted state input is bounded at 64 MiB.
Key derivation belongs to the key provider, not this envelope: the store uses the
resolved random or passphrase-derived AES key and never stores the passphrase.

Legacy nonce-prefixed AES-GCM remains readable. A normal locked load authenticates
and parses it, assigns any missing IDs, then atomically rewrites the current format
before returning. `Inspect`/doctor authenticates without performing this migration.
A legacy nonce that happens to begin with the entire eight-byte format magic is
reserved as an envelope and fails closed rather than triggering format fallback.
Old application versions cannot read newly written envelopes. Use an explicit
plaintext export (protected separately) for interoperability with an older binary.

## Payload and key metadata

The payload is the `ConnectionFile` YAML object with `version: "1.0"` and
`connections`. Legacy connection lists and objects without a version are readable;
unknown explicit schema versions are rejected rather than silently interpreted.
Random stable IDs are unrelated to aliases, hosts, or usernames.

`key-storage.yaml` has its own versioned journal, store ID, active backend, key
fingerprint, and optional nonsecret passphrase metadata. Keyring mode stores one
AES key under the application service and store-specific account. File mode uses
`secret.key`, either a 32-byte raw key or PBKDF2-SHA256 metadata. Existing PBKDF2
metadata is retained for compatibility; iteration counts are bounded to avoid
unbounded work from damaged metadata. Argon2id and provider refactoring remain
roadmap work.

## Transactions

All cooperating mutations hold `conn.lock` through read/modify/commit. Writes stage
0600 files in the destination directory, sync contents, rename atomically, and
attempt to sync the directory. Restore stages encrypted `conn.restore-pending` and
records checkpoints in the key journal. On restart, normal store operations resume
or cancel according to that journal. Never delete the key journal to fix a failed
migration; restore original key/state or an independently protected backup.

See [backup envelope](security-boundaries.md#portable-encrypted-backups) for the
separate portable backup format and [SECURITY.md](../SECURITY.md) for assumptions.
