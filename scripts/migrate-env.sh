#!/usr/bin/env bash
# ==============================================================================
# Bisync Hub Environment Migration Utility
# Non-destructively scans deployed bisync.env against canonical template,
# creates timestamped backup, and appends missing configuration keys and comments.
# ==============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

ENV_FILE="${ENV_FILE:-${REPO_ROOT}/bisync.env}"
TEMPLATE_FILE="${TEMPLATE_FILE:-${REPO_ROOT}/config/bisync.env.example}"
DRY_RUN=false

print_usage() {
    cat <<EOF
Usage: ./scripts/migrate-env.sh [OPTIONS]

Non-destructively audit and upgrade a deployed bisync.env file against the
canonical config/bisync.env.example template.

Options:
  -n, --dry-run             Report missing keys without modifying files
  -e, --env-file <path>     Path to deployed bisync.env (default: bisync.env)
  -t, --template <path>     Path to canonical template (default: config/bisync.env.example)
  -h, --help                Show this help documentation and exit

Examples:
  ./scripts/migrate-env.sh --dry-run                    # Audit bisync.env for missing keys
  ./scripts/migrate-env.sh                              # Apply migration and create .bak.<ts>
  ./scripts/migrate-env.sh -e /etc/bisync.env           # Migrate a custom deployed env file
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)
            print_usage
            exit 0
            ;;
        -n|--dry-run)
            DRY_RUN=true
            shift
            ;;
        -e|--env-file)
            ENV_FILE="$2"
            shift 2
            ;;
        -t|--template)
            TEMPLATE_FILE="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1" >&2
            print_usage
            exit 1
            ;;
    esac
done

export PATH="${HOME}/go/bin:/usr/local/go/bin:/usr/local/bin:${PATH}"

CLI_ARGS=("--env-file=${ENV_FILE}" "--template=${TEMPLATE_FILE}")
if [ "${DRY_RUN}" = true ]; then
    CLI_ARGS+=("--dry-run")
fi

if [ -x "${REPO_ROOT}/rclone-bisync" ]; then
    exec "${REPO_ROOT}/rclone-bisync" migrate-env "${CLI_ARGS[@]}"
else
    exec go run "${REPO_ROOT}/main.go" migrate-env "${CLI_ARGS[@]}"
fi
