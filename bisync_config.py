#!/usr/bin/env python3
"""
Rclone Bisync Configuration
Centralized configuration for bidirectional Google Drive sync
"""

import os

# --- Base Directory ---
# All sync-related files are stored here for consistency
BASE_DIR = os.path.join(os.path.expanduser("~"), "rclone-bisync")

# --- Paths ---
LOCAL_PATH = os.path.expanduser("~/gdrive")
REMOTE_PATH = "gdrive:"

# --- File Locations ---
LOG_FILE = os.path.join(BASE_DIR, "bisync.log")
LOCK_FILE = os.path.join(BASE_DIR, "bisync_gdrive.lock")
NOTIFICATION_STATE_FILE = os.path.join(BASE_DIR, "notification_state.json")
METRICS_FILE = os.path.join(BASE_DIR, "metrics.jsonl")
RATE_LIMIT_BACKOFF_FILE = os.path.join(BASE_DIR, "rate_limit.flag")
DAEMON_HEALTH_FILE = os.path.join(BASE_DIR, "daemon_health.json")

# --- Rclone Daemon Settings ---
# CRITICAL: These settings MUST match the rclone-daemon.service configuration
# 
# In rclone-daemon.service, the ExecStart line contains:
#   --rc-addr localhost:5572
# 
# The RC_HOST and RC_PORT values below must match that setting exactly.
# If you change the port in the service file, update RC_PORT here as well.
# 
# To verify: systemctl --user cat rclone-daemon.service | grep rc-addr

RC_HOST = "localhost"
RC_PORT = 5572  # Must match --rc-addr in rclone-daemon.service
RC_API_URL = f"http://{RC_HOST}:{RC_PORT}"

# Optional: Enable if you configured authentication in rclone-daemon.service
# If you uncommented --rc-user and --rc-pass in the service file, set this to True
# and update the credentials below to match
RC_AUTH_ENABLED = True
RC_USER = "csaba"         # Must match --rc-user in rclone-daemon.service
RC_PASS = "W4rl0ck86+"  # Must match --rc-pass in rclone-daemon.service

# --- Notification Settings ---
NOTIFICATION_APP_NAME = "Rclone Bisync"
SUCCESS_NOTIFICATION_INTERVAL_MIN = 30  # Only notify success every 30 min
ERROR_NOTIFICATION_REPEAT_HOURS = 6     # Re-notify persistent errors every 6 hours

# --- Rclone Arguments ---
RCLONE_COMMON_ARGS = [
    # Logging
    "--use-json-log",
    "--color", "NEVER",
    
    # Google Drive specific
    "--drive-export-formats", "docx,xlsx,pptx",
    "--drive-import-formats", "docx,xlsx,pptx",
    "--drive-skip-gdocs",           # Skip non-downloadable Google Docs
    "--drive-use-trash",            # Move to trash instead of permanent delete
    "--drive-acknowledge-abuse",    # Download files marked as abusive
    "--drive-skip-shortcuts",
    
    # Bisync reliability
    # Note: --check-access requires an 'RCLONE_TEST' file on both local and remote.
    # Uncomment only if you have created RCLONE_TEST on both paths.
    "--check-access",
    "--check-first",                # Compare before copying (saves bandwidth)
    "--track-renames",              # Detect renames to avoid re-upload
    "--remove-empty-dirs",          # Clean up empty directories
    "--resilient",                  # Continue on errors
    "--recover",                    # Automatically recover from interruptions
    
    # Retry logic
    "--retries", "4",
    "--retries-sleep", "15s",
    "--low-level-retries", "10",
    "--tpslimit", "10",
    "--tpslimit-burst", "10",
    
    # Performance
    "--fast-list",                  # Use recursive list if possible
    "--buffer-size", "32M",         # Larger buffer for better performance
    
    # Conflict handling
    "--conflict-resolve", "newer",  # Use newer file in conflicts
    "--conflict-loser", "num",      # Rename losing file with number suffix
]

# --- Log Rotation Settings ---
MAX_LOG_SIZE_MB = 5
LOG_BACKUP_COUNT = 5

# --- Daemon Settings ---
DAEMON_STARTUP_TIMEOUT_SEC = 30     # How long to wait for daemon to be ready
DAEMON_HEALTH_CHECK_RETRIES = 2     # Fast retry attempts before reporting offline
RETRY_DELAY = 1                     # Delay between health check retries (seconds)

# --- Job Monitoring Settings ---
JOB_POLL_INITIAL_SLEEP = 2          # Initial polling interval (seconds)
JOB_POLL_MAX_SLEEP = 10             # Maximum polling interval (seconds)
JOB_POLL_BACKOFF_MULTIPLIER = 1.5   # Exponential backoff multiplier
JOB_TIMEOUT_SECONDS = 1800          # 30 minutes max for any job

# --- Rate Limit Settings ---
RATE_LIMIT_BACKOFF_HOURS = 1        # Wait 1 hour after rate limit hit
RATE_LIMIT_RETRY_AFTER_BACKOFF = True  # Whether to retry automatically

# --- Lock File Settings ---
LOCK_STALE_TIMEOUT_SECONDS = 3600   # Consider lock stale after 1 hour
CHECK_STALE_LOCKS = True            # Enable stale lock detection

# --- Metrics Settings ---
ENABLE_METRICS = True               # Enable metrics collection
METRICS_MAX_ENTRIES = 1000          # Rotate metrics file after this many entries

# --- Cleanup Settings ---
# Files to clean up on successful sync (e.g., old conflict files)
# Example: ["**/*conflict*.txt", "**/*.tmp"]
FILES_TO_CLEANUP = []

# --- Validation Settings ---
VALIDATE_CONFIG_ON_STARTUP = True   # Verify configuration before running
REQUIRE_LOCAL_PATH_EXISTS = True    # Fail if local path doesn't exist

# --- Advanced Settings ---
# Enable debug mode (more verbose logging)
DEBUG_MODE = False

# Webhook URL for external monitoring (set to None to disable)
WEBHOOK_URL = None
WEBHOOK_TIMEOUT = 5  # Seconds to wait for webhook response

# --- Feature Flags ---
FEATURES = {
    "stale_lock_detection": True,
    "rate_limit_backoff": True,
    "daemon_health_check": True,
    "metrics_collection": True,
    "smart_notifications": True,
    "webhook_alerts": False,  # Enable if WEBHOOK_URL is set
}
