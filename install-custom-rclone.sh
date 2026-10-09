#!/bin/bash

# --- Rclone Bisync Installation Script (Dashboard Architecture) ---
set -e

if [[ "$EUID" -ne 0 ]]; then
  echo "This script must be run with sudo. Run: sudo bash $0"
  exit 1
fi

REAL_USER=$(logname || echo $SUDO_USER)
REAL_HOME=$(eval echo ~$REAL_USER)
RCLONE_DIR="$REAL_HOME/rclone-bisync"
SYSTEMD_USER_DIR="$REAL_HOME/.config/systemd/user"
SHORTCUT_DIR="$REAL_HOME/.local/share/applications"

echo "Starting installation for $REAL_USER..."
echo "----------------------------------------------------"

# --- 1. Install System & Python Dependencies ---
echo "Installing system prerequisites (FastAPI, Uvicorn, and utilities)..."
apt-get update
apt-get install -y python3-requests python3-plyer python3-fastapi python3-uvicorn python3-psutil

# --- 2. Install Rclone Binary ---
echo "Installing/Updating Rclone binary..."
curl https://rclone.org/install.sh | bash || true

# --- 3. Setup Script & Dashboard Directory ---
echo "Creating directory layout: $RCLONE_DIR"
sudo -u "$REAL_USER" mkdir -p "$RCLONE_DIR"

# Verify base workspace configuration assets exist before copy
REQUIRED_FILES=("bisync_config.py" "bisync_gdrive_advanced.py" "rclone-daemon.service" "rclone-dashboard.service" "bisync-gdrive.service" "bisync-gdrive.timer" "rclone-bisync-manual.desktop" "rclone-dashboard.desktop")
for f in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$f" ]; then
        echo "Error: Required workspace configuration asset '$f' not found."
        exit 1
    fi
done

# Ensure local dashboard components are structured before migrating
if [ ! -d "dashboard" ]; then
    echo "Error: 'dashboard' directory containing backend/ and frontend/ assets is missing."
    exit 1
fi

cp bisync_config.py bisync_gdrive_advanced.py "$RCLONE_DIR/"
cp -r dashboard "$RCLONE_DIR/"
chown -R "$REAL_USER:$REAL_USER" "$RCLONE_DIR"
chmod +x "$RCLONE_DIR/bisync_gdrive_advanced.py"

# --- 4. Setup Systemd User Services ---
echo "Configuring systemd user services..."
sudo -u "$REAL_USER" mkdir -p "$SYSTEMD_USER_DIR"
cp rclone-daemon.service rclone-dashboard.service bisync-gdrive.service bisync-gdrive.timer "$SYSTEMD_USER_DIR/"
chown "$REAL_USER:$REAL_USER" "$SYSTEMD_USER_DIR/"*

# --- 5. Install Desktop Shortcuts ---
echo "Installing application workspace launchers..."
sudo -u "$REAL_USER" mkdir -p "$SHORTCUT_DIR"
# Clear out legacy shortcut if it exists
rm -f "$SHORTCUT_DIR/rclone-webgui.desktop"

cp rclone-bisync-manual.desktop rclone-dashboard.desktop "$SHORTCUT_DIR/"
chown "$REAL_USER:$REAL_USER" "$SHORTCUT_DIR/"rclone-*.desktop
chmod +x "$SHORTCUT_DIR/"rclone-*.desktop

# --- 6. Enable and Start Services ---
echo "Activating automated services..."
USER_ID=$(id -u "$REAL_USER")
# XDG_RUNTIME_DIR is required by modern systemd --user to locate the user's
# service manager socket. DBUS_SESSION_BUS_ADDRESS alone is not sufficient.
DBUS_ENV="XDG_RUNTIME_DIR=/run/user/$USER_ID DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$USER_ID/bus"

# Force systemd daemon reset under the target user session context
sudo -u "$REAL_USER" $DBUS_ENV systemctl --user daemon-reload

# Enable core components
sudo -u "$REAL_USER" $DBUS_ENV systemctl --user enable rclone-daemon.service rclone-dashboard.service bisync-gdrive.timer
sudo -u "$REAL_USER" $DBUS_ENV systemctl --user restart rclone-daemon.service rclone-dashboard.service
sudo -u "$REAL_USER" $DBUS_ENV systemctl --user start bisync-gdrive.timer

echo "----------------------------------------------------"
echo "✅ Installation Complete successfully!"
echo "Custom Telemetry UI is running live at http://127.0.0.1:8080"