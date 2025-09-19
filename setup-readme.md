# Rclone Bisync Setup Guide

This guide provides a step-by-step process to set up an automated Rclone bisync solution using a Python script, `systemd` timers, and D-Bus for desktop notifications.

---

### Prerequisites

* **Rclone**: Ensure Rclone is installed and configured with a remote named `gdrive`.
* **Python 3**: Verify Python 3 is installed on your system.
* **D-Bus**: Ensure `notify-send` and D-Bus are properly configured for your desktop environment.
* **`inotify-tools`**: For on-demand sync, install this package (e.g., `sudo apt install inotify-tools`).

---

### Step 1: Create the Sync Script

Create the Python script `bisync_gdrive.py` in your home directory (`/home/csaba/`). Copy the complete Python script provided previously into this file.

```bash
touch /home/csaba/bisync_gdrive.py
# Paste the script content into the file and save.
```

Make the script executable:

```bash
chmod +x /home/csaba/bisync_gdrive.py
```

### Step 2: Set up D-Bus for Notifications

This is crucial for the script to send desktop notifications.

1.  **Create the D-Bus Capture Script**:
   
    Create a file at `/home/csaba/dbus-vars.sh` and add the following content:

    ```bash
    #!/bin/bash
    if [[ -n "$DBUS_SESSION_BUS_ADDRESS" && -n "$DISPLAY" ]]; then
        mkdir -p "$HOME/.config"
        echo "export DBUS_SESSION_BUS_ADDRESS=\"$DBUS_SESSION_BUS_ADDRESS\"" > "$HOME/.config/dbus-vars"
        echo "export DISPLAY=\"$DISPLAY\"" >> "$HOME/.config/dbus-vars"
    fi
    ```

    Make the script executable:

    ```bash
    chmod +x /home/csaba/dbus-vars.sh
    ```

2.  **Create the Autostart File**:

    Create a file at `~/.config/autostart/dbus-vars.desktop` and add the following content:

    ```ini
    [Desktop Entry]
    Name=D-Bus Environment Variables
    Comment=Saves D-Bus session variables for cron jobs
    Exec=/home/csaba/dbus-vars.sh
    Terminal=false
    Type=Application
    X-KDE-AutostartScript=true
    ```

    **Log out and log back in** to ensure the `.desktop` file runs and creates the `~/.config/dbus-vars` file.

---

### Step 3: Configure `systemd` Timers

This step sets up the scheduled sync.

1.  **Create Service and Timer Directories**:

    ```bash
    mkdir -p ~/.config/systemd/user
    ```

2.  **Create the Service File (`bisync-gdrive.service`)**:

    Create the file `~/.config/systemd/user/bisync-gdrive.service` and add the following content. **Remember to replace `csaba` with your username**.

    ```ini
    [Unit]
    Description=Rclone Bisync with Google Drive

    [Service]
    User=csaba
    Group=csaba
    WorkingDirectory=/home/csaba/
    ExecStart=/usr/bin/python3 -u /home/csaba/bisync_gdrive.py
    Restart=on-failure
    StandardOutput=journal
    StandardError=journal
    ```

3.  **Create the Timer File (`bisync-gdrive.timer`)**:

    Create the file `~/.config/systemd/user/bisync-gdrive.timer` and add the following for an hourly sync:

    ```ini
    [Unit]
    Description=Run Rclone Bisync every hour

    [Timer]
    OnCalendar=hourly

    [Install]
    WantedBy=timers.target
    ```

---

### Step 4: Enable the Service

Tell `systemd` to use your new timer.

```bash
systemctl --user daemon-reload
systemctl --user enable bisync-gdrive.timer
systemctl --user start bisync-gdrive.timer
```

### Step 5: Verify the Setup

1.  **Check the timer status**: Use the following command to see if your timer is loaded and scheduled correctly.

    ```bash
    systemctl --user list-timers --all
    ```

2.  **View the logs**: To see the output of your script, including any errors, use `journalctl`. The `-f` flag "follows" the log output in real-time.

    ```bash
    journalctl --user -u bisync-gdrive.service -f
    ```

3.  **Test the notification**: You can manually run the Python script from your terminal to ensure desktop notifications are working. A notification should appear on your desktop.

    ```bash
    python3 /home/csaba/bisync_gdrive.py
    ```
