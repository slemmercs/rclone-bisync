#!/usr/bin/env python3
"""
Rclone Bisync Advanced Script
Bidirectional sync with Google Drive via Rclone daemon API
Includes error recovery, rate limit handling, and comprehensive monitoring
"""

import os
import sys
import time
import logging
import logging.handlers
import fcntl
import argparse
import json
import signal
import subprocess
import requests
from datetime import datetime
from pathlib import Path

# Import configuration
try:
    import bisync_config as config
except ImportError:
    print("ERROR: bisync_config.py not found. Ensure it's in the same directory.")
    sys.exit(1)

# Optional imports with fallback
try:
    import psutil
    PSUTIL_AVAILABLE = True
except ImportError:
    PSUTIL_AVAILABLE = False
    print("WARNING: psutil not installed. Stale lock detection will be limited.")
    print("Install with: pip install psutil")

try:
    from plyer import notification
    NOTIFICATIONS_AVAILABLE = True
except ImportError:
    NOTIFICATIONS_AVAILABLE = False
    print("WARNING: plyer not installed. Desktop notifications disabled.")
    print("Install with: pip install plyer")


# --- Global Variables ---
lock_file_handle = None
current_run_id = "unknown"
current_origin = "cli"


class JsonLogFormatter(logging.Formatter):
    """Formats log records as single-line JSON conforming to Rclone's JSON log schema."""
    def format(self, record):
        entry = {
            "time": datetime.fromtimestamp(record.created).astimezone().isoformat(),
            "level": record.levelname.lower(),
            "msg": record.getMessage(),
            "source": f"{record.filename}:{record.lineno}",
            "origin": getattr(record, "origin", current_origin),
            "run_id": getattr(record, "run_id", current_run_id)
        }
        if record.exc_info:
            entry["error"] = self.formatException(record.exc_info)
        return json.dumps(entry, ensure_ascii=False)


def setup_logging(verbose=False):
    """Configure logging with JSON file rotation and clean console output."""
    # Ensure log directory exists
    log_dir = os.path.dirname(config.LOG_FILE)
    os.makedirs(log_dir, mode=0o700, exist_ok=True)
    
    # Configure root logger
    root_logger = logging.getLogger()
    root_logger.setLevel(logging.DEBUG if (verbose or config.DEBUG_MODE) else logging.INFO)
    root_logger.handlers = []
    
    # 1. Rotating File Handler: UNIFIED JSON FORMAT (matches rclone-daemon.log)
    file_handler = logging.handlers.RotatingFileHandler(
        config.LOG_FILE,
        maxBytes=config.MAX_LOG_SIZE_MB * 1024 * 1024,
        backupCount=config.LOG_BACKUP_COUNT,
        encoding="utf-8"
    )
    file_handler.setFormatter(JsonLogFormatter())
    root_logger.addHandler(file_handler)
    
    # 2. Console Handler: Clean human-friendly text for interactive CLI runs
    console_formatter = logging.Formatter(
        "%(asctime)s [%(levelname)s] %(message)s",
        datefmt="%Y-%m-%d %H:%M:%S"
    )
    console_handler = logging.StreamHandler()
    console_handler.setFormatter(console_formatter)
    root_logger.addHandler(console_handler)
    
    logging.info("=" * 60)
    logging.info(f"Rclone Bisync Starting [Run: {current_run_id} | Origin: {current_origin}]")
    logging.info("=" * 60)


def load_json_file(filepath, default=None):
    """Load JSON file with error handling."""
    if default is None:
        default = {}
    
    # Ensure directory exists
    os.makedirs(os.path.dirname(filepath), exist_ok=True)
    
    if os.path.exists(filepath):
        try:
            with open(filepath, 'r') as f:
                return json.load(f)
        except (json.JSONDecodeError, IOError) as e:
            logging.warning(f"Failed to load {filepath}: {e}")
    
    return default


def write_json_file(filepath, data):
    """Write JSON file atomically with error handling."""
    tmp_path = f"{filepath}.tmp.{os.getpid()}"
    try:
        os.makedirs(os.path.dirname(filepath), exist_ok=True)
        with open(tmp_path, 'w') as f:
            json.dump(data, f, indent=2)
            f.flush()
            os.fsync(f.fileno())
        os.replace(tmp_path, filepath)
    except IOError as e:
        logging.error(f"Failed to write {filepath}: {e}")
        if os.path.exists(tmp_path):
            try:
                os.remove(tmp_path)
            except OSError:
                pass


def load_notification_state():
    """Load notification state from file."""
    return load_json_file(config.NOTIFICATION_STATE_FILE, {
        "last_success_timestamp": 0,
        "last_error_type": None,
        "last_error_timestamp": 0
    })


def write_notification_state(state):
    """Write notification state to file."""
    write_json_file(config.NOTIFICATION_STATE_FILE, state)


def send_desktop_notification(title, message, urgency='normal'):
    """Send desktop notification via native notify-send (Tuxedo OS / KDE) with plyer fallback."""
    # 1. Native notify-send on Linux (most reliable in KDE Wayland user sessions)
    notify_send_bin = "/usr/bin/notify-send"
    if os.path.exists(notify_send_bin):
        try:
            u = "critical" if urgency == 'critical' else "normal"
            subprocess.run(
                [notify_send_bin, "-a", config.NOTIFICATION_APP_NAME, "-u", u, "-i", "sync-synchronizing", title, message],
                check=False,
                timeout=3
            )
            logging.debug(f"Native notification sent via notify-send: {title}")
            return
        except Exception as e:
            logging.debug(f"notify-send call failed: {e}")

    # 2. Plyer fallback
    if not NOTIFICATIONS_AVAILABLE:
        logging.debug(f"Notification (disabled): {title}: {message}")
        return
    
    try:
        notification.notify(
            title=title,
            message=message,
            app_name=config.NOTIFICATION_APP_NAME,
            timeout=10,
            hints={'urgency': 2 if urgency == 'critical' else 1}
        )
        logging.debug(f"Notification sent via plyer: {title}")
    except Exception as e:
        logging.error(f"Failed to send notification: {e}")


def should_notify_error(err_type, notification_state):
    """Determine if we should send error notification based on state changes."""
    if not config.FEATURES.get("smart_notifications", True):
        return True  # Always notify if smart notifications disabled
    
    last_error = notification_state.get("last_error_type")
    last_error_time = notification_state.get("last_error_timestamp", 0)
    
    # Always notify new error types (state change)
    if err_type != last_error:
        return True
    
    # Re-notify persistent errors after configured interval
    hours_since_last = (time.time() - last_error_time) / 3600
    if hours_since_last >= config.ERROR_NOTIFICATION_REPEAT_HOURS:
        return True
    
    return False


def send_webhook(event_type, data):
    """Send webhook notification to external monitoring system."""
    if not config.WEBHOOK_URL or not config.FEATURES.get("webhook_alerts", False):
        return
    
    payload = {
        "event": event_type,
        "timestamp": time.time(),
        "hostname": os.uname().nodename,
        "data": data
    }
    
    try:
        response = requests.post(
            config.WEBHOOK_URL,
            json=payload,
            timeout=config.WEBHOOK_TIMEOUT
        )
        response.raise_for_status()
        logging.debug(f"Webhook sent: {event_type}")
    except Exception as e:
        logging.warning(f"Failed to send webhook: {e}")


def log_metrics(success, duration, error_type=None, files_synced=0):
    """Log metrics to JSONL file for analysis."""
    if not config.ENABLE_METRICS or not config.FEATURES.get("metrics_collection", True):
        return
    
    metrics = {
        "timestamp": time.time(),
        "datetime": datetime.now().isoformat(),
        "run_id": current_run_id,
        "origin": current_origin,
        "success": success,
        "duration_seconds": round(duration, 2),
        "error_type": error_type,
        "files_synced": files_synced
    }
    
    try:
        # Append to metrics file
        with open(config.METRICS_FILE, 'a') as f:
            json.dump(metrics, f)
            f.write('\n')
        
        # Rotate metrics file if too large
        if os.path.exists(config.METRICS_FILE):
            line_count = sum(1 for _ in open(config.METRICS_FILE))
            if line_count > config.METRICS_MAX_ENTRIES:
                backup_file = f"{config.METRICS_FILE}.{int(time.time())}"
                os.rename(config.METRICS_FILE, backup_file)
                logging.info(f"Metrics file rotated to {backup_file}")
    
    except IOError as e:
        logging.warning(f"Failed to write metrics: {e}")


def check_pid_exists(pid):
    """Check if a process with given PID exists."""
    if PSUTIL_AVAILABLE:
        try:
            return psutil.pid_exists(pid)
        except:
            pass
    
    # Fallback if psutil not available or fails
    try:
        os.kill(pid, 0)
        return True
    except OSError:
        return False


def acquire_lock():
    """Acquire file lock with stale lock detection.

    Race-condition fix: we must NOT truncate the lock file before holding the
    fcntl lock.  Using open('w') did exactly that — a second process could zero
    out the first process's PID before the first process even called lockf(),
    making stale-lock detection read an empty string and fail silently.

    Fix: open with O_RDWR | O_CREAT (no implicit truncation), acquire the
    exclusive lock, and only then truncate + write our PID under lock
    protection.
    """
    global lock_file_handle

    # Ensure lock directory exists
    os.makedirs(os.path.dirname(config.LOCK_FILE), mode=0o700, exist_ok=True)

    # Open without truncating — the existing PID must stay readable until we
    # actually hold the lock.
    fd = os.open(config.LOCK_FILE, os.O_RDWR | os.O_CREAT, 0o600)
    lock_file_handle = os.fdopen(fd, 'r+')

    try:
        fcntl.lockf(lock_file_handle, fcntl.LOCK_EX | fcntl.LOCK_NB)
        # We own the lock — safe to truncate and stamp our PID.
        lock_file_handle.seek(0)
        lock_file_handle.truncate()
        lock_file_handle.write(str(os.getpid()))
        lock_file_handle.flush()
        logging.debug(f"Lock acquired (PID: {os.getpid()})")
        return True

    except IOError:
        # Lock is held by another process
        if not config.FEATURES.get("stale_lock_detection", True) or not config.CHECK_STALE_LOCKS:
            logging.warning("Another instance is running (lock held).")
            return False

        # Check for stale lock
        if os.path.exists(config.LOCK_FILE):
            try:
                with open(config.LOCK_FILE, 'r') as f:
                    content = f.read().strip()
                    if content:
                        old_pid = int(content)

                        # Check if process exists
                        if not check_pid_exists(old_pid):
                            logging.warning(f"Removing stale lock (PID {old_pid} not running)")
                            lock_file_handle.close()
                            os.remove(config.LOCK_FILE)
                            return acquire_lock()  # Retry

                        # Check if lock is too old (process might be hung)
                        lock_age = time.time() - os.path.getmtime(config.LOCK_FILE)
                        if lock_age > config.LOCK_STALE_TIMEOUT_SECONDS:
                            logging.warning(
                                f"Lock is {lock_age/3600:.1f} hours old (PID {old_pid}). "
                                "Process may be hung. Removing stale lock."
                            )
                            lock_file_handle.close()
                            os.remove(config.LOCK_FILE)
                            return acquire_lock()  # Retry
                        else:
                            logging.warning(f"Another instance is running (PID {old_pid})")
                            return False

            except (ValueError, IOError) as e:
                logging.warning(f"Could not read lock file: {e}")
                return False

        logging.warning("Another instance is running.")
        return False


def release_lock():
    """Release file lock and clean up."""
    global lock_file_handle
    
    if lock_file_handle:
        try:
            fcntl.lockf(lock_file_handle, fcntl.LOCK_UN)
            lock_file_handle.close()
        except Exception as e:
            logging.warning(f"Error releasing lock: {e}")
    
    if os.path.exists(config.LOCK_FILE):
        try:
            os.remove(config.LOCK_FILE)
            logging.debug("Lock file removed")
        except Exception as e:
            logging.warning(f"Could not remove lock file: {e}")


def check_rate_limit_backoff():
    """Check if we're in rate limit backoff period."""
    if not config.FEATURES.get("rate_limit_backoff", True):
        return False
    
    if not os.path.exists(config.RATE_LIMIT_BACKOFF_FILE):
        return False
    
    try:
        with open(config.RATE_LIMIT_BACKOFF_FILE, 'r') as f:
            backoff_start = float(f.read().strip())
        
        elapsed_hours = (time.time() - backoff_start) / 3600
        
        if elapsed_hours < config.RATE_LIMIT_BACKOFF_HOURS:
            remaining = config.RATE_LIMIT_BACKOFF_HOURS - elapsed_hours
            logging.info(
                f"In rate limit backoff period. "
                f"{remaining:.1f} hours remaining. Skipping sync."
            )
            return True
        else:
            # Backoff period expired, remove flag
            os.remove(config.RATE_LIMIT_BACKOFF_FILE)
            logging.info("Rate limit backoff period expired. Resuming sync.")
            return False
    
    except (ValueError, IOError) as e:
        logging.warning(f"Error reading rate limit backoff file: {e}")
        # Remove corrupted file
        try:
            os.remove(config.RATE_LIMIT_BACKOFF_FILE)
        except:
            pass
        return False


def set_rate_limit_backoff():
    """Set rate limit backoff flag."""
    try:
        os.makedirs(os.path.dirname(config.RATE_LIMIT_BACKOFF_FILE), exist_ok=True)
        with open(config.RATE_LIMIT_BACKOFF_FILE, 'w') as f:
            f.write(str(time.time()))
        logging.info(f"Rate limit backoff set for {config.RATE_LIMIT_BACKOFF_HOURS} hour(s)")
    except IOError as e:
        logging.error(f"Failed to set rate limit backoff: {e}")


def check_daemon_health():
    """Check if rclone daemon is responsive.

    Writes health metrics to DAEMON_HEALTH_FILE on both success and failure so
    the dashboard reflects real daemon status.
    """
    if not config.FEATURES.get("daemon_health_check", True):
        return True, None

    retries = getattr(config, "DAEMON_HEALTH_CHECK_RETRIES", 2)
    retry_delay = getattr(config, "RETRY_DELAY", 1)
    last_error = "Daemon health check failed after all retries"

    for attempt in range(retries):
        try:
            auth = None
            if config.RC_AUTH_ENABLED:
                auth = (config.RC_USER, config.RC_PASS)

            response = requests.post(
                f"{config.RC_API_URL}/core/stats",
                timeout=2,
                auth=auth
            )

            if response.status_code == 200:
                health_data = {
                    "last_check": time.time(),
                    "status": "healthy",
                    "response_time_ms": round(response.elapsed.total_seconds() * 1000, 2)
                }
                write_json_file(config.DAEMON_HEALTH_FILE, health_data)
                logging.debug(f"Daemon health check passed (attempt {attempt + 1})")
                return True, None
            else:
                last_error = f"Daemon returned HTTP status {response.status_code}"
                logging.warning(f"Daemon health check non-200 response: {response.status_code}")

        except requests.exceptions.ConnectionError as e:
            last_error = (
                f"Rclone daemon is not running on {config.RC_API_URL}. "
                "Ensure rclone-daemon.service is started."
            )
            if attempt < retries - 1:
                logging.warning(
                    f"Daemon not responding (attempt {attempt + 1}/{retries}). "
                    f"Retrying in {retry_delay}s..."
                )
                time.sleep(retry_delay)

        except Exception as e:
            last_error = str(e)
            logging.error(f"Daemon health check error: {e}")
            if attempt < retries - 1:
                time.sleep(retry_delay)

    # Record offline state to DAEMON_HEALTH_FILE so telemetry shows OFFLINE
    health_data = {
        "last_check": time.time(),
        "status": "offline",
        "response_time_ms": 0.0,
        "error": last_error
    }
    write_json_file(config.DAEMON_HEALTH_FILE, health_data)
    logging.error(f"Daemon health check failed: {last_error}")
    return False, last_error


def classify_error(error_msg):
    """Classify error message into categories for appropriate handling."""
    if not error_msg:
        return "UNKNOWN"
    
    msg_lower = error_msg.lower()
    
    # Stale lock
    if "prior lock file found" in msg_lower or "lock" in msg_lower:
        return "STALE_LOCK"
    
    # Missing listings (needs resync).
    # Only match when rclone explicitly reports the prior listing files are absent:
    #   "cannot find prior path1 or path2 listings"
    # Do NOT match "bisync aborted" (generic fatal indicator present on every rclone
    # bisync failure) or bare "listing" (too broad — appears in rate-limit logs, etc.)
    if "cannot find prior" in msg_lower:
        return "NO_LISTINGS"
    
    # Rate limiting
    if any(x in msg_lower for x in ["rate", "429", "quota exceeded", "403 forbidden"]):
        return "RATE_LIMIT"
    
    # Network issues
    if any(x in msg_lower for x in ["timeout", "connection", "network", "unreachable"]):
        return "NETWORK"
    
    # Permission issues
    if any(x in msg_lower for x in ["permission denied", "401", "403", "unauthorized"]):
        return "PERMISSION"
    
    # File conflicts
    if "conflict" in msg_lower:
        return "CONFLICT"
    
    # Daemon issues
    if "daemon" in msg_lower or "rcd" in msg_lower:
        return "DAEMON_ERROR"

    # Too many deletes (might need --force)
    if "too many deletes" in msg_lower:
        return "TOO_MANY_DELETES"
    
    return "UNKNOWN"


def monitor_daemon_job(job_id):
    """
    Poll the daemon for job completion with exponential backoff.
    Returns: (success, error_type, error_detail, files_synced)

    The finished RC job/status response has the shape:
        {"finished": true, "success": true, "output": {"stats": {"transfers": N, ...}}}
    We extract ``transfers`` from there so the caller can pass a real value to
    log_metrics() instead of the hardcoded 0 it used before.
    """
    start_time = time.time()
    sleep_duration = config.JOB_POLL_INITIAL_SLEEP

    logging.info(f"Monitoring job {job_id}...")

    while True:
        # Check timeout
        elapsed = time.time() - start_time
        if elapsed > config.JOB_TIMEOUT_SECONDS:
            logging.error(f"Job {job_id} exceeded timeout ({config.JOB_TIMEOUT_SECONDS}s)")
            return False, "TIMEOUT", f"Job exceeded {config.JOB_TIMEOUT_SECONDS}s timeout", 0

        try:
            auth = None
            if config.RC_AUTH_ENABLED:
                auth = (config.RC_USER, config.RC_PASS)

            response = requests.post(
                f"{config.RC_API_URL}/job/status",
                json={"jobid": job_id},
                timeout=10,
                auth=auth
            )
            response.raise_for_status()
            status = response.json()

            if status.get("finished"):
                success = status.get("success")
                error_msg = status.get("error", "")

                # Extract transfer count from the finished job output.
                # Rclone embeds stats directly under output (or output.stats in some commands).
                files_synced = 0
                try:
                    output_data = status.get("output") or {}
                    files_synced = int(
                        output_data.get("transfers")
                        if output_data.get("transfers") is not None
                        else output_data.get("stats", {}).get("transfers", 0)
                    )
                except (TypeError, ValueError):
                    pass

                if success:
                    duration = status.get("duration", elapsed)
                    logging.info(
                        f"Job {job_id} completed successfully in {duration:.1f}s "
                        f"({files_synced} file(s) transferred)"
                    )
                    return True, None, None, files_synced

                # Error classification
                err_type = classify_error(error_msg)
                logging.error(f"Job {job_id} failed: {err_type} - {error_msg}")
                return False, err_type, error_msg, files_synced

            # Job still running
            logging.debug(f"Job {job_id} still running... (elapsed: {elapsed:.1f}s)")
            time.sleep(min(sleep_duration, config.JOB_POLL_MAX_SLEEP))
            sleep_duration *= config.JOB_POLL_BACKOFF_MULTIPLIER

        except requests.exceptions.RequestException as e:
            logging.error(f"Error polling job status: {e}")
            return False, "MONITOR_ERROR", str(e), 0

        except Exception as e:
            logging.error(f"Unexpected error monitoring job: {e}")
            return False, "MONITOR_ERROR", str(e), 0


# Mapping of CLI arguments to Rclone Remote Control (RC) API parameters.
# Top-level bisync parameters are applied directly to /sync/bisync.
# Engine options are passed within the '_config' dictionary.
CLI_TO_RC_MAP = {
    # Bisync-specific options
    "--conflict-resolve": ("param", "conflictResolve"),
    "--conflict-loser": ("param", "conflictLoser"),
    "--conflict-suffix": ("param", "conflictSuffix"),
    "--remove-empty-dirs": ("param_bool", "removeEmptyDirs"),
    "--resilient": ("param_bool", "resilient"),
    "--recover": ("param_bool", "recover"),
    "--check-access": ("param_bool", "checkAccess"),
    "--compare": ("param", "compare"),
    
    # Engine configuration options passed in '_config'
    "--check-first": ("config_bool", "CheckFirst"),
    "--track-renames": ("config_bool", "TrackRenames"),
    "--fast-list": ("config_bool", "UseListR"),
    "--retries": ("config_int", "Retries"),
    "--retries-sleep": ("config_str", "RetriesInterval"),
    "--low-level-retries": ("config_int", "LowLevelRetries"),
    "--tpslimit": ("config_float", "TPSLimit"),
    "--tpslimit-burst": ("config_int", "TPSLimitBurst"),
    "--buffer-size": ("config_str", "BufferSize"),
    "--color": ("config_str", "TerminalColorMode"),
    # Backend-specific options applied to remote connection path
    "--drive-export-formats": ("backend_str", "export_formats"),
    "--drive-import-formats": ("backend_str", "import_formats"),
    "--drive-skip-gdocs": ("backend_bool", "skip_gdocs"),
    "--drive-use-trash": ("backend_bool", "use_trash"),
    "--drive-acknowledge-abuse": ("backend_bool", "acknowledge_abuse"),
    "--drive-skip-shortcuts": ("backend_bool", "skip_shortcuts"),
    "--use-json-log": ("config_bool", "LogJSON")
}


def build_bisync_rc_payload(resync_attempt=False, force_attempt=False, dry_run_attempt=False):
    """
    Translates RCLONE_COMMON_ARGS into valid parameters, backend connection
    string options, and _config settings for the Rclone Remote Control (RC)
    API endpoint (/sync/bisync).
    """
    bisync_params = {
        "path1": config.REMOTE_PATH,
        "path2": config.LOCAL_PATH,
        "resync": bool(resync_attempt),
        "force": bool(force_attempt),
        "dryRun": bool(dry_run_attempt),
    }
    engine_config = {}
    backend_options = []

    args_list = getattr(config, "RCLONE_COMMON_ARGS", [])
    i = 0
    while i < len(args_list):
        arg = args_list[i]
        if arg in CLI_TO_RC_MAP:
            action, target_key = CLI_TO_RC_MAP[arg]
            if action == "param_bool":
                bisync_params[target_key] = True
            elif action == "config_bool":
                engine_config[target_key] = True
            elif action == "backend_bool":
                backend_options.append(f"{target_key}=true")
            elif action == "backend_str":
                i += 1
                if i < len(args_list):
                    val = args_list[i]
                    if "," in val or " " in val:
                        backend_options.append(f'{target_key}="{val}"')
                    else:
                        backend_options.append(f"{target_key}={val}")
            elif action == "param":
                i += 1
                if i < len(args_list):
                    bisync_params[target_key] = args_list[i]
            elif action == "config_str":
                i += 1
                if i < len(args_list):
                    engine_config[target_key] = args_list[i]
            elif action == "config_int":
                i += 1
                if i < len(args_list):
                    try:
                        engine_config[target_key] = int(args_list[i])
                    except ValueError:
                        engine_config[target_key] = args_list[i]
            elif action == "config_float":
                i += 1
                if i < len(args_list):
                    try:
                        engine_config[target_key] = float(args_list[i])
                    except ValueError:
                        engine_config[target_key] = args_list[i]
        else:
            if not arg.startswith("-"):
                logging.warning(f"Unexpected argument in RCLONE_COMMON_ARGS: {arg}")
            else:
                if i + 1 < len(args_list) and not args_list[i + 1].startswith("-"):
                    logging.warning(f"Unmapped CLI option '{arg} {args_list[i + 1]}' skipped for RC API")
                    i += 1
                else:
                    logging.warning(f"Unmapped CLI flag '{arg}' skipped for RC API")
        i += 1

    # Attach backend-specific connection string parameters to path1
    if backend_options:
        remote_path = bisync_params["path1"]
        if remote_path.endswith(":"):
            base_remote = remote_path[:-1]
            opts_str = ",".join(backend_options)
            bisync_params["path1"] = f"{base_remote},{opts_str}:"

    # Safety check: checkAccess strictly requires RCLONE_TEST on both local and remote
    if bisync_params.get("checkAccess"):
        test_file = os.path.join(config.LOCAL_PATH, "RCLONE_TEST")
        if not os.path.exists(test_file):
            logging.warning(
                f"--check-access requested, but 'RCLONE_TEST' not found in {config.LOCAL_PATH}. "
                "Disabling checkAccess to prevent sync abort."
            )
            bisync_params["checkAccess"] = False

    if engine_config:
        bisync_params["_config"] = engine_config

    return bisync_params


def run_rclone_bisync(resync_attempt=False, force_attempt=False, dry_run_attempt=False):
    """
    Trigger bisync via RC API.
    Returns: (success, error_type, error_detail, files_synced)
    """
    # Check daemon health first
    daemon_ok, daemon_error = check_daemon_health()
    if not daemon_ok:
        return False, "DAEMON_OFFLINE", daemon_error, 0

    # Prepare API request
    url = f"{config.RC_API_URL}/sync/bisync?_async=true"

    # Build proper RC parameter payload
    params = build_bisync_rc_payload(resync_attempt=resync_attempt, force_attempt=force_attempt, dry_run_attempt=dry_run_attempt)

    if dry_run_attempt:
        logging.info("Simulating bidirectional sync in DRY-RUN mode (no filesystem modifications will be made).")
    if force_attempt:
        logging.info("Bypassing delete safety checks using --force flag.")
    if resync_attempt:
        logging.info("Requesting full database state reconstruction using --resync flag.")

    try:
        logging.info(f"Sending bisync command to daemon ({config.RC_API_URL})...")
        logging.info(f"  Remote: {params.get('path1')}")
        logging.info(f"  Local:  {params.get('path2')}")
        logging.info(f"  Resync: {params.get('resync')}")
        logging.info(f"  Force:  {params.get('force')}")
        if "conflictResolve" in params:
            logging.info(f"  Conflict Resolution: {params.get('conflictResolve')} (loser: {params.get('conflictLoser', 'num')})")
        if "_config" in params:
            logging.debug(f"  _config options: {json.dumps(params['_config'])}")

        auth = None
        if config.RC_AUTH_ENABLED:
            auth = (config.RC_USER, config.RC_PASS)

        response = requests.post(
            url,
            json=params,
            timeout=30,
            auth=auth
        )
        response.raise_for_status()
        response_json = response.json()

        logging.debug(f"Daemon response: {response_json}")

        job_id = response_json.get("jobid")

        if job_id is not None:
            logging.info(f"Bisync job started (Job ID: {job_id})")
            return monitor_daemon_job(int(job_id))
        else:
            # Synchronous completion (rare) — no stats available in this path
            logging.info("Bisync completed synchronously")
            return True, None, None, 0

    except requests.exceptions.ConnectionError:
        error_msg = f"Cannot connect to rclone daemon at {config.RC_API_URL}"
        logging.error(error_msg)
        return False, "DAEMON_OFFLINE", error_msg, 0

    except requests.exceptions.HTTPError as e:
        try:
            err_json = e.response.json()
            error_msg = err_json.get("error", e.response.text)
        except Exception:
            error_msg = e.response.text or str(e)
        err_type = classify_error(error_msg)
        logging.error(f"Daemon HTTP error: {err_type} - {error_msg}")
        return False, err_type, error_msg, 0

    except requests.exceptions.Timeout:
        error_msg = "Request to daemon timed out"
        logging.error(error_msg)
        return False, "DAEMON_TIMEOUT", error_msg, 0

    except requests.exceptions.RequestException as e:
        error_msg = f"Request error: {e}"
        logging.error(error_msg)
        return False, "DAEMON_ERROR", error_msg, 0

    except Exception as e:
        error_msg = f"Unexpected error: {e}"
        logging.error(error_msg)
        return False, "UNKNOWN", error_msg, 0


def validate_configuration():
    """Validate configuration before running."""
    if not config.VALIDATE_CONFIG_ON_STARTUP:
        return True
    
    logging.info("Validating configuration...")
    
    # Check if local path exists
    if config.REQUIRE_LOCAL_PATH_EXISTS:
        if not os.path.exists(config.LOCAL_PATH):
            logging.error(f"Local path does not exist: {config.LOCAL_PATH}")
            return False
        if not os.path.isdir(config.LOCAL_PATH):
            logging.error(f"Local path is not a directory: {config.LOCAL_PATH}")
            return False
    
    # Check if base directory is writable
    try:
        os.makedirs(config.BASE_DIR, mode=0o700, exist_ok=True)
        test_file = os.path.join(config.BASE_DIR, ".write_test")
        with open(test_file, 'w') as f:
            f.write("test")
        os.remove(test_file)
    except Exception as e:
        logging.error(f"Base directory is not writable: {config.BASE_DIR}: {e}")
        return False
    
    # Check daemon connectivity and warn about port mismatch
    logging.info(f"Checking daemon connectivity at {config.RC_API_URL}...")
    daemon_ok, daemon_error = check_daemon_health()
    if not daemon_ok:
        logging.error(f"Daemon health check failed: {daemon_error}")
        logging.error("")
        logging.error("Common causes:")
        logging.error("  1. rclone-daemon.service is not running")
        logging.error("     Fix: systemctl --user start rclone-daemon.service")
        logging.error("")
        logging.error("  2. Port mismatch between bisync_config.py and rclone-daemon.service")
        logging.error(f"     Config expects: {config.RC_API_URL}")
        logging.error("     Service file should have: --rc-addr localhost:5572")
        logging.error("     Check with: systemctl --user cat rclone-daemon.service | grep rc-addr")
        logging.error("")
        logging.error("  3. Daemon is still starting up")
        logging.error("     Wait a few seconds and try again")
        logging.error("")
        return False
    
    logging.info("Configuration validation passed")
    return True


def cleanup_on_success():
    """Perform cleanup tasks after successful sync."""
    if not config.FILES_TO_CLEANUP:
        return
    
    logging.info("Running post-sync cleanup...")
    import glob
    
    for pattern in config.FILES_TO_CLEANUP:
        try:
            files = glob.glob(os.path.join(config.LOCAL_PATH, pattern), recursive=True)
            for file in files:
                if os.path.isfile(file):
                    os.remove(file)
                    logging.info(f"Cleaned up: {file}")
        except Exception as e:
            logging.warning(f"Cleanup failed for {pattern}: {e}")


def signal_handler(signum, frame):
    """Handle SIGTERM gracefully.

    Exit with code 143 (128 + 15) so systemd and the calling shell correctly
    distinguish a signal termination from a clean exit (0) or a runtime
    failure (1).

    SIGINT is intentionally NOT registered here — letting it raise
    KeyboardInterrupt allows main()'s ``except KeyboardInterrupt`` block to
    run (logging + metrics) before exiting with code 130.
    """
    logging.warning(f"Received signal {signum} (SIGTERM). Cleaning up...")
    release_lock()
    sys.exit(143)  # 128 + SIGTERM(15)


def main():
    """Main entry point."""
    # Set up signal handlers.
    # SIGTERM → signal_handler (exits 143 so systemd sees a proper kill)
    # SIGINT  → default Python behaviour (raises KeyboardInterrupt, caught below)
    signal.signal(signal.SIGTERM, signal_handler)
    
    # Parse arguments
    parser = argparse.ArgumentParser(
        description="Rclone Bisync Client - Bidirectional Google Drive Sync",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s                    # Normal sync run
  %(prog)s --verbose          # Verbose logging
  %(prog)s --resync           # Force full resync (use with caution)
  %(prog)s --no-notify        # Disable desktop notifications
  %(prog)s --force            # Skip rate limit backoff

Configuration:
  Edit bisync_config.py to change sync paths, daemon port, and other settings.
  
  IMPORTANT: Ensure RC_PORT in bisync_config.py matches --rc-addr port
  in rclone-daemon.service file.

Logs:
  %(prog)s logs are written to: {}
  Daemon logs: ~/rclone-bisync/rclone-daemon.log
  System logs: journalctl --user -t bisync-gdrive
        """.format(config.LOG_FILE)
    )
    parser.add_argument("--verbose", "-v", action="store_true",
                        help="Enable verbose logging")
    parser.add_argument("--resync", "-r", action="store_true",
                        help="Force a full resync (rebuilds sync state)")
    parser.add_argument("--no-notify", "-n", action="store_true",
                        help="Disable desktop notifications")
    parser.add_argument("--force", "-f", action="store_true",
                        help="Skip rate limit backoff check and force sync")
    parser.add_argument("--dry-run", "-d", action="store_true",
                        help="Simulate sync without making any changes (dry run)")
    parser.add_argument("--origin", choices=["timer", "dashboard", "cli", "manual"], default="cli",
                        help="Execution origin tag (default: cli)")
    args = parser.parse_args()
    
    # Generate unique run ID and track origin
    global current_run_id, current_origin
    current_origin = args.origin
    current_run_id = f"sync_{datetime.now().strftime('%y%m%d_%H%M%S')}_{os.urandom(2).hex()}"

    # Setup logging
    setup_logging(args.verbose)
    
    # Log script info
    logging.info(f"Script: {os.path.abspath(__file__)}")
    logging.info(f"PID: {os.getpid()}")
    logging.info(f"User: {os.getenv('USER', 'unknown')}")
    
    # Validate configuration
    if not validate_configuration():
        logging.error("Configuration validation failed. Exiting.")
        sys.exit(1)
    
    # Load notification state
    notification_state = load_notification_state()
    
    # Check rate limit backoff
    if not args.force and check_rate_limit_backoff():
        logging.info("Exiting due to rate limit backoff. Use --force to override.")
        sys.exit(0)
    
    # Acquire lock
    if not acquire_lock():
        logging.info("Exiting due to existing lock.")
        sys.exit(0)
    
    start_time = time.time()
    success = False
    err_type = None
    err_detail = None
    files_synced = 0

    try:
        # Run sync
        success, err_type, err_detail, files_synced = run_rclone_bisync(
            resync_attempt=args.resync,
            force_attempt=args.force,
            dry_run_attempt=args.dry_run
        )
        duration = time.time() - start_time

        if args.dry_run:
            logging.info("=" * 60)
            logging.info(f"DRY-RUN SIMULATION COMPLETED ({'SUCCESS' if success else 'FAILURE'})")
            if success:
                logging.info(f"Simulation completed cleanly in {duration:.1f}s (no files modified).")
            else:
                logging.warning(f"Simulation identified issue: {err_type} - {err_detail}")
            return

        if success:
            # ============================================================
            # SUCCESS PATH
            # ============================================================
            logging.info("=" * 60)
            logging.info("SYNC COMPLETED SUCCESSFULLY")

            # Cleanup
            cleanup_on_success()

            # Log metrics
            log_metrics(success=True, duration=duration, files_synced=files_synced)
            
            # Send webhook
            send_webhook("sync_success", {
                "duration": duration,
                "local_path": config.LOCAL_PATH,
                "remote_path": config.REMOTE_PATH
            })
            
            # Desktop notification (throttled)
            last_success = notification_state.get("last_success_timestamp", 0)
            time_since_last = (time.time() - last_success) / 60
            
            # Always update success timestamp for accurate dashboard telemetry
            notification_state["last_success_timestamp"] = time.time()
            notification_state["last_error_type"] = None
            write_notification_state(notification_state)
            
            if not args.no_notify and time_since_last >= config.SUCCESS_NOTIFICATION_INTERVAL_MIN:
                send_desktop_notification(
                    "Sync Success",
                    f"Google Drive synchronized successfully.\nDuration: {duration:.1f}s"
                )
            else:
                logging.debug(
                    f"Success notification skipped (last notification {time_since_last:.1f} min ago)"
                )
        
        else:
            # ============================================================
            # ERROR PATH
            # ============================================================
            logging.error("=" * 60)
            logging.error("SYNC FAILED")
            logging.error("=" * 60)
            logging.error(f"Error Type: {err_type}")
            logging.error(f"Error Detail: {err_detail}")
            
            # Log metrics
            log_metrics(success=False, duration=duration, error_type=err_type)
            
            # Send webhook
            send_webhook("sync_failure", {
                "error_type": err_type,
                "error_detail": err_detail,
                "duration": duration
            })
            
            # Handle specific error types
            if err_type == "RATE_LIMIT":
                logging.warning("Rate limit detected. Setting backoff period.")
                set_rate_limit_backoff()
                
                if not args.no_notify and should_notify_error(err_type, notification_state):
                    send_desktop_notification(
                        "Sync Deferred - Rate Limit",
                        f"Google Drive rate limit reached.\n"
                        f"Will resume in {config.RATE_LIMIT_BACKOFF_HOURS} hour(s).",
                        urgency="normal"
                    )
                    notification_state["last_error_type"] = err_type
                    notification_state["last_error_timestamp"] = time.time()
                    write_notification_state(notification_state)
            
            elif err_type == "NO_LISTINGS":
                if args.resync:
                    # Guard clause: If it failed even though --resync was set, stop to prevent an infinite loop
                    logging.error("Sync failed during an explicit resync execution. Manual intervention required.")
                else:
                    logging.warning("⚠️ Prior listings missing or corrupted. Triggering automated self-healing resync...")
                    if not args.no_notify:
                        send_desktop_notification(
                            "Sync Auto-Recovery",
                            "Cache tracking missing. Rebuilding synchronization baseline states...",
                            urgency="normal"
                        )
                    
                    # Intercept execution: re-run the transaction immediately with resync activated
                    retry_success, retry_err, retry_detail, retry_files = run_rclone_bisync(resync_attempt=True, force_attempt=args.force)

                    if retry_success:
                        logging.info("✅ Auto-recovery successful! Baseline sync state successfully rebuilt.")
                        success = True
                        duration = time.time() - start_time
                        cleanup_on_success()

                        # Update notification and telemetry states
                        notification_state["last_error_type"] = None
                        notification_state["last_success_timestamp"] = time.time()
                        write_notification_state(notification_state)

                        log_metrics(success=True, duration=duration, files_synced=retry_files)

                        send_webhook("sync_success", {
                            "duration": duration,
                            "local_path": config.LOCAL_PATH,
                            "remote_path": config.REMOTE_PATH,
                            "recovered": True
                        })

                        if not args.no_notify:
                            send_desktop_notification(
                                "Sync Auto-Recovery Complete",
                                f"Baseline sync state rebuilt successfully.\nDuration: {duration:.1f}s"
                            )
                        return
                    else:
                        # Re-assign error values if the recovery attempt also dropped out
                        err_type, err_detail = retry_err, retry_detail
                        logging.error(f"Auto-recovery failed: {err_type} - {err_detail}")
            
            elif err_type == "STALE_LOCK":
                logging.warning("Stale lock detected by rclone bisync.")
                logging.warning("The lock file will be automatically cleaned on next successful run.")
                
                if not args.no_notify and should_notify_error(err_type, notification_state):
                    send_desktop_notification(
                        "Sync Error - Stale Lock",
                        "Prior sync lock found.\n"
                        "Will retry automatically.",
                        urgency="normal"
                    )
                    notification_state["last_error_type"] = err_type
                    notification_state["last_error_timestamp"] = time.time()
                    write_notification_state(notification_state)
            
            elif err_type == "DAEMON_OFFLINE":
                logging.error("Rclone daemon is not running.")
                logging.error("Start it with: systemctl --user start rclone-daemon.service")
                
                if not args.no_notify and should_notify_error(err_type, notification_state):
                    send_desktop_notification(
                        "Sync Error - Daemon Offline",
                        "Rclone daemon is not running.\n"
                        "Start: systemctl --user start rclone-daemon.service",
                        urgency="critical"
                    )
                    notification_state["last_error_type"] = err_type
                    notification_state["last_error_timestamp"] = time.time()
                    write_notification_state(notification_state)
            
            elif err_type == "NETWORK":
                logging.warning("Network error detected. Will retry on next scheduled run.")
                
                if not args.no_notify and should_notify_error(err_type, notification_state):
                    send_desktop_notification(
                        "Sync Error - Network Issue",
                        "Network connectivity problem.\n"
                        "Will retry automatically.",
                        urgency="normal"
                    )
                    notification_state["last_error_type"] = err_type
                    notification_state["last_error_timestamp"] = time.time()
                    write_notification_state(notification_state)
            
            elif err_type == "PERMISSION":
                logging.error("Permission error. Check rclone authentication.")
                logging.error("You may need to re-authenticate with: rclone config")
                
                if not args.no_notify and should_notify_error(err_type, notification_state):
                    send_desktop_notification(
                        "Sync Error - Permission Denied",
                        "Authentication or permission error.\n"
                        "Run: rclone config reconnect gdrive:",
                        urgency="critical"
                    )
                    notification_state["last_error_type"] = err_type
                    notification_state["last_error_timestamp"] = time.time()
                    write_notification_state(notification_state)

            elif err_type == "TOO_MANY_DELETES":
                logging.error("Safety abort: Mass deletions detected.")
                logging.error("To authorize these deletions, run the script manually:")
                logging.error("  python3 ~/rclone-bisync/bisync_gdrive_advanced.py --force")
                
                if not args.no_notify and should_notify_error(err_type, notification_state):
                    send_desktop_notification(
                        "Sync Safety Abort",
                        "Mass deletions detected.\nRun with --force to approve.",
                        urgency="critical"
                    )
                    notification_state["last_error_type"] = err_type
                    notification_state["last_error_timestamp"] = time.time()
                    write_notification_state(notification_state)
            
            else:
                # Generic error handling
                if not args.no_notify and should_notify_error(err_type, notification_state):
                    send_desktop_notification(
                        f"Sync Error - {err_type}",
                        f"{err_detail[:200]}",
                        urgency="critical"
                    )
                    notification_state["last_error_type"] = err_type
                    notification_state["last_error_timestamp"] = time.time()
                    write_notification_state(notification_state)
    
    except KeyboardInterrupt:
        logging.warning("Sync interrupted by user (Ctrl+C)")
        duration = time.time() - start_time
        log_metrics(success=False, duration=duration, error_type="INTERRUPTED")
        sys.exit(130)  # Standard exit code for Ctrl+C
    
    except Exception as e:
        # Unexpected exception
        logging.exception("Unexpected exception occurred:")
        duration = time.time() - start_time
        log_metrics(success=False, duration=duration, error_type="EXCEPTION")
        
        if not args.no_notify:
            send_desktop_notification(
                "Sync Error - Unexpected",
                f"Unexpected error: {str(e)[:200]}",
                urgency="critical"
            )
        
        send_webhook("sync_exception", {
            "error": str(e),
            "duration": duration
        })
        
        sys.exit(1)
    
    finally:
        # Always release lock
        release_lock()
        
        # Final summary
        duration = time.time() - start_time
        logging.info("=" * 60)
        logging.info(f"Total execution time: {duration:.2f}s")
        logging.info(f"Exit status: {'SUCCESS' if success else 'FAILURE'}")
        logging.info("=" * 60)
        
        # Exit with appropriate code
        sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()
