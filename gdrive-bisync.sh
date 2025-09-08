#!/bin/bash

# Make the script more robust by exiting on errors, unset variables, and pipe failures.
set -euo pipefail

LOG_FILE="/home/csaba/bisync-gdrive.log"
RCLONE_BIN="/usr/bin/rclone"
LOCAL_PATH="/home/csaba/gdrive"
REMOTE_PATH="gdrive:"

# --- Notification Settings ---
NOTIFICATION_APP_NAME="Rclone Bisync"
# -----------------------------

# --- Lock File Configuration ---
LOCK_FILE="/var/tmp/bisync_gdrive.lock"
# -------------------------------

# --- Ensure log directory and file exist ---
LOG_DIR=$(dirname "$LOG_FILE")
if [ ! -d "$LOG_DIR" ]; then
    mkdir -p "$LOG_DIR" || { echo "$(date +"%Y/%m/%d %H:%M:%S") ERROR: Failed to create log directory $LOG_DIR. Exiting." >&2; exit 1; }
    chmod 700 "$LOG_DIR"
fi
if [ ! -f "$LOG_FILE" ]; then
    touch "$LOG_FILE" || { echo "$(date +"%Y/%m/%d %H:%M:%S") ERROR: Failed to create log file $LOG_FILE. Exiting." >&2; exit 1; }
    chmod 600 "$LOG_FILE"
fi
# ----------------------------------------

# --- Acquire Lock ---
if ( set -o noclobber; echo "$$" > "$LOCK_FILE") 2> /dev/null; then
    # Trap to ensure the lock file is removed even if the script is interrupted
    trap "rm -f '$LOCK_FILE'; exit" INT TERM EXIT
else
    echo "$(date +"%Y/%m/%d %H:%M:%S") Sync already running. Exiting." >> "$LOG_FILE"
    exit 1
fi
# --------------------

# Function to send desktop notification
send_desktop_notification() {
    local title="$1"
    local message="$2"
    local urgency="${3:-normal}"

    local ENV_VARS_FILE="/home/csaba/.config/dbus-vars"
    if [ -f "$ENV_VARS_FILE" ]; then
        source "$ENV_VARS_FILE"
        if [ -n "$DBUS_SESSION_BUS_ADDRESS" ]; then
            /usr/bin/notify-send -u "$urgency" -a "$NOTIFICATION_APP_NAME" "$title" "$message"
        else
            echo "$(date +"%Y/%m/%d %H:%M:%S") Warning: Environment variables loaded but DBUS_SESSION_BUS_ADDRESS is empty. Desktop notification skipped." >> "$LOG_FILE"
        fi
    else
        echo "$(date +"%Y/%m/%d %H:%M:%S") Warning: D-Bus environment variables file not found at $ENV_VARS_FILE. Desktop notification skipped." >> "$LOG_FILE"
    fi
}

# Define the common rclone bisync arguments as an array
RCLONE_COMMON_ARGS=(
    --drive-export-formats docx,xlsx,pptx
    --drive-import-formats docx,xlsx,pptx
    --drive-skip-gdocs
    --remove-empty-dirs
    --check-access
    --fast-list
    --use-mmap
    --ignore-existing
    --workdir "/home/csaba/.cache/rclone/bisync"
)

# --- Real-Time Bisync Logic ---
echo "$(date +"%Y/%m/%d %H:%M:%S") Sync started" >> "$LOG_FILE"

# Function to run rclone and log output
run_rclone_bisync() {
    "$RCLONE_BIN" bisync "$REMOTE_PATH" "$LOCAL_PATH" "${RCLONE_COMMON_ARGS[@]}" "$@" 2>&1 | while read -r line; do
        echo "$(date +"%Y/%m/%d %H:%M:%S") $line" >> "$LOG_FILE"
    done
}

# Attempt the initial sync
if ! run_rclone_bisync; then
    SYNC_STATUS=$?
    echo "$(date +"%Y/%m/%d %H:%M:%S") Bisync failed. Checking for stale rclone lock file." >> "$LOG_FILE"

    # Check if the failure was due to a lock file
    if grep -q "prior lock file found" "$LOG_FILE"; then
        echo "$(date +"%Y/%m/%d %H:%M:%S") Stale rclone lock file detected. Deleting it and attempting --resync." >> "$LOG_FILE"
        # Extract the lock file path from the log and remove it. This is a robust way to handle any name changes.
        RCLONE_LOCK_FILE=$(grep "prior lock file found" "$LOG_FILE" | head -n 1 | sed -n 's/.*: \(.*\)/\1/p')
        if [ -n "$RCLONE_LOCK_FILE" ] && [ -f "$RCLONE_LOCK_FILE" ]; then
            rm -f "$RCLONE_LOCK_FILE"
        fi

        # Now, attempt the resync
        if ! run_rclone_bisync --resync; then
            SYNC_STATUS=$?
            echo "$(date +"%Y/%m/%d %H:%M:%S") Sync finished: ERROR (exit code $SYNC_STATUS)" >> "$LOG_FILE"
            send_desktop_notification "Rclone Bisync Alert: CRITICAL" "Sync failed (exit code $SYNC_STATUS). Manual intervention may be required!" "critical"
        else
            echo "$(date +"%Y/%m/%d %H:%M:%S") Sync finished: OK (resync successful)" >> "$LOG_FILE"
        fi
    else
        # If the failure was not a lock file issue, just report the error
        echo "$(date +"%Y/%m/%d %H:%M:%S") Sync finished: ERROR (exit code $SYNC_STATUS)" >> "$LOG_FILE"
        send_desktop_notification "Rclone Bisync Alert: CRITICAL" "Sync failed (exit code $SYNC_STATUS). Manual intervention may be required!" "critical"
    fi
else
    echo "$(date +"%Y/%m/%d %H:%M:%S") Sync finished: OK" >> "$LOG_FILE"
fi
# -----------------------------------------------

# --- Release Lock ---
rm -f "$LOCK_FILE"
# --------------------
