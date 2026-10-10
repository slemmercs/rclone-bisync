#!/usr/bin/env python3
"""
Rclone Bisync Configuration
Centralized configuration for bidirectional Google Drive sync
"""

import os

# --- Base Directory ---
BASE_DIR = os.path.join(os.path.expanduser("~"), "rclone-bisync")

# Load rc-secrets.env to synchronize credentials with systemd service
_secrets_path = os.path.join(BASE_DIR, "rc-secrets.env")
if not os.path.exists(_secrets_path):
    # Fallback to local directory if running from repo workspace
    _local_secrets = os.path.join(os.path.dirname(os.path.abspath(__file__)), "rc-secrets.env")
    if os.path.exists(_local_secrets):
        _secrets_path = _local_secrets

if os.path.exists(_secrets_path):
    with open(_secrets_path, "r", encoding="utf-8") as _f:
        for _line in _f:
            _line = _line.strip()
            if "=" in _line and not _line.startswith("#"):
                _k, _v = _line.split("=", 1)
                os.environ[_k.strip()] = _v.strip().strip('"\'')

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
PENDING_DELETIONS_FILE = os.path.join(BASE_DIR, "pending_deletions.json")

# --- Rclone Daemon Settings ---
RC_HOST = os.getenv("RC_HOST", "localhost")
RC_PORT = int(os.getenv("RC_PORT", "5572"))
RC_API_URL = f"http://{RC_HOST}:{RC_PORT}"

RC_AUTH_ENABLED = True
RC_USER = os.getenv("RC_USER", "rclone")
RC_PASS = os.getenv("RC_PASS")
if RC_AUTH_ENABLED and not RC_PASS:
    raise ValueError(
        "RC_PASS is not set in environment or rc-secrets.env. "
        "Please configure RC_PASS in rc-secrets.env to ensure authenticated access to Rclone RC."
    )

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
METRICS_MAX_BACKUPS = 5             # Maximum number of rotated metric archives to retain

# --- Cleanup Settings ---
FILES_TO_CLEANUP = []

# --- Validation Settings ---
VALIDATE_CONFIG_ON_STARTUP = True   # Verify configuration before running
REQUIRE_LOCAL_PATH_EXISTS = True    # Fail if local path doesn't exist

# --- Advanced Settings ---
DEBUG_MODE = False
WEBHOOK_URL = None
WEBHOOK_TIMEOUT = 5

# External watchdog ("Dead-Man's Snitch") URL (e.g., Healthchecks.io, Uptime Kuma)
WATCHDOG_PING_URL = os.getenv("WATCHDOG_PING_URL", None)
WATCHDOG_TIMEOUT = int(os.getenv("WATCHDOG_TIMEOUT", "5"))

# --- Feature Flags ---
FEATURES = {
    "stale_lock_detection": True,
    "rate_limit_backoff": True,
    "daemon_health_check": True,
    "metrics_collection": True,
    "smart_notifications": True,
    "webhook_alerts": False,
}
