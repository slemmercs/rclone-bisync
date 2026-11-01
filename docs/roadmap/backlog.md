# Bisync Hub - Architecture & Extended Roadmap Backlog

## 1. Completed Go Single-Binary Architecture Port (Phases 1–6)

The project has transitioned from Python to a native, single-binary Go architecture. This eliminates virtual environments (`.venv`), eliminates global pip/apt conflicts, provides embedded SPA dashboard distribution via `//go:embed`, reduces RAM footprint from ~60MB to ~12MB, and provides native Rclone RC ecosystem alignment.

* **Phase 1: `feat/go-core-engine-and-config` [COMPLETED - PR #9]**
  - Bootstrapped Go toolchain and initialized Go module (`github.com/slemmercs/rclone-bisync`).
  - Ported configuration loader (`config.go`): parsing `bisync.env` and `rc-secrets.env`, strict missing `RC_PASS` fail-fast validation.
  - Ported POSIX advisory lock manager (`lock.go`): `flock` semantics, PID liveness check, stale lock recovery.
  - Ported notification dispatcher (`notifier.go`): `notify-send` desktop integration with dynamic icon lookup and optional webhook delivery.
  - Ported unit tests for config, lock, and notifier.

* **Phase 2: `feat/go-rc-client-and-security` [COMPLETED - PR #9, PR #10]**
  - Ported Rclone RC HTTP client (`client.go`): HTTP connection pooling, basic auth, timeouts, error classification.
  - Implemented recursive deep secret sanitization engine (`security.go`): scrubbing nested maps, slices, structs for credentials and tokens.
  - Fixed empty JSON payload handling on RC POST endpoints (PR #10).
  - Ported unit tests for RC client and recursive secret redaction.

* **Phase 3: `feat/go-sync-service-and-explorer` [COMPLETED - PR #9]**
  - Ported `SyncService` & dual-pane explorer backend: path traversal jail enforcement (`filepath.Rel`), canary check and provisioning (`touch_file`), file/folder copy and deletion guards.
  - Ported pending deletion parsing and backward-seeking log reader.
  - Ported unit tests for services and explorer safety.

* **Phase 4: `feat/go-rest-api-and-embedded-ui` [COMPLETED - PR #9]**
  - Implemented standard library HTTP REST router with CSRF protection (`X-Dashboard-Request`, loopback origin checking).
  - Concrete Go struct request/response DTOs matching JSON contracts.
  - Embedded Alpine/Tailwind frontend SPA directly into the compiled executable via `//go:embed`.
  - Ported HTTP API integration tests.

* **Phase 5: `feat/go-systemd-and-installer` [COMPLETED - PR #9]**
  - Unified CLI command structure (`rclone-bisync sync`, `rclone-bisync dashboard`, `rclone-bisync canary`).
  - Updated systemd user units to target `./rclone-bisync` binary (eliminating `.venv` references).
  - Streamlined `install-custom-rclone.sh` to a fast single-binary installer.
  - Updated `scripts/check.sh` quality gate to run `go vet`, static checks, secret leak verification, and `go test ./...`.

* **Phase 6: `refactor/frontend-modularization` [COMPLETED - PR #11]**
  - Modularized frontend scripts into zero-build sub-modules: `utils.js`, `charts.js`, `remotes.js`, `explorer.js`, coordinated by `app.js`.
  - Maintained zero npm/Node build dependencies.

---

## 2. Completed Implementation Sprint: Test Coverage Uplift & Quality Gate (>80%) [COMPLETED - PR #17]

Targeted and achieved statement coverage >80% across all Go packages and resolved toolchain coverage caveats:

- Added `main_test.go` eliminating root package empty-test `covdata` toolchain error under `go test -cover ./...`.
- Uplifted `internal/core/state` (>90%): rate-limit backoff flag lifecycle, notification state persistence, metrics JSONL appending.
- Uplifted `internal/core/client` (>85%): mock tests for `TouchFile`, `CopyFile`, `CopyDir`, `DeleteFile`, `PurgeDir`, `ListRemotes`, `GetRemoteAbout`.
- Uplifted `internal/services` (>80%): added `telemetry_test.go` (stats, cached metrics, secret redaction, telemetry payload) and tested `CopyItem`, pending deletion log parsing, and remote actions.
- Uplifted `internal/engine/runner` & `internal/server` (>85%): lock contention, rate limit abort, job error classification, and full HTTP endpoint integration tests.
- Enhanced `scripts/check.sh` quality gate to an 8-stage verification pipeline with cross-compilation and automated coverage threshold enforcement.

---

## 3. Extended Roadmap & Active Backlog

### Recently Completed Epics

* **[COMPLETED] Sync Service Modularization & Domain Separation [COMPLETED - PR #18]**
  - Decoupled monolithic `internal/services/sync_service.go` into cohesive single-responsibility domain services: `TelemetryService`, `CanaryService`, `DeletionService`, `ExplorerService`, and `SyncService`.

* **[COMPLETED] Rebrand to Bisync Hub & Lucide SVG Standard [COMPLETED - PR #20]**
  - Renamed the project, dashboard, and user-facing assets to **Bisync Hub** (*"Review. Decide. Sync."*).
  - Standardized UI icons with 100% offline, locally-embedded Lucide SVGs.

* **[COMPLETED] Repository Quality Hardening & Security Jails [COMPLETED - PR #21]**
  - Stale lock PID verification guard, path confinement symlink evaluation (`filepath.EvalSymlinks`), and private LAN origin support.

* **[COMPLETED] Interactive Sync Review & Deletion Diff Visualizer [COMPLETED - PR #22]**
  - Core safety controller: dual backup staging (zero-byte POSIX hard links local, server-side remote copy) and saga rollback runners.

* **[COMPLETED] Heuristic Conflict Recommendation Engine [COMPLETED - 2026-10-23]**
  - Pure-Go offline analyzer pipeline ($G=4$) with confidence scoring ($0.00–1.00$) and risk classification.

* **[COMPLETED] Frontend Template Modularization (`html/template`) [COMPLETED - Refactor]**
  - Decomposed monolithic `index.html` into Go `html/template` partials served via `embed.FS`.

* **[COMPLETED] Immutable Audit Ledger & Unified Log Architecture [COMPLETED - PR #27]**
  - Cryptographically chained SQLite append-only audit ledger (`internal/audit`) using pure-Go `modernc.org/sqlite` and `pressly/goose/v3`.
  - Schema triggers (`BEFORE UPDATE`, `BEFORE DELETE`) enforcing write-once immutability.
  - In-process thread-safe size-bounded POSIX log rotator with dual-stream audit logging (`"type": "audit"`).
  - Web UI Audit tab with in-browser SHA-256 chain verification, action/actor filters, and CSV/JSONL compliance export.

* **[COMPLETED] Service Layer Decoupling & Sub-400 LOC Enforcement [COMPLETED - PR #28, PR #29, PR #31]**
  - Decomposed services into cohesive single-responsibility domain engines strictly under 400 LOC (`ReviewService`, `TelemetryService`, `CanaryService`, `DeletionService`, `ExplorerService`, `SyncService`).
  - Centralized frontend API client (`api.js`) and resolved Alpine boolean attribute reactivity edge cases.

* **[COMPLETED] Go CLI Quality Gate (`cmd/check`) & Headless Chrome E2E Testing [COMPLETED - PR #32]**
  - High-performance Go CLI check engine (`cmd/check`) with parallel check runners and deterministic POSIX exit codes.
  - Custom HTML attribute tokenizer detecting multi-line and dotted Alpine.js boolean attributes (`:disabled`, `:readonly`).
  - Headless Chrome end-to-end smoke test suite (`tests/e2e/dashboard_smoke_test.go` via `chromedp`) validating tab transitions and DOM state.
  - Verified overall statement test coverage at 88.3%.

* **[COMPLETED] Multi-Profile Engine Phase 1: Storage, Model & Scheduler [COMPLETED - PR #37]**
  - Pure-Go SQLite profile store (`internal/profiles/store.go`) with schema migrations (`profiles`, `profile_states`).
  - Isolated background scheduler (`internal/profiles/scheduler.go`) with per-profile goroutines, lock isolation (`/tmp/rclone-bisync-<id>.lock`), and profile telemetry collectors.
  - REST API routes (`/api/profiles`, `/api/profiles/{id}`, `/api/profiles/{id}/sync`, `/api/profiles/{id}/pause`).

* **[COMPLETED] Multi-Profile Engine Phase 2: Central DI Container & Subsystem Consolidation [COMPLETED - PR #40]**
  - Central dependency injection container (`internal/app/container.go`) managing lifecycle and service construction.
  - Engine consolidation: relocated review package to `internal/engine/review/` respecting bounded contexts.
  - Streaming secret redaction across all file sinks and backward log tailing (`filetail.go`).
  - CodeQL #15 dynamic allocation fix and test stabilization.

* **[COMPLETED] Multi-Profile Engine Phase 3: 3-Tier Operational Flags & Security Validation [COMPLETED - PR #42]**
  - 3-tier operational flags schema: Tier 1 (Execution Controls), Tier 2 (Profile Settings), Tier 3 (Custom Flags Escape Hatch).
  - POSIX shell argument tokenizer (`SplitCommandLine`) and prohibited flags blocklist (`--config`, `--rc`, `--use-mmap`, etc.).
  - Flag inheritance engine (`ResolveFlags`) implementing `Execution Override > Profile Override > Global Default > Hardcoded Fallback`.
  - Dynamic CLI argument builder (`BuildCLIArgs`) wired into profile execution runner.

* **[COMPLETED] Multi-Profile Engine Phase 4 & 5: Hybrid Fleet Overview, Provider Presets & Global Fleet Settings [COMPLETED - PR #41, PR #43 (`v1.0.0-beta.10`)]**
  - **Level 1 Fleet View (`view_fleet.html`)**: Fleet Health Summary banner with interactive filter pills, profile tile grid with topology, execution metrics, and quick-action buttons (`Sync Now`, `Review Diff`, `Settings`).
  - **Level 2 Scoped Workspace & Context Switcher**: Global header profile dropdown with status indicators and breadcrumbs to navigate between fleet view and scoped telemetry/explorer/audit panels.
  - **Visual Operational Flags & Provider Presets**: 3-tier flags configuration with inheritance badges (`Inherited from Global` vs `Profile Override`), provider-specific presets (`gdrive`, `onedrive`, `dropbox`, `s3`), and editable Global Fleet Defaults (`modal_global_settings.html`, `/api/config`).
  - **Deterministic Profile Sorting & Compare Mode Guidance**: Enforced `CreatedAt`/`Name` ordering and inline `--resync` guidance when switching `CompareMode`.


* **[COMPLETED] UI Design System, 10ms Telemetry, Inline Log Folding & Remote Hash Deduplication [COMPLETED - 2026-10-29]**
  - **Design System Foundation**: Semantic tokens (`design-tokens.css`), 7 Approved Status Vocabulary states, reusable partials (`partials/components/`, `partials/patterns/`, `partials/layouts/`), and forensic Slide-Over Entity Inspector Drawer.
  - **10ms Telemetry Pipeline**: Decoupled `rclone-daemon.log` parsing, cached `bisync.log` reads by `(mtime, size, limit)`, 60s TTL quota caching, and adaptive polling in `polling.js` (8s idle / 2s sync).
  - **50-Line On-Demand Pagination & O(N) Inline Log Folding**: Paginated API (`GET /api/logs/bisync?page=1&limit=50`), inline deduplication accordions with `×N` repeat counters, latest cluster occurrence timestamps, and reversed unified ledger (newest on top).
  - **Historical Deletion Scraper Bug Fix**: Remedied `deletion_service.go` to require an active circuit breaker abort without subsequent sync success, preventing historical completed deletions from surfacing as pending.
  - **Remote Hash Deduplication (`rclone dedupe --by-hash`)**: Safe consolidation of cloud file collisions (`modal_dedupe.html`, `/api/remotes/dedupe`) with dry-run preview, process lock isolation, and automated baseline cache resync.

* **[COMPLETED] Unified Logging, Profile-Scoped Execution Ledger & Telemetry De-Noising [COMPLETED - PR #49–#53 (v1.0.0-beta.14–beta.18)]**
  - Unified operational logging combining sync milestones, systemd daemon journal traces, and trigger origins (`CLI`, `CRON`, `UI`).
  - Demoted routine polling requests (`/api/status`, `/api/telemetry`) to DEBUG level to eliminate UI ledger noise.
  - Profile-scoped log isolation, reversed timeline ordering (newest on top), and real-time polling updates.

* **[COMPLETED] Pre-MCP Technical Debt Remediation & Strict Sub-400 LOC Enforcement [COMPLETED - PR #54 (v1.0.0-beta.19)]**
  - Modularized frontend scripts (`profiles.js` down to 154 LOC via `profiles_modal.js`, `review.js` down to 196 LOC via `review_actions.js`).
  - Decomposed `log_service.go` into `log_parser.go` and streamlined `server.go` to 375 LOC.
  - Enforced single-pass architectural LOC audit (`./scripts/check-loc.sh 400`).

* **[COMPLETED] Typed Service Contracts & Automated Copyleft License Guard [COMPLETED - PR #55 (v1.0.0-beta.20)]**
  - Introduced explicit Go model contracts (`models.DeleteItemRequest`, `models.CreateCanaryRequest`, `models.CanaryItemResponse`).
  - Uplifted `internal/remoteauth` test coverage to 92.3%.
  - Added native Go copyleft license check stage (`cmd/check/licenses.go`) and documented dependencies in `THIRD_PARTY_LICENSES.md`.

* **[COMPLETED] Native Model Context Protocol (MCP) Server (`rclone-bisync mcp`) [COMPLETED - PR #56 (v1.0.0-beta.20)]**
  - Built-in dual transport (`stdio` and `sse` over HTTP on `--port 8081`).
  - 10 structured read/write tools and 5 dynamic resources.
  - Cryptographic confirmation ticket engine with exact SHA-256 parameter fingerprint validation.
  - Token-optimized default status payload (195 characters, ~98.5% reduction) and 2,000-character context truncation on dedupe scans.
  - Maintained 84.5% statement test coverage across `internal/mcp`.

---

### Active Near-Term Backlog (Sequenced Priorities)

#### 1. Tier 1 Curated Options & Guided `+ Add Flag` Row Builder [ESTIMATE: 3-5h | 3 SP] - ACTIVE PRIORITY
Reclassifies provider options closer to Rclone standards, separates Tier 1 and Tier 2 user experiences, and replaces free-form text input with a guided, interactive flag builder:
* **Tier 1 Providers (`drive`, `onedrive`, `dropbox`, `s3`)**:
  - Keep curated, high-impact bisync toggles (e.g., for Google Drive: `Send Deletions to Trash`, `Skip Google Docs`, `Skip Shortcuts`, `Shared With Me`).
  - Fix default checkbox state persistence bug and ensure explicit boolean serialization (e.g., `--drive-use-trash=false`).
* **Tier 2 Providers (WebDAV, Nextcloud, SFTP, Generic)**:
  - Present an ultra-lean configuration profile with zero redundant UI boxes, directing advanced parameters to the flag builder.
* **Interactive `[ + Add Flag ]` Row Builder**:
  - Replace raw text input (`--fast-list --checkers 16`) with dynamic row-by-row additions (`[Flag Dropdown] [Value Input] [✕ Remove]`).
  - Provider-aware autocomplete (`<datalist>`) suggesting valid backend flags (`--drive-*`, `--onedrive-*`, etc.) and general tuning flags (`--fast-list`, `--transfers`, `--bwlimit`, `--tpslimit`) with inline tooltips.
  - Real-time POSIX tokenizer and security blocklist validation per row.

#### 2. Configuration & Setup Manager (Rollback & Snapshot Engine) [ESTIMATE: 5-7h | 4 SP]
Provides server-side configuration safety for global environments:
* **3-Layer Safety Pipeline**: Automated snapshot backups, SHA-256 integrity checks, and JSON/env schema validation before applying global configuration changes.
* **One-Click Rollback API**: Restore historical configuration snapshots on failed verification or operator rollback request.
* Note: Visual flag editing and profile onboarding are unified into Multi-Profile Phase 4.


#### 3. Unified Operational Log Ledger, Run ID Correlation & Deep Path Navigation [ESTIMATE: 4-5h | 3 SP] - LOCKED IN
Provides high-density, structured operational visibility aligned with the Audit Ledger:
* **Log Ledger Restructuring**: Align `bisync.log` unified viewer card to structured table format matching `view_audit.html` (`TIME`, `LEVEL`, `SOURCE`, `RUN ID`, `MESSAGE`, `ACTIONS`).
* **Run ID & Rev ID Cross-Correlation**: Clickable Run ID pills (`#run-...`) jumping directly to filtered Audit Ledger records and correlated daemon execution traces. Historical Review IDs (`@rev-...`) directly open the interactive review resolution modal.
* **Deep Path Tokenizer & Navigation**: Contextual path parser extracting file references (`Duplicate ignored`, `Deleted`, `Transferring`) with deep links that switch to File Explorer, load the target directory, and scroll/flash the target file row.
* Refer to [operational_visibility_and_queued_upload_specification.md](../design/operational_visibility_and_queued_upload_specification.md).

#### 4. Remote Cloud Web Link Navigation ("Open on Remote in New Tab") [ESTIMATE: 2-3h | 2 SP] - LOCKED IN
Enables instant one-click navigation to cloud provider web interfaces:
* **Rclone RC Public Link Integration**: Connect to Rclone RC `operations/publiclink` to retrieve canonical web URLs for cloud files and directories (Google Drive, OneDrive).
* **Explorer & Log Integration**: "Open on Remote" quick-action icon embedded in remote File Explorer table rows and rendered next to remote path chips in the unified log ledger.
* **Graceful Degradation**: Capability detection that dynamically hides or disables external link actions on storage tiers lacking web interfaces (local, SFTP, S3) without throwing errors.
* Refer to [operational_visibility_and_queued_upload_specification.md](../design/operational_visibility_and_queued_upload_specification.md).

#### 5. Queued Dual-Pane File Explorer Direct Upload [ESTIMATE: 5-6h | 4 SP] - LOCKED IN
Enables direct file upload capabilities with zero sync lock collisions:
* **Dual-Pane Upload Targets**: Support uploading to both Local (jailed) and Remote directories via file selector and drag-and-drop dropzones.
* **Non-Blocking Asynchronous Upload Queue**: Automatically queues file uploads during active synchronization (`telemetry.is_syncing === true` or POSIX flock contention) in a floating dock instead of rejecting them with HTTP 409.
* **Sequential Auto-Resume**: Automatically triggers sequential uploads once active sync runs finish and lock contention releases.
* **Mid-Upload Lock Protection**: Advisory upload coordination delaying scheduled sync runs while active multipart streams are in-flight.
* Refer to [operational_visibility_and_queued_upload_specification.md](../design/operational_visibility_and_queued_upload_specification.md).

#### 6. In-App User Guidance, Info & Help Documents [ESTIMATE: 3-5h | 3 SP]
Improves operational transparency and onboarding for new users:
* **Contextual Tooltips & Policy Banners**: Embeds inline tooltips across `partials/view_*.html` explaining safety mechanisms (`TOO_MANY_DELETES`, `MANDATORY_SUBDIR`, heuristic risk scores) in non-technical terms.
* **In-Dashboard Reference Manual**: A slide-over help panel summarizing CLI commands, Rclone flag impacts, and step-by-step conflict recovery workflows.
* **Diagnostic Documentation**: Integrated guides for resolving OAuth token drops, path permissions errors, and systemd service status.

#### 7. Remote Mount Lifecycle Manager & Pooled Storage Engine (`internal/mount`) [ESTIMATE: 8-10h | 8 SP]
Programmatic management of VFS virtual drive mounts and multi-cloud storage pooling:
* **Stage A (Remote Mount Manager)**: Mount, inspect, configure, and unmount individual remotes to local paths via Rclone RC `/mount/*` endpoints. Keepalive probes with automatic reconnect on transient network drops.
* **Stage B (Pooled Storage Engine)**: Provision stacked multi-cloud topologies (`union` + `chunker` + `crypt`) over RC `/config/create`. Combines disparate storage tiers into a single encrypted virtual drive guarded by Bisync Hub's safety probes.
* Refer to [`docs/design/remote_mount_manager_design_specification.md`](../design/remote_mount_manager_design_specification.md).

---

## 4. Medium & Infrastructure Backlog

* **Native Browser Notifications [ESTIMATE: 3-4h | 2 SP]**
  - Integrate HTML5 Web Notification API.
  - Alert operator when safety barriers trigger (`TOO_MANY_DELETES`), sync operations fail, or long runs finish.
  - Includes window focus handlers to bring Bisync Hub dashboard to front when clicking notifications.

* **File Explorer Content Preview [ESTIMATE: 4-5h | 3 SP]**
  - Modal content viewer for text/code files (up to 2MB) with syntax highlighting.
  - Safe inline rendering for common image formats (PNG, JPG, SVG, WebP up to 5MB) directly from local jail and remote streams.

* **Rclone Remote Control (RC) Framework Hardening [ESTIMATE: 4-5h | 3 SP]**
  - HTTP connection pooling, retry backoff with jitter for HTTP 429 throttling, dynamic API version compatibility checks.

* **Remote Mount Manager & Pooled Storage Engine (`internal/mount`) [ESTIMATE: 8-10h | 8 SP]**
  - Programmatic VFS mounting (`/mount/*`) and multi-cloud storage pooling (`union` + `chunker` + `crypt`) via RC `/config/create`.

* **Local Inter-Process Communication via Unix Sockets [ESTIMATE: 4-6h | 3 SP]**
  - Communicate over `unix:///tmp/rclone.sock` with strict `0600` permissions and graceful process cleanup.

* **Auto-Generated Cryptographic Credentials Engine [ESTIMATE: 3-4h | 2 SP]**
  - Automated installer script (`install.sh`) generating 32-character random credentials with `0600` permissions.

* **Desktop Application Packaging via Wails [ESTIMATE: 3-4h | 3 SP]**
  - Package Go backend and embedded frontend as a standalone native desktop app.
