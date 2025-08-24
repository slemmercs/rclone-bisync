#!/bin/bash

# Make the script more robust by exiting on errors, unset variables, and pipe failures.
set -euo pipefail

NOW=$(printf "%(%Y/%m/%d %H:%M:%S)T")
LOG_FILE="/home/csaba/bisync-gdrive.log"
RCLONE_BIN="/usr/bin/rclone"
LOCAL_PATH="/home/csaba/gdrive"
REMOTE_PATH="gdrive:"

# --- Notification Settings ---
NOTIFICATION_APP_NAME="Rclone Bisync"
# -----------------------------

# --- Lock File Configuration ---
LOCK_FILE="/var/tmp/bisync_gdrive.lock" # Using /var/tmp or /tmp for system-wide locks
# -------------------------------

# --- Ensure log directory and file exist ---
LOG_DIR=$(dirname "$LOG_FILE")
if [ ! -d "$LOG_DIR" ]; then
    mkdir -p "$LOG_DIR" || { echo "$NOW ERROR: Failed to create log directory $LOG_DIR. Exiting." >&2; exit 1; }
    chmod 700 "$LOG_DIR" # Example: Only owner can read/write/execute
fi
if [ ! -f "$LOG_FILE" ]; then
    touch "$LOG_FILE" || { echo "$NOW ERROR: Failed to create log file $LOG_FILE. Exiting." >&2; exit 1; }
    chmod 600 "$LOG_FILE" # Example: Only owner can read/write
fi
# ----------------------------------------

# --- Acquire Lock ---
if ( set -o noclobber; echo "$$" > "$LOCK_FILE") 2> /dev/null; then
    # Trap to ensure the lock file is removed even if the script is interrupted
    trap "rm -f '$LOCK_FILE'; exit" INT TERM EXIT
else
    echo "$NOW Sync already running. Exiting." >> "$LOG_FILE"
    exit 1
fi
# --------------------

# Function to send desktop notification
send_desktop_notification() {
    local title="$1"
    local message="$2"
    local urgency="${3:-normal}"

    # Source the environment variables from the saved file
    # This is the key change to make it work with cron
    local ENV_VARS_FILE="/home/csaba/.config/dbus-vars"
    if [ -f "$ENV_VARS_FILE" ]; then
        source "$ENV_VARS_FILE"
        if [ -n "$DBUS_SESSION_BUS_ADDRESS" ]; then
            /usr/bin/notify-send -u "$urgency" -a "$NOTIFICATION_APP_NAME" "$title" "$message"
        else
            echo "$NOW Warning: Environment variables loaded but DBUS_SESSION_BUS_ADDRESS is empty. Desktop notification skipped." >> "$LOG_FILE"
        fi
    else
        echo "$NOW Warning: D-Bus environment variables file not found at $ENV_VARS_FILE. Desktop notification skipped." >> "$LOG_FILE"
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

# --- NEW Bisync Logic ---
echo "$NOW Sync started" >> "$LOG_FILE"

# Execute the normal bisync command and capture the output
# NOTE: Removed --log-file to allow capturing output for error checking.
RCLONE_SYNC_OUTPUT=$("$RCLONE_BIN" bisync "$REMOTE_PATH" "$LOCAL_PATH" "${RCLONE_COMMON_ARGS[@]}" 2>&1)
SYNC_STATUS=$?

# Check for the specific critical error
if [ "$SYNC_STATUS" -ne 0 ] && echo "$RCLONE_SYNC_OUTPUT" | grep -q "cannot find prior Path1 or Path2 listings"; then
    echo "$NOW Bisync critical error detected. Forcing --resync." >> "$LOG_FILE"
    # Run a resync to fix the state files
    # NOTE: The --resync command's output will also be captured and appended to the log.
    RCLONE_SYNC_OUTPUT=$("$RCLONE_BIN" bisync "$REMOTE_PATH" "$LOCAL_PATH" "${RCLONE_COMMON_ARGS[@]}" --resync 2>&1)
    SYNC_STATUS=$?
fi

# Append the full rclone output to the log file
echo "$RCLONE_SYNC_OUTPUT" >> "$LOG_FILE"

if [ "$SYNC_STATUS" -eq 0 ]; then
    echo "$NOW Sync finished: OK" >> "$LOG_FILE"
#    send_desktop_notification "Rclone Bisync Alert" "Sync completed successfully." "normal"
else
    echo "$NOW Sync finished: ERROR (exit code $SYNC_STATUS)" >> "$LOG_FILE"
    send_desktop_notification "Rclone Bisync Alert: CRITICAL" "Sync failed (exit code $SYNC_STATUS). Manual intervention may be required!" "critical"
fi
# -----------------------------------------------
# --- Release Lock ---
rm -f "$LOCK_FILE"
# The trap will be removed automatically when the script exits,
# so explicit `trap - INT TERM EXIT` is not strictly necessary here.
# --------------------
