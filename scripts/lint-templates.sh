#!/usr/bin/env bash
# ==============================================================================
# Alpine.js Template Boolean Attribute Linter
# ==============================================================================
# In Alpine.js v3, evaluating a dotted expression (e.g., :disabled="item._copying")
# that yields undefined causes Alpine's evaluator to return "" (empty string).
# For HTML boolean attributes (disabled, checked, readonly, hidden, required, selected),
# an empty string is treated as truthy by the DOM, inadvertently enabling the attribute.
#
# This linter scans all HTML templates and strictly forbids dotted property access
# inside boolean attribute bindings. All dynamic boolean conditions must use
# top-level component helper methods (e.g., :disabled="isItemCopying(item)").
# ==============================================================================

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET_DIR="${1:-${ROOT_DIR}/dashboard/frontend}"

violations=0
pattern=':(disabled|checked|readonly|hidden|required|selected)="[^"]*\.[^"]*"'
pattern_sq=":(disabled|checked|readonly|hidden|required|selected)='[^']*\.[^']*'"

echo "==> Linting HTML templates for Alpine.js boolean attribute binding safety..."

while IFS= read -r html_file; do
    if [[ ! -f "${html_file}" ]]; then
        continue
    fi

    # Check double-quoted attributes
    matches_dq=$(grep -n -E "${pattern}" "${html_file}" || true)
    # Check single-quoted attributes
    matches_sq=$(grep -n -E "${pattern_sq}" "${html_file}" || true)

    if [[ -n "${matches_dq}" || -n "${matches_sq}" ]]; then
        rel_path="${html_file#"${ROOT_DIR}/"}"
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
done < <(find "${TARGET_DIR}" -type f -name "*.html")

if [[ ${violations} -gt 0 ]]; then
    echo ""
    echo "FAILED: Found ${violations} template file(s) with unsafe dotted boolean attribute bindings."
    exit 1
fi

echo "✓ All HTML templates passed Alpine boolean attribute safety checks (0 violations)."
exit 0
