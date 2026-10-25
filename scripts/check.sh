#!/usr/bin/env bash

set -u
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

BUILD_DIR="${REPO_ROOT}/.build"
LOG_DIR=""
export PATH="${HOME}/go/bin:/usr/local/go/bin:/usr/local/bin:${PATH}"

# ----------------------------------------------------------------------
# Configuration & Flag Parsing
# ----------------------------------------------------------------------

MIN_COVERAGE="${MIN_COVERAGE:-80.0}"
BUILD_TARGET="${BUILD_TARGET:-.}"
COVERAGE_FILE="${COVERAGE_FILE:-coverage.out}"
RUN_RACE="${RUN_RACE:-false}"

VERBOSE=0
STAGE="all"

for arg in "$@"; do
    case "${arg}" in
        -v|--verbose)
            VERBOSE=1
            ;;
        *)
            STAGE="${arg}"
            ;;
    esac
done

should_run() {
    [[ "${STAGE}" == "all" || "${STAGE}" == "$1" ]]
}

cleanup() {
    rm -rf "${BUILD_DIR}"
    if [[ -n "${LOG_DIR}" && -d "${LOG_DIR}" ]]; then
        rm -rf "${LOG_DIR}"
    fi
}
trap cleanup EXIT

LOG_DIR="$(mktemp -d)"
cd "${REPO_ROOT}"

# ----------------------------------------------------------------------
# Execution Engine
# ----------------------------------------------------------------------

FAILED_CHECKS=()
PASSED_CHECKS=()
FAILED_LOGS=()

log_verbose() {
    if [[ ${VERBOSE} -eq 1 ]]; then
        echo "$@"
    fi
}

execute_step() {
    local step_name="$1"
    shift

    local step_slug
    step_slug="$(echo "${step_name}" | tr '[:upper:] ' '[:lower:]_')"
    local step_log="${LOG_DIR}/${step_slug}.log"

    log_verbose "==> Running: ${step_name}..."

    if [[ ${VERBOSE} -eq 1 ]]; then
        if "$@"; then
            PASSED_CHECKS+=("${step_name}")
            log_verbose "✓ ${step_name} passed."
        else
            FAILED_CHECKS+=("${step_name}")
            log_verbose "✗ ${step_name} failed."
        fi
    else
        if "$@" >"${step_log}" 2>&1; then
            PASSED_CHECKS+=("${step_name}")
        else
            FAILED_CHECKS+=("${step_name}")
            FAILED_LOGS+=("${step_name}:${step_log}")
        fi
    fi
}

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
# Stage 2: Static Analysis & Security
# ----------------------------------------------------------------------

check_lint_and_security() {
    local failed=0

    log_verbose "--> Running go vet..."
    go vet ./... || failed=1

    if command -v staticcheck >/dev/null 2>&1; then
        log_verbose "--> Running staticcheck..."
        staticcheck ./... || failed=1
    fi

    if command -v govulncheck >/dev/null 2>&1; then
        log_verbose "--> Running govulncheck..."
        govulncheck ./... || failed=1
    fi

    if command -v betterleaks >/dev/null 2>&1; then
        log_verbose "--> Running betterleaks..."
        if [[ -f ".betterleaks.toml" ]]; then
            betterleaks dir --config .betterleaks.toml . || failed=1
        elif [[ -f ".gitleaks.toml" ]]; then
            betterleaks dir --config .gitleaks.toml . || failed=1
        else
            betterleaks dir . || failed=1
        fi
    fi

    if command -v esbuild >/dev/null 2>&1; then
        log_verbose "--> Running esbuild JS syntax check..."
        JS_FILES="$(git ls-files '*.js')"
        if [[ -n "${JS_FILES}" ]]; then
            TMP_OUT="$(mktemp -d)"
            echo "${JS_FILES}" | xargs esbuild --outdir="${TMP_OUT}" --log-level=error --log-limit=0 || failed=1
            rm -rf "${TMP_OUT}"
        fi
    fi

    if command -v tidy >/dev/null 2>&1; then
        log_verbose "--> Running HTML Tidy structural check..."
        HTML_FILES="$(git ls-files '*.html')"
        if [[ -n "${HTML_FILES}" ]]; then
            while IFS= read -r html_file; do
                if [[ -f "${html_file}" ]]; then
                    tidy -e -q "${html_file}" 2>/dev/null
                    RC=$?
                    if [[ $RC -ge 2 ]]; then
                        echo "Severe HTML structural error in: ${html_file}"
                        tidy -e -q "${html_file}" || true
                        failed=1
                    fi
                fi
            done <<< "${HTML_FILES}"
        fi
    else
        log_verbose "[WARN] tidy binary not found in PATH."
    fi

    if [[ -x "${REPO_ROOT}/scripts/lint-templates.sh" ]]; then
        log_verbose "--> Running Alpine template boolean attribute safety check..."
        "${REPO_ROOT}/scripts/lint-templates.sh" || failed=1
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
    TEST_ARGS=("-count=1" "-covermode=count" "-coverpkg=./internal/..." "-coverprofile=${COVERAGE_FILE}")
    if [[ "${RUN_RACE}" == "true" ]]; then TEST_ARGS+=("-race" "-shuffle=on"); fi

    go test "${TEST_ARGS[@]}" ./... || return 1

    COVER_OUT="$(go tool cover -func="${COVERAGE_FILE}")"
    COVERAGE="$(awk '/^total:/ { sub(/%/, "", $3); print $3 }' <<< "${COVER_OUT}")"
    log_verbose "Coverage: ${COVERAGE}% (Required: ${MIN_COVERAGE}%)"

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
# Summary
# ----------------------------------------------------------------------

TOTAL_CHECKS=$(( ${#PASSED_CHECKS[@]} + ${#FAILED_CHECKS[@]} ))

if (( ${#FAILED_CHECKS[@]} == 0 )); then
    echo "OK: All quality gate checks passed (${#PASSED_CHECKS[@]}/${TOTAL_CHECKS})."
    exit 0
fi

echo "FAIL: Quality gate failed (${#FAILED_CHECKS[@]}/${TOTAL_CHECKS} failed)."
echo "Failed step(s): ${FAILED_CHECKS[*]}"

if [[ ${VERBOSE} -eq 0 && ${#FAILED_LOGS[@]} -gt 0 ]]; then
    echo
    for entry in "${FAILED_LOGS[@]}"; do
        step_name="${entry%%:*}"
        log_file="${entry#*:}"
        echo "--- Error log for '${step_name}' ---"
        cat "${log_file}"
        echo "------------------------------------"
    done
fi

exit 1