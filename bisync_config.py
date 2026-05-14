#!/usr/bin/env python3

import os

# --- Paths ---
LOCAL_PATH = os.path.expanduser("~/gdrive")
REMOTE_PATH = "gdrive:"
LOG_FILE = os.path.join(os.path.expanduser("~"), "rclone-bisync", "bisync.log")
LOCK_FILE = os.path.join(os.path.expanduser("~"), "rclone-bisync", "bisync_gdrive.lock")

# --- Notification Settings ---
NOTIFICATION_APP_NAME = "Rclone Bisync"
SUCCESS_NOTIFICATION_INTERVAL_MIN = 30

# --- Rclone Arguments ---
RCLONE_COMMON_ARGS = [
    "--use-json-log",
    "--color", "NEVER",
    "--drive-export-formats", "docx,xlsx,pptx",
    "--drive-import-formats", "docx,xlsx,pptx",
    "--drive-skip-gdocs",
    "--drive-use-trash",
    "--check-access",
    "--remove-empty-dirs",
    "--resilient",
    "--recover",
    "--retries", "4",
    "--retries-sleep", "15s"
]

# --- Log Rotation Settings ---
MAX_LOG_SIZE_MB = 5
LOG_BACKUP_COUNT = 5

# --- Cleanup Settings ---
FILES_TO_CLEANUP = []