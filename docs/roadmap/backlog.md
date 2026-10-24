# Bisync Hub - Architecture & Extended Roadmap Backlog

## 1. Completed Go Single-Binary Architecture Port (Phases 1–6)

The project has transitioned from Python to a native, single-binary Go architecture. This eliminates virtual environments (`.venv`), eliminates global pip/apt conflicts, provides embedded SPA dashboard distribution via `//go:embed`, reduces RAM footprint from ~60MB to ~12MB, and native Rclone RC ecosystem alignment.

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

## 3. Historical Branches Reference (Python Implementation)

* **Branch 1: `refactor/clean-env-breaking-change` [MERGED]**
* **Branch 2: `refactor/service-layer-and-rc-correlation` [MERGED]**
* **Branch 3: `feat/security-pydantic-dtos-and-redaction` [MERGED]**
* **Branch 5: `feat/cockpit-workspace-and-file-explorer` [MERGED]**
* **Branch 7: `test/backend-integration-suite` [MERGED]**

---

## 4. Extended Roadmap & Backlog

### Completed Strategic Go Epics

* **[COMPLETED] Sync Service Modularization & Domain Separation [COMPLETED - PR #18]**

  - Decoupled monolithic `internal/services/sync_service.go` into cohesive single-responsibility domain services: `TelemetryService`, `CanaryService`, `DeletionService`, `ExplorerService`, and a lean `SyncService`.
  - Maintained strict path traversal jails, atomic operations, backward-compatible delegators, and >80% statement test coverage across all domain services (internal/services: 85.4%, internal/server: 83.2%).
  - Established prerequisite foundation for Phase 4 Interactive Sync Review and Multi-Profile Hub.

* **[COMPLETED] Rebrand to Bisync Hub & Lucide SVG Standard [COMPLETED - PR #20]**

  - Renamed the project, dashboard, documentation, and user-facing assets to **Bisync Hub**.
  - Adopted the project positioning: *"The safety-first dashboard for Rclone Bisync"*.
  - Introduced the long-term vision: *"Review. Decide. Sync."*.
  - Added `THIRD_PARTY_LICENSES.md` providing legal attribution for Lucide Icons (ISC/MIT), Alpine.js (MIT), Chart.js (MIT), Tailwind CSS (MIT), and Go sys (BSD 3-Clause).
  - Standardized and replaced all dashboard UI icons with 100% offline, locally-embedded [Lucide](https://lucide.dev) SVGs (enforcing uniform stroke width 2, 24x24 viewBox, and slate/indigo integration without npm/Node build dependencies).
  - Aligned README, deployment systemd units (`deploy/systemd/rclone-dashboard.service`), desktop launcher (`deploy/desktop/rclone-dashboard.desktop`), installer (`deploy/install.sh`), and AI disclosure.

* **[COMPLETED] Centralized OAuth Connection Manager & Google Drive Migration Wizard [ESTIMATE: 3-4h | 3 SP]**

  - Backend: Implemented `RemoteAuthService` covering auth health assessment, desktop-integrated OAuth reconnect runner (`rclone config reconnect {name}: --auto-confirm`), credentials updater, provider validator, and thread-safe audit logger (`internal/remoteauth`).
  - Endpoints: `GET /api/remotes/auth`, `POST /api/remotes/{name}/reconnect`, `POST /api/remotes/{name}/credentials`, `POST /api/remotes/{name}/validate`, `GET /api/remotes/auth/audit`, `POST /api/remotes/auth/oauth-stop`.
  - Frontend: Integrated Authentication Health overview bar, remote card badges, Google Drive client ID migration wizard, reconnect progress popup, and localized audit trail viewer.
  - Refer to [oauth-connection-manager.md](../design/oauth-connection-manager.md)

* **[COMPLETED] Structured Daemon Log Modal Refinement [ESTIMATE: 3-4h | 3 SP]**

  - Refined `internal/services/log_service.go` to unwrap nested JSON log strings from `bisync/rc.go` wrapper entries and match individual event timestamps.
  - Enhanced frontend `utils.js` and `app.js` with structured metadata parsing (`objectType`, `source`, `details`), comparison settings grid, and defense-in-depth unwrapping.


---

* **[COMPLETED] Repository Quality Hardening, Security Jails & Architectural Modularization [COMPLETED - PR #21]**

  - Stale lock PID verification guard preventing lockfile removal while owning PID is alive (`internal/core/lock/lock.go`).
  - Path confinement symlink evaluation (`filepath.EvalSymlinks`) blocking symlink jail escapes, and canary directory protection preventing purges of folders holding `RCLONE_TEST` (`internal/services/explorer_service.go`).
  - Private homelab LAN origin support (RFC 1918 subnets + `.local` mDNS) and graceful 5-second HTTP shutdown drain (`internal/server/server.go`, `main.go`).
  - Non-mutating environment file parser `ParseEnvFile()` and deep secret redaction unification (`internal/config/config.go`, `internal/core/security/security.go`).
  - Client package decomposition into `jobs.go`, `remotes.go`, `fs_ops.go`, `monitor.go`, and base `client.go` (`internal/core/client/`).
  - Server route modularization into dedicated handlers (`handlers_telemetry.go`, `handlers_explorer.go`, `handlers_auth.go`, `handlers_sync.go`) with concrete DTO response models (`internal/models/api.go`).
  - Quality gate uplift maintaining >82% statement coverage across all packages.

* **[COMPLETED] Phase 4 — Interactive Sync Review & Deletion Diff Visualizer [COMPLETED - PR #22]**

  - Primary strategic feature transforming Bisync Hub from an observer dashboard into an interactive safety controller.
  - Review domain models & WAL log contracts (`internal/models/review.go`).
  - Review service & persistent journal engine with atomic JSON journal and append-only WAL (`internal/review/journal.go`, `internal/services/review_service.go`).
  - Abort candidate discovery from bisync exit outputs (`internal/review/discovery.go`).
  - Reversible resolution executor with dual backup staging (zero-byte POSIX hard links for local files and cloud-to-cloud server-side copy for remotes) and saga rollback runner (`internal/review/backup.go`, `executor.go`, `rollback.go`).
  - REST API endpoints (`/api/review/session`, `/api/review/action`, `/api/review/execute`, `/api/review/rollback`, `/api/review/dismiss`) in `internal/server/handlers_review.go`.
  - Frontend dual-pane explorer status highlighting (red, amber, green) and interactive Review Modal ("Review. Decide. Sync.") in `dashboard/frontend/js/review.js`.

---

### Active Strategic Epic: Heuristic Conflict Recommendation Engine

Augments Phase 4 Interactive Sync Review with an intelligent decision pipeline. Evaluates candidate conflict file pairs concurrently using a chain of deterministic pure-Go heuristics, assigning actionable recommendations (`keep-local`, `keep-remote`, `keep-both`, `manual-review`), normalized confidence scores (0.00 to 1.00), and risk classifications (`LOW`, `MEDIUM`, `HIGH`, `CRITICAL`). Refer to [heuristic_conflict_recommendation_engine_design_specification.md](../design/heuristic_conflict_recommendation_engine_design_specification.md).

* **Micro-Task 1: Heuristic Domain Contracts & Recommendation Models (`internal/models/review.go`)**
  - Add `RuleResult`, `ConflictRecommendation`, and `BatchReviewActionRequest` DTOs.
  - Enrich `ConflictCandidate` with optional `Recommendation`.

* **Micro-Task 2: Pure-Go Analyzers & Scoring Engine (`internal/engine/heuristics/`)**
  - Implement `types.go`, `analyzers.go`, and `engine.go`.
  - Analyzers: `TempLockAnalyzer` (locks/tmp files), `HashMatchAnalyzer` (identical content hash), `TruncationAnalyzer` (0-byte / mass shrinkage guard), `ClockSkewAnalyzer` (1h/2h DST or $\le 2\text{s}$ drift), `SizeDivergenceAnalyzer` (>20% size difference), `TimestampRecencyAnalyzer` (timestamp delta).
  - Scoring aggregator: weighted sum, risk-override guardrails, and parallel worker pool ($G=4$).

* **Micro-Task 3: Unit Testing Suite & Concurrency Validation (`internal/engine/heuristics/engine_test.go`)**
  - Unit tests covering all analyzers, critical risk override, and race-free concurrent evaluation (>85% statement coverage).

* **Micro-Task 4: Service Facade Integration (`internal/services/review_service.go`)**
  - Wire `heuristics.Engine` into conflict discovery, automatically enriching conflict candidates with recommendations.
  - Implement batch decision application method `ApplyBatchRecommendations`.

* **Micro-Task 5: Server REST Batch Endpoint (`internal/server/handlers_review.go`, `server.go`)**
  - Mount `POST /api/review/action-batch` endpoint for one-click auto-staging of high-confidence, low-risk recommendations.

* **Micro-Task 6: Frontend Recommendation Badges & Batch Button (`dashboard/frontend/`)**
  - Display recommendation badges and reasoning in Review Modal.
  - Add "Apply High-Confidence Recommendations" batch button.

*Status: Delivered & Integrated [2026-10-23]*

---

### Active Refactor: Frontend Template Modularization (`html/template`) [`refactor/modularize-frontend-templates`]

Decomposes the monolithic 1,500+ LOC `dashboard/frontend/index.html` into clean Go `html/template` partials to establish a scalable foundation for subsequent feature tabs and modals without introducing Node.js or npm dependencies.

* **Micro-Task 1: Template Structure & Partials Extraction**
  - Create `dashboard/frontend/partials/` and `dashboard/frontend/partials/modals/`.
  - Extract `view_telemetry.html`, `view_remotes.html`, `view_explorer.html`.
  - Extract `modal_review.html`, `modal_daemon.html`, `modal_delete.html`, `modal_remote_reconnect.html`, `modal_remote_credentials.html`.
  - Streamline `index.html` to a ~120 LOC master shell.

* **Micro-Task 2: Server-Side Template Caching (`internal/server/server.go`)**
  - Parse master template and all partials once at server startup with `template.ParseFS`.
  - Render pre-parsed template on `GET /{$}` with zero per-request parsing latency.

* **Micro-Task 3: Automated Quality Gate Verification**
  - Validate HTTP endpoint response contracts and run `./scripts/check.sh`.

---

### Phase 4/5 Governance Priority: Immutable Audit Ledger & Unified Log Architecture

Cryptographically chained append-only SQLite ledger (`internal/audit`) paired with unified dual-logging in `bisync.log` and in-process POSIX log rotation.

* **Component 1: Pure-Go SQLite Ledger (`modernc.org/sqlite` + `pressly/goose/v3`)**
  - Zero-CGO SQLite driver with WAL mode, `BEFORE UPDATE`/`BEFORE DELETE` abort triggers, and embedded schema migrations.
  - Deterministic SHA-256 hash chaining (`prev_hash` $\rightarrow$ `hash`) with full chain integrity verification.
  - Strict separation of transient review sessions (`state/review-sessions/`) from permanent audit records.

* **Component 2: Unified Dual-Logging & In-Process Log Rotation (`internal/core/logger`)**
  - Dual-emit audit milestones into `bisync.log` (`"type": "audit"`) alongside sync runs.
  - POSIX log rotator with bounded file size (10 MB) and backup retention (5 archives).

* **Component 3: Dual-Stream Cockpit & Dedicated Audit Workspace (Option 1 UI)**
  - Cockpit multi-stream terminal switcher (`[All Events]`, `[Sync Runs]`, `[Audit Milestones]`).
  - 4th top-level navigation tab `[Audit Ledger]` implemented as a modular partial (`partials/view_audit.html`).
  - Cryptographic verification badge and JSONL/CSV streaming exports.

---

### Phase 5: Core Enterprise Feature Extensions (Medium-Term)

* **Multi-Profile Synchronization Hub [ESTIMATE: 6-8h | 5 SP]**

  - Single background Go daemon spawning lightweight goroutines per profile (Personal GDrive, Work OneDrive, etc.) with isolated flock files and telemetry streams.
  - Refer to [profile-manager.md](../design/profile-manager.md).

* **In-Dashboard Configuration Editor & 3-Layer Safety Pipeline [ESTIMATE: 6-8h | 5 SP]**

  - Visual parameter adjustment for `--buffer-size`, `--tpslimit`, and conflict resolution strategies.
  - 3-Layer Safety Pipeline: Automated snapshot backup, SHA-256 integrity match, and schema validation.
  - Rollback API: One-click restore of historical configurations.

---

### Phase 6: Local MVP Deployment Hardening (Core Infrastructure Requirements)

* **Local Inter-Process Communication via Unix Sockets [ESTIMATE: 4-6h | 3 SP]**

  - Update `rclone rcd` launch configuration and Go HTTP client (`net.DialContext`) to communicate over a local Unix domain socket (`unix:///tmp/rclone.sock`) with strict `0600` permissions and graceful process cleanup.
  - Support dual deployment modes: bare-metal user socket management and Docker container volume bind-mounting (`/tmp/rclone-ipc`) with container startup healthchecks.

* **Host File Permissions & Ownership Alignment [ESTIMATE: 3-4h | 2 SP]**

  - Enforce systemd user unit execution (`User=%I`, `Group=%I`, `UMask=0022`) to guarantee standard file permissions on synced data.
  - Provide Docker compose setups with dynamic host user passing (`user: "${UID}:${GID}"`), non-root container base images, and entrypoint write-permission verification.

* **Auto-Generated Cryptographic Credentials Engine [ESTIMATE: 3-4h | 2 SP]**

  - Automated installer script (`install.sh`) generating 32-character random credentials in `/etc/rclone-dashboard/env` with `0600` permissions.
  - Go CLI rotation command (`rclone-dashboard rotate-creds`) using `crypto/rand`, systemd `ExecStartPre` auto-generation, and Docker container boot wrapper for environment isolation.

---

### Phase 7: Desktop Application Packaging

* **Desktop Mode via Wails / Native Window [ESTIMATE: 3-4h | 3 SP]**

  - Package Go backend and embedded frontend as a standalone borderless native desktop app using Wails.

* **Portable Bundle Generator [ESTIMATE: 2-3h | 2 SP]**

  - Export endpoint packaging sanitized configs, scripts, and systemd units into `.tar.gz` for cross-device migration
