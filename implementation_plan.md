# Implementation Plan: Enterprise-Grade Bisync Hub Installer

## Overview
Transform `deploy/install.sh` into an enterprise-grade, idempotent installer for Bisync Hub featuring robust CLI option parsing, explicit opt-in Rclone installation flags, semantic version inspection with extensible boundary validation, multi-distro dependency resolution, and strict fail-fast error handling.

The current `deploy/install.sh` unconditionally downloads and executes Rclone's external installation script via `curl | bash` on every run, overwriting any existing Rclone binaries without user consent or version checks. It also runs a blind `apt-get update` regardless of whether prerequisites already exist, lacks CLI argument parsing (`--help`, `--install-rclone`, `--dry-run`), and assumes a Debian/Ubuntu-only package management environment.

This implementation re-engineers `deploy/install.sh` into a POSIX-compliant, defensive Bash installer adhering to enterprise software distribution standards. It introduces strict pre-flight checks, selective dependency installation, color-coded structured logging (with `NO_COLOR` support), safe secrets generation for unconfigured deployments, dynamic systemd user service provisioning with DBus validation, and a self-contained test suite ensuring zero regressions under `< 350 LOC`.

---

## Types
Since this change operates in Bash (`deploy/install.sh`), type contracts are defined via strict Shell variable constraints, standard exit codes, and structured CLI options:

### Exit Codes
- `0`: Success — Bisync Hub installed, services active and healthy.
- `1`: General fatal error or missing prerequisite (e.g. Rclone missing without `--install-rclone`).
- `2`: Invalid CLI option or missing argument.
- `3`: Missing root / `sudo` privileges.
- `4`: Unsupported operating system or architecture.
- `5`: Systemd user daemon initialization failure.

### Configuration Struct (Shell State Representation)
```bash
INSTALL_RCLONE=false          # Flag: -i, --install-rclone (default: false)
DRY_RUN=false                 # Flag: -n, --dry-run (default: false)
SKIP_SERVICES=false           # Flag: --skip-services (default: false)
DASHBOARD_PORT=8080           # Flag: --port <port> (default: 8080)
TARGET_DIR=""                 # Flag: -d, --target-dir <path> (default: $REAL_HOME/rclone-bisync)
MIN_RCLONE_VERSION=""         # Flag: --min-rclone-version <semver> (hook placeholder)
MAX_RCLONE_VERSION=""         # Flag: --max-rclone-version <semver> (hook placeholder)
NON_INTERACTIVE=false         # Flag: -y, --yes, --non-interactive (default: false)
```

---

## Files

### Modified Files
- `deploy/install.sh`:
  - Complete re-architecture of the installation script incorporating structured CLI flags, Rclone presence detection, semantic versioning extraction and validation hooks, multi-distro prerequisite checking, atomic configuration staging, and robust systemd health validation.
  - Target line count budget: strictly $\le$ 350 LOC.

### New Test Files
- `tests/unit/install_script_test.sh`:
  - Dedicated unit and regression test suite for `deploy/install.sh`.
  - Tests argument parsing (`--help`, unknown flags, `--install-rclone`), version extraction regexes, semver comparison edge cases, mock missing Rclone fail-fast behavior, and dry-run safety.

---

## Functions

### Shell Functions in `deploy/install.sh`

1. **`print_usage()`**: Emits formatted CLI help documentation with available options, environment variables, exit codes, and usage examples to stdout. Supports running without `sudo`.
2. **`log_info()`, `log_success()`, `log_warn()`, `log_error()`, `log_step()`**: Standardized, colored, timestamped output adhering to 12-factor CLI specifications. Supports `NO_COLOR=1`.
3. **`parse_arguments()`**: Parses CLI arguments (`-h|--help`, `-i|--install-rclone`, `-n|--dry-run`, `-d|--target-dir`, `--port`, `--skip-services`, `--min-rclone-version`, `-y|--non-interactive`).
4. **`detect_real_user()`**: Resolves the true invoking user (`SUDO_USER` vs `LOGNAME` vs `id -un`) and home directory via `getent passwd`.
5. **`ensure_prerequisites()`**: Checks if `curl` is present on PATH. If missing, detects package manager (`apt-get`, `dnf`, `yum`, `pacman`, `zypper`) and installs `curl`.
6. **`extract_rclone_version()`**: Executes `<binary_path> version` and extracts clean semantic version string (e.g. `v1.76.0` $\rightarrow$ `1.76.0`).
7. **`compare_semver()`**: Pure Bash numerical 3-part semantic version comparison returning `0` (equal), `1` ($v1 > v2$), or `2` ($v1 < v2$).
8. **`validate_rclone_installation()`**:
   - Probes `command -v rclone`.
   - If installed: extracts version, prints `Found existing Rclone installation: <path> (<version>)`, evaluates against `MIN_RCLONE_VERSION` / `MAX_RCLONE_VERSION` hooks, and skips installation.
   - If not installed and `--install-rclone` is passed: executes official install script, verifies success, and captures version.
   - If not installed and `--install-rclone` is NOT passed: halts immediately with `exit 1` and informative guidance.
9. **`generate_secure_secrets()`**: Copies `rc-secrets.env.example` to `rc-secrets.env` if not already present, replacing default `RC_PASS` with a cryptographically generated 24-character random secret.
10. **`configure_and_start_services()`**: Copies and templates systemd user service units, sets up DBus session variables, reloads user daemon, and performs a 5-step health polling loop.

---

## Verification Plan

- `bash tests/unit/install_script_test.sh`
- `bash -n deploy/install.sh`
- `./scripts/check.sh fmt`
- `./scripts/check-loc.sh 400 deploy/install.sh`
