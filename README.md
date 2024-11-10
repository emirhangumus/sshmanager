# SSH Manager

This Go application allows you to manage SSH connections securely using encryption. You can add SSH connection details, encrypt them, and easily connect to your SSH servers through the command line.

## Features

- **Add SSH Connections**: Save SSH connection details securely.
- **Connect to SSH**: Select from saved SSH connections and connect automatically.
- **Encryption**: All connection details are encrypted using NaCl `secretbox` to ensure security.

## Prerequisites

- Go 1.19 or higher
- SSH and SSHPass installed on your system
  ```bash
  sudo apt install openssh-client sshpass
  ```

## Installation

1. Clone the repository
   ```bash
   git clone https://github.com/emirhangumus/sshmanager.git && cd sshmanager
   ```
2. Build the application
   ```bash
    go build -o sshmanager
   ```
3. Run the application
   ```bash
   ./sshmanager
   ```
4. Add the executable to your PATH to run the application from anywhere (Optional)

## Flags

- **--clean**: Clean the saved SSH connections (Inculding the encrypted file)
  ```bash
  ./sshmanager --clean
  ```

### Add SSH Connection

1. Run the application
   ```bash
   ./sshmanager
   ```
2. Select `Add SSH Connection`

3. Enter the connection details

   - **Host**: The hostname or IP address of the SSH server
   - **Username**: The username to connect to the SSH server
   - **Password**: The password to connect to the SSH server

4. The connection details will be encrypted and saved to the `~/.sshmanager/conn` file.

### Connect to SSH

1. Run the application
   ```bash
   ./sshmanager
   ```
2. Select `Connect to SSH`

3. Select the SSH connection you want to connect to

4. The application will automatically connect to the SSH server using the saved connection details
