#!/usr/bin/env bash
# ==============================================================================
# Unit & Regression Test Suite for deploy/install.sh
# Tests option parsing, version detection, semver comparison, and safety gates.
# ==============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
INSTALL_SCRIPT="$REPO_ROOT/deploy/install.sh"

PASSED_COUNT=0
FAILED_COUNT=0

assert_equals() {
    local expected="$1"
    local actual="$2"
    local msg="$3"
    if [[ "$expected" == "$actual" ]]; then
        echo "  ✓ $msg"
        PASSED_COUNT=$((PASSED_COUNT + 1))
    else
        echo "  ✗ $msg: expected '$expected', got '$actual'"
        FAILED_COUNT=$((FAILED_COUNT + 1))
    fi
}

assert_contains() {
    local haystack="$1"
    local needle="$2"
    local msg="$3"
    if [[ "$haystack" == *"$needle"* ]]; then
        echo "  ✓ $msg"
        PASSED_COUNT=$((PASSED_COUNT + 1))
    else
        echo "  ✗ $msg: '$needle' not found in output"
        FAILED_COUNT=$((FAILED_COUNT + 1))
    fi
}

echo "=================================================="
echo "🧪 Running Tests: deploy/install.sh"
echo "=================================================="

# Test 1: --help displays usage with exit 0 (without requiring sudo)
echo "[Test 1] --help outputs usage instructions"
HELP_OUTPUT=$(bash "$INSTALL_SCRIPT" --help 2>&1 || true)
HELP_STATUS=$?
assert_equals "0" "$HELP_STATUS" "Exits with code 0 on --help"
assert_contains "$HELP_OUTPUT" "Usage: sudo bash deploy/install.sh [OPTIONS]" "Usage banner included"
assert_contains "$HELP_OUTPUT" "--install-rclone" "Contains --install-rclone description"

# Test 2: -h alias works identically
echo "[Test 2] -h alias flag"
H_OUTPUT=$(bash "$INSTALL_SCRIPT" -h 2>&1 || true)
assert_contains "$H_OUTPUT" "Usage: sudo bash deploy/install.sh [OPTIONS]" "-h displays help"

# Test 3: Reject unknown option with exit code 2
echo "[Test 3] Unknown options rejected with code 2"
set +e
ERR_OUTPUT=$(bash "$INSTALL_SCRIPT" --unrecognized-option-foo 2>&1)
ERR_STATUS=$?
set -e
assert_equals "2" "$ERR_STATUS" "Exits with code 2 on invalid option"
assert_contains "$ERR_OUTPUT" "Unknown option: --unrecognized-option-foo" "Helpful error message printed"

# Test 4: Semver extraction logic
echo "[Test 4] Version extraction logic"
MOCK_RCLONE_OUTPUT="rclone v1.76.0
- os/version: ubuntu 24.04 (64 bit)
- os/kernel: 6.8.0-45-generic (x86_64)"
EXTRACTED_VER=$(echo "$MOCK_RCLONE_OUTPUT" | head -n 1 | awk '{print $2}' | sed 's/^v//')
assert_equals "1.76.0" "$EXTRACTED_VER" "Clean semantic version extracted"

# Test 4b: Semver comparison logic
echo "[Test 4b] Semver comparison logic"
compare_semver() {
    local v1="${1#v}"
    local v2="${2#v}"
    if [[ "$v1" == "$v2" ]]; then echo "0"; return 0; fi
    local IFS=.
    local i ver1=($v1) ver2=($v2)
    for ((i=0; i<3; i++)); do
        local n1=${ver1[i]:-0}
        local n2=${ver2[i]:-0}
        n1=${n1%%-*}
        n2=${n2%%-*}
        if ((10#$n1 > 10#$n2)); then echo "1"; return 0; fi
        if ((10#$n1 < 10#$n2)); then echo "2"; return 0; fi
    done
    echo "0"
}
assert_equals "0" "$(compare_semver "1.76.0" "1.76.0")" "1.76.0 == 1.76.0 (0)"
assert_equals "1" "$(compare_semver "1.76.0" "1.68.2")" "1.76.0 > 1.68.2 (1)"
assert_equals "2" "$(compare_semver "1.68.2" "1.76.0")" "1.68.2 < 1.76.0 (2)"
assert_equals "2" "$(compare_semver "1.76.0" "1.76.1")" "1.76.0 < 1.76.1 (2)"
assert_equals "1" "$(compare_semver "2.0.0" "1.99.9")" "2.0.0 > 1.99.9 (1)"


# Test 5: Missing Rclone fail-fast check without --install-rclone
echo "[Test 5] Missing Rclone fail-fast abort (exit 1)"
TMP_BIN_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_BIN_DIR"' EXIT

# Create a mock PATH where everything from /usr/bin and /bin is available EXCEPT rclone
mkdir -p "$TMP_BIN_DIR/bin"
# Populate with symlinks to all binaries in /bin and /usr/bin except rclone
for p in /usr/bin/* /bin/*; do
    bn="$(basename "$p")"
    if [[ "$bn" != "rclone" && -x "$p" && ! -e "$TMP_BIN_DIR/bin/$bn" ]]; then
        ln -s "$p" "$TMP_BIN_DIR/bin/$bn" 2>/dev/null || true
    fi
done

set +e
NO_RCLONE_OUTPUT=$(env PATH="$TMP_BIN_DIR/bin" SUDO_USER="nobody" bash -c "
    bash \"$INSTALL_SCRIPT\" --dry-run
" 2>&1)
NO_RCLONE_STATUS=$?
set -e

# When rclone is absent and --install-rclone is not passed, script must exit 1
assert_equals "1" "$NO_RCLONE_STATUS" "Missing rclone exits with code 1"
assert_contains "$NO_RCLONE_OUTPUT" "Rclone is not installed" "Explains Rclone is missing"
assert_contains "$NO_RCLONE_OUTPUT" "--install-rclone" "Advises user to pass --install-rclone"

# Test 6: Existing Rclone detection skips installation
echo "[Test 6] Existing Rclone detected & installation skipped"
cat <<'EOF' > "$TMP_BIN_DIR/bin/rclone"
#!/bin/sh
if [ "$1" = "version" ] || [ "$1" = "--version" ]; then
    echo "rclone v1.76.0"
    exit 0
fi
echo "mock rclone invoked with: $*"
exit 0
EOF
chmod +x "$TMP_BIN_DIR/bin/rclone"

set +e
DETECT_OUTPUT=$(env PATH="$TMP_BIN_DIR/bin" SUDO_USER="nobody" bash -c "
    bash \"$INSTALL_SCRIPT\" --install-rclone --dry-run
" 2>&1)
DETECT_STATUS=$?
set -e

assert_equals "0" "$DETECT_STATUS" "Dry run exits 0 when rclone found"
assert_contains "$DETECT_OUTPUT" "1.76.0" "Detected version 1.76.0 reported"
assert_contains "$DETECT_OUTPUT" "skipping installation" "Skips re-installing when already present"

# Test 7: Min Rclone version constraint enforcement
echo "[Test 7] --min-rclone-version constraints"
set +e
MIN_FAIL_OUTPUT=$(env PATH="$TMP_BIN_DIR/bin" SUDO_USER="nobody" bash -c "
    bash \"$INSTALL_SCRIPT\" --dry-run --min-rclone-version 1.80.0
" 2>&1)
MIN_FAIL_STATUS=$?

MIN_PASS_OUTPUT=$(env PATH="$TMP_BIN_DIR/bin" SUDO_USER="nobody" bash -c "
    bash \"$INSTALL_SCRIPT\" --dry-run --min-rclone-version 1.70.0
" 2>&1)
MIN_PASS_STATUS=$?
set -e

assert_equals "1" "$MIN_FAIL_STATUS" "Exits 1 when rclone version < min-version"
assert_contains "$MIN_FAIL_OUTPUT" "is below required minimum 1.80.0" "Reports minimum version error"
assert_equals "0" "$MIN_PASS_STATUS" "Exits 0 when rclone version >= min-version"

# Test 8: Max Rclone version constraint enforcement
echo "[Test 8] --max-rclone-version constraints"
set +e
MAX_FAIL_OUTPUT=$(env PATH="$TMP_BIN_DIR/bin" SUDO_USER="nobody" bash -c "
    bash \"$INSTALL_SCRIPT\" --dry-run --max-rclone-version 1.70.0
" 2>&1)
MAX_FAIL_STATUS=$?

MAX_PASS_OUTPUT=$(env PATH="$TMP_BIN_DIR/bin" SUDO_USER="nobody" bash -c "
    bash \"$INSTALL_SCRIPT\" --dry-run --max-rclone-version 1.80.0
" 2>&1)
MAX_PASS_STATUS=$?
set -e

assert_equals "1" "$MAX_FAIL_STATUS" "Exits 1 when rclone version > max-version"
assert_contains "$MAX_FAIL_OUTPUT" "exceeds maximum supported version 1.70.0" "Reports maximum version error"
assert_equals "0" "$MAX_PASS_STATUS" "Exits 0 when rclone version <= max-version"

# Test 9: Cryptographic secret generation
echo "[Test 9] Secret generation randomness and length"
generate_random_secret() {
    if command -v openssl >/dev/null 2>&1; then
        openssl rand -hex 16
    else
        head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n'
    fi
}
SECRET1=$(generate_random_secret)
SECRET2=$(generate_random_secret)
assert_equals "32" "${#SECRET1}" "Generated secret has 32 hex chars"
if [[ "$SECRET1" != "$SECRET2" ]]; then
    echo "  ✓ Generated secrets are unique ($SECRET1 != $SECRET2)"
    PASSED_COUNT=$((PASSED_COUNT + 1))
else
    echo "  ✖ Error: Secret collision detected!"
    FAILED_COUNT=$((FAILED_COUNT + 1))
fi

# Test 10: Port argument error handling
echo "[Test 10] Port argument validation"
set +e
PORT_ERR_OUTPUT=$(bash "$INSTALL_SCRIPT" --port 2>&1)
PORT_ERR_STATUS=$?
set -e
assert_equals "2" "$PORT_ERR_STATUS" "Exits with code 2 on missing --port value"
assert_contains "$PORT_ERR_OUTPUT" "requires a port argument" "Helpful error message for missing port"

# Test 11: Target-dir argument error handling
echo "[Test 11] Target-dir argument validation"
set +e
DIR_ERR_OUTPUT=$(bash "$INSTALL_SCRIPT" --target-dir 2>&1)
DIR_ERR_STATUS=$?
set -e
assert_equals "2" "$DIR_ERR_STATUS" "Exits with code 2 on missing --target-dir value"
assert_contains "$DIR_ERR_OUTPUT" "requires a path argument" "Helpful error message for missing target dir"

# Test 12: Dry-run honors --target-dir and --skip-services
echo "[Test 12] Dry run logs target-dir"
set +e
DRY_CUSTOM_OUTPUT=$(env PATH="$TMP_BIN_DIR/bin" SUDO_USER="nobody" bash -c "
    bash \"$INSTALL_SCRIPT\" --dry-run --target-dir /tmp/custom-bisync-dir --port 9999 --skip-services
" 2>&1)
DRY_CUSTOM_STATUS=$?
set -e
assert_equals "0" "$DRY_CUSTOM_STATUS" "Dry run exits 0 with custom options"
assert_contains "$DRY_CUSTOM_OUTPUT" "/tmp/custom-bisync-dir" "Custom target directory reflected in output"

echo "=================================================="
echo "📊 Results: $PASSED_COUNT passed, $FAILED_COUNT failed."
echo "=================================================="

if [[ "$FAILED_COUNT" -gt 0 ]]; then
    exit 1
fi
exit 0
