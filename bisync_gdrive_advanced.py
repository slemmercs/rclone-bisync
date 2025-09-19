#!/usr/bin/env python3

import os
import sys
import time
import subprocess
import logging
import fcntl
import re
import tempfile
import argparse
import shutil
import json
import bisync_config

# --- Configuration (Loaded from bisync_config.py) ---
LOCAL_PATH = bisync_config.LOCAL_PATH
REMOTE_PATH = bisync_config.REMOTE_PATH
LOG_FILE = bisync_config.LOG_FILE
LOCK_FILE = bisync_config.LOCK_FILE
NOTIFICATION_APP_NAME = bisync_config.NOTIFICATION_APP_NAME
RCLONE_COMMON_ARGS = bisync_config.RCLONE_COMMON_ARGS
SUCCESS_NOTIFICATION_INTERVAL_MIN = bisync_config.SUCCESS_NOTIFICATION_INTERVAL_MIN

# Location for the persistent notification state file
NOTIFICATION_STATE_FILE = os.path.join(os.path.expanduser("~"), ".cache", "bisync", "notification_state.json")

def setup_logging(verbose=False):
    """Configures logging to write to a file and console."""
    log_formatter = logging.Formatter("%(asctime)s %(levelname)s: %(message)s")

    # Ensure log directory exists
    log_dir = os.path.dirname(LOG_FILE)
    os.makedirs(log_dir, mode=0o700, exist_ok=True)

    # Configure file handler
    file_handler = logging.FileHandler(LOG_FILE)
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
    Sends a desktop notification using notify-send by checking for
    DBus variables in the environment, as loaded by systemd.
    """
    if 'DBUS_SESSION_BUS_ADDRESS' in os.environ and 'DISPLAY' in os.environ:
        try:
            subprocess.run(
                ["/usr/bin/notify-send", "-u", urgency, "-a", NOTIFICATION_APP_NAME, title, message],
                env=os.environ,
                check=True,
                timeout=10
            )
        except (subprocess.CalledProcessError, FileNotFoundError, subprocess.TimeoutExpired) as e:
            logging.warning(f"Failed to send desktop notification: {e}")
    else:
        logging.warning("D-Bus environment variables not found in the service environment. Skipping notification.")

def run_rclone_bisync(*args, resync_attempt=False):
    """
    Runs the rclone bisync command with specified arguments and logs output.
    Returns True on success, False on failure.
    """
    # Find the rclone binary dynamically
    rclone_bin = shutil.which("rclone")
    if not rclone_bin:
        logging.critical("Rclone binary not found in PATH. Please ensure it is installed.")
        return False

    command = [rclone_bin, "bisync", REMOTE_PATH, LOCAL_PATH] + RCLONE_COMMON_ARGS + list(args)

    if resync_attempt:
        command.append("--resync")

    try:
        logging.debug(f"Running command: {' '.join(command)}")
        result = subprocess.run(command, capture_output=True, text=True, check=False)

        # Log stdout and distinguish stderr notices from critical errors
        for line in result.stdout.splitlines():
            logging.info(line)
        for line in result.stderr.splitlines():
            if "NOTICE:" in line:
                logging.info(line)
            else:
                logging.error(line)

        # Check for success based on exit code and content of stderr
        if result.returncode == 0:
            return True
        else:
            logging.error(f"Rclone bisync failed with exit code {result.returncode}")
            return False

    except Exception as e:
        logging.critical(f"An unexpected error occurred: {e}")
        return False

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

        if not run_rclone_bisync(*rclone_args):
            logging.warning("Initial bisync failed. Checking for stale rclone lock file.")

            with open(LOG_FILE, 'r') as f:
                log_content = f.read()
            match = re.search(r'prior lock file found: (\S+)', log_content)

            if match:
                stale_lock_file = match.group(1)
                logging.info(f"Stale rclone lock file detected: {stale_lock_file}. Attempting to remove it.")

                try:
                    os.remove(stale_lock_file)
                    logging.info("Stale rclone lock file removed.")

                    if run_rclone_bisync(*rclone_args, resync_attempt=True):
                        logging.info("Sync finished: OK (resync successful)")
                        if not args.no_notify:
                            send_desktop_notification("Rclone Bisync: Success", "Resync completed successfully.")
                            notification_state["last_success_timestamp"] = time.time()
                    else:
                        logging.error("Sync finished: ERROR (resync failed). Manual intervention may be required.")
                        if not args.no_notify:
                            send_desktop_notification("Rclone Bisync Alert: CRITICAL", "Resync failed. Manual intervention may be required!", "critical")
                except OSError as e:
                    logging.error(f"Failed to remove stale lock file {stale_lock_file}: {e}")
                    if not args.no_notify:
                        send_desktop_notification("Rclone Bisync Alert: CRITICAL", "Failed to remove stale lock file. Manual intervention required!", "critical")
            else:
                logging.error("Sync finished: ERROR. Manual intervention may be required!")
                if not args.no_notify:
                    send_desktop_notification("Rclone Bisync Alert: CRITICAL", "Sync failed. Manual intervention may be required!", "critical")
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
