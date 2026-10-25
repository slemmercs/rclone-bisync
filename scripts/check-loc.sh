#!/usr/bin/env bash
# scripts/check-loc.sh - Audit codebase for files exceeding LOC thresholds across Go, JS, and HTML

THRESHOLD="${1:-400}"

audit_loc() {
    local label="$1"
    local name_pattern="$2"
    local exclude_pattern="${3:-}"

    echo "=== ${label}: Files Exceeding ${THRESHOLD} LOC ==="

    local find_opts=("-type" "f" "-name" "${name_pattern}")
    if [[ -n "${exclude_pattern}" ]]; then
        find_opts+=("-not" "-name" "${exclude_pattern}")
    fi

    # Exclude hidden dirs, node_modules, and site build/docs outputs
    local results
    results="$(find . -not -path "*/.*" -not -path "*/node_modules/*" -not -path "*/site/*" -not -path "*/_site/*" "${find_opts[@]}" -exec wc -l {} + 2>/dev/null | \
      awk -v max="${THRESHOLD}" '$1 >= max && $2 != "total" { printf "%5d lines  %s\n", $1, $2 }' | \
      sort -nr)"

    if [[ -n "${results}" ]]; then
        echo "${results}"
    else
        echo "  (No files exceeding threshold)"
    fi
    echo
}

audit_loc "GO" "*.go" "*_test.go"
audit_loc "JAVASCRIPT" "*.js" "*.min.js"
audit_loc "HTML" "*.html"