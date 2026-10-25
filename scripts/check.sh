#!/usr/bin/env bash
# rclone-bisync Quality Gate Launcher
# Delegates to high-performance Go check engine (cmd/check)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

export PATH="${HOME}/go/bin:/usr/local/go/bin:/usr/local/bin:${PATH}"

exec go run "${REPO_ROOT}/cmd/check" "$@"
