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
LOC_THRESHOLD="${LOC_THRESHOLD:-400}"

VERBOSE=0
STAGE="all"
STAGE_SET=0
EXTRA_ARGS=()

show_help() {
    cat <<'EOF'
Usage: ./scripts/check.sh [STAGE] [OPTIONS]

Stages:
  all         Run all quality gate stages (default)
  fmt         Check Go formatting and validate shell scripts
  lint        Run Go static analysis, secret scanning, JS & HTML linting
  templates   Lint HTML templates for Alpine.js boolean attribute safety
  loc [LIMIT] Audit codebase for files exceeding LOC threshold (default: 400)
  test        Execute unit tests and verify code coverage
  build       Verify native and cross-platform compilation

Options:
  -v, --verbose   Stream live check output instead of capturing logs
  -h, --help      Display this help message

Environment Variables:
  LOC_THRESHOLD   Maximum lines of code per file (default: 400)
  MIN_COVERAGE    Minimum required test coverage percentage (default: 80.0)
  RUN_RACE        Enable race detection during tests (default: false)
  BUILD_TARGET    Target package for builds (default: .)
EOF
}

for arg in "$@"; do
    case "${arg}" in
        -v|--verbose)
            VERBOSE=1
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        [0-9]*)
            LOC_THRESHOLD="${arg}"
            ;;
        *)
            if [[ ${STAGE_SET} -eq 0 ]]; then
                STAGE="${arg}"
                STAGE_SET=1
            else
                EXTRA_ARGS+=("${arg}")
            fi
            ;;
    esac
done

validate_stage() {
    case "${STAGE}" in
        all|fmt|format|lint|static|security|templates|template|lint-templates|template-lint|loc|check-loc|lines|test|tests|unit|build|builds)
            return 0
            ;;
        *)
            echo "Unknown stage: '${STAGE}'"
            echo "Run './scripts/check.sh --help' for available stages."
            exit 1
            ;;
    esac
}
validate_stage

should_run() {
    local target="$1"
    if [[ "${STAGE}" == "all" ]]; then
        return 0
    fi
    case "${target}" in
        fmt)
            [[ "${STAGE}" == "fmt" || "${STAGE}" == "format" ]]
            ;;
        lint)
            [[ "${STAGE}" == "lint" || "${STAGE}" == "static" || "${STAGE}" == "security" ]]
            ;;
        templates)
            [[ "${STAGE}" == "templates" || "${STAGE}" == "template" || "${STAGE}" == "lint-templates" || "${STAGE}" == "template-lint" ]]
            ;;
        loc)
            [[ "${STAGE}" == "loc" || "${STAGE}" == "check-loc" || "${STAGE}" == "lines" ]]
            ;;
        test)
            [[ "${STAGE}" == "test" || "${STAGE}" == "tests" || "${STAGE}" == "unit" ]]
            ;;
        build)
            [[ "${STAGE}" == "build" || "${STAGE}" == "builds" ]]
            ;;
        *)
            [[ "${STAGE}" == "${target}" ]]
            ;;
    esac
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

    return ${failed}
}

if should_run "lint"; then
    execute_step "Static Analysis & Security" check_lint_and_security
fi

# ----------------------------------------------------------------------
# Stage 3: Alpine Template Boolean Attribute Safety
# ----------------------------------------------------------------------

check_alpine_templates() {
    local target_dir="${1:-${REPO_ROOT}/dashboard/frontend}"
    local violations=0
    local pattern=':(disabled|checked|readonly|hidden|required|selected)="[^"]*\.[^"]*"'
    local pattern_sq=":(disabled|checked|readonly|hidden|required|selected)='[^']*\.[^']*'"

    log_verbose "--> Linting HTML templates for Alpine.js boolean attribute binding safety..."

    while IFS= read -r html_file; do
        if [[ ! -f "${html_file}" ]]; then
            continue
        fi

        local matches_dq matches_sq
        matches_dq=$(grep -n -E "${pattern}" "${html_file}" || true)
        matches_sq=$(grep -n -E "${pattern_sq}" "${html_file}" || true)

        if [[ -n "${matches_dq}" || -n "${matches_sq}" ]]; then
            local rel_path="${html_file#"${REPO_ROOT}/"}"
            echo ""
            echo "❌ Template Linter Violation in: ${rel_path}"
            if [[ -n "${matches_dq}" ]]; then
                echo "${matches_dq}" | while IFS= read -r line; do
                    echo "   line ${line}"
                done
            fi
            if [[ -n "${matches_sq}" ]]; then
                echo "${matches_sq}" | while IFS= read -r line; do
                    echo "   line ${line}"
                done
            fi
            echo "   👉 Alpine v3 quirk: Dotted expression in boolean attribute coerces undefined to '' (truthy)."
            echo "   👉 Remediation: Replace dotted path with a component helper method (e.g., :disabled=\"isItemCopying(item)\")."
            violations=$((violations + 1))
        fi
    done < <(find "${target_dir}" -type f -name "*.html")

    if [[ ${violations} -gt 0 ]]; then
        echo ""
        echo "FAILED: Found ${violations} template file(s) with unsafe dotted boolean attribute bindings."
        return 1
    fi

    echo "✓ All HTML templates passed Alpine boolean attribute safety checks (0 violations)."
    return 0
}

if should_run "templates"; then
    execute_step "Alpine Template Safety" check_alpine_templates
fi

# ----------------------------------------------------------------------
# Stage 4: Lines of Code (LOC) Audit
# ----------------------------------------------------------------------

check_loc() {
    local threshold="${LOC_THRESHOLD}"
    local total_oversized=0

    audit_loc_type() {
        local label="$1"
        local name_pattern="$2"
        local exclude_pattern="${3:-}"

        echo "=== ${label}: Files Exceeding ${threshold} LOC ==="

        local find_opts=("-type" "f" "-name" "${name_pattern}")
        if [[ -n "${exclude_pattern}" ]]; then
            find_opts+=("-not" "-name" "${exclude_pattern}")
        fi

        local results
        results="$(find . -not -path "*/.*" -not -path "*/node_modules/*" -not -path "*/site/*" -not -path "*/_site/*" "${find_opts[@]}" -exec wc -l {} + 2>/dev/null | \
          awk -v max="${threshold}" '$1 >= max && $2 != "total" { printf "%5d lines  %s\n", $1, $2 }' | \
          sort -nr)"

        if [[ -n "${results}" ]]; then
            echo "${results}"
            local count
            count="$(echo "${results}" | wc -l)"
            total_oversized=$((total_oversized + count))
        else
            echo "  (No files exceeding threshold)"
        fi
        echo
    }

    audit_loc_type "GO" "*.go" "*_test.go"
    audit_loc_type "JAVASCRIPT" "*.js" "*.min.js"
    audit_loc_type "HTML" "*.html"

    if [[ ${total_oversized} -gt 0 ]]; then
        echo "FAILED: Found ${total_oversized} file(s) exceeding ${threshold} lines of code."
        return 1
    fi

    echo "✓ All audited source files are within ${threshold} lines of code."
    return 0
}

if should_run "loc"; then
    execute_step "Lines of Code Audit" check_loc
fi

# ----------------------------------------------------------------------
# Stage 5: Unit Tests & Coverage
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
# Stage 6: Build Verification
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