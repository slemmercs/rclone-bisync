#!/usr/bin/env bash

set -euo pipefail

# =============================================================================
# Local Go Toolchain Maintenance
#
# Local use only.
#
# Usage:
#   ./update-tools.sh
#   ./update-tools.sh --status
#   ./update-tools.sh --dry-run
#   ./update-tools.sh --rollback
#   ./update-tools.sh --go-version 1.26.11
# =============================================================================

# ----------------------------------------------------------------------
# Configuration
# ----------------------------------------------------------------------

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

GO_VERSION_DEFAULT="1.26.10"
STATICCHECK_VERSION="2025.1.1"
GOVULNCHECK_VERSION="v1.9.0"
GOSEC_VERSION="v2.30.0"
BETTERLEAKS_VERSION="v1.9.0"
ESBUILD_VERSION="v0.25.0"

DRY_RUN=false
ROLLBACK=false
STATUS=false
TOOLS_ONLY=false
INSTALL_FRONTEND=false

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
    echo "[ERROR] ${1:-Unspecified failure occurred}" >&2
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

  $(basename "$0") [OPTIONS]

Options:

  --status
      Show installed versions and backups across Go, Node.js, npm, linters, and runtime engines

  --dry-run
      Show actions without making changes

  --rollback
      Restore latest Go installation backup

  --go-version VERSION
      Install specific Go version (default: ${GO_VERSION_DEFAULT})

  --tools-only
      Skip Go compiler download and update Go security tools and frontend dependencies only

  --install-frontend
      Install/update frontend npm packages in dashboard/frontend

  --help
      Show this help documentation

Examples:

  $(basename "$0")
  $(basename "$0") --status
  $(basename "$0") --tools-only
  $(basename "$0") --install-frontend
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
            if [[ $# -lt 2 ]]; then
                fatal "--go-version requires a version argument"
            fi

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

        --tools-only)
            TOOLS_ONLY=true
            shift
            ;;

        --install-frontend)
            INSTALL_FRONTEND=true
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

    echo "Node.js & npm (Frontend Toolchain)"
    if command -v node >/dev/null 2>&1; then
        echo "  Node.js   : $(node -v) ($(command -v node))"
    else
        echo "  Node.js   : NOT INSTALLED (required for frontend dev/rebuild)"
    fi
    if command -v npm >/dev/null 2>&1; then
        echo "  npm       : v$(npm -v) ($(command -v npm))"
    else
        echo "  npm       : NOT INSTALLED"
    fi
    if [[ -d "${REPO_ROOT}/dashboard/frontend/node_modules" ]]; then
        echo "  Packages  : INSTALLED (${REPO_ROOT}/dashboard/frontend/node_modules)"
    else
        echo "  Packages  : NOT INSTALLED (run: npm --prefix dashboard/frontend install)"
    fi
    echo

    echo "Rclone (Core Engine Runtime)"
    if command -v rclone >/dev/null 2>&1; then
        echo "  Installed : $(rclone version 2>/dev/null | head -n 1)"
        echo "  Path      : $(command -v rclone)"
    else
        echo "  Installed : NOT INSTALLED (install via: sudo apt install rclone or curl https://rclone.org/install.sh | sudo bash)"
    fi
    echo

    echo "ShellCheck (Bash Quality Gate)"
    if command -v shellcheck >/dev/null 2>&1; then
        echo "  Installed : $(shellcheck --version 2>/dev/null | grep version: || echo 'installed')"
        echo "  Path      : $(command -v shellcheck)"
    else
        echo "  Installed : NOT INSTALLED (install via: sudo apt install shellcheck)"
    fi
    echo

    echo "tidy (HTML Quality Gate)"
    if command -v tidy >/dev/null 2>&1; then
        echo "  Installed : $(tidy -v 2>/dev/null || echo 'installed')"
        echo "  Path      : $(command -v tidy)"
    else
        echo "  Installed : NOT INSTALLED (install via: sudo apt install tidy)"
    fi
    echo

    echo "Browser (Headless E2E Smoke Tests)"
    browser_found=false
    for b in google-chrome chromium chromium-browser; do
        if command -v "$b" >/dev/null 2>&1; then
            echo "  Browser   : $($b --version 2>/dev/null || echo "$b") ($(command -v "$b"))"
            browser_found=true
            break
        fi
    done
    if [[ "${browser_found}" == "false" ]]; then
        echo "  Browser   : NOT INSTALLED (install chromium or google-chrome for E2E tests)"
    fi
    echo

    echo "Go Security & Static Analysis Tools"
    for tool in govulncheck gosec staticcheck betterleaks esbuild; do
        if command -v "${tool}" >/dev/null 2>&1; then
            echo "  ✓ ${tool} : $(command -v "${tool}")"
        else
            echo "  ✗ ${tool} : NOT INSTALLED"
        fi
    done
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

    if [[ ${#backups[@]} -eq 0 ]]; then
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

    if [[ ${#backups[@]} -eq 0 ]]; then
        fatal "No Go backup found."
    fi

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
        sudo mv /usr/local/go /usr/local/go.rollback-bad
    fi

    sudo mv "${latest_backup}" /usr/local/go
    success "Rollback completed"

    /usr/local/go/bin/go version
}

if [[ "${ROLLBACK}" == "true" ]]; then
    rollback
    exit 0
fi

# ----------------------------------------------------------------------
# Prerequisites & Go Compiler Installation
# ----------------------------------------------------------------------

if [[ "${TOOLS_ONLY}" == "true" ]]; then
    log "Skipping Go compiler installation (--tools-only specified)"
else
    for tool in curl tar sha256sum sudo; do
        command -v "${tool}" >/dev/null 2>&1 \
            || fatal "Required tool not found: ${tool}"
    done

    GO_TARBALL="go${GO_VERSION}.${OS}-${ARCH}.tar.gz"
    GO_URL="https://dl.google.com/go/${GO_TARBALL}"
    GO_SHA256_URL="${GO_URL}.sha256"

    TMP_DIR=""

    cleanup() {
        local rc=$?
        if [[ -n "${TMP_DIR}" && -d "${TMP_DIR}" ]]; then
            rm -rf "${TMP_DIR}"
        fi
        if [[ ${rc} -ne 0 ]]; then
            warn "Script exited with errors"
        fi
    }

    trap cleanup EXIT

    echo
    log "Target Version : ${GO_VERSION}"
    log "Platform       : ${OS}/${ARCH}"

    TMP_DIR="$(mktemp -d)"
    GO_ARCHIVE="${TMP_DIR}/${GO_TARBALL}"

    if [[ "${DRY_RUN}" == "true" ]]; then
        echo "[DRY-RUN] Download ${GO_URL}"
    else
        log "Downloading Go..."
        curl -fSL --retry 3 -o "${GO_ARCHIVE}" "${GO_URL}"
        success "Download complete"
    fi

# ----------------------------------------------------------------------
# Verify SHA256
# ----------------------------------------------------------------------

if [[ "${DRY_RUN}" == "false" ]]; then
    log "Verifying checksum..."

    EXPECTED_HASH="$(curl -fsSL "${GO_SHA256_URL}" | tr -d '[:space:]')"
    ACTUAL_HASH="$(sha256sum "${GO_ARCHIVE}" | awk '{print $1}')"

    [[ -n "${EXPECTED_HASH}" ]] || fatal "Failed to download SHA256 checksum from ${GO_SHA256_URL}"

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
    run sudo mv /usr/local/go "${BACKUP_DIR}"
    success "Backup: ${BACKUP_DIR}"
fi

# ----------------------------------------------------------------------
# Install Go
# ----------------------------------------------------------------------

log "Installing Go..."
run sudo tar -C /usr/local -xzf "${GO_ARCHIVE}"

# ----------------------------------------------------------------------
# Verify Go
# ----------------------------------------------------------------------

if [[ "${DRY_RUN}" == "false" ]]; then
    [[ -x /usr/local/go/bin/go ]] || fatal "Go installation verification failed"

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
    echo 'export PATH=/usr/local/go/bin:$PATH'
fi

# ----------------------------------------------------------------------
# Backup Retention
# ----------------------------------------------------------------------

if [[ "${DRY_RUN}" == "false" ]]; then
    shopt -s nullglob
    backups=(/usr/local/go.backup.*)

    if [[ ${#backups[@]} -gt 3 ]]; then
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

fi # End of if [[ "${TOOLS_ONLY}" != "true" ]]

# ----------------------------------------------------------------------
# Install Tooling
# ----------------------------------------------------------------------

export PATH="/usr/local/go/bin:${PATH}"
export GOTOOLCHAIN=local

GOBIN="$(go env GOBIN 2>/dev/null || true)"
if [[ -z "${GOBIN}" ]]; then
    GOBIN="$(go env GOPATH 2>/dev/null || true)/bin"
fi
if [[ -n "${GOBIN}" ]]; then
    mkdir -p "${GOBIN}"
    export PATH="${GOBIN}:${PATH}"
fi

if [[ "${DRY_RUN}" == "false" ]]; then
    log "Installing govulncheck..."
    go install "golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION}"

    log "Installing staticcheck..."
    go install "honnef.co/go/tools/cmd/staticcheck@${STATICCHECK_VERSION}"

    log "Installing gosec..."
    go install "github.com/securego/gosec/v2/cmd/gosec@${GOSEC_VERSION}"

    log "Installing betterleaks..."
    go install "github.com/betterleaks/betterleaks@${BETTERLEAKS_VERSION}"

    log "Installing esbuild..."
    go install "github.com/evanw/esbuild/cmd/esbuild@${ESBUILD_VERSION}"
else
    echo "[DRY-RUN] go install golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION}"
    echo "[DRY-RUN] go install honnef.co/go/tools/cmd/staticcheck@${STATICCHECK_VERSION}"
    echo "[DRY-RUN] go install github.com/securego/gosec/v2/cmd/gosec@${GOSEC_VERSION}"
    echo "[DRY-RUN] go install github.com/betterleaks/betterleaks@${BETTERLEAKS_VERSION}"
    echo "[DRY-RUN] go install github.com/evanw/esbuild/cmd/esbuild@${ESBUILD_VERSION}"
fi

# ----------------------------------------------------------------------
# Verify Tooling
# ----------------------------------------------------------------------

if [[ "${DRY_RUN}" == "false" ]]; then
    command -v govulncheck >/dev/null 2>&1 || fatal "govulncheck installation failed"
    command -v staticcheck >/dev/null 2>&1 || fatal "staticcheck installation failed"
    command -v gosec >/dev/null 2>&1 || fatal "gosec installation failed"
    command -v betterleaks >/dev/null 2>&1 || fatal "betterleaks installation failed"
    command -v esbuild >/dev/null 2>&1 || fatal "esbuild installation failed"
fi

# ----------------------------------------------------------------------
# Frontend Dependencies (Node.js & npm)
# ----------------------------------------------------------------------

if [[ "${INSTALL_FRONTEND}" == "true" || "${TOOLS_ONLY}" == "true" ]]; then
    echo
    log "Frontend Dependencies Maintenance"
    if command -v npm >/dev/null 2>&1; then
        if [[ "${DRY_RUN}" == "true" ]]; then
            echo "[DRY-RUN] npm --prefix ${REPO_ROOT}/dashboard/frontend install"
        else
            log "Running npm install in dashboard/frontend..."
            npm --prefix "${REPO_ROOT}/dashboard/frontend" install
            success "Frontend dependencies installed"
        fi
    else
        warn "npm not found; skipping frontend dependency installation."
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
echo "gosec:"
echo "  ${GOSEC_VERSION}"
echo
echo "betterleaks:"
echo "  ${BETTERLEAKS_VERSION}"
echo
echo "esbuild:"
echo "  ${ESBUILD_VERSION}"
echo

show_status