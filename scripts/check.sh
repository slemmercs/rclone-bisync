#!/usr/bin/env bash
#
# scripts/check.sh
#
# Repository Quality Gate
#
# Validates:
#   - Required tooling
#   - Runner architecture
#   - Shell script syntax
#   - Go formatting
#   - Go static analysis
#   - Unit tests
#   - Native build
#   - Cross-platform build verification
#   - Executable validation
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
BUILD_DIR="${REPO_ROOT}/.build"

trap 'rm -rf "${BUILD_DIR}"' EXIT

# ----------------------------------------------------------------------
# Colors
# ----------------------------------------------------------------------

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# ----------------------------------------------------------------------
# Logging helpers
# ----------------------------------------------------------------------

info() {
    echo -e "${BLUE}$1${NC}"
}

success() {
    echo -e "${GREEN}$1${NC}"
}

warning() {
    echo -e "${YELLOW}$1${NC}"
}

fail() {
    echo -e "${RED}$1${NC}"
    exit 1
}

require_tool() {
    local tool="$1"

    if ! command -v "$tool" >/dev/null 2>&1; then
        fail "[FAIL] Required tool not found: ${tool}"
    fi
}

# ----------------------------------------------------------------------
# Header
# ----------------------------------------------------------------------

echo -e "${BLUE}====================================================${NC}"
echo -e "${BLUE}              Repository Quality Gate              ${NC}"
echo -e "${BLUE}====================================================${NC}"

cd "${REPO_ROOT}"

# ----------------------------------------------------------------------
# 1. Tool Validation
# ----------------------------------------------------------------------

info "[1/7] Validating required tooling..."

for tool in bash git go gofmt; do
    require_tool "$tool"
done

success "Required tooling available."
echo

# ----------------------------------------------------------------------
# Architecture Detection
# ----------------------------------------------------------------------

ARCH="$(uname -m)"

case "${ARCH}" in
    x86_64)
        PLATFORM="linux/amd64"
        ;;
    aarch64|arm64)
        PLATFORM="linux/arm64"
        ;;
    *)
        PLATFORM="unknown"
        ;;
esac

info "Runner architecture: ${ARCH} (${PLATFORM})"

if [[ "${PLATFORM}" == "unknown" ]]; then
    warning "Unknown architecture detected. Continuing."
fi

echo

# ----------------------------------------------------------------------
# 2. Shell Script Validation
# ----------------------------------------------------------------------

info "[2/7] Validating shell scripts..."

FOUND_SCRIPT=false

while IFS= read -r script; do
    FOUND_SCRIPT=true
    bash -n "${script}"
    echo "  ✓ ${script}"
done < <(
    find . \
        -type f \
        -name "*.sh" \
        -not -path "./.git/*" \
        -not -path "./vendor/*"
)

if [[ "${FOUND_SCRIPT}" == "false" ]]; then
    warning "No shell scripts found."
fi

success "Shell script validation passed."
echo

# ----------------------------------------------------------------------
# 3. Go Formatting
# ----------------------------------------------------------------------

info "[3/7] Checking Go formatting..."

UNFORMATTED="$(gofmt -l .)"

if [[ -n "${UNFORMATTED}" ]]; then
    echo
    echo "${UNFORMATTED}"
    echo
    fail "[FAIL] Go formatting issues detected. Run: gofmt -w ."
fi

success "Go formatting check passed."
echo

# ----------------------------------------------------------------------
# 4. Go Static Analysis
# ----------------------------------------------------------------------

info "[4/7] Running go vet..."

go vet ./...

success "Go static analysis passed."
echo

# ----------------------------------------------------------------------
# 5. Unit Tests
# ----------------------------------------------------------------------

info "[5/7] Running unit tests..."

go test -count=1 ./...

success "All unit tests passed."
echo

# ----------------------------------------------------------------------
# 6. Build Verification
# ----------------------------------------------------------------------

mkdir -p "${BUILD_DIR}"

info "[6/7] Verifying native build..."

go build -v -o "${BUILD_DIR}/native-build" .

success "Native build passed."
echo

# ----------------------------------------------------------------------
# 7. Cross-Platform Build Verification
# ----------------------------------------------------------------------

info "[7/7] Verifying cross-platform builds..."

GOOS=linux GOARCH=amd64 go build \
    -o "${BUILD_DIR}/rclone-bisync-linux-amd64" .

GOOS=linux GOARCH=arm64 go build \
    -o "${BUILD_DIR}/rclone-bisync-linux-arm64" .

success "Cross-platform build verification passed."
echo

# ----------------------------------------------------------------------
# Executable Validation
# ----------------------------------------------------------------------

info "Validating executable CLI..."

"${BUILD_DIR}/native-build" --help >/dev/null

success "Executable validation passed."
echo

# ----------------------------------------------------------------------
# Summary
# ----------------------------------------------------------------------

echo -e "${GREEN}====================================================${NC}"
echo -e "${GREEN}      Quality Gate Passed - Ready to Merge         ${NC}"
echo -e "${GREEN}====================================================${NC}"
