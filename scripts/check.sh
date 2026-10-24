#!/usr/bin/env bash

set -u
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

BUILD_DIR="${REPO_ROOT}/.build"
export PATH="${HOME}/go/bin:/usr/local/go/bin:/usr/local/bin:${PATH}"

# ----------------------------------------------------------------------
# Configuration & Stage Selection
# ----------------------------------------------------------------------

MIN_COVERAGE="${MIN_COVERAGE:-80.0}"
BUILD_TARGET="${BUILD_TARGET:-.}"
COVERAGE_FILE="${COVERAGE_FILE:-coverage.out}"
RUN_RACE="${RUN_RACE:-false}"
STAGE="${1:-all}"

should_run() {
    [[ "${STAGE}" == "all" || "${STAGE}" == "$1" ]]
}

# ----------------------------------------------------------------------
# Tracking & Summary System
# ----------------------------------------------------------------------

FAILED_CHECKS=()
PASSED_CHECKS=()

cleanup() { rm -rf "${BUILD_DIR}"; }
trap cleanup EXIT

info() { echo -e "\033[0;34m$1\033[0m"; }
success() { echo -e "\033[0;32m$1\033[0m"; }
warning() { echo -e "\033[1;33m$1\033[0m"; }
fail_step() { echo -e "\033[0;31m[FAIL] $1\033[0m"; }

execute_step() {
    local step_name="$1"
    shift
    
    info "==> ${step_name}..."
    if "$@"; then
        success "✓ ${step_name} passed."
        PASSED_CHECKS+=("${step_name}")
    else
        fail_step "✗ ${step_name} failed."
        FAILED_CHECKS+=("${step_name}")
    fi
    echo
}

cd "${REPO_ROOT}"

# ----------------------------------------------------------------------
# Stage 1: Validation & Formatting
# ----------------------------------------------------------------------

check_fmt_and_scripts() {
    local failed=0
    while IFS= read -r script; do
        if [[ -f "${script}" ]]; then
            bash -n "${script}" || failed=1
        fi
    done < <(git ls-files '*.sh')

    UNFORMATTED="$(git ls-files -z '*.go' | xargs -0 gofmt -l -s 2>/dev/null || true)"
    if [[ -n "${UNFORMATTED}" ]]; then
        echo "Unformatted Go files:"
        echo "${UNFORMATTED}"
        failed=1
    fi
    return ${failed}
}

if should_run "fmt"; then
    execute_step "Go Formatting & Shell Validation" check_fmt_and_scripts
fi

# ----------------------------------------------------------------------
# Stage 2: Static Analysis & Security (Go, JS, Secrets & HTML Tidy)
# ----------------------------------------------------------------------

check_lint_and_security() {
    local failed=0
    
    info "--> Running go vet..."
    go vet ./... || failed=1

    if command -v staticcheck >/dev/null 2>&1; then
        info "--> Running staticcheck..."
        staticcheck ./... || failed=1
    fi

    if command -v govulncheck >/dev/null 2>&1; then
        info "--> Running govulncheck..."
        govulncheck ./... || failed=1
    fi

    if command -v betterleaks >/dev/null 2>&1; then
        info "--> Running betterleaks..."
        if [[ -f ".betterleaks.toml" ]]; then
            betterleaks dir --config .betterleaks.toml . || failed=1
        else
            betterleaks dir . || failed=1
        fi
    fi

    if command -v esbuild >/dev/null 2>&1; then
        info "--> Running esbuild JS syntax check..."
        JS_FILES="$(git ls-files '*.js')"
        if [[ -n "${JS_FILES}" ]]; then
            TMP_OUT="$(mktemp -d)"
            echo "${JS_FILES}" | xargs esbuild --outdir="${TMP_OUT}" --log-level=error --log-limit=0 || failed=1
            rm -rf "${TMP_OUT}"
        fi
    fi

    if command -v tidy >/dev/null 2>&1; then
        info "--> Running HTML Tidy structural check..."
        HTML_FILES="$(git ls-files '*.html')"
        if [[ -n "${HTML_FILES}" ]]; then
            while IFS= read -r html_file; do
                if [[ -f "${html_file}" ]]; then
                    tidy -e -q "${html_file}" 2>/dev/null
                    RC=$?
                    # Exit Code 0 = Clean, 1 = Warnings (Alpine/SVG attributes), >= 2 = Structural Failure
                    if [[ $RC -ge 2 ]]; then
                        echo "❌ Severe HTML structural error in: ${html_file}"
                        tidy -e -q "${html_file}" || true
                        failed=1
                    fi
                fi
            done <<< "${HTML_FILES}"
        fi
    else
        warning "[WARN] tidy binary not found in PATH. Install via: sudo apt install tidy"
    fi

    return ${failed}
}

if should_run "lint"; then
    execute_step "Static Analysis & Security" check_lint_and_security
fi

# ----------------------------------------------------------------------
# Stage 3: Unit Tests & Coverage
# ----------------------------------------------------------------------

check_tests_and_coverage() {
    local failed=0
    TEST_ARGS=("-count=1" "-covermode=count" "-coverpkg=./..." "-coverprofile=${COVERAGE_FILE}")
    if [[ "${RUN_RACE}" == "true" ]]; then TEST_ARGS+=("-race" "-shuffle=on"); fi

    go test "${TEST_ARGS[@]}" ./... || return 1

    COVER_OUT="$(go tool cover -func="${COVERAGE_FILE}")"
    COVERAGE="$(awk '/^total:/ { sub(/%/, "", $3); print $3 }' <<< "${COVER_OUT}")"
    echo "Coverage: ${COVERAGE}% (Required: ${MIN_COVERAGE}%)"

    awk -v cov="${COVERAGE}" -v min="${MIN_COVERAGE}" 'BEGIN { exit (cov >= min ? 0 : 1) }' || failed=1
    return ${failed}
}

if should_run "test"; then
    execute_step "Unit Tests & Coverage" check_tests_and_coverage
fi

# ----------------------------------------------------------------------
# Stage 4: Build Verification
# ----------------------------------------------------------------------

check_builds() {
    local failed=0
    mkdir -p "${BUILD_DIR}"
    go build -trimpath -o "${BUILD_DIR}/native-build" "${BUILD_TARGET}" || failed=1
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o "${BUILD_DIR}/app-linux-amd64" "${BUILD_TARGET}" || failed=1
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o "${BUILD_DIR}/app-linux-arm64" "${BUILD_TARGET}" || failed=1
    "${BUILD_DIR}/native-build" --help >/dev/null 2>&1 || failed=1
    return ${failed}
}

if should_run "build"; then
    execute_step "Build Verification" check_builds
fi

# ----------------------------------------------------------------------
# Final Summary Output
# ----------------------------------------------------------------------

echo "===================================================="
echo "               Quality Gate Summary"
echo "===================================================="

if (( ${#PASSED_CHECKS[@]} > 0 )); then
    success "Passed Checks (${#PASSED_CHECKS[@]}):"
    for check in "${PASSED_CHECKS[@]}"; do
        echo -e "  \033[0;32m✓\033[0m ${check}"
    done
fi

if (( ${#FAILED_CHECKS[@]} > 0 )); then
    echo
    fail_step "Failed Checks (${#FAILED_CHECKS[@]}):"
    for check in "${FAILED_CHECKS[@]}"; do
        echo -e "  \033[0;31m✗\033[0m ${check}"
    done
    echo "===================================================="
    exit 1
fi

echo "===================================================="
success "All quality gate checks passed successfully!"