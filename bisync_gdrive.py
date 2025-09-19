#!/usr/bin/env python3

import os
import sys
import time
import subprocess
import logging
import fcntl
import re

# --- Configuration ---
# All paths should be absolute
RCLONE_BIN = "/usr/bin/rclone"
LOCAL_PATH = "/home/csaba/gdrive"
REMOTE_PATH = "gdrive:"

# Logging
LOG_FILE = "/home/csaba/rclone-bisync/bisync-gdrive.log"

# Lock file to prevent concurrent script execution
LOCK_FILE = "/var/tmp/bisync_gdrive.lock"

# Desktop Notification Settings
NOTIFICATION_APP_NAME = "Rclone Bisync"
DBUS_VARS_FILE = "/home/csaba/.config/dbus-vars"

# Common rclone bisync arguments
RCLONE_COMMON_ARGS = [
    "--drive-export-formats", "docx,xlsx,pptx",
    "--drive-import-formats", "docx,xlsx,pptx",
    "--drive-skip-gdocs",
    "--remove-empty-dirs",
    "--check-access",
    "--fast-list",
    "--use-mmap",
    "--ignore-existing",
    "--workdir", "/home/csaba/.cache/rclone/bisync",
]
# --------------------

def setup_logging():
    """Configures logging to write to a file and console."""
    log_formatter = logging.Formatter("%(asctime)s %(levelname)s: %(message)s")
    
    # Ensure log directory exists
    log_dir = os.path.dirname(LOG_FILE)
    os.makedirs(log_dir, mode=0o700, exist_ok=True)

    # Configure file handler
    file_handler = logging.FileHandler(LOG_FILE)
    file_handler.setFormatter(log_formatter)
    
    # Configure root logger
    logging.basicConfig(level=logging.INFO, handlers=[file_handler])

def send_desktop_notification(title, message, urgency="normal"):
    """Sends a desktop notification using notify-send."""
    if os.path.exists(DBUS_VARS_FILE):
        env_vars = {}
        with open(DBUS_VARS_FILE, 'r') as f:
            for line in f:
                if '=' in line:
                    key, value = line.strip().split('=', 1)
                    env_vars[key] = value.strip('"')
        
        if 'DBUS_SESSION_BUS_ADDRESS' in env_vars:
            try:
                subprocess.run(
                    ["/usr/bin/notify-send", "-u", urgency, "-a", NOTIFICATION_APP_NAME, title, message],
                    env=dict(os.environ, **env_vars),
                    check=True,
                    timeout=10
                )
            except (subprocess.CalledProcessError, FileNotFoundError, subprocess.TimeoutExpired) as e:
                logging.warning(f"Failed to send desktop notification: {e}")
        else:
            logging.warning(f"D-Bus environment variable not found in {DBUS_VARS_FILE}. Skipping notification.")
    else:
        logging.warning(f"D-Bus environment variables file not found at {DBUS_VARS_FILE}. Skipping notification.")

def run_rclone_bisync(*args, resync_attempt=False):
    """
    Runs the rclone bisync command with specified arguments and logs output.
    Returns True on success, False on failure.
    """
    command = [RCLONE_BIN, "bisync", REMOTE_PATH, LOCAL_PATH] + RCLONE_COMMON_ARGS + list(args)
    
    if resync_attempt:
        command.append("--resync")

    try:
        result = subprocess.run(command, capture_output=True, text=True, check=False)
        
        # Log both stdout and stderr
        for line in result.stdout.splitlines():
            logging.info(line)
        for line in result.stderr.splitlines():
            logging.error(line)

        # Check for specific rclone failure codes
        if result.returncode == 0:
            return True
        else:
            logging.error(f"Rclone bisync failed with exit code {result.returncode}")
            return False

    except FileNotFoundError:
        logging.critical(f"Rclone binary not found at {RCLONE_BIN}. Please check the path.")
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
        logging.warning("Sync already running. Exiting.")
        return None

def main():
    """Main function to orchestrate the sync process."""
    setup_logging()
    
    lock_file_handle = acquire_lock()
    if not lock_file_handle:
        sys.exit(1)

    # Use a try-finally block to ensure the lock is always released
    start_time = time.time() # Start the timer
    
    try:
        logging.info("Sync started")
        
        # Attempt the initial sync
        if not run_rclone_bisync():
            logging.warning("Initial bisync failed. Checking for stale rclone lock file.")
            
            # Use regex to find the lock file path from the logs
            with open(LOG_FILE, 'r') as f:
                log_content = f.read()
            match = re.search(r'prior lock file found: (\S+)', log_content)
            
            if match:
                stale_lock_file = match.group(1)
                logging.info(f"Stale rclone lock file detected: {stale_lock_file}. Attempting to remove it.")
                
                try:
                    os.remove(stale_lock_file)
                    logging.info("Stale rclone lock file removed.")
                    
                    # Now, attempt the resync
                    if run_rclone_bisync(resync_attempt=True):
                        logging.info("Sync finished: OK (resync successful)")
                    else:
                        logging.error("Sync finished: ERROR (resync failed). Manual intervention may be required.")
                        send_desktop_notification("Rclone Bisync Alert: CRITICAL", "Resync failed. Manual intervention may be required!", "critical")
                except OSError as e:
                    logging.error(f"Failed to remove stale lock file {stale_lock_file}: {e}")
                    send_desktop_notification("Rclone Bisync Alert: CRITICAL", "Failed to remove stale lock file. Manual intervention required!", "critical")
            else:
                logging.error("Sync finished: ERROR. Manual intervention may be required!")
                send_desktop_notification("Rclone Bisync Alert: CRITICAL", "Sync failed. Manual intervention may be required!", "critical")
        else:
            logging.info("Sync finished: OK")

    finally:
        end_time = time.time()
        duration = end_time - start_time
        logging.info(f"Sync duration: {duration:.2f} seconds.")
        
        # Release the lock by closing the file handle
        logging.info("Releasing script lock.")
        if lock_file_handle:
            try:
                os.remove(LOCK_FILE)
            except OSError as e:
                logging.warning(f"Failed to remove lock file {LOCK_FILE}: {e}")
            lock_file_handle.close()
            
if __name__ == "__main__":
    main()