---
layout: default
title: UI Gallery
nav_order: 4
description: "Visual tour of the Bisync Hub embedded web cockpit, sync review workflow, immutable audit ledger, and storage remotes."
---

# Dashboard & UI Gallery

Bisync Hub features a zero-dependency, compiled-in Web Cockpit built with Alpine.js and Tailwind CSS. The interface operates completely self-contained from the single Go binary with zero external npm or Node.js runtime requirements.

---

## 1. Hybrid Fleet Overview & Profile Management

The Fleet Overview provides real-time workload orchestration across multiple sync profiles. Operators can track fleet health counters (`Total`, `Healthy`, `Needs Review`, `Failing`, `Syncing`), trigger dry-runs or sync cycles, and manage operational flag inheritance.

<div class="screenshot-frame">
  <img src="{{ '/assets/images/screenshots/bisync-hub_fleet-overview.png' | relative_url }}" alt="Bisync Hub Multi-Profile Fleet Overview" />
  <div class="caption">
    <span class="title">Hybrid Fleet Overview</span>
    <span>Aggregate health status counters, interactive profile cards, and fleet-wide quick actions</span>
  </div>
</div>

<div class="screenshot-grid">
  <div class="screenshot-frame">
    <img src="{{ '/assets/images/screenshots/bisync-hub_profile-manager.png' | relative_url }}" alt="Bisync Hub Profile Manager" />
    <div class="caption">
      <span class="title">Profile Manager</span>
      <span>Add, edit, archive, and configure independent sync workloads</span>
    </div>
  </div>
  <div class="screenshot-frame">
    <img src="{{ '/assets/images/screenshots/bisync-hub_profile-settings.png' | relative_url }}" alt="Bisync Hub Profile Settings & Flags" />
    <div class="caption">
      <span class="title">Operational Flags</span>
      <span>3-tier flag overrides, provider presets, and security blocklist validation</span>
    </div>
  </div>
</div>

---

## 2. Operations & Telemetry Cockpit

The primary operations cockpit provides real-time visibility into the health of synchronization pairs, active transfer speeds, cumulative data transferred, and engine status indicators.

<div class="screenshot-frame">
  <img src="{{ '/assets/images/screenshots/bisync-hub_engine-telemetry.png' | relative_url }}" alt="Bisync Hub Real-Time Telemetry Cockpit" />
  <div class="caption">
    <span class="title">Production Cockpit</span>
    <span>Real-time transfer velocity, synchronization metrics, and engine status indicators</span>
  </div>
</div>

---

## 3. Interactive Sync Review & Safety Barrier

When unexpected deletion spikes or non-trivial conflicts exceed configured thresholds, Bisync Hub halts execution and requires human-in-the-loop sign-off before destructive changes touch replicas.

### Step 1: Sync Review Barrier Active
The engine detects threshold breaches during dry-run analysis, trips the circuit breaker, and engages the review barrier:

<div class="screenshot-frame">
  <img src="{{ '/assets/images/screenshots/bisync-hub_sync-review-barrier-active.png' | relative_url }}" alt="Sync Review Barrier Active" />
  <div class="caption">
    <span class="title">1. Sync Review Barrier Engaged</span>
    <span>Circuit breaker trips when deletion or conflict thresholds are exceeded, halting automated execution</span>
  </div>
</div>

### Step 2: Granular Resolution & Heuristics
Operators inspect proposed changes, view heuristic recommendations (e.g., Office lock files, local user deletions, canary protection), and choose per-file decisions or batch-apply recommendations:

<div class="screenshot-frame">
  <img src="{{ '/assets/images/screenshots/bisync-hub_sync-review-resolution.png' | relative_url }}" alt="Sync Review Resolution Modal" />
  <div class="caption">
    <span class="title">2. Granular Triage & Heuristic Recommendations</span>
    <span>Side-by-side metadata inspection, heuristic scoring, per-file resolutions, and batch actions</span>
  </div>
</div>

### Step 3: Pre-Mutation Backups & Automated Sync
Upon clicking "Apply Resolutions & Sync", the system stages atomic backups of all mutated files, marks the review session complete, and triggers an authorized force sync:

<div class="screenshot-frame">
  <img src="{{ '/assets/images/screenshots/bisync-hub_sync-review-starting.png' | relative_url }}" alt="Sync Review Starting Execution" />
  <div class="caption">
    <span class="title">3. Staged Backups & Force Sync Execution</span>
    <span>Modal auto-closes with toast notification; pre-mutation backups are saved and force sync is triggered</span>
  </div>
</div>

### Step 4: Multi-Stream Correlated Daemon Logs
The correlated daemon log modal allows operators to follow live execution across journal logs and engine streams, confirming safe convergence:

<div class="screenshot-frame">
  <img src="{{ '/assets/images/screenshots/bisync-hub_correlated-daemon-log.png' | relative_url }}" alt="Correlated Multi-Stream Daemon Logs" />
  <div class="caption">
    <span class="title">4. Correlated Multi-Stream Diagnostics</span>
    <span>Correlated execution output confirming clean sync convergence and barrier dismissal</span>
  </div>
</div>

---

## 4. Cryptographic Immutable Audit Ledger

A mathematically verifiable, SHA-256 hash-chained compliance ledger tracking sync events, operator resolutions, and file mutations. Backed by pure-Go SQLite with schema triggers enforcing append-only immutability.

### Unified Audit Ledger & Verification
Operators can inspect the full chronological ledger, run live cryptographic verification, filter by action or actor, and export to CSV or JSONL:

<div class="screenshot-frame">
  <img src="{{ '/assets/images/screenshots/bisync-hub_audit-ledger.png' | relative_url }}" alt="Unified Cryptographic Audit Ledger" />
  <div class="caption">
    <span class="title">Unified Cryptographic Audit Ledger</span>
    <span>Live SHA-256 hash-chain verification, action/actor filtering, pagination, and CSV/JSONL compliance export</span>
  </div>
</div>

### Tamper-Evident Block Inspection
Every ledger block links cryptographically to its predecessor (`H_n = SHA256(H_{n-1} || payload || timestamp)`). Clicking any record displays the full forensic block breakdown with redacted secrets:

<div class="screenshot-frame">
  <img src="{{ '/assets/images/screenshots/bisync-hub_audit-block.png' | relative_url }}" alt="Cryptographic Audit Block Inspection" />
  <div class="caption">
    <span class="title">Forensic Block Inspection</span>
    <span>Sequence ID, previous hash, current block hash, and recursive secret-sanitized payload</span>
  </div>
</div>

---

## 5. Storage Remotes & OAuth Management

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

## 6. Dual-Pane File Explorer

Inspect directory trees, file sizes, and modification timestamps across both synchronization roots side-by-side without leaving the browser or invoking CLI commands.

<div class="screenshot-frame">
  <img src="{{ '/assets/images/screenshots/bisync-hub_file-explorer.png' | relative_url }}" alt="Bisync Hub Dual-Pane File Explorer" />
  <div class="caption">
    <span class="title">Dual-Pane Explorer</span>
    <span>Side-by-side browsing of local directories and remote cloud storage hierarchies</span>
  </div>
</div>

---

## 7. Live Execution & Correlated Daemon Diagnostics

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


