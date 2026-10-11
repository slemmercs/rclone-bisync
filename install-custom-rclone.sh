#!/bin/bash

# --- Rclone Bisync Installation Script (Dashboard Architecture) ---
set -e

if [[ "$EUID" -ne 0 ]]; then
  echo "This script must be run with sudo. Run: sudo bash $0"
  exit 1
fi

REAL_USER="${SUDO_USER:-$(id -un)}"
REAL_HOME="$(getent passwd "$REAL_USER" | cut -d: -f6)"
RCLONE_DIR="$REAL_HOME/rclone-bisync"
SYSTEMD_USER_DIR="$REAL_HOME/.config/systemd/user"
SHORTCUT_DIR="$REAL_HOME/.local/share/applications"

echo "Starting installation for $REAL_USER..."
echo "----------------------------------------------------"

# --- 1. Install System Prerequisites & Virtual Environment Tooling ---
echo "Installing system prerequisites (python3-venv, curl)..."
apt-get update
apt-get install -y python3-venv curl

# --- 2. Install Rclone Binary ---
echo "Installing/Updating Rclone binary..."
curl https://rclone.org/install.sh | bash || true

# --- 3. Setup Script & Dashboard Directory ---
echo "Creating directory layout: $RCLONE_DIR"
sudo -u "$REAL_USER" mkdir -p "$RCLONE_DIR"

# Verify base workspace configuration assets exist before copy
if [ ! -f "rc-secrets.env" ]; then
    if [ -f "rc-secrets.env.example" ]; then
        echo "Notice: rc-secrets.env not found. Creating from rc-secrets.env.example..."
        cp rc-secrets.env.example rc-secrets.env
        chmod 600 rc-secrets.env
    else
        echo "Error: Neither rc-secrets.env nor rc-secrets.env.example found."
        exit 1
    fi
fi

if [ ! -f "bisync.env" ]; then
    if [ -f "bisync.env.example" ]; then
        echo "Notice: bisync.env not found. Creating from bisync.env.example..."
        cp bisync.env.example bisync.env
    else
        echo "Error: Neither bisync.env nor bisync.env.example found."
        exit 1
    fi
fi

REQUIRED_FILES=("bisync.env" "bisync.env.example" "bisync_gdrive_advanced.py" "requirements.txt" "rc-secrets.env" "rclone-daemon.service" "rclone-dashboard.service" "bisync-gdrive.service" "bisync-gdrive.timer" "rclone-bisync-manual.desktop" "rclone-dashboard.desktop")
for f in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$f" ]; then
        echo "Error: Required workspace configuration asset '$f' not found."
        exit 1
    fi
done

if [ ! -d "dashboard" ]; then
    echo "Error: 'dashboard' directory containing backend/ and frontend/ assets is missing."
    exit 1
fi

if [ ! -d "rclone_bisync" ]; then
    echo "Error: 'rclone_bisync' package directory is missing."
    exit 1
fi

cp bisync_gdrive_advanced.py requirements.txt "$RCLONE_DIR/"
cp -r rclone_bisync "$RCLONE_DIR/"

if [ ! -f "$RCLONE_DIR/bisync.env" ]; then
    cp bisync.env "$RCLONE_DIR/"
else
    echo "Preserving existing configuration file: $RCLONE_DIR/bisync.env"
fi

if [ ! -f "$RCLONE_DIR/rc-secrets.env" ]; then
    cp rc-secrets.env "$RCLONE_DIR/"
else
    echo "Preserving existing secrets file: $RCLONE_DIR/rc-secrets.env"
fi
cp -r dashboard "$RCLONE_DIR/"

# Bootstrap isolated Python virtual environment
echo "Setting up Python virtual environment in $RCLONE_DIR/.venv..."
if [ ! -d "$RCLONE_DIR/.venv" ]; then
    sudo -u "$REAL_USER" python3 -m venv "$RCLONE_DIR/.venv"
fi
echo "Installing application dependencies into .venv..."
sudo -u "$REAL_USER" "$RCLONE_DIR/.venv/bin/pip" install --upgrade pip
sudo -u "$REAL_USER" "$RCLONE_DIR/.venv/bin/pip" install -r "$RCLONE_DIR/requirements.txt"

chown -R "$REAL_USER:$REAL_USER" "$RCLONE_DIR"
chmod +x "$RCLONE_DIR/bisync_gdrive_advanced.py"
chmod 600 "$RCLONE_DIR/rc-secrets.env"
chmod 644 "$RCLONE_DIR/bisync.env"

# --- 4. Setup Systemd User Services ---
echo "Configuring systemd user services..."
sudo -u "$REAL_USER" mkdir -p "$SYSTEMD_USER_DIR"
cp rclone-daemon.service rclone-dashboard.service bisync-gdrive.service bisync-gdrive.timer "$SYSTEMD_USER_DIR/"

# Dynamically resolve system binaries for systemd units
RCLONE_BIN=$(command -v rclone || echo "/usr/bin/rclone")
CURL_BIN=$(command -v curl || echo "/usr/bin/curl")
sed -i "s|/usr/bin/rclone|$RCLONE_BIN|g" "$SYSTEMD_USER_DIR/rclone-daemon.service"
sed -i "s|/usr/bin/curl|$CURL_BIN|g" "$SYSTEMD_USER_DIR/rclone-daemon.service"

chown "$REAL_USER:$REAL_USER" "$SYSTEMD_USER_DIR/"*

# --- 5. Install Desktop Shortcuts ---
echo "Installing application workspace launchers..."
sudo -u "$REAL_USER" mkdir -p "$SHORTCUT_DIR"
rm -f "$SHORTCUT_DIR/rclone-webgui.desktop"

cp rclone-bisync-manual.desktop rclone-dashboard.desktop "$SHORTCUT_DIR/"
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