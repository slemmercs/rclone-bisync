"""
Rclone Bisync Configuration Loader
Environment-based configuration loader reading bisync.env and rc-secrets.env.
"""

import os
import shlex
from typing import List, Dict, Any

def _str2bool(val: Any, default: bool = False) -> bool:
    if val is None:
        return default
    if isinstance(val, bool):
        return val
    s = str(val).strip().lower()
    if s in ("true", "1", "t", "yes", "y", "on"):
        return True
    if s in ("false", "0", "f", "no", "n", "off", ""):
        return False
    return default


def load_env_file(filepath: str):
    """Loads key-value pairs from an environment file into os.environ."""
    if not os.path.exists(filepath):
        return
    try:
        with open(filepath, "r", encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line or line.startswith("#"):
                    continue
                if "=" in line:
                    key, val = line.split("=", 1)
                    key = key.strip()
                    val = val.strip().strip("\"'")
                    if key not in os.environ or os.environ[key] == "":
                        os.environ[key] = val
    except Exception:
        pass


# --- Base Directory Resolution ---
BASE_DIR = os.path.join(os.path.expanduser("~"), "rclone-bisync")

# Search locations for environment files: 1) BASE_DIR, 2) package root directory
_search_dirs = [
    BASE_DIR,
    os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
]

for _dir in _search_dirs:
    load_env_file(os.path.join(_dir, "rc-secrets.env"))
    load_env_file(os.path.join(_dir, "bisync.env"))

# --- Paths & Remote Mapping ---
LOCAL_PATH = os.path.expanduser(os.getenv("LOCAL_PATH", "~/gdrive"))
REMOTE_PATH = os.getenv("REMOTE_PATH", "gdrive:")

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
        "RC_PASS is not set in environment, bisync.env, or rc-secrets.env. "
        "Please configure RC_PASS in rc-secrets.env to ensure authenticated access to Rclone RC."
    )

# --- Notification Settings ---
NOTIFICATION_APP_NAME = os.getenv("NOTIFICATION_APP_NAME", "Rclone Bisync")
SUCCESS_NOTIFICATION_INTERVAL_MIN = float(os.getenv("SUCCESS_NOTIFICATION_INTERVAL_MIN", "30"))
ERROR_NOTIFICATION_REPEAT_HOURS = float(os.getenv("ERROR_NOTIFICATION_REPEAT_HOURS", "6"))

# --- Rclone Bisync Arguments (Parsed from single-line string via shlex) ---
_default_args = (
    "--use-json-log --color NEVER "
    "--drive-export-formats docx,xlsx,pptx --drive-import-formats docx,xlsx,pptx "
    "--drive-skip-gdocs --drive-use-trash --drive-acknowledge-abuse --drive-skip-shortcuts "
    "--check-access --check-first --track-renames --remove-empty-dirs --resilient --recover "
    "--retries 4 --retries-sleep 15s --low-level-retries 10 --tpslimit 10 --tpslimit-burst 10 "
    "--fast-list --buffer-size 32M --conflict-resolve newer --conflict-loser num"
)
_raw_args = os.getenv("RCLONE_COMMON_ARGS", _default_args)
RCLONE_COMMON_ARGS: List[str] = shlex.split(_raw_args)

# --- Log Rotation Settings ---
MAX_LOG_SIZE_MB = int(os.getenv("MAX_LOG_SIZE_MB", "5"))
LOG_BACKUP_COUNT = int(os.getenv("LOG_BACKUP_COUNT", "5"))

# --- Daemon Health & Startup ---
DAEMON_STARTUP_TIMEOUT_SEC = int(os.getenv("DAEMON_STARTUP_TIMEOUT_SEC", "30"))
DAEMON_HEALTH_CHECK_RETRIES = int(os.getenv("DAEMON_HEALTH_CHECK_RETRIES", "2"))
RETRY_DELAY = float(os.getenv("RETRY_DELAY", "1"))

# --- Job Monitoring Settings ---
JOB_POLL_INITIAL_SLEEP = float(os.getenv("JOB_POLL_INITIAL_SLEEP", "2"))
JOB_POLL_MAX_SLEEP = float(os.getenv("JOB_POLL_MAX_SLEEP", "10"))
JOB_POLL_BACKOFF_MULTIPLIER = float(os.getenv("JOB_POLL_BACKOFF_MULTIPLIER", "1.5"))
JOB_TIMEOUT_SECONDS = int(os.getenv("JOB_TIMEOUT_SECONDS", "1800"))

# --- Rate Limit Settings ---
RATE_LIMIT_BACKOFF_HOURS = float(os.getenv("RATE_LIMIT_BACKOFF_HOURS", "1"))
RATE_LIMIT_RETRY_AFTER_BACKOFF = _str2bool(os.getenv("RATE_LIMIT_RETRY_AFTER_BACKOFF", "true"))

# --- Concurrency & Process Lock File ---
LOCK_STALE_TIMEOUT_SECONDS = int(os.getenv("LOCK_STALE_TIMEOUT_SECONDS", "3600"))
CHECK_STALE_LOCKS = _str2bool(os.getenv("CHECK_STALE_LOCKS", "true"))

# --- Telemetry & Metrics Settings ---
ENABLE_METRICS = _str2bool(os.getenv("ENABLE_METRICS", "true"))
METRICS_MAX_ENTRIES = int(os.getenv("METRICS_MAX_ENTRIES", "1000"))
METRICS_MAX_BACKUPS = int(os.getenv("METRICS_MAX_BACKUPS", "5"))

# --- Post-Sync Cleanup Settings ---
_cleanup_raw = os.getenv("FILES_TO_CLEANUP", "")
FILES_TO_CLEANUP = [p.strip() for p in _cleanup_raw.split(",") if p.strip()]

# --- Validation Settings ---
VALIDATE_CONFIG_ON_STARTUP = _str2bool(os.getenv("VALIDATE_CONFIG_ON_STARTUP", "true"))
REQUIRE_LOCAL_PATH_EXISTS = _str2bool(os.getenv("REQUIRE_LOCAL_PATH_EXISTS", "true"))

# --- Advanced & External Alerts ---
DEBUG_MODE = _str2bool(os.getenv("DEBUG_MODE", "false"))
WEBHOOK_URL = os.getenv("WEBHOOK_URL") or None
WEBHOOK_TIMEOUT = int(os.getenv("WEBHOOK_TIMEOUT", "5"))

WATCHDOG_PING_URL = os.getenv("WATCHDOG_PING_URL") or None
WATCHDOG_TIMEOUT = int(os.getenv("WATCHDOG_TIMEOUT", "5"))

# --- Feature Flags ---
FEATURES: Dict[str, bool] = {
    "stale_lock_detection": _str2bool(os.getenv("FEATURE_STALE_LOCK_DETECTION", "true")),
    "rate_limit_backoff": _str2bool(os.getenv("FEATURE_RATE_LIMIT_BACKOFF", "true")),
    "daemon_health_check": _str2bool(os.getenv("FEATURE_DAEMON_HEALTH_CHECK", "true")),
    "metrics_collection": _str2bool(os.getenv("FEATURE_METRICS_COLLECTION", "true")),
    "smart_notifications": _str2bool(os.getenv("FEATURE_SMART_NOTIFICATIONS", "true")),
    "webhook_alerts": _str2bool(os.getenv("FEATURE_WEBHOOK_ALERTS", "false")),
}
