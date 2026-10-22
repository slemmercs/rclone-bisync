#!/bin/bash

# --- Rclone Bisync Installation Script (Dashboard Architecture) ---
set -e

if [[ "$EUID" -ne 0 ]]; then
  echo "This script must be run with sudo. Run: sudo bash $0"
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

REAL_USER="${SUDO_USER:-$(id -un)}"
REAL_HOME="$(getent passwd "$REAL_USER" | cut -d: -f6)"
RCLONE_DIR="$REAL_HOME/rclone-bisync"
SYSTEMD_USER_DIR="$REAL_HOME/.config/systemd/user"
SHORTCUT_DIR="$REAL_HOME/.local/share/applications"

echo "Starting installation for $REAL_USER..."
echo "Repository root resolved to: $REPO_ROOT"
echo "----------------------------------------------------"

# --- 1. Install System Prerequisites ---
echo "Installing system prerequisites (curl, build tools if needed)..."
apt-get update
apt-get install -y curl

# --- 2. Install Rclone Binary ---
echo "Installing/Updating Rclone binary..."
curl https://rclone.org/install.sh | bash || true

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
    "$REPO_ROOT/deploy/systemd/bisync-gdrive.service"
    "$REPO_ROOT/deploy/systemd/bisync-gdrive.timer"
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
cp "$REPO_ROOT/deploy/systemd/rclone-daemon.service" \
   "$REPO_ROOT/deploy/systemd/rclone-dashboard.service" \
   "$REPO_ROOT/deploy/systemd/bisync-gdrive.service" \
   "$REPO_ROOT/deploy/systemd/bisync-gdrive.timer" \
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
USER_ID=$(id -u "$REAL_USER")
DBUS_ENV="XDG_RUNTIME_DIR=/run/user/$USER_ID DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$USER_ID/bus"

# Reload systemd and clear any existing failed states / start-rate limits
sudo -u "$REAL_USER" $DBUS_ENV systemctl --user daemon-reload
sudo -u "$REAL_USER" $DBUS_ENV systemctl --user reset-failed

# Enable components
sudo -u "$REAL_USER" $DBUS_ENV systemctl --user enable rclone-daemon.service rclone-dashboard.service bisync-gdrive.timer

# Restart services sequentially
sudo -u "$REAL_USER" $DBUS_ENV systemctl --user restart rclone-daemon.service
sudo -u "$REAL_USER" $DBUS_ENV systemctl --user restart rclone-dashboard.service
sudo -u "$REAL_USER" $DBUS_ENV systemctl --user start bisync-gdrive.timer

# Verify daemon status
if sudo -u "$REAL_USER" $DBUS_ENV systemctl --user is-active --quiet rclone-daemon.service; then
    echo "----------------------------------------------------"
    echo "✅ Installation Complete successfully!"
    echo "Custom Telemetry UI is running live at http://127.0.0.1:8080"
else
    echo "----------------------------------------------------"
    echo "⚠️ Warning: rclone-daemon.service failed to report active status."
    echo "Run: systemctl --user status rclone-daemon.service for details."
fi
