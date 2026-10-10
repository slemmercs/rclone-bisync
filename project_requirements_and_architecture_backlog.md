# Rclone Bisync Enterprise Automation Hub - Architecture & Requirements Backlog

## 1. WebUI Remote Management & Discovery
* **Active Remote Discovery:**
  * Endpoint: `GET /api/remotes` leveraging Rclone RC `POST /config/listremotes`.
  * Returns all configured backends (Google Drive, OneDrive, S3, WebDAV) with provider type and connection status.
* **Remote Health & Quota Probing:**
  * Endpoint: `GET /api/remotes/{name}/about` leveraging Rclone RC `POST /operations/about` (`fs="{name}:"`).
  * Queries total storage, used capacity, trash usage, and free capacity to render visual quota telemetry bars in the UI.
* **Interactive Remote Re-Authentication:**
  * WebUI action triggering OAuth re-authentication (`rclone config reconnect {name}:`) or updating credentials with masked UI inputs (`********`).

## 2. Interactive Remote File & Canary Browser
* **Hierarchical Remote Exploration:**
  * Endpoint: `POST /api/browse` accepting `{ "remote": "gdrive:", "path": "Documents" }`.
  * Calls Rclone RC `POST /operations/list` with `{ "fs": "{remote}", "remote": "{path}", "opt": { "recurse": false } }`.
  * Renders folders and file objects with human-readable file sizes, MIME icons, and modification timestamps.
* **Canary & Baseline Inspection:**
  * Visual status pill in the UI indicating whether the required `RCLONE_TEST` canary file is present on the selected remote and local directory paths.
  * One-click option to provision `RCLONE_TEST` when creating a new synchronization baseline.
* **Side-by-Side Conflict & Deletion Diff:**
  * When a `TOO_MANY_DELETES` safety abort occurs, the WebUI opens the file browser highlighting proposed deletions in red and conflicts in amber before requesting authorization with `--force`.

## 3. WebUI Configuration Management & Safe Re-Tuning
* **Visual Parameter Topology:**
  * Throughput controls: Interactive sliders and input masks for `--buffer-size` (16M, 32M, 64M), `--tpslimit`, and `--retries`.
  * Conflict resolution selector: Strategy dropdowns for `--conflict-resolve` (`newer`, `older`, `path1`, `path2`) and `--conflict-loser` (`num`, `pathname`).
  * Drive backend options: Toggles for `--drive-skip-gdocs`, `--drive-use-trash`, and export formats.
* **3-Layer Configuration Safety Pipeline:**
  * *Automated Snapshot Generation:* Timestamped backups of configuration files (`config.py`) stored in `config_backups/` prior to any write operation, paired with sliding retention pruning.
  * *Verification Gate 1 (Integrity):* Bit-for-bit SHA-256 hash matching between source and backup files.
  * *Verification Gate 2 (Syntax):* Abstract Syntax Tree (`ast.parse`) validation ensuring parseable Python code prior to saving.
  * *Verification Gate 3 (Schema):* Structural key-value schema validation.
* **Rollback Management API:** Backend endpoints (`/api/config/backups`, `/api/config/rollback`) enabling instant, verified restoration of historical configuration revisions.

## 4. Multi-Profile Synchronization Hub
* **Profile Parameterization:**
  * Central profile registry allowing multiple independent remotes (e.g., Google Drive personal, OneDrive work, Nextcloud).
  * Scoped locking (`bisync_<profile>.lock`) and isolated metrics ledgers (`metrics_<profile>.jsonl`).
* **WebUI Multi-Profile Tabs:**
  * Multi-tab navigation in the dashboard allowing users to switch between profile views, trigger individual syncs, and monitor dedicated performance charts.

## 5. Structured JSON Safety Abortion Reports
* **Structured Deletion Ledgers:**
  * Atomic generation of `pending_deletions.json` containing exact deleted file lists, source directory, and target action whenever `TOO_MANY_DELETES` aborts.
  * Eliminates fragile text scraping of daemon logs, serving instant deletion modals via `/api/deletions/pending`.

## 6. Headless Watchdog ("Dead-Man's Snitch")
* **Heartbeat Dispatcher:**
  * Lightweight HTTP GET/POST ping sent to external monitoring endpoints (Healthchecks.io, ntfy, Uptime Kuma) upon successful sync completion.
* **Silent Failure Detection:**
  * Triggers immediate alerting if a laptop stays suspended, a network drops, or systemd timer fails to wake up on schedule.

## 7. Dual-Mode Installation & Portable Export
* **Desktop vs. Headless Mode:** Launch flag support (`--mode=desktop` vs `--mode=headless`) to spawn a native window via PyWebView or run as a pure network-bound server.
* **Portable Bundle Generator:** A backend export endpoint packaging a sanitized `rclone.conf`, core sync scripts, and systemd unit templates into a downloadable archive for cross-computer deployment.

## 8. Testing, Contracts & Quality Gates
* **Backend Contract Tests (`pytest`):** Validating Pydantic schemas, telemetry payloads, error classification, fcntl locks, and the 3-layer configuration safety pipeline.
* **Frontend Test Integration:** Ensuring UI components correctly handle mock telemetry contracts and reactive state changes.