# Bisync Hub Technical & Support Reference Manual

> **Review. Decide. Sync.** — Comprehensive documentation of features, configuration settings, operational workflows, and troubleshooting procedures.

---

## 1. System Overview & Architecture

Bisync Hub is a production-grade safety orchestrator and bidirectional synchronization control plane built around [Rclone Bisync](https://rclone.org/bisync/). It wraps bidirectional file replication with an unyielding safety envelope: human-in-the-loop conflict and deletion reviews, POSIX process flocking, path traversal jails, canary integrity validation, and a tamper-evident cryptographic audit ledger.

### 1.1 Architecture Stack
- **Engine Core**: Native Go static binary with embedded assets (`//go:embed`). No external language runtimes (Node.js, Python, Ruby) or system daemons required.
- **Control Plane**: Finite State Machine (FSM) managing workload lifecycles (`idle` → `validating` → `running` → `review_required` / `completed` / `failed` / `aborted`).
- **Execution Queue**: Concurrency orchestrator ensuring atomic profile execution, avoiding POSIX advisory lock thrashing.
- **Rclone Integration**: Direct interaction with Rclone via CLI subprocess execution and the Rclone Remote Control (RC) JSON-RPC HTTP API.
- **Cockpit SPA**: Reactive Single Page Application built with Vue 3.4 and PrimeVue v4, featuring Pinia domain stores and real-time telemetry streaming over SSE.
- **Model Context Protocol (MCP)**: Native stdio/SSE agent server enabling AI agents (Claude, Cursor, Cline) to inspect storage, analyze conflicts, and execute gated syncs.

---

## 2. Global System Settings

Global settings establish engine-wide operational defaults, resource constraints, telemetry retention, and external notification targets. Access them via **Cockpit → Global Settings** or through `GET/PUT /api/config`.

### 2.1 Global Defaults (`global_defaults`)
These defaults are inherited by all synchronization profiles unless explicitly overridden at the profile level.

| Setting | Type | Default | Description & Operational Specifics |
| :--- | :--- | :--- | :--- |
| `max_delete` | Integer | `10` | **Safety deletion threshold percentage (0–100%)**. If a synchronization run proposes deleting more than this percentage of total files on either replica, the run aborts immediately with safety violation status, preventing data loss. |
| `canary_validation` | Boolean | `true` | **Canary verification guard**. Verifies the presence of `RCLONE_TEST` marker files on both endpoints before initiating sync. If missing (e.g. unmounted network shares or empty remotes), execution is blocked. |
| `conflict_resolve` | String | `newer` | **Default conflict heuristic**. Options: `path1` (local/Path1 wins), `path2` (remote/Path2 wins), `newer` (most recently modified wins), `older` (least recently modified wins), `larger` (largest file wins), `smaller` (smallest file wins). |
| `compare_mode` | String | `size,modtime` | **File divergence comparison strategy**. Options: `size,modtime` (fastest, standard), `size,modtime,checksum` (recommended for critical integrity), `checksum` (hash only), `modtime` (timestamp only). |
| `remove_empty_dirs` | Boolean | `true` | When enabled, bisync cleans up empty directory structures left behind after file migrations or deletions. |
| `create_empty_dirs` | Boolean | `true` | When enabled, bisync replicates empty directories from source to destination replicas. |
| `lock_explorer_remote` | Boolean | `false` | When enabled, prevents arbitrary remote path browsing in the dual-pane file explorer, locking users to configured profile roots. |
| `custom_flags` | String | `""` | Pass-through CLI flags appended to every Rclone bisync execution (e.g. `--fast-list`, `--checkers 8`). |
| `default_local_path` | String | `~/sync` | Suggested default base directory when creating new local replica endpoints. |
| `default_remote_path` | String | `""` | Suggested default remote prefix/bucket for new profiles. |

---

### 2.2 Daemon Engine & Process Settings (`daemon`)
Controls background worker scheduling, polling intervals, timeouts, and backoff behavior.

| Setting | Type | Default | Description & Operational Specifics |
| :--- | :--- | :--- | :--- |
| `job_poll_initial_sleep_sec`| Float | `1.0` | Initial polling sleep interval (seconds) while waiting for long-running Rclone RC background tasks. |
| `job_poll_max_sleep_sec` | Float | `10.0` | Upper bound for exponential polling backoff to reduce CPU/socket load during multi-gigabyte transfers. |
| `job_poll_backoff_multiplier` | Float | `1.5` | Exponential multiplier applied to polling intervals between successive status queries. |
| `job_timeout_seconds` | Integer | `3600` | Absolute ceiling for a single sync execution (1 hour default). If exceeded, the process tree is canceled. |
| `lock_stale_timeout_seconds` | Integer | `7200` | Process lock expiry threshold (2 hours default). If an advisory lock remains active with a dead PID, it is pruned. |
| `rate_limit_backoff_hours` | Float | `1.0` | Automatic cooling-off duration triggered when cloud API HTTP 429 / rate limit throttling errors are detected. |
| `rate_limit_retry_after_backoff` | Boolean | `true` | Automatically reschedule suspended profiles after the rate-limit backoff window expires. |
| `check_stale_locks` | Boolean | `true` | Proactively audit lock file timestamps on daemon heartbeat and reap orphaned locks. |

---

### 2.3 Logging & Rotation Settings (`logging`)

| Setting | Type | Default | Description & Operational Specifics |
| :--- | :--- | :--- | :--- |
| `max_log_size_mb` | Integer | `10` | Maximum file size in megabytes before bisync and daemon log files are automatically rotated. |
| `log_backup_count` | Integer | `5` | Number of historical rotated log archives (`.log.1`, `.log.2`) retained on disk before oldest files are purged. |

---

### 2.4 Telemetry & Metrics Settings (`telemetry`)

| Setting | Type | Default | Description & Operational Specifics |
| :--- | :--- | :--- | :--- |
| `enable_metrics` | Boolean | `true` | Enables high-resolution timeseries metrics collection (transfer speeds, latency, IOPS, memory usage). |
| `metrics_max_entries` | Integer | `1000` | In-memory ring buffer capacity for telemetry datapoints displayed on the Live Performance Telemetry graph. |
| `metrics_max_backups` | Integer | `10` | Retention count for persisted daily telemetry snapshots stored in `telemetry_history.json`. |

---

### 2.5 External Notifications & Webhook Alerts (`notifications`)

| Setting | Type | Default | Description & Operational Specifics |
| :--- | :--- | :--- | :--- |
| `app_name` | String | `Bisync Hub`| Identifying user-agent header and payload label included in external webhook notifications. |
| `webhook_url` | String | `""` | HTTP/HTTPS endpoint for outgoing event payloads (Slack, Discord, Teams, or custom webhook handlers). |
| `webhook_timeout_sec` | Integer | `10` | Network timeout for outgoing HTTP POST alert requests. |
| `success_interval_min` | Float | `60.0` | Throttling throttle window: prevents alert noise by only dispatching success notifications once per interval. |
| `error_repeat_hours` | Float | `4.0` | Re-notification window for persistent repeating errors, preventing alerting fatigue on sustained outages. |
| `watchdog_ping_url` | String | `""` | Periodic dead-man's-switch heartbeat URL (e.g. Healthchecks.io, Better Uptime) pinged upon successful daemon cycles. |
| `watchdog_timeout_sec` | Integer | `10` | Network timeout for watchdog heartbeat pings. |

---

### 2.6 Subsystem Feature Toggles (`features`)

| Flag | Default | Functional Scope |
| :--- | :--- | :--- |
| `debug_mode` | `false` | Enables verbose debug logging (`--verbose -vv`) across Rclone subprocesses and internal HTTP endpoints. |
| `mcp_read_only` | `false` | Restricts Model Context Protocol (MCP) server: disables mutating tools (`bisync_trigger_sync`, `bisync_delete_item`, `bisync_purge_dir`). |
| `stale_lock_detection` | `true` | Activates background inspection of `/tmp/rclone-bisync-*.lock` files against active OS PID tables. |
| `rate_limit_backoff` | `true` | Automatically pauses profiles upon detecting HTTP 429 or provider quota rate limits. |
| `daemon_health_check` | `true` | Regularly validates background systemd/supervisor process responsiveness. |
| `metrics_collection` | `true` | Streams live speed, latency, and throughput counters to the cockpit dashboard. |
| `smart_notifications` | `true` | Suppresses redundant alerts and groups consecutive related status changes. |
| `webhook_alerts` | `true` | Enables outgoing HTTP notifications when sync operations fail or require review. |


---

## 3. Profile Configuration & Sync Policies

Profiles represent individual, isolated synchronization workloads mapping a local filesystem path to a remote cloud storage backend.

### 3.1 Path Specifications & Jail Confinement
- **Local Path (`local_path`)**: Absolute or relative directory on the host machine.
  - *Jail Confinement Guarantee*: All file exploration, preview generation, and deletion operations scoped to a profile are strictly confined to `local_path`. Traversal attempts (`../../etc/shadow`) return immediate security violations.
- **Remote Target (`remote`)**: Configured Rclone remote (e.g. `gdrive:`, `s3_backup:`, `dropbox:`).
- **Remote Path (`remote_path`)**: Optional subfolder within the remote target (e.g. `backups/workstation`).

### 3.2 Scheduling Presets
Profiles can be executed manually or scheduled via internal timers / systemd integration:
- `manual`: Executed only via cockpit trigger, CLI command, or MCP agent call.
- `15m`: Every 15 minutes.
- `30m`: Every 30 minutes.
- `1h`: Hourly execution.
- `4h`: Every 4 hours.
- `daily`: Daily sync at 00:00 local time.

### 3.3 Review Modes (`review_mode`)
- `disabled`: Fully automated sync; changes are committed according to conflict and safety heuristics.
- `deletions`: Sync halts and requires operator review if deletions are proposed on either replica.
- `conflicts`: Sync halts and prompts for review when two files diverge and cannot be cleanly resolved.
- `all`: Dry-run pre-flight execution on every cycle; operator must review and confirm all pending transfers and deletions.

### 3.4 Safety Thresholds
- **Max Delete Threshold (`delete_threshold_pct` / `max_delete`)**: Percentage of total files (e.g. `10.0` for 10%) or absolute count (`200c`). If proposed deletions exceed this value, bisync aborts immediately.
- **Consecutive Failure Circuit Breaker**: If a profile fails 3 consecutive times, it is placed in an error-backoff state to prevent cloud API bans or spamming alert channels.
- **Canary Validation**: Requires `RCLONE_TEST` canary files to exist at both Path1 and Path2 roots.

---

## 4. Interactive Review & Conflict Resolution Workflow

When a sync run flags conflicts or exceeds deletion thresholds in `review_mode`, Bisync Hub halts execution before mutating storage and initiates an interactive review session.

### 4.1 Write-Ahead Log (WAL) & Pre-Mutation Staging
1. **Candidate Discovery**: The engine isolates diverging files, comparing modified timestamps, sizes, and cryptographic hashes.
2. **Pre-Mutation Backup**: Files marked for overwrite or deletion are safely staged in `.review_backups/<session_id>/` before disk operations commence.
3. **Session State Ledger**: Decisions are staged in SQLite and can be revised before committing.

### 4.2 Resolution Decisions
Operators can select from 6 granular resolution strategies:
- `keep-local`: Overwrite remote replica with the local version.
- `keep-remote`: Overwrite local replica with the remote version.
- `keep-both`: Rename candidate files with `.local.<timestamp>` and `.remote.<timestamp>` suffixes, preserving both versions.
- `accept-delete`: Confirm and execute pending deletion.
- `restore-item`: Restore deleted item from the staged pre-mutation backup.
- `skip`: Defer decision; exclude file from the current synchronization cycle.

---

## 5. Remote & Local Deduplication Workspace

Bisync Hub includes a visual deduplication engine that resolves duplicate files on cloud remotes or local directories without requiring destructive shell scripts.

### 5.1 Dual Matching Strategies
1. **Cryptographic Hash Mode (`by_hash: true`)**:
   - Calculates cryptographic checksums (SHA-256 or MD5 depending on backend support).
   - Guarantees 100% data identity even when filenames or modification dates differ.
2. **Name & Size Mode (`by_hash: false`)**:
   - Matches identical filenames having identical byte sizes.
   - Recommended for backends where computing remote hashes is slow, costly, or unsupported.

### 5.2 Resolution Actions
- **List / Inspect**: Safe dry-run scan clustering duplicates into collapsible groups with path, size, and timestamp breakdown.
- **Oldest**: Retains the oldest file replica based on modified timestamp and deletes newer duplicates.
- **Newest**: Retains the most recently modified replica and deletes older duplicates.
- **Interactive Staging**: Stage specific deletions into the bottom action drawer and apply them in batch with a confirmation token.


---

## 6. Dual-Pane Filesystem Explorer

The Cockpit features an integrated dual-pane file manager for side-by-side replica inspection:
- **Jail-Confined Browsing**: Local explorer pane cannot navigate outside the configured profile directory.
- **Remote Inspection**: Queries remote directories via the Rclone RC `operations/list` API with metadata caching.
- **Drag-and-Drop Operations**: Copy or move files between local and remote panes with confirmation safeguards.
- **Safe Previews**: Text, markdown, JSON, and image files can be previewed without executing external helper binaries.

---

## 7. Cryptographic Immutable Audit Ledger

Every profile update, sync execution, file deletion, and configuration rollback is recorded in an immutable SQLite audit ledger (`audit.db`).

### 7.1 Security Invariants
- **Database Trigger Enforcement**: `BEFORE UPDATE` and `BEFORE DELETE` triggers reject any attempt to modify or delete historical records with SQL abort exceptions.
- **SHA-256 Block Chaining**:
  `hash_n = SHA256(hash_{n-1} || timestamp || action || profile_id || status || details)`
- **In-Browser Chain Audit**: The Cockpit audit viewer runs a client-side verification engine walking the hash chain from Genesis to head, visually confirming that the ledger has not been tampered with.
- **Compliance Export**: Ledger records can be exported directly to CSV or JSONL formats for external auditing.

---

## 8. Model Context Protocol (MCP) AI Agent Server

Bisync Hub includes an integrated MCP server allowing LLM agents (Claude Desktop, Cursor, Cline) to manage and monitor synchronization workloads safely.

### 8.1 Dual Transport Modes
- **Standard Input/Output (`stdio`)**: Default mode for local agent desktop environments.
  ```bash
  ./bin/rclone-bisync mcp
  ```
- **Server-Sent Events (`sse`)**: HTTP streaming transport for distributed agent deployments.
  ```bash
  ./bin/rclone-bisync mcp --transport sse --port 8081
  ```

### 8.2 Operational Tools Reference

| MCP Tool Name | Description | Blast-Radius Safeguards |
| :--- | :--- | :--- |
| `bisync_get_status` | Health summary, active syncs, daemon metrics, canaries | Read-only |
| `bisync_list_profiles` | Lists profiles, schedules, paths, and status | Read-only |
| `bisync_list_remotes` | Lists configured remotes and storage backend types | Read-only |
| `bisync_list_providers` | Catalogs supported Rclone storage backends | Read-only |
| `bisync_explore_path` | Navigates files under local jail or remote targets | Jail-confined |
| `bisync_preview_file` | Reads metadata and text previews of files | Jail-confined |
| `bisync_get_sync_history`| Queries immutable audit ledger records | Read-only |
| `bisync_get_logs` | Tails filtered execution and daemon log streams | Read-only |
| `bisync_validate_flags` | Previews and validates compiled Rclone CLI flags | Read-only |
| `bisync_trigger_sync` | Dispatches sync execution on a profile | Requires confirmation token for live/force sync |
| `bisync_copy_item` | Copies a file/directory between storage targets | Jail-confined |
| `bisync_move_item` | Moves or renames a file/directory | Jail-confined |
| `bisync_delete_item` | Safely deletes a file or directory | Canary check + confirmation token |
| `bisync_purge_dir` | Recursively purges a directory | Requires confirmation token |
| `bisync_dedupe_by_hash` | Runs deduplication by hash or size/name | Confirmation token for mutations |
| `bisync_create_profile` | Provisions a new bidirectional sync profile | Path & canary validation |
| `bisync_update_profile` | Updates settings on an existing profile | Re-validates configuration |
| `bisync_delete_profile` | Permanently deletes a profile | Requires confirmation token |
| `bisync_rollback_profile`| Reverts profile to a historical version snapshot | Requires confirmation token |
| `bisync_inspect_conflict_session`| Retrieves pending conflict review candidates | Read-only |
| `bisync_resolve_review` | Stages or applies conflict review decisions | Confirmation token for `execute: true` |

### 8.3 Cryptographic Confirmation Tickets
Destructive or high-impact agent tools require a confirmation ticket:
1. Agent invokes tool (e.g. `bisync_trigger_sync` with `dry_run: false`).
2. Server computes parameter fingerprint: `SHA256(canonical(params))` and issues a short-lived ticket (`TTL = 300s`).
3. Agent must return the exact ticket alongside identical parameters. Altering parameters invalidates the ticket.


---

## 9. Troubleshooting & Support Playbook

### 9.1 Stale Advisory Locks
- **Symptom**: Sync attempts fail with `ErrLockHeld: another sync process holds advisory lock`.
- **Cause**: Previous bisync process was killed forcefully (`kill -9`) or system rebooted ungracefully while lock was held.
- **Resolution**:
  1. Check if the process is genuinely running:
     ```bash
     ps aux | grep rclone-bisync
     ```
  2. If dead, remove the lock file:
     ```bash
     rm /tmp/rclone-bisync-*.lock
     ```
  3. Ensure `check_stale_locks: true` is enabled in Global Daemon Settings so the engine cleans orphaned locks automatically.

---

### 9.2 Canary File Missing (`RCLONE_TEST`)
- **Symptom**: Sync fails with `CanaryNotFound: RCLONE_TEST missing on Path1/Path2`.
- **Cause**: Remote storage mount dropped, directory was purged, or canary was never initialized.
- **Resolution**:
  1. Verify the remote is mounted and accessible:
     ```bash
     rclone lsd remote_name:
     ```
  2. If the path is correct and intentionally empty, create the canary marker:
     ```bash
     touch /path/to/local/root/RCLONE_TEST
     rclone touch remote_name:remote_path/RCLONE_TEST
     ```

---

### 9.3 Cloud Rate Limiting (HTTP 429 / Quota Exceeded)
- **Symptom**: Sync job fails with `rateLimit: User Rate Limit Exceeded`.
- **Cause**: Cloud provider (e.g. Google Drive, OneDrive) throttled requests due to excessive API queries.
- **Resolution**:
  1. The engine automatically enters backoff cooling for `rate_limit_backoff_hours` (default 1 hour).
  2. In Global Settings → Engine Settings, reduce concurrency: append `--tpslimit 10` or `--transfers 4` to Custom Flags.

---

### 9.4 Resync Required After Major Directory Changes
- **Symptom**: Rclone reports `bisync cannot continue: path changed without history`.
- **Cause**: Upstream files were renamed or modified outside of bisync awareness.
- **Resolution**:
  1. In Cockpit → Profile Actions, select **Resync (Rebuild Cache)**.
  2. Bisync runs a single-direction baseline generation to rebuild internal listing caches.

---

## 10. CLI Reference & Service Management

### 10.1 CLI Commands

```bash
# Start web cockpit dashboard
./bin/rclone-bisync dashboard --port 8080

# Run sync on a specific profile
./bin/rclone-bisync sync --profile <profile_id>

# Run dry-run sync without altering files
./bin/rclone-bisync sync --profile <profile_id> --dry-run

# Start MCP server over stdio
./bin/rclone-bisync mcp

# Start MCP server over SSE
./bin/rclone-bisync mcp --transport sse --port 8081

# Verify automated quality gate stages
./scripts/check.sh all
```

### 10.2 Systemd User Service
Bisync Hub runs as a user-level systemd service:

```bash
# Check service status
systemctl --user status rclone-dashboard.service

# Restart service after binary updates
systemctl --user restart rclone-dashboard.service

# Inspect live journal logs
journalctl --user -u rclone-dashboard.service -f
```

