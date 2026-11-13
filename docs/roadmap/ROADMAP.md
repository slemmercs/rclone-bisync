# Bisync Hub Strategic Roadmap

This roadmap outlines the strategic direction, architectural horizons, and forward-looking milestones for **Bisync Hub**.

---

## Vision & Guiding Principles

**"Review. Decide. Sync."**

Bisync Hub exists to be the safest, most transparent, and most resilient way to operate Rclone Bisync across desktop and homelab environments.

### Guiding Principles

1. **User Safety & Data Loss Prevention First**: Destructive actions always require deterministic barriers, dry-run previews, or explicit operator confirmation.
2. **Crash-Resilient Atomicity**: All state mutations, telemetry writes, and review decisions use atomic file replacement and race-free POSIX advisory locking (`flock`).
3. **Operational Transparency**: System state, failure causes (`TOO_MANY_DELETES`, missing canaries, lock contention), and flag effects must be comprehensible to operators without guesswork.
4. **Zero Global Pollution**: Native Go single binary distribution with embedded assets and zero external runtime dependencies.

---

## Delivered Releases & Milestone History

All historical releases, delivered epics, and version changelogs (from initial architecture port through `v1.0.0-beta.31`) are documented in detail in our official release notes:

👉 **[View All Delivered Releases on GitHub](https://github.com/slemmercs/rclone-bisync/releases)**

### Key Recently Delivered Capabilities (beta.28 – beta.32)
- **Deep File Path Navigation & Pinned Log Audit Dock (`v1.0.0-beta.32`)**: Contextual path tokenizer extracting file mutations from logs, synchronized dual-pane profile switching & directory loading, transient pulse row highlight animation, and IDE-style collapsible pinned Log Audit Dock with height presets and N/P keyboard stepping.
- **Configuration Snapshot & One-Click Rollback (`v1.0.0-beta.28`)**: Automated configuration backups, SHA-256 integrity verification, timestamped JSON snapshots under `config_snapshots/`, and one-click rollback REST API.
- **Unified Operational Log Ledger & Run ID Correlation (`v1.0.0-beta.29`)**: Structured high-density table view aligning `bisync.log` events with Audit Ledger records, Run ID correlation chips (`#run-...`), and daemon trace inspection.
- **Remote Cloud Web Link Navigation ("Open on Remote") (`v1.0.0-beta.30`)**: Canonical cloud URL resolution via Rclone RC `operations/publiclink` with capability filtering for Google Drive and OneDrive.
- **FullRemoteTarget Local Path Preservation & Go 1.26.11 (`v1.0.0-beta.31`)**: Raw filesystem path handling for local destinations, Go 1.26.11 vulnerability patch (`GO-2026-6859`), and elevated test coverage (>85% under race detection).

---

## Forward-Looking Horizons

### Horizon 1: Target v1.0.0 General Availability (GA)

The immediate focus leading to the production `v1.0.0` GA milestone centers on deep operational navigation, homelab production hardening, and queued workflows.

#### 1. Homelab Production Hardening & Standards Observability (OTel/OTLP) (Active / In-Flight)
- **Reverse Proxy Header Normalization**: Support for standard proxy headers (`X-Forwarded-For`, `X-Real-IP`, `X-Forwarded-Proto`) with configurable trusted CIDR proxy subnets.
- **Authentication Gateway Support**: Optional HTTP basic authentication or forward-auth middleware integration for secure exposure behind reverse proxies (Traefik, Caddy, Nginx).
- **Native OpenTelemetry Instrumentation**: Export sync run durations, transferred byte counts, lock contention wait periods, error rates, and daemon health metrics directly over standard OTLP (gRPC / HTTP) to Prometheus OTel receivers, OpenTelemetry Collectors, or Grafana Tempo/Mimir.

---

### Horizon 2: Advanced Operational Visibility & Queued Workflows

Expands forensic visibility and smooths file transfer operations without lock collisions.

#### 3. Queued Dual-Pane File Explorer Direct Upload
- **Non-Blocking Upload Dock**: Asynchronous upload queue accepting file uploads during active synchronization (`telemetry.is_syncing === true` or POSIX flock contention) rather than returning HTTP 409 errors.
- **Sequential Auto-Resume**: Automatically dispatches queued transfers once active sync runs finish and locks are released.
- **Mid-Upload Lock Coordination**: Advisory upload coordination delaying scheduled sync triggers while active multipart uploads are in-flight.


---

### Horizon 3: Storage Expansion & Native Infrastructure

Broadens platform integration and system storage capabilities.

#### 6. Remote Mount Lifecycle Manager & Multi-Cloud Pooling
- **Stage A (Remote Mount Manager)**: Pure-Go daemon managing Rclone VFS mount lifecycles on Linux (FUSE) with health keepalive probes, background daemon supervision, and clean unmount handlers (`fusermount -u`).
- **Stage B (Pooled Storage Engine)**: Programmatic provisioning of stacked multi-cloud topologies (`union` + `chunker` + `crypt`) via Rclone RC `/config/create`. Combines disparate storage backends into a unified virtual drive guarded by Bisync Hub safety barriers.

#### 7. Local IPC via Unix Domain Sockets
- Replaces loopback TCP ports with local Unix domain sockets (`unix:///run/user/$UID/rclone.sock`) with strict `0600` permissions.
- Eliminates local port contention and unauthorized loopback snooping in multi-user and containerized environments.

#### 8. Native Desktop Application Packaging (Wails)
- Standalone native desktop application bundle with native OS system tray integration, sleep/wake network reconnect hooks, and offline desktop notifications.

---

## Long-Term Direction & Non-Goals

To maintain high code quality and focus, Bisync Hub is explicitly **not** intended to become:
- **A multi-tenant cloud SaaS platform**: It is optimized as a sovereign, self-hosted homelab and desktop hub.
- **A proprietary replacement for upstream Rclone**: Bisync Hub wraps and orchestrates official Rclone RC engines rather than re-implementing cloud storage protocols.
- **An arbitrary text/binary diff merge editor**: File content diff merges are intentionally deferred to specialized diff tools.
