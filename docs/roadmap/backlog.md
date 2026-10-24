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

---

### Active Near-Term Backlog (Sequenced Priorities)

#### 1. Immutable Audit Ledger & Unified Log Architecture [ESTIMATE: 6-8h | 5 SP]
Cryptographically chained append-only SQLite ledger (`internal/audit`) paired with unified dual-logging in `bisync.log` and in-process POSIX log rotation:
* **Pure-Go SQLite Ledger (`modernc.org/sqlite` + `pressly/goose/v3`)**: Zero-CGO SQLite driver with WAL mode, `BEFORE UPDATE`/`BEFORE DELETE` abort triggers, and SHA-256 hash chaining (`prev_hash` $\rightarrow$ `hash`).
* **Unified Dual-Logging**: Dual-emit audit milestones into `bisync.log` (`"type": "audit"`) alongside sync runs.
* **POSIX Log Rotator**: In-process size-bounded rotation (10 MB, 5 archives).
* **Dedicated Audit Workspace**: Multi-stream terminal switcher plus a dedicated 4th top-level tab `[Audit Ledger]` (`partials/view_audit.html`) with cryptographic verification badges.

#### 2. Multi-Profile Workspace Manager [ESTIMATE: 6-8h | 5 SP]
Supports independent synchronization workloads from a single background daemon and dashboard session:
* Single Go daemon spawning lightweight goroutines per profile (Personal GDrive, Work OneDrive, NAS backup).
* Pairs independent local folders to different remotes (e.g., `/home/user/Documents` $\rightarrow$ `gdrive:`, `/home/user/Photos` $\rightarrow$ `onedrive:`).
* Strict flock isolation (`/tmp/rclone-bisync-<profile>.lock`) and dedicated state directories.
* Refer to [profile-manager.md](../design/profile-manager.md).

#### 3. In-Dashboard Configuration & Setup Manager [ESTIMATE: 5-7h | 4 SP]
Provides visual setup, flag customization, and safety validation:
* **Visual Parameter Editor**: In-dashboard adjustment for `--buffer-size`, `--tpslimit`, conflict strategies, and webhook endpoints.
* **Remote Setup & Onboarding Wizard**: Guided visual setup wizard for initial Rclone RC remote pairing and environment configuration.
* **3-Layer Safety Pipeline**: Automated snapshot backups, SHA-256 integrity checks, and JSON/env schema validation before applying new settings.
* **Rollback API**: One-click restore of historical configurations.

#### 4. In-App User Guidance, Info & Help Documents [ESTIMATE: 3-5h | 3 SP]
Improves operational transparency and onboarding for new users:
* **Contextual Tooltips & Policy Banners**: Embeds inline tooltips across `partials/view_*.html` explaining safety mechanisms (`TOO_MANY_DELETES`, `MANDATORY_SUBDIR`, heuristic risk scores) in non-technical terms.
* **In-Dashboard Reference Manual**: A slide-over help panel summarizing CLI commands, Rclone flag impacts, and step-by-step conflict recovery workflows.
* **Diagnostic Documentation**: Integrated guides for resolving OAuth token drops, path permissions errors, and systemd service status.

---

## 4. Medium & Infrastructure Backlog

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
