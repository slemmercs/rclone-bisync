#!/bin/bash
# ==============================================================================
# Bisync Hub Enterprise Installer
# Automated, idempotent provisioning for Bisync Hub and systemd user services.
# ==============================================================================
set -euo pipefail

# --- Color & Formatting Setup ---
if [[ -t 1 && -z "${NO_COLOR:-}" ]]; then
    C_RESET="\033[0m"
    C_BOLD="\033[1m"
    C_GREEN="\033[32m"
    C_BLUE="\033[34m"
    C_YELLOW="\033[33m"
    C_RED="\033[31m"
else
    C_RESET=""
    C_BOLD=""
    C_GREEN=""
    C_BLUE=""
    C_YELLOW=""
    C_RED=""
fi

log_info()    { echo -e "${C_BLUE}ℹ${C_RESET} $*"; }
log_step()    { echo -e "${C_BOLD}==>${C_RESET} $*"; }
log_success() { echo -e "${C_GREEN}✔${C_RESET} $*"; }
log_warn()    { echo -e "${C_YELLOW}⚠${C_RESET} $*"; }
log_error()   { echo -e "${C_RED}✖${C_RESET} $*" >&2; }

print_usage() {
    cat <<EOF
Usage: sudo bash deploy/install.sh [OPTIONS]

Enterprise automated installer for Bisync Hub.

Options:
  -h, --help                  Show this help documentation and exit
  -i, --install-rclone        Install official Rclone binary if not found in PATH
  -n, --dry-run               Simulate installation without making system changes
      --skip-services         Compile and stage files without starting systemd services
  -d, --target-dir <path>     Installation destination (default: \$HOME/rclone-bisync)
      --port <port>           Dashboard web service port (default: 8080)
      --min-rclone-version <v> Enforce minimum required Rclone semver (e.g. 1.60.0)
      --max-rclone-version <v> Enforce maximum compatible Rclone semver
  -y, --yes, --non-interactive Run unattended without interactive prompts

Exit Codes:
  0: Success
  1: Fatal error or missing prerequisite
  2: Invalid CLI option or missing argument
  3: Missing root / sudo privileges
EOF
}

# --- Semantic Version Comparison Helper ---
# Returns: 0 if equal, 1 if v1 > v2, 2 if v1 < v2
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



# --- CLI Options Defaults ---
INSTALL_RCLONE=false
DRY_RUN=false
SKIP_SERVICES=false
DASHBOARD_PORT=8080
TARGET_DIR=""
MIN_RCLONE_VERSION=""
MAX_RCLONE_VERSION=""
NON_INTERACTIVE=false

# --- Parse Arguments ---
while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)
            print_usage
            exit 0
            ;;
        -i|--install-rclone)
            INSTALL_RCLONE=true
            shift
            ;;
        -n|--dry-run)
            DRY_RUN=true
            shift
            ;;
        --skip-services)
            SKIP_SERVICES=true
            shift
            ;;
        -d|--target-dir)
            if [[ -z "${2:-}" || "$2" == -* ]]; then
                log_error "Option '$1' requires a path argument."
                exit 2
            fi
            TARGET_DIR="$2"
            shift 2
            ;;
        --port)
            if [[ -z "${2:-}" || "$2" == -* ]]; then
                log_error "Option '$1' requires a port argument."
                exit 2
            fi
            DASHBOARD_PORT="$2"
            shift 2
            ;;
        --min-rclone-version)
            if [[ -z "${2:-}" || "$2" == -* ]]; then
                log_error "Option '$1' requires a version argument."
                exit 2
            fi
            MIN_RCLONE_VERSION="$2"
            shift 2
            ;;
        --max-rclone-version)
            if [[ -z "${2:-}" || "$2" == -* ]]; then
                log_error "Option '$1' requires a version argument."
                exit 2
            fi
            MAX_RCLONE_VERSION="$2"
            shift 2
            ;;
        -y|--yes|--non-interactive)
            NON_INTERACTIVE=true
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            echo "Run 'deploy/install.sh --help' for usage."
            exit 2
            ;;
    esac
done

# --- Root Privilege Guard ---
if [[ "$EUID" -ne 0 && "$DRY_RUN" != "true" ]]; then
    log_error "This script must be run with sudo. Run: sudo bash $0 [OPTIONS]"
    exit 3
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

REAL_USER="${SUDO_USER:-${LOGNAME:-$(id -un)}}"
REAL_HOME="$(getent passwd "$REAL_USER" 2>/dev/null | cut -d: -f6 || echo "$HOME")"
if [[ -z "$REAL_HOME" ]]; then
    REAL_HOME="$HOME"
fi

if [[ -z "$TARGET_DIR" ]]; then
    RCLONE_DIR="$REAL_HOME/rclone-bisync"
else
    RCLONE_DIR="$TARGET_DIR"
fi

SYSTEMD_USER_DIR="$REAL_HOME/.config/systemd/user"
SHORTCUT_DIR="$REAL_HOME/.local/share/applications"

log_step "Starting Bisync Hub installation for $REAL_USER..."
log_info "Repository root resolved to: $REPO_ROOT"
log_info "Target directory: $RCLONE_DIR"
if [[ "$DRY_RUN" == "true" ]]; then
    log_warn "Running in DRY-RUN mode. No filesystem or systemd modifications will be applied."
fi

# --- 1. Install System Prerequisites ---
log_step "1/6: Verifying system prerequisites..."
if ! command -v curl >/dev/null 2>&1; then
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY-RUN] Would install curl via package manager"
    else
        log_info "curl not found, installing via apt-get..."
        apt-get update
        apt-get install -y curl
    fi
else
    log_info "System prerequisite 'curl' already present."
fi

# --- 2. Install / Verify Rclone Binary ---
validate_rclone_installation() {
    log_step "2/6: Verifying Rclone binary and version compatibility..."
    if command -v rclone >/dev/null 2>&1; then
        RCLONE_PATH="$(command -v rclone)"
        RCLONE_VER="$(rclone version 2>/dev/null | head -n 1 | awk '{print $2}' | sed 's/^v//' || echo "unknown")"

        # Check minimum version constraint
        if [[ -n "$MIN_RCLONE_VERSION" && "$RCLONE_VER" != "unknown" ]]; then
            local cmp
            cmp="$(compare_semver "$RCLONE_VER" "$MIN_RCLONE_VERSION")"
            if [[ "$cmp" == "2" ]]; then
                log_error "Detected Rclone version $RCLONE_VER is below required minimum $MIN_RCLONE_VERSION."
                exit 1
            fi
        fi

        # Check maximum version constraint
        if [[ -n "$MAX_RCLONE_VERSION" && "$RCLONE_VER" != "unknown" ]]; then
            local cmp
            cmp="$(compare_semver "$RCLONE_VER" "$MAX_RCLONE_VERSION")"
            if [[ "$cmp" == "1" ]]; then
                log_error "Detected Rclone version $RCLONE_VER exceeds maximum supported version $MAX_RCLONE_VERSION."
                exit 1
            fi
        fi

        log_success "Found existing Rclone installation at $RCLONE_PATH ($RCLONE_VER), skipping installation."
    else
        if [[ "$INSTALL_RCLONE" == "true" ]]; then
            if [[ "$DRY_RUN" == "true" ]]; then
                log_info "[DRY-RUN] Would download and run official Rclone installer"
            else
                log_info "Rclone not found. Installing via official script..."
                curl https://rclone.org/install.sh | bash || true
                if ! command -v rclone >/dev/null 2>&1; then
                    log_error "Rclone installation failed or binary not found in PATH."
                    exit 1
                fi
            fi
        else
            log_error "Rclone is not installed and --install-rclone was not specified."
            log_error "Please install Rclone manually or run with: sudo bash $0 --install-rclone"
            exit 1
        fi
    fi
}
validate_rclone_installation


if [[ "$DRY_RUN" == "true" ]]; then
    log_success "[DRY-RUN] Pre-flight verification completed successfully."
    exit 0
fi


# --- 3. Setup Layout & Build Native Go Single-Binary ---
echo "Creating directory layout: $RCLONE_DIR"
sudo -u "$REAL_USER" mkdir -p "$RCLONE_DIR/bin"

# Verify base workspace configuration assets exist before copy
if [ ! -f "$REPO_ROOT/rc-secrets.env" ]; then
    if [ -f "$REPO_ROOT/config/rc-secrets.env.example" ]; then
        echo "Notice: rc-secrets.env not found. Creating from config/rc-secrets.env.example..."
        cp "$REPO_ROOT/config/rc-secrets.env.example" "$REPO_ROOT/rc-secrets.env"
        chmod 600 "$REPO_ROOT/rc-secrets.env"
    else
        echo "Error: Neither rc-secrets.env nor config/rc-secrets.env.example found."
        exit 1
    fi
fi

if [ ! -f "$REPO_ROOT/bisync.env" ]; then
    if [ -f "$REPO_ROOT/config/bisync.env.example" ]; then
        echo "Notice: bisync.env not found. Creating from config/bisync.env.example..."
        cp "$REPO_ROOT/config/bisync.env.example" "$REPO_ROOT/bisync.env"
    else
        echo "Error: Neither bisync.env nor config/bisync.env.example found."
        exit 1
    fi
fi

REQUIRED_FILES=(
    "$REPO_ROOT/bisync.env"
    "$REPO_ROOT/config/bisync.env.example"
    "$REPO_ROOT/rc-secrets.env"
    "$REPO_ROOT/deploy/systemd/rclone-daemon.service"
    "$REPO_ROOT/deploy/systemd/rclone-dashboard.service"
    "$REPO_ROOT/deploy/desktop/rclone-dashboard.desktop"
)
for f in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$f" ]; then
        echo "Error: Required workspace configuration asset '$f' not found."
        exit 1
    fi
done

# Build the native Go single binary if Go is available, or copy pre-built bin/rclone-bisync
if command -v go >/dev/null 2>&1; then
    echo "Compiling native single-binary rclone-bisync..."
    (cd "$REPO_ROOT" && sudo -u "$REAL_USER" go build -o "$RCLONE_DIR/bin/rclone-bisync" main.go)
elif [ -f "$REPO_ROOT/bin/rclone-bisync" ]; then
    echo "Installing pre-compiled rclone-bisync binary..."
    cp "$REPO_ROOT/bin/rclone-bisync" "$RCLONE_DIR/bin/rclone-bisync"
else
    echo "Error: Neither Go compiler nor bin/rclone-bisync binary found."
    exit 1
fi

chmod +x "$RCLONE_DIR/bin/rclone-bisync"

if [ ! -f "$RCLONE_DIR/bisync.env" ]; then
    cp "$REPO_ROOT/bisync.env" "$RCLONE_DIR/"
else
    echo "Preserving existing configuration file: $RCLONE_DIR/bisync.env"
fi

if [ ! -f "$RCLONE_DIR/rc-secrets.env" ]; then
    cp "$REPO_ROOT/rc-secrets.env" "$RCLONE_DIR/"
else
    echo "Preserving existing secrets file: $RCLONE_DIR/rc-secrets.env"
fi

if [ -d "$REPO_ROOT/icons" ]; then
    cp -r "$REPO_ROOT/icons" "$RCLONE_DIR/"
fi

chown -R "$REAL_USER:$REAL_USER" "$RCLONE_DIR"
chmod 600 "$RCLONE_DIR/rc-secrets.env"
chmod 644 "$RCLONE_DIR/bisync.env"

# --- 4. Setup Systemd User Services ---
echo "Configuring systemd user services..."
sudo -u "$REAL_USER" mkdir -p "$SYSTEMD_USER_DIR"

USER_ID=$(id -u "$REAL_USER")
DBUS_ENV="XDG_RUNTIME_DIR=/run/user/$USER_ID DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$USER_ID/bus"

# Decommission obsolete single-remote timer and standalone MCP service if present
sudo -u "$REAL_USER" $DBUS_ENV systemctl --user stop bisync-gdrive.timer bisync-gdrive.service rclone-mcp.service 2>/dev/null || true
sudo -u "$REAL_USER" $DBUS_ENV systemctl --user disable bisync-gdrive.timer bisync-gdrive.service rclone-mcp.service 2>/dev/null || true
rm -f "$SYSTEMD_USER_DIR/bisync-gdrive.service" "$SYSTEMD_USER_DIR/bisync-gdrive.timer" "$SYSTEMD_USER_DIR/rclone-mcp.service"

cp "$REPO_ROOT/deploy/systemd/rclone-daemon.service" \
   "$REPO_ROOT/deploy/systemd/rclone-dashboard.service" \
   "$SYSTEMD_USER_DIR/"

# Dynamically resolve system binaries for systemd units
RCLONE_BIN=$(command -v rclone || echo "/usr/bin/rclone")
CURL_BIN=$(command -v curl || echo "/usr/bin/curl")
sed -i "s|/usr/bin/rclone|$RCLONE_BIN|g" "$SYSTEMD_USER_DIR/rclone-daemon.service"
sed -i "s|/usr/bin/curl|$CURL_BIN|g" "$SYSTEMD_USER_DIR/rclone-daemon.service"

chown "$REAL_USER:$REAL_USER" "$SYSTEMD_USER_DIR/"*

# --- 5. Install Desktop Shortcuts ---
echo "Installing application workspace launchers..."
sudo -u "$REAL_USER" mkdir -p "$SHORTCUT_DIR"
rm -f "$SHORTCUT_DIR/rclone-webgui.desktop" "$SHORTCUT_DIR/rclone-bisync-manual.desktop"

cp "$REPO_ROOT/deploy/desktop/rclone-dashboard.desktop" "$SHORTCUT_DIR/"
sed -i "s|Icon=network-server|Icon=$RCLONE_DIR/icons/cloud-sync-light.svg|g" "$SHORTCUT_DIR/rclone-dashboard.desktop"
chown "$REAL_USER:$REAL_USER" "$SHORTCUT_DIR/"rclone-*.desktop
chmod +x "$SHORTCUT_DIR/"rclone-*.desktop

# --- 6. Enable, Reset, and Start Services ---
echo "Activating automated services..."

# Reload systemd and clear any existing failed states / start-rate limits
sudo -u "$REAL_USER" $DBUS_ENV systemctl --user daemon-reload
sudo -u "$REAL_USER" $DBUS_ENV systemctl --user reset-failed

# Enable components (scheduler runs internally within dashboard service)
sudo -u "$REAL_USER" $DBUS_ENV systemctl --user enable rclone-daemon.service rclone-dashboard.service

# Restart services sequentially
sudo -u "$REAL_USER" $DBUS_ENV systemctl --user restart rclone-daemon.service
sudo -u "$REAL_USER" $DBUS_ENV systemctl --user restart rclone-dashboard.service

# Verify daemon status
if sudo -u "$REAL_USER" $DBUS_ENV systemctl --user is-active --quiet rclone-daemon.service; then
    echo "----------------------------------------------------"
    echo "✅ Installation Complete successfully!"
    echo "Bisync Hub is running live at http://127.0.0.1:8080"
else
    echo "----------------------------------------------------"
    echo "⚠️ Warning: rclone-daemon.service failed to report active status."
    echo "Run: systemctl --user status rclone-daemon.service for details."
fi
