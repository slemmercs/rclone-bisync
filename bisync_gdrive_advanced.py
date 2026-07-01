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
import re
import argparse
import json
import signal
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


def setup_logging(verbose=False):
    """Configure logging with rotation and appropriate verbosity."""
    log_formatter = logging.Formatter(
        "%(asctime)s [%(levelname)s] %(message)s",
        datefmt="%Y-%m-%d %H:%M:%S"
    )
    
    # Ensure log directory exists
    log_dir = os.path.dirname(config.LOG_FILE)
    os.makedirs(log_dir, mode=0o700, exist_ok=True)
    
    # Configure root logger
    root_logger = logging.getLogger()
    root_logger.setLevel(logging.DEBUG if (verbose or config.DEBUG_MODE) else logging.INFO)
    root_logger.handlers = []
    
    # File handler with rotation
    file_handler = logging.handlers.RotatingFileHandler(
        config.LOG_FILE,
        maxBytes=config.MAX_LOG_SIZE_MB * 1024 * 1024,
        backupCount=config.LOG_BACKUP_COUNT
    )
    file_handler.setFormatter(log_formatter)
    root_logger.addHandler(file_handler)
    
    # Console handler
    console_handler = logging.StreamHandler()
    console_handler.setFormatter(log_formatter)
    root_logger.addHandler(console_handler)
    
    logging.info("=" * 60)
    logging.info("Rclone Bisync Starting")
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
    """Write JSON file with error handling."""
    try:
        os.makedirs(os.path.dirname(filepath), exist_ok=True)
        with open(filepath, 'w') as f:
            json.dump(data, f, indent=2)
    except IOError as e:
        logging.error(f"Failed to write {filepath}: {e}")


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
    """Send desktop notification if available."""
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
        logging.debug(f"Notification sent: {title}")
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
    """Acquire file lock with stale lock detection."""
    global lock_file_handle
    
    # Ensure lock directory exists
    os.makedirs(os.path.dirname(config.LOCK_FILE), mode=0o700, exist_ok=True)
    
    lock_file_handle = open(config.LOCK_FILE, 'w')
    
    try:
        fcntl.lockf(lock_file_handle, fcntl.LOCK_EX | fcntl.LOCK_NB)
        # Write our PID to lock file
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
    """Check if rclone daemon is responsive."""
    if not config.FEATURES.get("daemon_health_check", True):
        return True, None
    
    for attempt in range(config.DAEMON_HEALTH_CHECK_RETRIES):
        try:
            auth = None
            if config.RC_AUTH_ENABLED:
                auth = (config.RC_USER, config.RC_PASS)
            
            response = requests.post(
                f"{config.RC_API_URL}/core/stats",
                timeout=5,
                auth=auth
            )
            
            if response.status_code == 200:
                # Update health file
                health_data = {
                    "last_check": time.time(),
                    "status": "healthy",
                    "response_time_ms": response.elapsed.total_seconds() * 1000
                }
                write_json_file(config.DAEMON_HEALTH_FILE, health_data)
                
                logging.debug(f"Daemon health check passed (attempt {attempt + 1})")
                return True, None
        
        except requests.exceptions.ConnectionError:
            if attempt < config.DAEMON_HEALTH_CHECK_RETRIES - 1:
                logging.warning(
                    f"Daemon not responding (attempt {attempt + 1}/{config.DAEMON_HEALTH_CHECK_RETRIES}). "
                    f"Retrying in 5 seconds..."
                )
                time.sleep(5)
            else:
                error_msg = (
                    f"Rclone daemon is not running on {config.RC_API_URL}. "
                    "Ensure rclone-daemon.service is started."
                )
                logging.error(error_msg)
                return False, error_msg
        
        except Exception as e:
            logging.error(f"Daemon health check error: {e}")
            return False, str(e)
    
    return False, "Daemon health check failed after all retries"


def classify_error(error_msg):
    """Classify error message into categories for appropriate handling."""
    if not error_msg:
        return "UNKNOWN"
    
    msg_lower = error_msg.lower()
    
    # Stale lock
    if "prior lock file found" in msg_lower or "lock" in msg_lower:
        return "STALE_LOCK"
    
    # Missing listings (needs resync)
    if any(x in msg_lower for x in ["cannot find prior", "listing", "bisync aborted"]):
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
    Returns: (success, error_type, error_detail)
    """
    start_time = time.time()
    sleep_duration = config.JOB_POLL_INITIAL_SLEEP
    
    logging.info(f"Monitoring job {job_id}...")
    
    while True:
        # Check timeout
        elapsed = time.time() - start_time
        if elapsed > config.JOB_TIMEOUT_SECONDS:
            logging.error(f"Job {job_id} exceeded timeout ({config.JOB_TIMEOUT_SECONDS}s)")
            return False, "TIMEOUT", f"Job exceeded {config.JOB_TIMEOUT_SECONDS}s timeout"
        
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
                
                if success:
                    duration = status.get("duration", elapsed)
                    logging.info(f"Job {job_id} completed successfully in {duration:.1f}s")
                    return True, None, None
                
                # Error classification
                err_type = classify_error(error_msg)
                logging.error(f"Job {job_id} failed: {err_type} - {error_msg}")
                return False, err_type, error_msg
            
            # Job still running
            logging.debug(f"Job {job_id} still running... (elapsed: {elapsed:.1f}s)")
            time.sleep(min(sleep_duration, config.JOB_POLL_MAX_SLEEP))
            sleep_duration *= config.JOB_POLL_BACKOFF_MULTIPLIER
        
        except requests.exceptions.RequestException as e:
            logging.error(f"Error polling job status: {e}")
            return False, "MONITOR_ERROR", str(e)
        
        except Exception as e:
            logging.error(f"Unexpected error monitoring job: {e}")
            return False, "MONITOR_ERROR", str(e)


def run_rclone_bisync(resync_attempt=False, force_attempt=False):
    """
    Trigger bisync via RC API.
    Returns: (success, error_type, error_detail)
    """
    # Check daemon health first
    daemon_ok, daemon_error = check_daemon_health()
    if not daemon_ok:
        return False, "DAEMON_OFFLINE", daemon_error
    
    # Prepare API request
    url = f"{config.RC_API_URL}/sync/bisync?_async=true"
    
    # Safely copy common args and append flags explicitly to the engine arguments list
    rclone_args = list(config.RCLONE_COMMON_ARGS)
    if force_attempt:
        rclone_args.append("--force")
        logging.info("Bypassing delete safety checks using --force flag.")
    if resync_attempt:
        rclone_args.append("--resync")
        logging.info("Requesting full database state reconstruction using --resync flag.")
    
    params = {
        "path1": config.REMOTE_PATH,
        "path2": config.LOCAL_PATH,
        "resync": resync_attempt,
        "force": force_attempt,
        "args": rclone_args
    }
    
    try:
        logging.info(f"Sending bisync command to daemon ({config.RC_API_URL})...")
        logging.info(f"  Remote: {config.REMOTE_PATH}")
        logging.info(f"  Local:  {config.LOCAL_PATH}")
        logging.info(f"  Resync: {resync_attempt}")
        logging.info(f"  Force:  {force_attempt}")
        
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
            # Synchronous completion (rare)
            logging.info("Bisync completed synchronously")
            return True, None, None
    
    except requests.exceptions.ConnectionError:
        error_msg = f"Cannot connect to rclone daemon at {config.RC_API_URL}"
        logging.error(error_msg)
        return False, "DAEMON_OFFLINE", error_msg
    
    except requests.exceptions.Timeout:
        error_msg = "Request to daemon timed out"
        logging.error(error_msg)
        return False, "DAEMON_TIMEOUT", error_msg
    
    except requests.exceptions.RequestException as e:
        error_msg = f"Request error: {e}"
        logging.error(error_msg)
        return False, "DAEMON_ERROR", error_msg
    
    except Exception as e:
        error_msg = f"Unexpected error: {e}"
        logging.error(error_msg)
        return False, "UNKNOWN", error_msg


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
    """Handle termination signals gracefully."""
    logging.warning(f"Received signal {signum}. Cleaning up...")
    release_lock()
    sys.exit(0)


def main():
    """Main entry point."""
    # Set up signal handlers
    signal.signal(signal.SIGINT, signal_handler)
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
    args = parser.parse_args()
    
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
    
    try:
        # Run sync
        success, err_type, err_detail = run_rclone_bisync(resync_attempt=args.resync, force_attempt=args.force)
        duration = time.time() - start_time
        
        if success:
            # ============================================================
            # SUCCESS PATH
            # ============================================================
            logging.info("=" * 60)
            logging.info("SYNC COMPLETED SUCCESSFULLY")
            logging.info("=" * 60)
            
            # Cleanup
            cleanup_on_success()
            
            # Log metrics
            log_metrics(success=True, duration=duration)
            
            # Send webhook
            send_webhook("sync_success", {
                "duration": duration,
                "local_path": config.LOCAL_PATH,
                "remote_path": config.REMOTE_PATH
            })
            
            # Desktop notification (throttled)
            last_success = notification_state.get("last_success_timestamp", 0)
            time_since_last = (time.time() - last_success) / 60
            
            if not args.no_notify and time_since_last >= config.SUCCESS_NOTIFICATION_INTERVAL_MIN:
                send_desktop_notification(
                    "Sync Success",
                    f"Google Drive synchronized successfully.\nDuration: {duration:.1f}s"
                )
                notification_state["last_success_timestamp"] = time.time()
                write_notification_state(notification_state)
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
                    retry_success, retry_err, retry_detail = run_rclone_bisync(resync_attempt=True, force_attempt=args.force)
                    
                    if retry_success:
                        logging.info("✅ Auto-recovery successful! Baseline sync state successfully rebuilt.")
                        success = True
                        cleanup_on_success()
                        
                        # Clear old error states since recovery wiped out the fault
                        notification_state["last_error_type"] = None
                        write_notification_state(notification_state)
                        
                        log_metrics(success=True, duration=time.time() - start_time)
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
