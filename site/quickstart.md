---
layout: default
title: Quickstart Guide
nav_order: 2
description: "Install, configure, and automate Bisync Hub on Linux and macOS in under 5 minutes."
---

# Quickstart Guide

Get up and running with Bisync Hub in less than 5 minutes.

> **Prerequisites:** Rclone (`v1.65+`) configured with at least one remote (`rclone config`), or pass `--install-rclone` to the automated installer below.

---

## 1. Installation

### Option A: Automated Installer (Linux)
Downloads the latest signed binary, verifies checksums, and configures systemd integration:

```bash
# Standard installation (uses existing Rclone on PATH)
curl -sSL https://raw.githubusercontent.com/slemmercs/rclone-bisync/main/deploy/install.sh | sudo bash

# Or install Rclone automatically if not present
curl -sSL https://raw.githubusercontent.com/slemmercs/rclone-bisync/main/deploy/install.sh | sudo bash -s -- --install-rclone
```

<details>
<summary><strong>View Installer CLI Options</strong></summary>

| Option | Description | Default |
| :--- | :--- | :--- |
| `-i, --install-rclone` | Install official Rclone binary if not found in `PATH` | `false` |
| `-n, --dry-run` | Simulate installation without modifying system | `false` |
| `-d, --target-dir <path>` | Installation target directory | `$HOME/bisync-hub` |
| `--port <port>` | Dashboard HTTP service port | `8080` |
| `--skip-services` | Stage binary and config without enabling systemd units | `false` |
| `--min-rclone-version <v>` | Minimum required Rclone semver | |
| `--max-rclone-version <v>` | Maximum compatible Rclone semver | |
| `-y, --non-interactive` | Run unattended with defaults | `false` |

</details>

### Option B: Manual Binary Download (Linux & macOS)

Download the standalone binary directly from GitHub Releases:

```bash
# Linux (x86_64) — use 'darwin-arm64' or 'darwin-amd64' for macOS
curl -LO https://github.com/slemmercs/rclone-bisync/releases/latest/download/bisync-hub-linux-amd64
chmod +x bisync-hub-linux-amd64
sudo mv bisync-hub-linux-amd64 /usr/local/bin/bisync-hub
```

---

## 2. Configuration (`bisync.env`)

Create a working directory and generate your `bisync.env` configuration file:

```bash
mkdir -p ~/bisync && cd ~/bisync
```

```env
# ------------------------------------------------------------------------------
# Bisync Hub Core Configuration
# ------------------------------------------------------------------------------
PORT=8080
BIND_ADDR=127.0.0.1

# Default Sync Paths (Additional profiles can be added in the Web UI)
PATH1=/home/operator/data
PATH2=onedrive:data_backup

# Safety & Circuit Breakers
SAFETY_MAX_DELETE_COUNT=25
SAFETY_MAX_DELETE_RATIO=0.10
SAFETY_ENABLE_CANARY=true
CANARY_FILENAME=RCLONE_TEST

# Conflict Engine Settings
CONFLICT_AUTO_RESOLVE_MIN_CONFIDENCE=0.85
CONFLICT_BACKUP_LOSERS=true

# Embedded MCP Server
MCP_READ_ONLY=false

# Notification Webhooks
# WEBHOOK_URL=https://discord.com/api/webhooks/...
```

---

## 3. Starting the Server

Launch the management cockpit interactively or run a one-off dry-run simulation:

```bash
# Start the operational dashboard & scheduler
bisync-hub dashboard --env-file bisync.env

# Or test synchronization safely in dry-run mode
bisync-hub sync --dry-run --env-file bisync.env
```

Open `http://localhost:8080` in your browser to access the dashboard, verify remote OAuth status, and trigger your initial baseline sync.

> **First-Time Sync Note:** If syncing two non-empty directories for the first time, trigger an initial **Resync** from the profile card in the dashboard (or pass `--resync` on the CLI) to establish the baseline state cache.

<div class="screenshot-grid">
  <div class="screenshot-frame">
    <img src="{{ '/assets/images/screenshots/bisync-hub_storage-remotes.png' | relative_url }}" alt="Bisync Hub Storage Remotes" />
    <div class="caption">
      <span class="title">Storage Remotes</span>
      <span>Remote health checks & OAuth indicators</span>
    </div>
  </div>
  <div class="screenshot-frame">
    <img src="{{ '/assets/images/screenshots/bisync-hub_file-explorer.png' | relative_url }}" alt="Bisync Hub Dual-Pane File Explorer" />
    <div class="caption">
      <span class="title">Dual-Pane Explorer</span>
      <span>Browse local and remote directories</span>
    </div>
  </div>
</div>

---

## 4. Running as a Background Service (Systemd)

If you used manual binary installation on Linux, create a systemd unit for 24/7 background operation (replace `operator` with your Linux username):

```ini
# /etc/systemd/system/bisync-hub.service
[Unit]
Description=Bisync Hub Cloud Orchestrator
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=operator
WorkingDirectory=/home/operator/bisync
ExecStart=/usr/local/bin/bisync-hub dashboard --env-file /home/operator/bisync/bisync.env
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now bisync-hub
```

---

## 5. Connecting AI Agents (MCP)

Bisync Hub includes a native **Model Context Protocol (MCP)** server supporting both local `stdio` and embedded HTTP `SSE` transports.

### Local Agents (`stdio` — Cline, Claude Desktop, Cursor)

Add Bisync Hub to your MCP client configuration (e.g., `~/.cline/data/settings/cline_mcp_settings.json`):

```json
{
  "mcpServers": {
    "bisync-hub": {
      "command": "/usr/local/bin/bisync-hub",
      "args": ["mcp", "--env-file", "/home/operator/bisync/bisync.env"],
      "disabled": false
    }
  }
}
```

### Network / Remote Agents (Embedded `SSE`)

When `bisync-hub dashboard` is running, the embedded MCP SSE endpoint is automatically served on port `8081` alongside the daemon—no separate background process required. Open the **MCP Inspector** pill in the dashboard header for one-click client snippets and live tool inspection.

