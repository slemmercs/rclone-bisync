#!/bin/bash

# --- Rclone Bisync Installation Script (API Architecture) ---
set -e

# --- 1. Privilege Check ---
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

# --- 2. Install Rclone Binary ---
echo "Installing/Updating Rclone binary..."
curl https://rclone.org/install.sh | bash

# --- 3. Setup Script Directory ---
echo "Creating directory: $RCLONE_DIR"
sudo -u "$REAL_USER" mkdir -p "$RCLONE_DIR"

REQUIRED_FILES=("bisync_config.py" "bisync_gdrive_advanced.py" "rclone-daemon.service" "bisync-gdrive.service" "bisync-gdrive.timer" "rclone-bisync-manual.desktop" "rclone-webgui.desktop")

for f in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$f" ]; then
        echo "Error: Required file '$f' not found in current directory."
        exit 1
    fi
done

cp bisync_config.py bisync_gdrive_advanced.py "$RCLONE_DIR/"
chown -R "$REAL_USER:$REAL_USER" "$RCLONE_DIR"
chmod +x "$RCLONE_DIR/bisync_gdrive_advanced.py"

# --- 4. Setup Systemd User Services ---
echo "Configuring systemd user services..."
sudo -u "$REAL_USER" mkdir -p "$SYSTEMD_USER_DIR"
cp rclone-daemon.service bisync-gdrive.service bisync-gdrive.timer "$SYSTEMD_USER_DIR/"
chown "$REAL_USER:$REAL_USER" "$SYSTEMD_USER_DIR/"*

# --- 5. Install Desktop Shortcuts ---
echo "Installing desktop shortcuts..."
sudo -u "$REAL_USER" mkdir -p "$SHORTCUT_DIR"
cp rclone-bisync-manual.desktop rclone-webgui.desktop "$SHORTCUT_DIR/"
chown "$REAL_USER:$REAL_USER" "$SHORTCUT_DIR/"*.desktop
chmod +x "$SHORTCUT_DIR/"*.desktop

# --- 6. Enable and Start Services ---
echo "Activating services..."
USER_ID=$(id -u "$REAL_USER")
DBUS_ENV="DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$USER_ID/bus"

sudo -u "$REAL_USER" $DBUS_ENV systemctl --user daemon-reload
sudo -u "$REAL_USER" $DBUS_ENV systemctl --user enable rclone-daemon.service bisync-gdrive.timer
sudo -u "$REAL_USER" $DBUS_ENV systemctl --user restart rclone-daemon.service bisync-gdrive.timer

echo "----------------------------------------------------"
echo "Installation Complete!"
echo "1. Daemon active on port 5572."
echo "2. Timer active (15-minute intervals)."
echo "3. Shortcuts 'Rclone Force Bisync' and 'Rclone Web GUI' added to menu."