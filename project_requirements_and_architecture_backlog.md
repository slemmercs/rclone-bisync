# Rclone Bisync Enterprise Automation Hub - Architecture & Requirements Backlog

## 1. Bulletproof Configuration Safety & Recovery
* **Automated Snapshot Generation:** Timestamped backups of configuration files (`bisync_config.py`) stored in a dedicated `config_backups/` directory prior to any write operation, paired with sliding retention pruning.
* **3-Layer Verification Pipeline:**
  * *Layer 1 (Integrity):* Bit-for-bit SHA-256 hash matching between source and backup files.
  * *Layer 2 (Syntax):* Abstract Syntax Tree (`ast.parse`) validation ensuring parseable Python code.
  * *Layer 3 (Schema):* Key-value structural validation post-copy and post-restore.
* **Rollback Management API:** Backend endpoints (`/api/config/backups`, `/api/config/rollback`) enabling instant, verified restoration of historical configuration states.

## 2. Multi-Provider Bisync Abstraction & Hybrid Plugin Architecture
* **Hybrid Fallback Pattern:** Specialized tuning for core storage providers (e.g., Google Drive) with seamless fallback to a safe `GenericProvider` for any unknown, niche, or newly released rclone backend.
* **Modular Hooks (`Pre-Sync` / `Post-Sync`):** Isolated plugin classes handling provider-specific pre-flight checks, token validations, and cleanups.
* **Dynamic Command Builder:** Assembling execution commands dynamically based on active remote capabilities rather than relying on hardcoded provider scripts.

## 3. Pre-Flight "Dry-Run" & Custom Safety Guardrail Engine
* **Dry-Run Diff Simulation:** Automatically executing rclone's `--dry-run --verbose` prior to live sync execution to map out incoming file modifications (uploads, downloads, deletes, conflicts).
* **Custom Rule Builder (Guardrails):** A dedicated rule configuration engine allowing users to set deterministic safety thresholds:
  * *Deletion Limits:* Automatic abortion if a run proposes deleting more than a specified file count or percentage of the repository.
  * *Anomaly Detection:* Flagging unusual bulk changes or recursive empty-directory risks.
  * *Path Exclusions:* Rule-based filtering to ignore temporary system files or unwanted extensions.

## 4. Linux Native Notifications (`libnotify`) & Webhooks
* **Desktop Integration:** Triggering native Linux desktop notifications via `notify-send` on Tuxedo OS whenever a sync finishes or encounters errors.
* **Remote/Headless Webhooks:** Flexible notification dispatchers (Discord, Telegram, Gotify, Slack) for mobile alerting during headless server deployments.

## 5. The "Dead-Man's Snitch" Watchdog
* **Silent Failure Protection:** Lightweight heartbeat pings sent to an external service (Healthchecks.io, ntfy, or local endpoints) upon successful sync completion.
* **Timer Failure Detection:** Automated alerts if a scheduled systemd timer fails to report in due to laptop sleep, network dropouts, or process stalls.

## 6. Interactive Rclone RC File Browser
* **Side-by-Side Exploration:** Utilizing rclone RC endpoints (`/operations/list`, `/operations/about`) to browse local directories and remote storage buckets side-by-side.
* **Granular Control:** Inspecting file metadata, verifying folder structures, or triggering single-file operations directly within the dashboard UI.

## 7. Unified Log Correlation & Origin Tracking
* **Universal Run ID Generation:** Generating a unique tracing identifier (`run_id`) at the start of every sync execution to logically bind disparate log streams.
* **Origin Differentiation:** Automatically tagging execution sources as **`user`** (ad-hoc dashboard triggers) or **`timer`** (automated systemd schedules).
* **Log & Metrics Enrichment:** Prefacing wrapper logs (`bisync.log`) with the run ID and injecting correlation keys directly into the metrics ledger (`metrics.jsonl`).
* **UI Filtering:** Click-to-filter workflows in the dashboard to isolate log streams by run ID or trace.

## 8. Dual-Mode Installation & Portable Export
* **Desktop vs. Headless Mode:** Launch flag support (`--mode=desktop` vs `--mode=headless`) to spawn a native window via PyWebView or run as a pure network-bound server.
* **Portable Bundle Generator:** A backend export endpoint packaging a sanitized `rclone.conf`, core sync scripts, and systemd unit templates into a downloadable `.zip` archive for cross-computer deployment.

## 9. Frontend Framework & Architecture
* **Tech Stack:** **Vite + Svelte + Tailwind CSS** SPA architecture for reactive state management, rich form handling, and modal workflows.
* **Desktop Packaging:** Bundling the FastAPI backend and compiled Svelte frontend into a standalone native binary via PyInstaller and PyWebView.

## 10. Testing, Contracts & Quality Gates
* **Backend Contract Tests (`pytest`):** Validating Pydantic schemas, telemetry payloads, and the 3-layer configuration safety pipeline.
* **Frontend Test Integration (`Vitest` + Testing Library):** Ensuring UI components correctly handle mock telemetry contracts and reactive state changes.
* **System-Level Polish:** Systemd service hardening, log rotation (`logrotate`), and write-only masked UI inputs (`********`) for sensitive OAuth credentials.