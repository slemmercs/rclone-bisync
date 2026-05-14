#!/usr/bin/env python3

import os
import sys
import time
import subprocess
import logging
import logging.handlers
import fcntl
import re
import argparse
import shutil
import json
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

# New log rotation settings loaded from bisync_config.py
MAX_LOG_SIZE_MB = bisync_config.MAX_LOG_SIZE_MB
LOG_BACKUP_COUNT = bisync_config.LOG_BACKUP_COUNT

# Location for the persistent notification state file
NOTIFICATION_STATE_FILE = os.path.join(os.path.expanduser("~"), ".cache", "bisync", "notification_state.json")

def setup_logging(verbose=False):
    """Configures logging to write to a file and console with rotation."""
    log_formatter = logging.Formatter("%(asctime)s %(levelname)s: %(message)s")

    # Ensure log directory exists
    log_dir = os.path.dirname(LOG_FILE)
    os.makedirs(log_dir, mode=0o700, exist_ok=True)

    # Configure rotating file handler
    max_log_size_bytes = MAX_LOG_SIZE_MB * 1024 * 1024
    backup_count = LOG_BACKUP_COUNT
    file_handler = logging.handlers.RotatingFileHandler(
        LOG_FILE,
        maxBytes=max_log_size_bytes,
        backupCount=backup_count,
    )
    file_handler.setFormatter(log_formatter)

    # Configure root logger
    log_level = logging.DEBUG if verbose else logging.INFO
    logging.basicConfig(level=log_level, handlers=[file_handler])

def read_notification_state():
    """Reads the last notification timestamp from the state file."""
    if os.path.exists(NOTIFICATION_STATE_FILE):
        try:
            with open(NOTIFICATION_STATE_FILE, 'r') as f:
                return json.load(f)
        except (IOError, json.JSONDecodeError):
            return {"last_success_timestamp": 0}
    return {"last_success_timestamp": 0}

def write_notification_state(state):
    """Writes the current notification timestamp to the state file."""
    os.makedirs(os.path.dirname(NOTIFICATION_STATE_FILE), exist_ok=True)
    with open(NOTIFICATION_STATE_FILE, 'w') as f:
        json.dump(state, f)

def send_desktop_notification(title, message, urgency="normal"):
    """
    Sends a desktop notification using the cross-platform plyer library.
    """
    try:
        notification.notify(
            title=title,
            message=message,
            app_name=NOTIFICATION_APP_NAME,
        )
    except Exception as e:
        logging.warning(f"Failed to send desktop notification using plyer: {e}")

def run_rclone_bisync(*args, resync_attempt=False):
    """
    Runs the rclone bisync command, parses JSON logs in memory,
    and returns a tuple: (success_boolean, error_type_string, error_details)
    """
    rclone_bin = shutil.which("rclone")
    if not rclone_bin:
        logging.critical("Rclone binary not found in PATH. Please ensure it is installed.")
        return False, "RCLONE_MISSING", None

    command = [rclone_bin, "bisync", REMOTE_PATH, LOCAL_PATH] + RCLONE_COMMON_ARGS + list(args)

    if resync_attempt:
        command.append("--resync")

    try:
        logging.debug(f"Running command: {' '.join(command)}")
        # capture_output=True loads everything into RAM immediately
        result = subprocess.run(command, capture_output=True, text=True, check=False)

        err_type = "UNKNOWN"
        err_detail = None

        # Process standard output (usually dry-run stats or raw text)
        for line in result.stdout.splitlines():
            logging.info(line)

        # Process JSON logs from standard error
        for line in result.stderr.splitlines():
            try:
                # Parse the JSON log line
                log_entry = json.loads(line)
                level = log_entry.get("level", "info")
                msg = log_entry.get("msg", "")

                # Map Rclone log levels to Python's logging system
                if level == "info" or level == "notice":
                    logging.info(msg)
                elif level == "debug":
                    logging.debug(msg)
                elif level == "warning":
                    logging.warning(msg)
                elif level == "error":
                    logging.error(f"Rclone Error: {msg}")
                    
                    # Identify specific critical errors for main() to handle
                    if "prior lock file found" in msg:
                        err_type = "STALE_LOCK"
                        # Extract just the lock file path
                        match = re.search(r'prior lock file found: (\S+)', msg)
                        err_detail = match.group(1) if match else msg
                        
                    elif "cannot find prior Path1 or Path2 listings" in msg:
                        err_type = "NO_LISTINGS"
                        
                    elif any(limit in msg for limit in ["Rate Limit Exceeded", "rateLimitExceeded", "429 Too Many Requests", "quotaExceeded", "403 Forbidden"]):
                        err_type = "RATE_LIMIT"
                        err_detail = msg

            except json.JSONDecodeError:
                # Fallback if rclone crashes and outputs raw text instead of JSON
                logging.error(f"Raw stderr: {line}")

        if result.returncode == 0:
            return True, None, None
        else:
            return False, err_type, err_detail

    except Exception as e:
        logging.critical(f"An unexpected error occurred: {e}")
        return False, "EXCEPTION", str(e)

def acquire_lock():
    """Acquires an exclusive lock on the lock file."""
    try:
        lock_file_handle = open(LOCK_FILE, 'w')
        # Use a non-blocking lock (LOCK_EX | LOCK_NB)
        fcntl.flock(lock_file_handle, fcntl.LOCK_EX | fcntl.LOCK_NB)
        # Write PID to the lock file
        lock_file_handle.write(str(os.getpid()))
        return lock_file_handle
    except (IOError, OSError):
        logging.info("Sync already running. Exiting.")
        return None

def main():
    """Main function to orchestrate the sync process."""

    # Setup command-line argument parsing
    parser = argparse.ArgumentParser(description="A robust script to run rclone bisync operations.")
    parser.add_argument("--dry-run", action="store_true", help="Perform a dry run without making any changes.")
    parser.add_argument("--no-notify", action="store_true", help="Disable desktop notifications.")
    parser.add_argument("--verbose", action="store_true", help="Enable verbose logging.")
    args = parser.parse_args()

    setup_logging(verbose=args.verbose)

    lock_file_handle = acquire_lock()
    if not lock_file_handle:
        sys.exit(1)

    start_time = time.time()
    notification_state = read_notification_state()

    try:
        logging.info("Sync started")

        rclone_args = ["--dry-run"] if args.dry_run else []

        # Unpack the tuple returned by the updated function
        success, err_type, err_detail = run_rclone_bisync(*rclone_args)

        if not success:
            # Look at the error type identified in memory, no disk reading required
            
            if err_type == "STALE_LOCK":
                logging.info(f"Stale rclone lock file detected: {err_detail}. Attempting to remove it.")
                try:
                    os.remove(err_detail)
                    logging.info("Stale rclone lock file removed.")
                    
                    # Retry with resync
                    retry_success, _, _ = run_rclone_bisync(*rclone_args, resync_attempt=True)
                    if retry_success:
                        logging.info("Sync finished: OK (resync successful)")
                        if not args.no_notify:
                            send_desktop_notification("Rclone Bisync: Success", "Resync completed successfully.")
                            notification_state["last_success_timestamp"] = time.time()
                    else:
                        logging.error("Sync finished: ERROR (resync failed).")
                        if not args.no_notify:
                            send_desktop_notification("Rclone Bisync Alert", "Resync failed. Manual intervention required!", "critical")
                except OSError as e:
                    logging.error(f"Failed to remove stale lock file {err_detail}: {e}")
                    if not args.no_notify:
                        send_desktop_notification("Rclone Bisync Alert", "Failed to remove lock file!", "critical")

            elif err_type == "NO_LISTINGS":
                logging.info("Initial bisync failed due to no prior listings. Attempting a one-time resync.")
                retry_success, _, _ = run_rclone_bisync(*rclone_args, resync_attempt=True)
                if retry_success:
                    logging.info("Sync finished: OK (initial resync successful)")
                    if not args.no_notify:
                        send_desktop_notification("Rclone Bisync: Success", "Initial resync completed successfully.")
                        notification_state["last_success_timestamp"] = time.time()
                else:
                    logging.error("Sync finished: ERROR (initial resync failed).")
                    if not args.no_notify:
                        send_desktop_notification("Rclone Bisync Alert", "Initial resync failed!", "critical")
            
            elif err_type == "RATE_LIMIT":
                logging.warning(f"Sync paused due to Google Drive rate limiting: {err_detail}")
                if not args.no_notify:
                    send_desktop_notification(
                        "Rclone Bisync: Rate Limited", 
                        "Google API limit hit. Will retry automatically on next schedule.", 
                        "normal"
                    )

            else:
                logging.error(f"Sync finished: ERROR ({err_type}). Manual intervention may be required!")
                if not args.no_notify:
                    send_desktop_notification("Rclone Bisync Alert: CRITICAL", "Sync failed. Manual intervention required!", "critical")
                    
        else:
            logging.info("Sync finished: OK")
            time_since_last_success = (time.time() - notification_state.get("last_success_timestamp", 0)) / 60
            if not args.no_notify and time_since_last_success >= SUCCESS_NOTIFICATION_INTERVAL_MIN:
                send_desktop_notification("Rclone Bisync: Success", "Sync completed successfully.")
                notification_state["last_success_timestamp"] = time.time()
            elif args.no_notify:
                 pass
            else:
                logging.info(f"Successful sync notification skipped. Next notification in approximately {SUCCESS_NOTIFICATION_INTERVAL_MIN - time_since_last_success:.0f} minutes.")
    finally:
        write_notification_state(notification_state)
        end_time = time.time()
        duration = end_time - start_time
        logging.info(f"Sync duration: {duration:.2f} seconds.")

        logging.info("Releasing script lock.")
        if lock_file_handle:
            try:
                os.remove(LOCK_FILE)
            except OSError as e:
                logging.warning(f"Failed to remove lock file {LOCK_FILE}: {e}")
            lock_file_handle.close()

if __name__ == "__main__":
    main()
