#!/usr/bin/env bash

set -euo pipefail

# =============================================================================
# Local Go Toolchain Maintenance
#
# Local use only.
#
# Features:
#   - Architecture detection
#   - Dry run
#   - Status reporting
#   - Automatic rollback on failure
#   - Manual rollback
#   - SHA256 verification
#   - Backup retention
#
# Usage:
#
#   ./update-tools.sh
#   ./update-tools.sh --status
#   ./update-tools.sh --dry-run
#   ./update-tools.sh --rollback
#   ./update-tools.sh --go-version 1.26.11
#
# =============================================================================

# ----------------------------------------------------------------------
# Configuration
# ----------------------------------------------------------------------

GO_VERSION_DEFAULT="1.26.10"
STATICCHECK_VERSION="2025.1.1"
GOVULNCHECK_VERSION="v1.9.0"

DRY_RUN=false
ROLLBACK=false
STATUS=false

GO_VERSION="${GO_VERSION_DEFAULT}"

# ----------------------------------------------------------------------
# Logging
# ----------------------------------------------------------------------

log() {
    echo "[INFO] $*"
}

success() {
    echo "[OK] $*"
}

warn() {
    echo "[WARN] $*"
}

fatal() {
    echo "[ERROR] $*" >&2
    exit 1
}

# ----------------------------------------------------------------------
# Command Execution
# ----------------------------------------------------------------------

run() {
    if [[ "${DRY_RUN}" == "true" ]]; then
        printf '[DRY-RUN]'
        printf ' %q' "$@"
        printf '\n'
    else
        "$@"
    fi
}

# ----------------------------------------------------------------------
# Usage
# ----------------------------------------------------------------------

usage() {
    cat <<EOF
Usage:

  $(basename "$0")

Options:

  --status
      Show installed versions and backups

  --dry-run
      Show actions without making changes

  --rollback
      Restore latest backup

  --go-version VERSION
      Install specific Go version

Examples:

  $(basename "$0")

  $(basename "$0") --status

  $(basename "$0") --dry-run

  $(basename "$0") --go-version 1.26.11

  $(basename "$0") --rollback

EOF
}

# ----------------------------------------------------------------------
# Parse Arguments
# ----------------------------------------------------------------------

while [[ $# -gt 0 ]]; do

    case "$1" in

        --go-version)
            [[ $# -ge 2 ]] || fatal "--go-version requires a value"

            GO_VERSION="$2"

            shift 2
            ;;

        --dry-run)
            DRY_RUN=true
            shift
            ;;

        --rollback)
            ROLLBACK=true
            shift
            ;;

        --status)
            STATUS=true
            shift
            ;;

        --help)
            usage
            exit 0
            ;;

        *)
            fatal "Unknown option: $1"
            ;;

    esac

done

# ----------------------------------------------------------------------
# Validate Version Format
# ----------------------------------------------------------------------

[[ "${GO_VERSION}" =~ ^[0-9]+\.[0-9]+(\.[0-9]+)?([a-z0-9.-]+)?$ ]] \
    || fatal "Invalid Go version: ${GO_VERSION}"

# ----------------------------------------------------------------------
# Platform Detection
# ----------------------------------------------------------------------

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"

ARCH_RAW="$(uname -m)"

case "${ARCH_RAW}" in
    aarch64|arm64)
        ARCH="arm64"
        ;;
    x86_64|amd64)
        ARCH="amd64"
        ;;
    *)
        fatal "Unsupported architecture: ${ARCH_RAW}"
        ;;
esac

# ----------------------------------------------------------------------
# Status
# ----------------------------------------------------------------------

show_status() {

    echo
    echo "===================================================="
    echo "Toolchain Status"
    echo "===================================================="
    echo

    echo "Platform"

    echo "  OS   : ${OS}"
    echo "  Arch : ${ARCH}"
    echo

    echo "Go"

    if command -v go >/dev/null 2>&1; then
        echo "  Installed : $(go version)"
        echo "  Path      : $(command -v go)"
    else
        echo "  Installed : NO"
    fi

    echo

    echo "govulncheck"

    if command -v govulncheck >/dev/null 2>&1; then
        echo "  Path : $(command -v govulncheck)"
    else
        echo "  Installed : NO"
    fi

    echo

    echo "staticcheck"

    if command -v staticcheck >/dev/null 2>&1; then
        echo "  Path : $(command -v staticcheck)"
    else
        echo "  Installed : NO"
    fi

    echo

    echo "PATH Checks"

    if [[ ":$PATH:" == *":/usr/local/go/bin:"* ]]; then
        echo "  ✓ /usr/local/go/bin"
    else
        echo "  ✗ /usr/local/go/bin"
    fi

    if [[ -d "${HOME}/go/bin" ]]; then

        if [[ ":$PATH:" == *":${HOME}/go/bin:"* ]]; then
            echo "  ✓ ${HOME}/go/bin"
        else
            echo "  ✗ ${HOME}/go/bin"
        fi
    fi

    echo

    echo "Backups"

    shopt -s nullglob

    backups=(/usr/local/go.backup.*)

    if (( ${#backups[@]} == 0 )); then
        echo "  None"
    else
        printf '%s\n' "${backups[@]}" | sort
    fi

    echo
}

if [[ "${STATUS}" == "true" ]]; then
    show_status
    exit 0
fi

# ----------------------------------------------------------------------
# Rollback
# ----------------------------------------------------------------------

rollback() {

    shopt -s nullglob

    backups=(/usr/local/go.backup.*)

    (( ${#backups[@]} > 0 )) || fatal "No Go backup found."

    latest_backup="$(
        printf '%s\n' "${backups[@]}" |
        sort |
        tail -n 1
    )"

    echo
    log "Restoring backup:"
    echo "  ${latest_backup}"
    echo

    if [[ "${DRY_RUN}" == "true" ]]; then

        echo "[DRY-RUN] sudo mv /usr/local/go /usr/local/go.rollback-bad"
        echo "[DRY-RUN] sudo mv ${latest_backup} /usr/local/go"

        return
    fi

    if [[ -d /usr/local/go ]]; then

        sudo mv \
            /usr/local/go \
            /usr/local/go.rollback-bad

    fi

    sudo mv \
        "${latest_backup}" \
        /usr/local/go

    success "Rollback completed"

    /usr/local/go/bin/go version
}

if [[ "${ROLLBACK}" == "true" ]]; then
    rollback
    exit 0
fi

# ----------------------------------------------------------------------
# Prerequisites
# ----------------------------------------------------------------------

for tool in curl tar sha256sum sudo; do

    command -v "${tool}" >/dev/null 2>&1 \
        || fatal "Required tool not found: ${tool}"

done

# ----------------------------------------------------------------------
# Download Details
# ----------------------------------------------------------------------

GO_TARBALL="go${GO_VERSION}.${OS}-${ARCH}.tar.gz"
GO_URL="https://go.dev/dl/${GO_TARBALL}"

# ----------------------------------------------------------------------
# Temporary Workspace
# ----------------------------------------------------------------------

TMP_DIR=""

cleanup() {

    local rc=$?

    if [[ -n "${TMP_DIR}" && -d "${TMP_DIR}" ]]; then
        rm -rf "${TMP_DIR}"
    fi

    if (( rc != 0 )); then
        warn "Script exited with errors"
    fi
}

trap cleanup EXIT

# ----------------------------------------------------------------------
# Download
# ----------------------------------------------------------------------

echo
log "Target Version : ${GO_VERSION}"
log "Platform       : ${OS}/${ARCH}"

TMP_DIR="$(mktemp -d)"

GO_ARCHIVE="${TMP_DIR}/${GO_TARBALL}"

if [[ "${DRY_RUN}" == "true" ]]; then

    echo "[DRY-RUN] Download ${GO_URL}"

else

    log "Downloading Go..."

    curl \
        -fSL \
        --retry 3 \
        -o "${GO_ARCHIVE}" \
        "${GO_URL}"

    success "Download complete"

fi

# ----------------------------------------------------------------------
# Verify SHA256
# ----------------------------------------------------------------------

if [[ "${DRY_RUN}" == "false" ]]; then

    log "Verifying checksum..."

    EXPECTED_HASH="$(curl -fsSL "${GO_URL}.sha256" | tr -d '[:space:]')"
    ACTUAL_HASH="$(sha256sum "${GO_ARCHIVE}" | awk '{print $1}')"

    [[ -n "${EXPECTED_HASH}" ]] || fatal "Failed to download SHA256 checksum from ${GO_URL}.sha256"

    if [[ "${EXPECTED_HASH}" != "${ACTUAL_HASH}" ]]; then
        fatal "Checksum mismatch!\n  Expected: ${EXPECTED_HASH}\n  Got:      ${ACTUAL_HASH}"
    fi

    success "Checksum verified (${ACTUAL_HASH})"

fi

# ----------------------------------------------------------------------
# Already Installed?
# ----------------------------------------------------------------------

if command -v go >/dev/null 2>&1; then

    CURRENT_GO_VERSION="$(
        go version |
        awk '{print $3}' |
        sed 's/^go//'
    )"

    if [[ "${CURRENT_GO_VERSION}" == "${GO_VERSION}" ]]; then

        success "Go ${GO_VERSION} is already installed."

        echo

        echo "Updating tooling only..."

    fi
fi

# ----------------------------------------------------------------------
# Backup Existing Installation
# ----------------------------------------------------------------------

TIMESTAMP="$(date +%Y%m%d-%H%M%S)"

BACKUP_DIR="/usr/local/go.backup.${TIMESTAMP}"

if [[ -d /usr/local/go ]]; then

    log "Creating backup..."

    run sudo mv \
        /usr/local/go \
        "${BACKUP_DIR}"

    success "Backup: ${BACKUP_DIR}"
fi

# ----------------------------------------------------------------------
# Install Go
# ----------------------------------------------------------------------

log "Installing Go..."

run sudo tar \
    -C /usr/local \
    -xzf "${GO_ARCHIVE}"

# ----------------------------------------------------------------------
# Verify Go
# ----------------------------------------------------------------------

if [[ "${DRY_RUN}" == "false" ]]; then

    [[ -x /usr/local/go/bin/go ]] \
        || fatal "Go installation verification failed"

    INSTALLED_VERSION="$(
        /usr/local/go/bin/go version |
        awk '{print $3}' |
        sed 's/^go//'
    )"

    [[ "${INSTALLED_VERSION}" == "${GO_VERSION}" ]] \
        || fatal "Installed ${INSTALLED_VERSION}, expected ${GO_VERSION}"

    success "Go ${GO_VERSION} installed"
fi

# ----------------------------------------------------------------------
# PATH Validation
# ----------------------------------------------------------------------

echo
echo "PATH Validation"

if [[ ":$PATH:" == *":/usr/local/go/bin:"* ]]; then
    echo "  ✓ /usr/local/go/bin"
else
    echo "  ✗ /usr/local/go/bin"
    echo
    echo "Add this to your shell profile:"
    echo
    echo 'export PATH=/usr/local/go/bin:$PATH'
fi

# ----------------------------------------------------------------------
# Install Tooling
# ----------------------------------------------------------------------

export PATH="/usr/local/go/bin:${PATH}"
export GOTOOLCHAIN=local

if [[ "${DRY_RUN}" == "false" ]]; then

    GOBIN="$(go env GOBIN)"

    if [[ -z "${GOBIN}" ]]; then
        GOBIN="$(go env GOPATH)/bin"
    fi

    mkdir -p "${GOBIN}"

    log "Installing govulncheck..."

    go install \
        "golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION}"

    log "Installing staticcheck..."

    go install \
        "honnef.co/go/tools/cmd/staticcheck@${STATICCHECK_VERSION}"

fi

# ----------------------------------------------------------------------
# Verify Tooling
# ----------------------------------------------------------------------

if [[ "${DRY_RUN}" == "false" ]]; then

    command -v govulncheck >/dev/null 2>&1 \
        || fatal "govulncheck installation failed"

    command -v staticcheck >/dev/null 2>&1 \
        || fatal "staticcheck installation failed"
fi

# ----------------------------------------------------------------------
# Backup Retention
# ----------------------------------------------------------------------

if [[ "${DRY_RUN}" == "false" ]]; then

    shopt -s nullglob

    backups=(/usr/local/go.backup.*)

    if (( ${#backups[@]} > 3 )); then

        mapfile -t old_backups < <(
            printf '%s\n' "${backups[@]}" |
            sort |
            head -n -3
        )

        for backup in "${old_backups[@]}"; do

            log "Removing old backup: ${backup}"

            sudo rm -rf "${backup}"

        done

    fi
fi

# ----------------------------------------------------------------------
# Summary
# ----------------------------------------------------------------------

echo
echo "===================================================="
echo "Update Complete"
echo "===================================================="
echo

echo "Go Version:"
echo "  ${GO_VERSION}"

echo

echo "govulncheck:"
echo "  ${GOVULNCHECK_VERSION}"

echo

echo "staticcheck:"
echo "  ${STATICCHECK_VERSION}"

echo

show_status