# SSH Manager

SSH Manager is a terminal application for storing and connecting to SSH hosts from an interactive menu or alias command.

## Demo

![Demo](demo.gif)

## Features

- AES-GCM encrypted storage for saved connections
- Add, edit, remove, and connect from an interactive menu
- Direct alias connection (`sshmanager myserver`)
- Configurable post-SSH behavior (`behaviour.continueAfterSSHExit`)
- Shell completion support for Bash and Zsh
- Best-effort secure cleanup (`-clean`) for connection and key files

## Requirements

- Go `1.23.2+`
- `sshpass`
- OpenSSH client (`ssh`)

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

### Flags

- Clean data:

```bash
sshmanager -clean
```

- Show version:

```bash
sshmanager -version
```

- Set config value:

```bash
sshmanager -set behaviour.continueAfterSSHExit true
```

- Completion candidates (used by shell completion scripts):

```bash
sshmanager -complete [prefix]
```

- Print completion script:

```bash
sshmanager -completion bash
sshmanager -completion zsh
```

- Install completion script (explicit opt-in):

```bash
sshmanager -completion install bash
sshmanager -completion install zsh
```

For Bash, reload your shell after installation:

```bash
source ~/.bashrc
```

## Configuration

| Key | Default | Type | Description |
|---|---|---|---|
| `behaviour.continueAfterSSHExit` | `false` | boolean | If `true`, return to menu after SSH exits. If `false`, exit the app after SSH session ends. |

## Data files

SSH Manager stores files under:

```text
~/.sshmanager/
```

Files:

- `conn` (encrypted connection file)
- `secret.key` (AES-256 key, file mode `0600`)
- `config.yaml` (configuration)

## Development

```bash
make test
make vet
make lint
```

## Security notes

- Connection data is encrypted at rest using AES-GCM.
- Key files are validated and stored with restrictive permissions.
- SSH passwords are passed to `sshpass` via environment variable (`SSHPASS`) instead of CLI args.
- Secure deletion is best-effort and may not provide full guarantees on all filesystems.
- SSH keys/agent are preferred over password authentication when possible.

## License

Licensed under the [Apache License 2.0](LICENSE).
