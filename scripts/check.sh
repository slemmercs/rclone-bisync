#!/usr/bin/env bash
# scripts/check.sh - Fast local pre-flight quality and safety gate (Go Single Binary)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}====================================================${NC}"
echo -e "${BLUE}    Rclone Bisync Pre-Flight Quality Gate          ${NC}"
echo -e "${BLUE}====================================================${NC}"

# 1. Check Shell Script Syntax
echo -e "${BLUE}[1/4] Checking shell script syntax (bash -n)...${NC}"
SHELL_SCRIPTS=$(find "${REPO_ROOT}" -maxdepth 2 -name "*.sh" -not -path "*/.*")
for script in $SHELL_SCRIPTS; do
    bash -n "$script"
    echo -e "  ${GREEN}✓${NC} $(basename "$script")"
done
echo -e "${GREEN}Shell script syntax OK.${NC}\n"

# 2. Check for Tracked Secrets & Leaks
echo -e "${BLUE}[2/4] Verifying secret isolation & git hygiene...${NC}"
if git -C "${REPO_ROOT}" ls-files --error-unmatch rc-secrets.env 2>/dev/null; then
    echo -e "${RED}[FAIL] rc-secrets.env is tracked by git! Untrack it immediately.${NC}"
    exit 1
fi
echo -e "  ${GREEN}✓${NC} rc-secrets.env properly untracked/gitignored"

# Scan tracked files for raw password assignments (excluding docs, examples, and test fixtures)
SUSPECT_LEAKS=$(git -C "${REPO_ROOT}" grep -n -I -E "(RC_PASS|password)\s*=\s*['\"][^'\"]{6,}['\"]" -- ':!*.example' ':!*.md' || true)
if [[ -n "$SUSPECT_LEAKS" ]]; then
    echo -e "${RED}[FAIL] Potential hardcoded secret found in tracked files:${NC}"
    echo "$SUSPECT_LEAKS"
    exit 1
fi
echo -e "  ${GREEN}✓${NC} No hardcoded passwords detected in tracked files"
echo -e "${GREEN}Secret isolation OK.${NC}\n"

# 3. Execute Go Static Analysis (go vet)
echo -e "${BLUE}[3/4] Running Go static analysis (go vet ./...)...${NC}"
if command -v go >/dev/null 2>&1; then
    cd "${REPO_ROOT}"
    if ! go vet ./...; then
        echo -e "${RED}[FAIL] go vet static analysis failed!${NC}"
        exit 1
    fi
    echo -e "${GREEN}Go static analysis clean.${NC}\n"
else
    echo -e "${RED}[FAIL] Go toolchain not found in PATH.${NC}"
    exit 1
fi

# 4. Execute Go Automated Unit Test Suite
echo -e "${BLUE}[4/4] Running Go automated unit test suite (go test -count=1 ./...)...${NC}"
cd "${REPO_ROOT}"
if ! go test -count=1 ./...; then
    echo -e "${RED}[FAIL] Go unit tests failed!${NC}"
    exit 1
fi
echo -e "${GREEN}All Go unit tests passed.${NC}\n"

echo -e "${GREEN}====================================================${NC}"
echo -e "${GREEN}    Pre-Flight Passed: Ready to commit & push!     ${NC}"
echo -e "${GREEN}====================================================${NC}"
