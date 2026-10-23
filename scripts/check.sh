#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

BUILD_DIR="${REPO_ROOT}/.build"

# ----------------------------------------------------------------------
# Colors
# ----------------------------------------------------------------------

if [[ -t 1 && -z "${NO_COLOR:-}" ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

# ----------------------------------------------------------------------
# Configuration
# ----------------------------------------------------------------------

MIN_COVERAGE="${MIN_COVERAGE:-80.0}"
BUILD_TARGET="${BUILD_TARGET:-.}"
COVERAGE_FILE="${COVERAGE_FILE:-coverage.out}"

# ----------------------------------------------------------------------
# Cleanup
# ----------------------------------------------------------------------

cleanup() {
    rm -rf "${BUILD_DIR}"
}

trap cleanup EXIT

# ----------------------------------------------------------------------
# Logging
# ----------------------------------------------------------------------

CURRENT_STEP="unknown"

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
    echo -e "${RED}[FAIL] $1${NC}" >&2
    echo "::error::$1"
    exit 1
}

on_error() {
    fail "Quality gate failed during: ${CURRENT_STEP}"
}

trap on_error ERR

require_tool() {
    local tool="$1"

    command -v "$tool" >/dev/null 2>&1 \
        || fail "Required tool not found: ${tool}"
}

# ----------------------------------------------------------------------
# Header
# ----------------------------------------------------------------------

echo "===================================================="
echo "              Repository Quality Gate"
echo "===================================================="

cd "${REPO_ROOT}"

# ----------------------------------------------------------------------
# 1. Tool Validation
# ----------------------------------------------------------------------

CURRENT_STEP="Tool Validation"

info "[1/8] Validating required tooling..."

for tool in bash git go gofmt; do
    require_tool "${tool}"
done

success "Required tooling available."
echo

# ----------------------------------------------------------------------
# Platform Information
# ----------------------------------------------------------------------

PLATFORM="$(go env GOOS)/$(go env GOARCH)"

info "Runner Platform: ${PLATFORM}"
echo

# ----------------------------------------------------------------------
# 2. Module Verification
# ----------------------------------------------------------------------

CURRENT_STEP="Module Verification"

info "[2/8] Verifying Go modules..."

go mod verify
go mod tidy

git diff --exit-code go.mod go.sum \
    || fail "go.mod/go.sum not tidy"

success "Go modules verified."
echo

# ----------------------------------------------------------------------
# 3. Shell Validation
# ----------------------------------------------------------------------

CURRENT_STEP="Shell Validation"

info "[3/8] Validating shell scripts..."

while IFS= read -r script; do
    bash -n "${script}"
    echo "  ✓ ${script}"
done < <(git ls-files '*.sh')

success "Shell validation passed."
echo

# ----------------------------------------------------------------------
# 4. Go Formatting
# ----------------------------------------------------------------------

CURRENT_STEP="Go Formatting"

info "[4/8] Checking Go formatting..."

GO_FILES="$(git ls-files '*.go')"

if [[ -n "${GO_FILES}" ]]; then
    UNFORMATTED="$(
        git ls-files -z '*.go' |
        xargs -0 gofmt -l -s
    )"

    if [[ -n "${UNFORMATTED}" ]]; then
        echo "${UNFORMATTED}"
        fail "Go formatting issues detected. Run: gofmt -w ."
    fi
fi

success "Go formatting check passed."
echo

# ----------------------------------------------------------------------
# 5. Static Analysis
# ----------------------------------------------------------------------

CURRENT_STEP="Go Vet"

info "[5/8] Running go vet..."

go vet ./...

success "Go vet passed."
echo

# ----------------------------------------------------------------------
# 6. Tests & Coverage
# ----------------------------------------------------------------------

CURRENT_STEP="Unit Tests & Coverage"

info "[6/8] Running unit tests with coverage..."

go test \
    -count=1 \
    -covermode=count \
    -coverpkg=./... \
    -coverprofile="${COVERAGE_FILE}" \
    ./...

success "Unit tests passed."
echo

info "Coverage breakdown:"

COVER_OUT="$(go tool cover -func="${COVERAGE_FILE}")"

echo "${COVER_OUT}"

COVERAGE="$(
    awk '/^total:/ {
        sub(/%/, "", $3)
        print $3
    }' <<< "${COVER_OUT}"
)"

echo
echo "Coverage : ${COVERAGE}%"
echo "Required : ${MIN_COVERAGE}%"

awk -v cov="${COVERAGE}" -v min="${MIN_COVERAGE}" '
BEGIN {
    exit (cov >= min ? 0 : 1)
}' || fail "Coverage ${COVERAGE}% is below required ${MIN_COVERAGE}%"

success "Coverage threshold met."
echo

# ----------------------------------------------------------------------
# 7. Native Build
# ----------------------------------------------------------------------

CURRENT_STEP="Native Build"

mkdir -p "${BUILD_DIR}"

info "[7/8] Verifying native build..."

go build \
    -trimpath \
    -o "${BUILD_DIR}/native-build" \
    "${BUILD_TARGET}"

success "Native build passed."
echo

# ----------------------------------------------------------------------
# 8. Cross Platform Builds
# ----------------------------------------------------------------------

CURRENT_STEP="Cross Builds"

info "[8/8] Verifying cross-platform builds..."

info "Building linux/amd64..."

CGO_ENABLED=0 \
GOOS=linux \
GOARCH=amd64 \
go build \
    -trimpath \
    -o "${BUILD_DIR}/app-linux-amd64" \
    "${BUILD_TARGET}"

success "linux/amd64 build passed."

info "Building linux/arm64..."

CGO_ENABLED=0 \
GOOS=linux \
GOARCH=arm64 \
go build \
    -trimpath \
    -o "${BUILD_DIR}/app-linux-arm64" \
    "${BUILD_TARGET}"

success "linux/arm64 build passed."
echo

# ----------------------------------------------------------------------
# CLI Validation
# ----------------------------------------------------------------------

CURRENT_STEP="CLI Validation"

info "Validating executable..."

"${BUILD_DIR}/native-build" --help >/dev/null

success "CLI validation passed."
echo

# ----------------------------------------------------------------------
# Summary
# ----------------------------------------------------------------------

echo "===================================================="
echo "Quality Gate Passed"
echo "Coverage: ${COVERAGE}%"
echo "Coverage Report: ${COVERAGE_FILE}"
echo "===================================================="