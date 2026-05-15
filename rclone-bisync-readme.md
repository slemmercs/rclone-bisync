# Rclone Bisync API-Based Setup Guide

This solution provides a robust, automated bidirectional synchronization between your local machine and Google Drive. Unlike standard scripts, this uses the **Rclone Remote Control (RC) API** for cleaner job management and background execution.

## Core Components
1. **Rclone Daemon**: An always-on background process managing the connection.
2. **Python Master Script**: Communicates with the API to trigger and monitor sync jobs.
3. **Systemd Timer**: Schedules the sync every 15 minutes.
4. **Desktop Shortcuts**: For manual triggers and GUI access.

## Prerequisites
- **Rclone**: Installed and configured with a remote named `gdrive:`.
- **System Dependencies**:
    ```bash
    sudo apt update
    sudo apt install python3-requests python3-plyer
    ```

## Installation
1. Navigate to your repository folder.
2. Run the automated installer:
    ```bash
    sudo bash install-custom-rclone.sh
    ```

## Usage & Monitoring
- **Automatic**: The systemd timer handles everything every 10 minutes.
- **Manual Sync**: Launch "Rclone Force Bisync" from your application menu.
- **Web GUI**: Launch "Rclone Web GUI" to view statistics and manage files at `http://localhost:5572`.

## Monitoring Logs
To see the synchronization progress in real-time:
```bash
journalctl --user -u bisync-gdrive.service -f