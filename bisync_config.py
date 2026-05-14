#!/usr/bin/env python3

import os

# --- Paths ---
# The local directory to be synced.
# The `os.path.expanduser` function handles dynamic paths (e.g., '~').
LOCAL_PATH = os.path.expanduser("~/gdrive")
# The remote Rclone path.
REMOTE_PATH = "gdrive:"
# The location for log files.
LOG_FILE = os.path.join(os.path.expanduser("~"), "rclone-bisync", "bisync.log")
# The lock file to prevent concurrent script execution.
LOCK_FILE = os.path.join(os.path.expanduser("~"), "rclone-bisync", "bisync_gdrive.lock")

# --- Notification Settings ---
# A friendly name for the notification application.
NOTIFICATION_APP_NAME = "Rclone Bisync"
# Set the minimum interval (in minutes) for sending a successful sync notification.
# A value of 0 will send a notification on every successful run.
SUCCESS_NOTIFICATION_INTERVAL_MIN = 30

# --- Rclone Arguments ---
# Common arguments passed to every `rclone bisync` command.
# It's recommended to include --dry-run for initial tests.
RCLONE_COMMON_ARGS = [
    "--use-json-log",
    "--drive-export-formats", "docx,xlsx,pptx",
    "--drive-import-formats", "docx,xlsx,pptx",
    "--drive-skip-gdocs",
    "--drive-use-trash",
    "--check-access",
    "--remove-empty-dirs",
    "--resilient",
    "--recover",
    "--retries", "4",
    "--retries-sleep", "15s",
    "--quiet"
]

# --- Log Rotation Settings ---
# Maximum size of a log file before rotation (in MB).
MAX_LOG_SIZE_MB = 5
# Number of backup log files to keep.
LOG_BACKUP_COUNT = 5

# --- Cleanup Settings ---
# A list of files to be cleaned up in case of a failed bisync run.
# This should include the rclone bisync workdir file, which is critical for
# preventing resync loops.
CLEANUP_FILES = [
    os.path.join(os.path.expanduser("~"), ".cache", "rclone", "bisync", "bisync_gdrive_advanced_workdir")
]
