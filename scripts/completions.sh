#!/usr/bin/env bash
#
# Generates the Bash and Z-shell completion scripts used by GoReleaser.
# Called from .goreleaser.yml → before.hooks.
#
#   ./scripts/completions.sh
#
# It assumes the repository layout:
#   cmd/sshmanager        → your main package
#   completions/          → target directory committed to the repo
#
# You can run it manually or let GoReleaser call it automatically.

set -euo pipefail

ROOT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )/.." && pwd )"
COMPL_DIR="${ROOT_DIR}/completions"

mkdir -p "${COMPL_DIR}"

echo "▶ generating Bash completion…"
go run ./cmd/sshmanager -completion bash > "${COMPL_DIR}/sshmanager.bash"

echo "▶ generating Z-shell completion…"
go run ./cmd/sshmanager -completion zsh  > "${COMPL_DIR}/sshmanager.zsh"

echo "✓ completions written to ${COMPL_DIR}/"
