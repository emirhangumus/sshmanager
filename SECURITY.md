# Security policy

## Supported versions

Security fixes target the latest released 2.x version and the current `main` branch.
Older releases are not maintained with security backports. Reports affecting older
versions are welcome, but reproduce them on a maintained version when possible.

## Report a vulnerability privately

Use [GitHub private vulnerability reporting](https://github.com/emirhangumus/sshmanager/security/advisories/new).
Do not publish exploitable details, passwords, encryption keys, backups, or private
infrastructure information in public issues, pull requests, or logs.

Include the affected version/commit, OS, key-storage backend, reproduction steps
using synthetic credentials, impact, and a minimal proof of concept. Maintainers
will coordinate investigation, a fix, and disclosure through the private report.
The acknowledgement target is seven business days; this volunteer project cannot
guarantee a response or remediation deadline. Follow up in the same private report
if you have not received a response.

## Threat model

SSH Manager protects connection and jump passwords, connection metadata, and notes
at rest using authenticated AES-256-GCM encryption. In normal keyring mode the
random encryption key is stored in the user's OS keyring rather than next to the
ciphertext. Passphrase-encrypted backups are independent of the installation key.
File-key mode stores a raw key beside the store unless passphrase protection was
explicitly selected; copying both files defeats that mode's confidentiality.

Atomic writes avoid exposing a partly written replacement file. OS-backed process
locks serialize cooperating datastore mutations, migrations, restore, and cleanup.
Restore/key migrations have durable checkpoints for interrupted-operation recovery.
These mechanisms reduce accidental corruption and lost updates; they are not a
substitute for backups and cannot guarantee durability against every filesystem or
power-loss failure. Permissions are created as 0600 files and 0700 directories on
POSIX systems. Windows access protection depends on the user's filesystem ACLs.

The trusted computing base includes the OS, OS keyring, user account, HOME and all
ancestor directories of the application data directory, application binaries,
PATH, OpenSSH executables/configuration, and remote systems. An attacker able to
replace these or act as the same user is outside the protection boundary.

The application does not protect against:

- Malicious root, administrators, or processes running as the same user.
- Compromised OS keyrings, SSH clients/configuration, or remote hosts.
- Credentials in a running process, child environments, terminals, or Go strings.
- Shell history/process inspection when explicit `*-unsafe` secret flags are used.
- Plaintext exports or explicitly plaintext backups, screenshots, and external logs.
- Theft of both a raw file key and its ciphertext.
- Offline guessing of weak backup/master passphrases, replay of an older valid store,
  or an attacker deleting files and backups.
- Complete memory erasure or secure deletion on SSDs, snapshots, journaled storage,
  backups, and copy-on-write filesystems.

Secrets use hidden terminal prompts, stdin, or inherited descriptors by default.
Password authentication still uses child-scoped sshpass environment channels. The
master-passphrase environment interface remains supported and must also be treated
as a credential channel. Mutable buffers are cleared where practical as best-effort
lifetime reduction. Normal configuration cannot enable credential printing; old
`showCredentialsOnConnect` fields are ignored and omitted on the next config save.

OpenSSH owns host-key verification and private-key handling. SSH Manager does not
implement SSH or automatically disable host-key checks. Explicit insecure saved
OpenSSH options remain the user's choice. Local invocation uses argument slices,
validated destinations, and controlled extra options. Remote exec strings
intentionally use the remote shell. User OpenSSH config may itself execute commands.

State-file operations reject symlink/nonregular leaves and an immediately symlinked
state directory; opened-file identity is checked before reads and overwrites.
These checks do not provide a complete same-user TOCTOU defense. Ancestor directory
replacement and hard links remain outside the trusted-directory assumption.

See [storage format](docs/storage-format.md), [architecture](docs/architecture.md),
and [execution/backup boundaries](docs/security-boundaries.md) for details.
