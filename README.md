# Rclone Bisync Automation Engine & Telemetry Hub

This solution provides an automated, bidirectional synchronization between your local machine and Google Drive using the **Rclone Remote Control (RC) API** paired with a custom FastAPI operational dashboard.

## Core Components

1. **Rclone Daemon (`rclone-daemon.service`)**: Background controller exposing internal statistics APIs.
2. **Telemetry UI Backend (`rclone-dashboard.service`)**: High-performance FastAPI server running continuously to aggregate log files, operational state vectors, and runtime charts.
3. **Python Execution Engine (`bisync_gdrive_advanced.py`)**: Script triggered by timers to run the sync cycle safely with locking mechanisms.
4. **Systemd Timer (`bisync-gdrive.timer`)**: Automatically runs the sync iteration every 15 minutes.

## Prerequisites & Dependencies

The custom dashboard relies on FastAPI and Uvicorn. To avoid pathing issues inside background systemd daemons, these should be installed directly through your system package manager:

```bash
sudo apt update
sudo apt install python3-requests python3-plyer python3-fastapi python3-uvicorn python3-psutil
```

## Installation

1. Move to the source workspace directory.
2. Run the automated deployment script with administrative elevations:

    ```bash
    sudo bash install-custom-rclone.sh
    ```

## Access & Monitoring Commands

- **Telemetry UI Instance**: Open your Application Launcher and select **Rclone Custom Dashboard**, or directly navigate to `http://localhost:8080` in your browser.
- **Manual Sync Execution**: Run **Rclone Force Bisync** from your app launcher or call:
    ```bash
    systemctl --user start bisync-gdrive.service
    ```
- **Real-Time Service Journal Streams**:

    ```bash
    # View dashboard web service engine logs
    journalctl --user -u rclone-dashboard.service -f
    
    # View synchronization runner history
    journalctl --user -u bisync-gdrive.service -f
    ```

## Development & Testing

Run the automated pre-flight quality and safety check before committing or opening a PR:

```bash
./scripts/check.sh
```

This verifies:
1. Shell script syntax linting (`bash -n`).
2. Python bytecode compilation across packages (`compileall`).
3. Secret isolation (preventing `rc-secrets.env` or plaintext credentials from entering git).
4. Full automated test suite execution (`pytest tests/`).

