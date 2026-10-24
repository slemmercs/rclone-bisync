---
layout: default
title: UI Gallery
nav_order: 4
description: "Visual tour of the Bisync Hub embedded web cockpit, telemetry charts, storage remotes, and log diagnostics."
---

# Dashboard & UI Gallery

Bisync Hub features a zero-dependency, compiled-in Web Cockpit built with Alpine.js and Tailwind CSS. The interface operates completely self-contained from the single Go binary with zero external npm or Node.js runtime requirements.

---

## 1. Operations & Telemetry Cockpit

The primary operations cockpit provides real-time visibility into the health of synchronization pairs, active transfer speeds, cumulative data transferred, and engine status indicators.

<div class="screenshot-frame">
  <img src="{{ '/assets/images/screenshots/bisync-hub_engine-telemetry.png' | relative_url }}" alt="Bisync Hub Real-Time Telemetry Cockpit" />
  <div class="caption">
    <span class="title">Production Cockpit</span>
    <span>Real-time transfer velocity, synchronization metrics, and engine status indicators</span>
  </div>
</div>

---

## 2. Storage Remotes & OAuth Management

Manage cloud endpoints and local storage directories from a centralized interface. The panel displays reachability status, authentication health, and latency diagnostics.

<div class="screenshot-frame">
  <img src="{{ '/assets/images/screenshots/bisync-hub_storage-remotes.png' | relative_url }}" alt="Storage Remotes Management" />
  <div class="caption">
    <span class="title">Storage Remotes</span>
    <span>Endpoint connectivity overview, active remotes, and OAuth authentication health</span>
  </div>
</div>

### Guided OAuth Configuration Modals

Configure and renew OAuth2 credentials for major cloud storage backends with clear, guided credential forms:

<div class="screenshot-grid">
  <div class="screenshot-frame">
    <img src="{{ '/assets/images/screenshots/bisync-hub_google-drive-oauth-configuration.png' | relative_url }}" alt="Google Drive OAuth Configuration Modal" />
    <div class="caption">
      <span class="title">Google Drive Setup</span>
      <span>Client ID, Secret & scope authorization</span>
    </div>
  </div>
  <div class="screenshot-frame">
    <img src="{{ '/assets/images/screenshots/bisync-hub_microsoft-onedrive-oauth-configuration.png' | relative_url }}" alt="Microsoft OneDrive OAuth Configuration Modal" />
    <div class="caption">
      <span class="title">OneDrive Setup</span>
      <span>Personal & Business Graph API credential setup</span>
    </div>
  </div>
</div>

---

## 3. Dual-Pane File Explorer

Inspect directory trees, file sizes, and modification timestamps across both synchronization roots side-by-side without leaving the browser or invoking CLI commands.

<div class="screenshot-frame">
  <img src="{{ '/assets/images/screenshots/bisync-hub_file-explorer.png' | relative_url }}" alt="Bisync Hub Dual-Pane File Explorer" />
  <div class="caption">
    <span class="title">Dual-Pane Explorer</span>
    <span>Side-by-side browsing of local directories and remote cloud storage hierarchies</span>
  </div>
</div>

---

## 4. Live Execution & Correlated Daemon Diagnostics

Comprehensive diagnostics allowing operators to inspect live Rclone engine outputs and systemd daemon execution logs without SSH session requirements.

<div class="screenshot-frame">
  <img src="{{ '/assets/images/screenshots/bisync-hub_bisync-log-panel.png' | relative_url }}" alt="Bisync Execution Log Panel" />
  <div class="caption">
    <span class="title">Engine Execution Log</span>
    <span>Live log stream with syntax-highlighted sync actions and dry-run outputs</span>
  </div>
</div>

<div class="screenshot-frame">
  <img src="{{ '/assets/images/screenshots/bisync-hub_correlated-daemon-log.png' | relative_url }}" alt="Correlated Daemon Log Modal" />
  <div class="caption">
    <span class="title">Correlated Daemon Log</span>
    <span>Unified service events, systemd journal entries, and trace diagnostics</span>
  </div>
</div>

---

## 5. Cryptographic Immutable Audit Ledger

Mathematically verifiable SHA-256 hash-chained compliance ledger tracking sync events, conflict resolutions, and deletion decisions with real-time verification and zero-leak secret redaction.

<div class="screenshot-frame">
  <img src="{{ '/assets/images/screenshots/bisync-hub_audit-ledger.png' | relative_url }}" alt="Cryptographic Immutable Audit Ledger" />
  <div class="caption">
    <span class="title">Cryptographic Audit Ledger</span>
    <span>Real-time SHA-256 hash chain verification, actor and action filtering, CSV/JSONL exports, and forensic payload inspection</span>
  </div>
</div>

