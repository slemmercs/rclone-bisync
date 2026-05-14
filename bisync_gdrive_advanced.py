#!/usr/bin/env python3

import os
import sys
import time
import logging
import logging.handlers
import fcntl
import re
import argparse
import json
import requests
import bisync_config
from plyer import notification

# --- Configuration (Loaded from bisync_config.py) ---
LOCAL_PATH = bisync_config.LOCAL_PATH
REMOTE_PATH = bisync_config.REMOTE_PATH
LOG_FILE = bisync_config.LOG_FILE
LOCK_FILE = bisync_config.LOCK_FILE
NOTIFICATION_APP_NAME = bisync_config.NOTIFICATION_APP_NAME
RCLONE_COMMON_ARGS = bisync_config.RCLONE_COMMON_ARGS
SUCCESS_NOTIFICATION_INTERVAL_MIN = bisync_config.SUCCESS_NOTIFICATION_INTERVAL_MIN
MAX_LOG_SIZE_MB = bisync_config.MAX_LOG_SIZE_MB
LOG_BACKUP_COUNT = bisync_config.LOG_BACKUP_COUNT

NOTIFICATION_STATE_FILE = os.path.join(os.path.expanduser("~"), ".cache", "bisync", "notification_state.json")
RC_API_URL = "http://localhost:5572"

def setup_logging(verbose=False):
    log_formatter = logging.Formatter("%(asctime)s %(levelname)s: %(message)s")
    log_dir = os.path.dirname(LOG_FILE)
    os.makedirs(log_dir, mode=0o700, exist_ok=True)

    root_logger = logging.getLogger()
    root_logger.setLevel(logging.DEBUG if verbose else logging.INFO)
    root_logger.handlers = []

    file_handler = logging.handlers.RotatingFileHandler(
        LOG_FILE, maxBytes=MAX_LOG_SIZE_MB * 1024 * 1024, backupCount=LOG_BACKUP_COUNT
    )
    file_handler.setFormatter(log_formatter)
    root_logger.addHandler(file_handler)

    console_handler = logging.StreamHandler()
    console_handler.setFormatter(log_formatter)
    root_logger.addHandler(console_handler)

def load_notification_state():
    os.makedirs(os.path.dirname(NOTIFICATION_STATE_FILE), exist_ok=True)
    if os.path.exists(NOTIFICATION_STATE_FILE):
        try:
            with open(NOTIFICATION_STATE_FILE, 'r') as f:
                return json.load(f)
        except (json.JSONDecodeError, IOError):
            pass
    return {"last_success_timestamp": 0}

def write_notification_state(state):
    try:
        with open(NOTIFICATION_STATE_FILE, 'w') as f:
            json.dump(state, f)
    except IOError as e:
        logging.error(f"Failed to write notification state: {e}")

def send_desktop_notification(title, message, urgency='normal'):
    try:
        notification.notify(
            title=title,
            message=message,
            app_name=NOTIFICATION_APP_NAME,
            timeout=10,
            hints={'urgency': 2 if urgency == 'critical' else 1}
        )
    except Exception as e:
        logging.error(f"Failed to send notification: {e}")

def run_rclone_bisync(*args, resync_attempt=False):
    """Triggers bisync via RC API and returns job results."""
    # Using the ?_async=true query parameter to ensure a Job ID is returned
    url = f"{RC_API_URL}/sync/bisync?_async=true"
    
    params = {
        "path1": REMOTE_PATH,
        "path2": LOCAL_PATH,
        "resync": resync_attempt,
        "args": RCLONE_COMMON_ARGS + list(args)
    }

    try:
        logging.info("Sending bisync command to Rclone Daemon...")
        response = requests.post(url, json=params)
        response.raise_for_status() # Check for HTTP errors
        response_json = response.json()
        
        logging.debug(f"Daemon Response: {response_json}")
        
        job_id = response_json.get("jobid")
        
        if job_id is not None:
            logging.info(f"Sync started (Job ID: {job_id})")
            return monitor_daemon_job(int(job_id))
        else:
            # Fallback for synchronous completion
            logging.info("Sync finished synchronously (no Job ID needed).")
            return True, None, None

    except requests.exceptions.ConnectionError:
        return False, "DAEMON_OFFLINE", "Rclone Daemon is not running on port 5572."
    except Exception as e:
        return False, "DAEMON_ERROR", str(e)

def monitor_daemon_job(job_id):
    """Polls the daemon for job completion and classifies errors."""
    while True:
        try:
            status_resp = requests.post(f"{RC_API_URL}/job/status", json={"jobid": job_id})
            status = status_resp.json()
            
            if status.get("finished"):
                success = status.get("success")
                error_msg = status.get("error", "")
                
                if success:
                    return True, None, None
                
                # Error classification for notifications
                err_type = "UNKNOWN"
                err_detail = error_msg
                msg_lower = error_msg.lower()

                if "prior lock file found" in msg_lower:
                    err_type = "STALE_LOCK"
                    match = re.search(r'prior lock file found: (\S+)', error_msg)
                    err_detail = match.group(1) if match else error_msg
                elif "cannot find prior" in msg_lower:
                    err_type = "NO_LISTINGS"
                elif any(x in msg_lower for x in ["rate", "429", "quota", "403"]):
                    err_type = "RATE_LIMIT"
                    err_detail = "Google Drive Rate Limit hit. Retrying later."

                return False, err_type, err_detail
            
            time.sleep(5)
        except Exception as e:
            return False, "MONITOR_ERROR", str(e)

def main():
    parser = argparse.ArgumentParser(description="Rclone Bisync Client")
    parser.add_argument("--verbose", action="store_true", help="Enable verbose logging")
    parser.add_argument("--resync", action="store_true", help="Force a resync")
    parser.add_argument("--no-notify", action="store_true", help="Disable notifications")
    args = parser.parse_args()

    setup_logging(args.verbose)
    notification_state = load_notification_state()
    
    # File locking
    lock_file_handle = open(LOCK_FILE, 'w')
    try:
        fcntl.lockf(lock_file_handle, fcntl.LOCK_EX | fcntl.LOCK_NB)
    except IOError:
        logging.warning("Another instance of the script is running. Exiting.")
        sys.exit(0)

    start_time = time.time()
    try:
        success, err_type, err_detail = run_rclone_bisync(resync_attempt=args.resync)

        if not success:
            if err_type == "RATE_LIMIT":
                logging.warning(f"Sync deferred: {err_detail}")
            else:
                logging.error(f"Sync failed: {err_type} - {err_detail}")
                if not args.no_notify:
                    send_desktop_notification("Sync Error", f"Type: {err_type}\n{err_detail}", "critical")
        else:
            logging.info("Sync completed successfully.")
            last_success = notification_state.get("last_success_timestamp", 0)
            if not args.no_notify and (time.time() - last_success) / 60 >= SUCCESS_NOTIFICATION_INTERVAL_MIN:
                send_desktop_notification("Sync Success", "Google Drive is synchronized.")
                notification_state["last_success_timestamp"] = time.time()
                write_notification_state(notification_state)

    finally:
        fcntl.lockf(lock_file_handle, fcntl.LOCK_UN)
        lock_file_handle.close()
        if os.path.exists(LOCK_FILE):
            os.remove(LOCK_FILE)
        logging.info(f"Finished in {time.time() - start_time:.2f}s")

if __name__ == "__main__":
    main()