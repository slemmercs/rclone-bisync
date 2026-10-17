# Rclone Bisync Automation Engine & Telemetry Hub

This solution provides an automated, bidirectional synchronization between your local machine and Google Drive using the **Rclone Remote Control (RC) API** paired with a single-binary native Go operational dashboard and sync orchestrator.

## Core Components

1. **Rclone Daemon (`rclone-daemon.service`)**: Background controller exposing internal statistics and control APIs.
2. **Telemetry UI Backend (`rclone-dashboard.service`)**: High-performance native Go web server (`rclone-bisync dashboard`) running continuously with embedded Alpine.js + Tailwind frontend, aggregating log streams, operational state vectors, and real-time charts.
3. **Synchronization Engine (`bisync-gdrive.service`)**: Native Go execution engine (`rclone-bisync sync`) triggered by systemd timer or manual command to execute bidirectional sync cycles safely with kernel-level flock locking and canary validation.
4. **Systemd Timer (`bisync-gdrive.timer`)**: Automatically runs the sync iteration every 15 minutes.

## Prerequisites & Installation

The application compiles into a self-contained, single-binary executable with no external Python runtime dependencies required for production operation.

### Installation

1. Move to the source workspace directory.
2. Run the automated deployment script with administrative elevations:

    ```bash
    sudo bash install-custom-rclone.sh
    ```

## CLI Commands

The unified single binary supports modular subcommands:

```bash
# Launch telemetry dashboard (embedded UI)
./bin/rclone-bisync dashboard --host 127.0.0.1 --port 8080

# Execute bidirectional sync cycle
./bin/rclone-bisync sync [--resync] [--force] [--dry-run] [--no-notify] [--origin=cli|timer|dashboard]

# Provision RCLONE_TEST canary file
./bin/rclone-bisync canary [--remote local|gdrive] [--path subfolder]
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

Run the automated pre-flight quality and safety gate before committing or opening a PR:

```bash
./scripts/check.sh
```

This verifies:
1. Shell script syntax linting (`bash -n`).
2. Python bytecode compilation across packages (`compileall`).
3. Secret isolation (preventing `rc-secrets.env` or plaintext credentials from entering git).
4. Go static analysis (`go vet ./...`) and complete Go unit test suite (`go test -count=1 ./...`).
5. Automated test suite execution (`pytest tests/`).


