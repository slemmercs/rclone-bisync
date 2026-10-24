---
layout: default
title: Quickstart Guide
nav_order: 6
description: "Install, configure, and automate Bisync Hub on Linux and macOS in under 5 minutes."
---

# Quickstart Guide

Get up and running with Bisync Hub in less than 5 minutes.

---

## 1. Quick Installation

Execute the automated installer to download the latest signed binary and setup basic system integration:

```bash
curl -sSL https://raw.githubusercontent.com/slemmercs/rclone-bisync/main/deploy/install.sh | bash
```

### Manual Binary Download
Alternatively, download the target binary directly from GitHub Releases:

```bash
# Example for Linux x86_64
curl -LO https://github.com/slemmercs/rclone-bisync/releases/latest/download/bisync-hub-linux-amd64
chmod +x bisync-hub-linux-amd64
sudo mv bisync-hub-linux-amd64 /usr/local/bin/bisync-hub
```

---

## 2. Configuration (`bisync.env`)

Generate a starting configuration or customize `bisync.env` in your working directory:

```env
# ------------------------------------------------------------------------------
# Bisync Hub Core Configuration
# ------------------------------------------------------------------------------
PORT=8080
BIND_ADDR=127.0.0.1

# Sync Paths
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

# Notification Webhooks (Planned Roadmap Feature)
# DISCORD_WEBHOOK_URL=
# TELEGRAM_BOT_TOKEN=
# TELEGRAM_CHAT_ID=
# HEALTHCHECKS_URL=
```

---

## 3. Starting the Server

Run the orchestrator interactively:

```bash
bisync-hub --config bisync.env
```

Open your browser to `http://localhost:8080` to access the real-time operational dashboard, inspect remote mounts, check canary health, and review pending sync events.

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

## 4. Running as a Systemd Service

For continuous 24/7 background operation on Linux:

```ini
# /etc/systemd/system/bisync-hub.service
[Unit]
Description=Bisync Hub Cloud Orchestrator
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=csaba
WorkingDirectory=/home/csaba/sync
ExecStart=/usr/local/bin/bisync-hub --config /home/csaba/sync/bisync.env
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
