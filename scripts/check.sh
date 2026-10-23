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
#   - Code coverage (minimum threshold)
#   - Native build
#   - Cross-platform build verification
#   - Executable validation
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
BUILD_DIR="${REPO_ROOT}/.build"

trap 'rm -rf "${BUILD_DIR}" coverage.out' EXIT

# ----------------------------------------------------------------------
# Colors
# ----------------------------------------------------------------------

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# ----------------------------------------------------------------------
# Logging Helpers
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

info "[1/8] Validating required tooling..."

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

info "[2/8] Validating shell scripts..."

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

info "[3/8] Checking Go formatting..."

UNFORMATTED="$(
    find . \
        -type f \
        -name "*.go" \
        -not -path "./vendor/*" \
        -exec gofmt -l {} +
)"

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

info "[4/8] Running go vet..."

go vet ./...

success "Go static analysis passed."
echo

# ----------------------------------------------------------------------
# 5. Unit Tests & Coverage
# ----------------------------------------------------------------------

info "[5/8] Running unit tests with coverage..."

go test \
    -count=1 \
    -coverprofile=coverage.out \
    ./...

success "All unit tests passed."
echo

info "Coverage breakdown:"

go tool cover -func=coverage.out

COVERAGE=$(
    go tool cover -func=coverage.out |
    awk '/^total:/ {print substr($3,1,length($3)-1)}'
)

MIN_COVERAGE=80.0

echo
echo "Total coverage: ${COVERAGE}%"
echo "Required coverage: ${MIN_COVERAGE}%"

awk -v cov="$COVERAGE" -v min="$MIN_COVERAGE" '
BEGIN {
    exit (cov >= min ? 0 : 1)
}
' || fail "[FAIL] Coverage ${COVERAGE}% is below required ${MIN_COVERAGE}%"

success "Coverage threshold met (${COVERAGE}%)."
echo

# ----------------------------------------------------------------------
# 6. Native Build Verification
# ----------------------------------------------------------------------

mkdir -p "${BUILD_DIR}"

info "[6/8] Verifying native build..."

go build -v -o "${BUILD_DIR}/native-build" .

success "Native build passed."
echo

# ----------------------------------------------------------------------
# 7. Cross-Platform Build Verification
# ----------------------------------------------------------------------

info "[7/8] Verifying cross-platform builds..."

info "Building linux/amd64..."

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
go build -o "${BUILD_DIR}/rclone-bisync-linux-amd64" .

success "linux/amd64 build passed."

info "Building linux/arm64..."

CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
go build -o "${BUILD_DIR}/rclone-bisync-linux-arm64" .

success "linux/arm64 build passed."

success "Cross-platform build verification passed."
echo

# ----------------------------------------------------------------------
# 8. Executable Validation
# ----------------------------------------------------------------------

info "[8/8] Validating executable CLI..."

"${BUILD_DIR}/native-build" --help >/dev/null

success "Executable validation passed."
echo

# ----------------------------------------------------------------------
# Summary
# ----------------------------------------------------------------------

echo -e "${GREEN}====================================================${NC}"
echo -e "${GREEN}      Quality Gate Passed - Ready to Merge         ${NC}"
echo -e "${GREEN}====================================================${NC}"
echo -e "${GREEN}Coverage: ${COVERAGE}%${NC}"
