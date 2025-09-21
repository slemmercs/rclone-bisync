# Rclone Bisync Setup Guide

This guide provides a comprehensive, step-by-step process for setting up an automated rclone bisync solution. It uses a robust Python script, systemd timers for reliable scheduling, and a single, external configuration file for easy management. This approach is highly recommended over older cron-based methods.

## Prerequisites

- **Rclone**: Ensure Rclone is installed and configured with a remote (e.g., gdrive:).
- **Python 3**: Verify that Python 3 is installed on your system.
- **plyer**: Install the plyer library for cross-platform desktop notifications. You can install it using pip:

    ```bash
    pip install plyer
    ```

## Step 1: Create the Sync Script and Configuration Files

1. First, create a dedicated directory for your script to keep your files organized. This guide uses `~/rclone-bisync`.

    ```bash
    mkdir -p ~/rclone-bisync
    cd ~/rclone-bisync
    ```

2. Create the Python Script (`bisync_gdrive_plyer.py`)

    Create the file `bisync_gdrive_plyer.py` and paste the complete Python script into it.

3. Create the Configuration File (`bisync_config.py`)

    Create the file `bisync_config.py` in the same directory and paste the complete configuration script into it. This is where you will customize all of the paths, rclone arguments, and notification settings.

4. Make the Script Executable

    Make the main Python script executable so it can be run by systemd:

    ```bash
    chmod +x bisync_gdrive_plyer.py
    ```

## Step 2: Configure `systemd` Timers

`systemd` is the preferred way to schedule tasks on modern Linux systems. It is more reliable and provides better logging and error handling than traditional cron jobs.

1. Create the systemd directory

    Create the user-specific systemd directory if it doesn't already exist.

    ```bash
    mkdir -p ~/.config/systemd/user
    ```

2. Create the Service File (`bisync-gdrive.service`)

    Create the file `~/.config/systemd/user/bisync-gdrive.service` and add the content from the provided file. This file tells systemd how to run your script.

3. Create the Timer File (`bisync-gdrive.timer`)

    Create the file `~/.config/systemd/user/bisync-gdrive.timer` and add the content from the provided file. This file tells systemd when to run the service. The provided timer is set to run every 10 minutes.

## Step 3: Enable the Service

After creating the files, you need to tell systemd to load your new timer and enable it.

1. Reload the systemd daemon to recognize the new files

```bash
systemctl --user daemon-reload
```

2. Enable the timer to start automatically at login

```bash
systemctl --user enable bisync-gdrive.timer
```

3. Start the timer immediately

```bash
systemctl --user start bisync-gdrive.timer
```

## Step 4: Verify the Setup

Check the timer status: Use the following command to see if your timer is loaded and scheduled correctly.

```bash
systemctl --user list-timers --all
```

View the logs: To see the output of your script, including any errors, use `journalctl`. The `-f` flag "follows" the log output in real-time.

```bash
journalctl --user -u bisync-gdrive.service -f
```

## Troubleshooting

**Error**: Rclone binary not found: Make sure rclone is in your system's PATH. You may need to provide the full path to the binary in the ExecStart command of the .service file if it's not.

**Notification errors**: If you encounter issues with notifications, ensure plyer is correctly installed. The plyer library handles the DBus environment variables automatically, so no manual setup is required.

**Permissions**: If the script or systemd files are not working, double-check that they have the correct permissions. The service and timer files should be readable, and the script should be executable (`chmod +x bisync_gdrive_plyer.py`).