# SSH Manager

A secure and simple command-line tool for managing and connecting to SSH servers. SSH Manager stores your SSH connection details using strong encryption and lets you connect to servers with a few keystrokes.

## Demo
![Demo](demo.gif)

## Features

- 🔐 **Encrypted Storage**: Uses NaCl `secretbox` for secure encryption of connection details.
- ➕ **Add SSH Connections**: Input and securely save SSH server details.
- ⚡ **Quick Connect**: Automatically connect to saved SSH servers using `sshpass`.
- 🧹 **Cleanup Option**: Clear all saved (encrypted) SSH connections with a single flag.

## Requirements

- **Go 1.19+**
- **sshpass**
- **OpenSSH client**

Install requirements on Debian/Ubuntu:
```bash
sudo apt install openssh-client sshpass
```

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/emirhangumus/sshmanager.git
cd sshmanager
```

### 2. Build the Binary

```bash
make build
```

### 3. (Optional) Install to `~/.local/bin`

```bash
make install
```

### 4. Run the App

```bash
sshmanager
```

## Usage

### Run the App

```bash
sshmanager
```

You'll see a menu with the following options:

* **Add SSH Connection**: Enter host, username, and password (encrypted and saved).
* **Connect to SSH**: Choose a saved connection to connect instantly.

### Flags

* `-clean` – Remove all saved SSH connections:

  ```bash
  sshmanager -clean
  ```

* `-version` – Show the current version of SSH Manager:

  ```bash
  sshmanager -version
  ```

* `-help` – Show help information:

  ```bash
  sshmanager -help
  ```
  
* `-set` – Set a SSHManager config.

  ```bash
  sshmanager -set <key> <value>
  ```
| Key                              | Default Value   | Value Type   | Description                                                                                                                        |
|----------------------------------|-----------------|--------------|------------------------------------------------------------------------------------------------------------------------------------|
| `behaviour.continueAfterSSHExit` | `false`         | boolean      | If set to `true`, SSH Manager will return to the main menu after exiting an SSH session. If `false`, it will exit the application. |

## File Structure

Encrypted connection data is saved to:

```
~/.sshmanager/conn
```

## Makefile Commands

| Command                   | Description                 |
| ------------------------- | --------------------------- |
| `make build`              | Build the binary            |
| `make build_compressed`   | Build and compress with UPX |
| `make install`            | Install the binary locally  |
| `make install_compressed` | Install compressed binary   |
| `make run`                | Builds and runs the binary  |
| `make clean`              | Remove build artifacts      |
| `make remove`             | Remove the installed binary |

## Security Notes

All SSH connection details are encrypted using `secretbox` (XSalsa20 + Poly1305). Your credentials never leave your machine and are stored in encrypted form only.

## License

Licensed under the [Apache License 2.0](LICENSE).