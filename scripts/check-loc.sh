#!/usr/bin/env bash
# scripts/check-loc.sh - Audit codebase for files exceeding LOC thresholds

THRESHOLD="${1:-400}"

echo "=== Go Files Exceeding ${THRESHOLD} LOC ==="
find internal/ -name "*.go" -not -name "*_test.go" -exec wc -l {} + | \
  awk -v max="${THRESHOLD}" '$1 >= max && $2 != "total" { printf "%5d lines  %s\n", $1, $2 }' | \
  sort -nr