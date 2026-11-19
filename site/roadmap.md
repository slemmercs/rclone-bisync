---
layout: default
title: Roadmap
nav_order: 3
description: "Bisync Hub strategic priorities, upcoming milestones, and core shipped capabilities."
---

# Project Roadmap

Bisync Hub is built through test-driven micro-iterations (>80% test coverage target) to turn `rclone bisync` into a safe, observable, zero-dependency synchronization control plane.

| Status | Focus Area | Key Deliverables |
| :--- | :--- | :--- |
| **In Progress** | **Homelab Edge Security & OTel** | Trusted reverse proxy headers, forward-auth support, and OpenTelemetry (OTLP) metrics/traces. |
| **Up Next** | **Remote Mount & Storage Pooling** | Self-healing FUSE/WinFsp mount lifecycle manager and stacked multi-cloud storage (`union` + `chunker` + `crypt`). |
| **Exploring** | **Distributed & Zero-Knowledge Sync** | Multi-node peer state coordination and end-to-end encrypted fleet telemetry. |
| **Shipped** | **Core v1.0 Foundation (`beta.1`–`beta.27`)** | FSM queue orchestrator, 4-analyzer conflict engine, cryptographic audit ledger, native MCP server, and embedded Vue 3 cockpit. |

---

## 🟡 Near-Term Strategic Priorities

### 1. Homelab Reverse Proxy Hardening & OpenTelemetry (OTel)
Harden production homelab deployments exposed behind reverse proxies while adding industry-standard observability:
* **Proxy-Aware Networking:** Configurable trusted CIDR subnets with normalization for `X-Forwarded-For`, `X-Real-IP`, and `X-Forwarded-Proto`.
* **Edge Authentication:** Built-in HTTP Basic Auth and forward-auth middleware compatibility for Traefik, Caddy, and Nginx.
* **Native OTel / OTLP Export:** Expose sync duration histograms, transferred byte counters, lock wait times, and daemon health metrics to Prometheus, Grafana, or Jaeger.

### 2. Remote Mount Lifecycle Manager & Pooled Storage (`internal/mount`)
Transform fragile OS mounts into self-healing virtual storage endpoints:
* **Lifecycle Daemon (Stage A):** Pure-Go supervisor managing `rclone mount` processes on Linux (FUSE) and Windows (WinFsp) with automated health probes and clean unmount handling.
* **Pooled Storage Engine (Stage B):** Provision stacked multi-cloud topologies (`union` + `chunker` + `crypt`) directly via the Rclone RC API.
* *Design Reference:* [Remote Mount Manager Design Specification](https://github.com/slemmercs/rclone-bisync/blob/main/docs/design/remote_mount_manager_design_specification.md)

---

## 🔮 Future Vision & Research

* **Peer-to-Peer State Coordination:** Multi-node cluster coordination for synchronizing across distributed homelab nodes or branch offices without central lock bottlenecks.
* **Encrypted Remote Telemetry:** Zero-knowledge status reporting that monitors multi-device sync health without exposing file paths or metadata.

---

## 🟢 Shipped Foundation (Available Now)

All core capabilities below are implemented, verified by automated Chromium E2E and real-Rclone sandbox suites, and shipped in the single-binary release:

### Safety, Conflict Resolution & Rollback
* **Interactive Deletion & Diff Review:** Dry-run mutation inspection (`+ CREATE`, `~ MODIFY`, `- DELETE`) that halts execution when deletion thresholds trip, backed by write-ahead log (WAL) rollback staging.
* **4-Analyzer Heuristic Conflict Engine:** Parallel evaluation of file size divergence, binary magic-byte signatures, provider clock skew (e.g., OneDrive 2-second windows), and edit velocity—auto-resolving high-confidence changes ($\ge 85\%$) and quarantining risky collisions ($< 60\%$).
* **Profile Configuration Snapshots:** Automated version history with 1-click rollback via the Web UI, REST API, or MCP tools.

### Control Plane & Multi-Profile Orchestration
* **FSM Lifecycle & Queue Manager:** Deterministic state machine (`idle` → `validating` → `running` → `review_required` / `completed`) with bounded worker concurrency and live SSE/WebSocket progress streaming.
* **3-Tier Operational Flags & Curated Cloud Toggles:** Inherited flag hierarchy (`Execution > Profile > Global > Default`) with guided CLI-to-RC compilation, rename tracking, and one-click presets for Google Drive, OneDrive, S3, and Dropbox.
* **Remote MD5 Deduplication:** Safe duplicate cleanup (`oldest`, `newest`, `list`) with automated baseline resyncing to prevent false deletion barriers.

<div class="screenshot-grid">
  <div class="screenshot-frame">
    <img src="{{ '/assets/images/screenshots/bisync-hub_google-drive-oauth-configuration.png' | relative_url }}" alt="Google Drive OAuth Configuration" />
    <div class="caption">
      <span class="title">Google Drive OAuth</span>
      <span>Guided provider setup & curated flags</span>
    </div>
  </div>
  <div class="screenshot-frame">
    <img src="{{ '/assets/images/screenshots/bisync-hub_microsoft-onedrive-oauth-configuration.png' | relative_url }}" alt="Microsoft OneDrive OAuth Configuration" />
    <div class="caption">
      <span class="title">Microsoft OneDrive OAuth</span>
      <span>Personal & Business Graph API integration</span>
    </div>
  </div>
</div>

### Native AI / MCP Integration & Cockpit UI
* **Embedded Dual-Transport MCP Server:** Pure-Go Model Context Protocol server (`stdio` and HTTP `sse`) exposing 11 token-optimized tools, read-only mode, structured abort diagnostics, and cryptographic confirmation tickets for destructive operations.
* **Single-Binary Vue 3 Cockpit:** Zero-dependency Go binary embedding a PrimeVue/Tailwind fleet dashboard, live MCP Inspector modal, and built-in diagnostic runbooks.

### Auditability & Security Hardening
* **Cryptographic Immutable Audit Ledger:** Append-only SQLite ledger protected by `BEFORE UPDATE/DELETE` triggers and SHA-256 hash chaining, complete with recursive secret redaction and CSV/JSONL compliance export.
* **Process & Path Isolation:** Advisory POSIX file locking (`flock`), local path traversal jails, and hardened input validation.
