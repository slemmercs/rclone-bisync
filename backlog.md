# Rclone Bisync Enterprise Automation Hub - Architecture & Roadmap Backlog

## 1. Active Implementation Sprint (Incremental Branch Roadmap)

* **Branch 1: `refactor/clean-env-breaking-change` [MERGED]**
  * Eliminate legacy `bisync_config.py` shim completely (breaking change to pure `.env` configuration).
  * Refactor `install-custom-rclone.sh`: remove destructive `apt-get remove` system package removals (Rule 3) and remove references to `bisync_config.py`.
  * Add `pytest.ini` (`pythonpath = .`) for frictionless CLI test execution.
  * Update `tests/test_config.py`: replace deprecated shim test with strict tests for `bisync.env` and `rc-secrets.env` parsing and missing `RC_PASS` exception handling.

* **Branch 2: `refactor/service-layer-and-rc-correlation` [MERGED]**
  * Extract domain logic out of `dashboard/backend/main.py` into `rclone_bisync/services/sync_service.py` (`SyncService` class):
    * Systemd transient execution (`run_bisync_systemd`) with origin tracking.
    * Process/lock actions (`start_sync`, `stop_sync`, `force_resync`, `force_sync`, `dry_run`, `remove_lock`, `clear_rate_limit`).
    * Cached daemon health checking and live latency probing.
    * Structured pending deletion parsing with fallback.
    * Backward-seeking log parsing with JSON unwrapping.
  * Correlate Rclone RC daemon logs with orchestrator `run_id`:
    * Pass `_group=f"bisync-{run_id}"` to Rclone RC `/sync/bisync`.
    * Record Rclone RC `job_id` into `bisync.log` and `metrics.jsonl` alongside `run_id`.
  * Streamline `dashboard/backend/main.py` to a thin HTTP controller.

* **Branch 3: `feat/security-pydantic-dtos-and-redaction`**
  * Define strict Pydantic response models in `rclone_bisync/models/api.py`:
    * `ActionResponse`, `RemotesResponse`, `PendingDeletionsResponse`, `BrowseResponse`, `ReconnectResponse`, `CanaryResponse`.
  * Implement recursive deep secret sanitization (`redact_secrets()`) to prevent credential, token, or webhook leakage over HTTP.
  * Enforce `response_model` across all FastAPI route handlers.

* **Branch 4: `feat/remote-oauth-reauth`**
  * Backend: Implement `reconnect_remote(name)` executing desktop-integrated OAuth flow (`rclone config reconnect {name}:`) preserving user session environment (`DBUS_SESSION_BUS_ADDRESS`, `DISPLAY`, `XDG_RUNTIME_DIR`).
  * Endpoint: `POST /api/remotes/{name}/reconnect`.
  * Frontend: Add "Reconnect / Re-Auth" action button to Remote Cards with reactive loading state and notification toasts.

* **Branch 5: `feat/cockpit-workspace-and-file-explorer` [IMPLEMENTED]**
  * Backend: Safe local and remote directory traversal with path traversal jails; `/api/browse/canary` provisioning via `/operations/touch`; `/api/browse/copy` file & folder transfers via `/operations/copyfile` & `/sync/copy`; `/api/browse/delete` file & directory deletion via `/operations/deletefile` & `/operations/purge` with jail checks, root guards, and canary protection.
  * Frontend: Cockpit-inspired 3-tab workspace navigation (`Engine & Telemetry`, `Storage Remotes`, `File Explorer`) with live pulsing latency and daemon health pills (`● LIVE | <latency>ms`).
  * Explorer:
    * Dual-pane layout (Local Left, Remote Right) with equalized heights, interactive breadcrumbs with one-click path clipboard copy, real-time client-side search filters, and canary status pills with one-click touch provisioning.
    * Classic multi-key column sorting (`Name`, `Type`, `Modified`, `Size`) with folders pinned first on **both** local and remote panes with interactive direction indicators (`▲`/`▼`) and formatted row modification dates.
    * Directional file/folder transfers via 1-click copy buttons and HTML5 drag-and-drop with drag-conflict prevention (`draggable="false"`, `@mousedown.stop`) and standard mouse hover cursor (`cursor-default`).
    * Configurable multi-step file & folder deletion: toolbar safety selector (`Double Confirmation (Default - Safe)`, `Single Confirmation`, `Direct`) persisted in `localStorage` with a 2-step review modal and automated pane refresh.


* **Branch 6: `feat/interactive-log-cross-filtering`**
  * UI Component: Render `run_id` as an interactive, clickable chip/pill in the Bisync log viewer.
  * Cross-Filter Action: Clicking a `run_id` chip highlights that execution cycle and filters the `rclone-daemon.log` panel to display only the daemon entries matching that run's execution window and `job_id`.
  * Filter Clear / Reset: Provide a visual pill `[Filter: run_id=xxx ✕]` allowing one-click return to the live unfiltered daemon log stream.

* **Branch 7: `test/backend-integration-suite` [IMPLEMENTED]**
  * Added `tests/test_dashboard_backend.py` using `fastapi.testclient.TestClient`.
  * Validated CSRF protections (`X-Dashboard-Request: 1`, loopback origin validation).
  * Validated endpoints (`/api/telemetry`, `/api/action`, `/api/remotes`, `/api/browse`, `/api/deletions/pending`).
  * Verified recursive secret redaction prevents token exposure.

* **Branch 8: `test/playwright-ui-automation`**
  * Add `playwright` to dev dependencies (`requirements.txt`).
  * Implement headless browser test suite (`tests/ui/test_dashboard_ui.py`) leveraging the native `/usr/bin/google-chrome-stable` binary.
  * Verify Alpine.js reactive components: health pills, tab navigation, remote cards, dual-pane explorer (left local, right remote), log cross-filtering, and pending deletions modal.

---

## 2. Future Releases & Extended Backlog

* **Multi-Profile Synchronization Hub:**
  * Profile registry supporting independent cloud remotes (e.g., Google Drive personal, OneDrive work, Nextcloud).
  * Profile-scoped locks (`bisync_<profile>.lock`) and isolated metric streams (`metrics_<profile>.jsonl`).
  * Multi-tab navigation in the UI to switch profile telemetry.

* **In-Dashboard Configuration Editor & 3-Layer Safety Pipeline:**
  * Visual parameter adjustment for `--buffer-size`, `--tpslimit`, and conflict resolution strategies.
  * 3-Layer Safety Pipeline: Automated snapshot backup, SHA-256 integrity match, and schema validation.
  * Rollback API: One-click restore of historical configurations.

* **Side-by-Side Deletion & Conflict Diff Visualizer:**
  * When `TOO_MANY_DELETES` aborts, automatically open File Explorer highlighting deletions in red and conflicts in amber before requesting authorization with `--force`.

* **Desktop Mode via PyWebView (`--mode=desktop`):**
  * Optional launch flag to spawn the dashboard in a borderless native window without needing an external browser.

* **Portable Bundle Generator:**
  * Export endpoint packaging sanitized configs, scripts, and systemd units into `.tar.gz` for cross-device migration.
