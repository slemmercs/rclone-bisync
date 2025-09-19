#!/usr/bin/env python3

import os

# --- Configuration for Rclone Bisync Script ---

# Rclone and Path Settings
# All paths should be relative to the user's home directory or absolute
LOCAL_PATH = os.path.join(os.path.expanduser("~"), "gdrive")
REMOTE_PATH = "gdrive:"
RCLONE_BIN_NAME = "rclone"  # Will be found in the system's PATH

# Logging
LOG_FILE = os.path.join(os.path.expanduser("~"), "rclone-bisync", "bisync-gdrive.log")

# Lock file to prevent concurrent script execution
LOCK_FILE = os.path.join(os.path.expanduser("~"), "rclone-bisync", "bisync_gdrive.lock")

# Desktop Notification Settings
NOTIFICATION_APP_NAME = "Rclone Bisync"
DBUS_VARS_FILE = os.path.join(os.path.expanduser("~"), ".config", "dbus-vars")

# Common rclone bisync arguments
RCLONE_COMMON_ARGS = [
    "--drive-export-formats", "docx,xlsx,pptx",
    "--drive-import-formats", "docx,xlsx,pptx",
    "--drive-skip-gdocs",
    "--remove-empty-dirs",
    "--check-access",
    "--fast-list",
    "--use-mmap",
    "--ignore-existing",
    "--workdir", os.path.join(os.path.expanduser("~"), ".cache", "rclone", "bisync"),
]
