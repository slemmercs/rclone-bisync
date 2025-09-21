#!/bin/bash

# --- Rclone Bisync Installation Script ---
# This script automates the installation of rclone, and the necessary
# Python scripts and systemd services for the automated bisync solution.

# Exit immediately if a command exits with a non-zero status.
set -e

# --- 1. Check for sudo privileges ---
if [[ "$EUID" -ne 0 ]]; then
  echo "This script must be run with sudo privileges. Please run with 'sudo bash install.sh'."
  exit 1
fi

# A directory to store the rclone-bisync files in the user's home directory.
RCLONE_DIR="$HOME/rclone-bisync"

echo "Starting automated installation for Rclone bisync..."
echo "----------------------------------------------------"

# --- 2. Install Rclone ---
echo "Installing Rclone..."
curl https://rclone.org/install.sh | bash

# --- 3. Copy the script files ---
echo "Creating the script directory: $RCLONE_DIR"
mkdir -p "$RCLONE_DIR"

echo "Copying Python scripts to $RCLONE_DIR..."
# Check for the existence of the python scripts before copying.
if [ ! -f "bisync_config.py" ] || [ ! -f "bisync_gdrive_advanced.py" ]; then
    echo "Error: 'bisync_config.py' or 'bisync_gdrive_advanced.py' not found in the current directory."
    echo "Please ensure these files are present before running the script."
    exit 1
fi

cp bisync_config.py "$RCLONE_DIR/"
cp bisync_gdrive_advanced.py "$RCLONE_DIR/"

# Ensure the main script is executable.
chmod +x "$RCLONE_DIR/bisync_gdrive_advanced.py"

# --- 4. Copy Systemd Service and Timer files ---
echo "Copying systemd user service and timer files..."
SYSTEMD_USER_DIR="$HOME/.config/systemd/user"
mkdir -p "$SYSTEMD_USER_DIR"

# Check for the existence of the service and timer files before copying.
if [ ! -f "bisync-gdrive.service" ] || [ ! -f "bisync-gdrive.timer" ]; then
    echo "Error: 'bisync-gdrive.service' or 'bisync-gdrive.timer' not found in the current directory."
    echo "Please ensure these files are present before running the script."
    exit 1
fi

cp bisync-gdrive.service "$SYSTEMD_USER_DIR/"
cp bisync-gdrive.timer "$SYSTEMD_USER_DIR/"

# --- 5. Enable and start services ---
echo "Enabling and starting systemd services and timers..."
systemctl --user daemon-reload
systemctl --user enable bisync-gdrive.timer
systemctl --user start bisync-gdrive.timer

echo "----------------------------------------------------"
echo "Installation complete!"
echo "You can view the logs with 'journalctl --user -u bisync-gdrive.service -f'."
echo ""
echo "IMPORTANT: You must now run 'rclone config' to set up your Google Drive remote."
