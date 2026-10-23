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

### Active Implementation Sprint: Repository Quality Hardening, Security Jails & Architectural Modularization

Foundational maintenance milestone resolving all findings from `docs/architecture/code-review.md` and preparing the codebase for the core differentiator: Interactive Sync Review.

* **Micro-Task 1: Stale Lock PID Verification Guard [Finding H-1]**
  - Prevent lockfile unlinking while the owning PID is alive, preventing dual-sync collisions during long-running syncs.
* **Micro-Task 2: Path Confinement Symlink Resolution & Canary Protection [Findings H-2, H-3]**
  - Integrate `filepath.EvalSymlinks()` into path jail validation to eliminate symlink escapes.
  - Guard recursive directory deletions against purging directories containing `RCLONE_TEST`.
* **Micro-Task 3: Homelab LAN Access & Graceful HTTP Drain [Findings M-2, M-3]**
  - Expand origin validation to allow RFC 1918 private subnets and `.local` mDNS domains.
  - Implement 5-second graceful HTTP shutdown drain in `main.go`.
* **Micro-Task 4: Secret Sanitization Unification & Non-Mutating Env Parser [Findings M-4, L-1]**
  - Unify config masking with `security.RedactSecrets()` and introduce `ParseEnvFile()`.
* **Micro-Task 5: Client Package Decomposition (`internal/core/client/`)**
  - Split `client.go` into `jobs.go`, `remotes.go`, `fs_ops.go`, and base `client.go`.
* **Micro-Task 6: Server Route Modularization & Schema Contract Hardening [Finding M-1]**
  - Split `server.go` into `handlers_telemetry.go`, `handlers_explorer.go`, `handlers_auth.go`, `handlers_sync.go`.
  - Replace loose `map[string]interface{}` with concrete Go models from `internal/models/api.go`.
* **Micro-Task 7: Quality Gate & Test Coverage Uplift (>= 85%)**
  - Run `./scripts/check.sh` and verify native/cross-compilation.

---

### Strategic Epic: Phase 4 — Interactive Sync Review & Deletion Diff Visualizer [DELIVERED - feat/interactive-sync-review]

Primary strategic feature transforming Bisync Hub from an observer dashboard into an interactive safety controller. When synchronization aborts due to safety circuit breakers (`TOO_MANY_DELETES` or unresolved file conflicts), the hub launches an interactive review session instead of requiring manual CLI `--force` intervention. Refer to [interactive-sync-review.md](../design/interactive-sync-review.md).

* **Micro-Task 1: Review Domain Models & Contract Definitions (`internal/models/review.go`)** [COMPLETED]
  - Concrete DTOs: `ReviewSession`, `DeletionCandidate`, `ConflictCandidate`, `ResolutionAction`, `ReviewSessionStatus`.
  - Action types: `keep-local`, `keep-remote`, `keep-both`, `accept-delete`, `restore-item`.

* **Micro-Task 2: Review Service & Persistent Journal Engine (`internal/services/review_service.go`, `internal/review/`)** [COMPLETED]
  - Thread-safe `ReviewService` and `ReviewStateMachine` managing session lifecycle, discovery, execution, backup snapshots, and rollback.
  - Atomic JSON journal persistence (`.review-session.<id>.json.tmp.<pid>` + `os.Rename`) and append-only write-ahead log (`WAL`) ensuring durability across daemon restarts.
  - Pre-sync barrier integration: `sync_service.go` blocks bisync triggering if active unresolved review sessions exist.

* **Micro-Task 3: Candidate Discovery & Abort Interceptor (`internal/review/discovery.go`)** [COMPLETED]
  - Parse candidate items from bisync execution output and abort states (`TOO_MANY_DELETES`, unhandled conflicts).
  - Populate candidate metadata (local/remote paths, sizes, mod times, detected condition).

* **Micro-Task 4: Resolution Executor, Backup Staging & Rollback Engine (`internal/review/backup.go`, `executor.go`, `rollback.go`)** [COMPLETED]
  - Safe execution of user decisions using zero-byte POSIX hard links for local files and cloud-to-cloud server-side copy (`operations/copyfile`) for remote files.
  - Pre-flight `statfs` validation checking local disk space before fallback copies.
  - Saga rollback runner compensating and restoring previous snapshots in reverse execution order.

* **Micro-Task 5: REST API Handlers & Routing (`internal/server/handlers_review.go`, `server.go`)** [COMPLETED]
  - `GET /api/review/session`: Retrieve active session status and candidates.
  - `POST /api/review/action`: Stage per-candidate decision or batch decisions.
  - `POST /api/review/execute`: Execute staged decisions with atomic progress tracking and pre-mutation backups.
  - `POST /api/review/rollback`: Roll back executed decisions from backups.
  - `POST /api/review/dismiss`: Dismiss/abort review session safely.

* **Micro-Task 6: Frontend Dual-Pane Explorer Highlighting & Review Modal (`dashboard/frontend/`)** [COMPLETED]
  - Visual status highlighting in Explorer: Red (deletion queued), Amber (conflict candidate), Green (decision staged/preserved).
  - Dedicated interactive Review Modal ("Review. Decide. Sync.") with per-item decisions, quick batch actions, safety callouts, and resolution status badges.
  - Synchronized polling in `app.js` and dynamic status banner in dashboard header.

* **Micro-Task 7: Automated Verification Suite & End-to-End Quality Gate** [COMPLETED]
  - Comprehensive unit and integration tests with `httptest.NewServer` RC mocking.
  - Verify statement coverage >80% across all packages (`./scripts/check.sh`).

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
