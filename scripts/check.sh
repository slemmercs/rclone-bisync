#!/usr/bin/env bash
# scripts/check.sh - Fast local pre-flight quality and safety gate
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

# 1. Resolve Python environment
VENV_PATH="${VENV_PATH:-/home/csaba/rclone-bisync/.venv}"
if [[ ! -d "$VENV_PATH" && -d "${REPO_ROOT}/.venv" ]]; then
    VENV_PATH="${REPO_ROOT}/.venv"
fi

if [[ -f "${VENV_PATH}/bin/python3" ]]; then
    PYTHON_BIN="${VENV_PATH}/bin/python3"
    PYTEST_BIN="${VENV_PATH}/bin/pytest"
else
    PYTHON_BIN="$(command -v python3 || true)"
    PYTEST_BIN="$(command -v pytest || true)"
fi

# 5. Run Go Unit Test Suite
if command -v go >/dev/null 2>&1; then
    echo -e "${BLUE}[5/5] Running Go unit test suite (go test ./...)...${NC}"
    if ! go test ./...; then
        echo -e "${RED}[FAIL] Go unit tests failed!${NC}"
        exit 1
    fi
    echo -e "  ${GREEN}✓${NC} Go test suite clean."
fi


if [[ -z "$PYTHON_BIN" || ! -x "$PYTHON_BIN" ]]; then
    echo -e "${RED}[FAIL] Python 3 interpreter not found.${NC}"
    exit 1
fi

echo -e "Using Python: ${YELLOW}${PYTHON_BIN}${NC}"
echo -e "Using Pytest: ${YELLOW}${PYTEST_BIN}${NC}"
echo ""

# 2. Check Shell Script Syntax
echo -e "${BLUE}[1/4] Checking shell script syntax (bash -n)...${NC}"
SHELL_SCRIPTS=$(find "${REPO_ROOT}" -maxdepth 2 -name "*.sh" -not -path "*/.*")
for script in $SHELL_SCRIPTS; do
    bash -n "$script"
    echo -e "  ${GREEN}✓${NC} $(basename "$script")"
done
echo -e "${GREEN}Shell script syntax OK.${NC}\n"

# 3. Check Python Syntax & Compilation
echo -e "${BLUE}[2/4] Compiling Python source files (compileall)...${NC}"
"$PYTHON_BIN" -m compileall -q "${REPO_ROOT}/rclone_bisync" "${REPO_ROOT}/dashboard" "${REPO_ROOT}/tests"
echo -e "${GREEN}Python compilation clean.${NC}\n"

# 4. Check for Tracked Secrets & Leaks
echo -e "${BLUE}[3/4] Verifying secret isolation & git hygiene...${NC}"
if git -C "${REPO_ROOT}" ls-files --error-unmatch rc-secrets.env 2>/dev/null; then
    echo -e "${RED}[FAIL] rc-secrets.env is tracked by git! Untrack it immediately.${NC}"
    exit 1
fi
echo -e "  ${GREEN}✓${NC} rc-secrets.env properly untracked/gitignored"

# Scan tracked files for raw password assignments (excluding docs, examples, and test fixtures)
SUSPECT_LEAKS=$(git -C "${REPO_ROOT}" grep -n -I -E "(RC_PASS|password)\s*=\s*['\"][^'\"]{6,}['\"]" -- ':!*.example' ':!tests/*' ':!*.md' || true)
if [[ -n "$SUSPECT_LEAKS" ]]; then
    echo -e "${RED}[FAIL] Potential hardcoded secret found in tracked files:${NC}"
    echo "$SUSPECT_LEAKS"
    exit 1
fi
echo -e "  ${GREEN}✓${NC} No hardcoded passwords detected in tracked files"
echo -e "${GREEN}Secret isolation OK.${NC}\n"

# 5. Execute Test Suite
echo -e "${BLUE}[4/4] Running automated unit test suite (pytest)...${NC}"
cd "${REPO_ROOT}"
if [[ -x "$PYTEST_BIN" ]]; then
    "$PYTEST_BIN" tests
else
    "$PYTHON_BIN" -m pytest tests
fi
echo -e "${GREEN}All unit tests passed.${NC}\n"

echo -e "${GREEN}====================================================${NC}"
echo -e "${GREEN}    Pre-Flight Passed: Ready to commit & push!     ${NC}"
echo -e "${GREEN}====================================================${NC}"
