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

## 2. Active Implementation Sprint: Test Coverage Uplift & Quality Gate (>80%)

Targeting statement coverage >80% across all Go packages and resolving toolchain coverage caveats:
- Add `main_test.go` to eliminate root package empty-test `covdata` toolchain error under `go test -cover ./...`.
- Uplift `internal/core/state` (>90%): rate-limit backoff flag lifecycle, notification state persistence, metrics JSONL appending.
- Uplift `internal/core/client` (>85%): mock tests for `TouchFile`, `CopyFile`, `CopyDir`, `DeleteFile`, `PurgeDir`, `ListRemotes`, `GetRemoteAbout`.
- Uplift `internal/services` (>80%): add `telemetry_test.go` (stats, cached metrics, secret redaction, telemetry payload) and test `CopyItem`, pending deletion log parsing, and remote actions.
- Uplift `internal/engine/runner` & `internal/server` (>85%): lock contention, rate limit abort, job error classification, and full HTTP endpoint integration tests.

---

## 3. Historical Branches Reference (Python Implementation)

* **Branch 1: `refactor/clean-env-breaking-change` [MERGED]**
* **Branch 2: `refactor/service-layer-and-rc-correlation` [MERGED]**
* **Branch 3: `feat/security-pydantic-dtos-and-redaction` [MERGED]**
* **Branch 5: `feat/cockpit-workspace-and-file-explorer` [MERGED]**
* **Branch 7: `test/backend-integration-suite` [MERGED]**

---

## 4. Extended Roadmap & Backlog

### Phase 3: Immediate Go Enhancements & Evaluation Tasks (Near-Term)

* **Rebrand to Bisync Hub**
  - Rebrand the UI to Bisync Hub

* **Remote OAuth Re-Authentication Flow [ESTIMATE: 3-4h | 3 SP]**
  - Backend: Implement `ReconnectRemote(name)` executing desktop-integrated OAuth flow (`rclone config reconnect {name}:`) preserving user session environment (`DBUS_SESSION_BUS_ADDRESS`, `DISPLAY`, `XDG_RUNTIME_DIR`).
  - Endpoint: `POST /api/remotes/{name}/reconnect`.
  - Frontend: Add "Reconnect / Re-Auth" action button to Remote Cards with reactive loading state and notification toasts.

* **[COMPLETED] Structured Daemon Log Modal Refinement [ESTIMATE: 3-4h | 3 SP]**
  - Refined `internal/services/log_service.go` to unwrap nested JSON log strings from `bisync/rc.go` wrapper entries and match individual event timestamps.
  - Enhanced frontend `utils.js` and `app.js` with structured metadata parsing (`objectType`, `source`, `details`), comparison settings grid, and defense-in-depth unwrapping.

* **Sync Service Modularization & Code Cleanup [ESTIMATE: 4-6h | 5 SP]**
  - Decouple `internal/services/sync_service.go` by extracting canary check operations, pending deletion parsing, and path validation into discrete domain components.

---

### Phase 4: Core Enterprise Feature Extensions (Medium-Term)

* **Side-by-Side Deletion & Conflict Diff Visualizer [ESTIMATE: 4-6h | 5 SP]**
  - When `TOO_MANY_DELETES` aborts, automatically open File Explorer highlighting deletions in red and conflicts in amber before requesting authorization with `--force`.
  - Refer to [interactive-sync-review.md](interactive-sync-review.md)

* **Multi-Profile Synchronization Hub [ESTIMATE: 6-8h | 5 SP]**
  - Single background Go daemon spawning lightweight goroutines per profile (Personal GDrive, Work OneDrive, etc.) with isolated flock files and telemetry streams.

* **In-Dashboard Configuration Editor & 3-Layer Safety Pipeline [ESTIMATE: 6-8h | 5 SP]**
  - Visual parameter adjustment for `--buffer-size`, `--tpslimit`, and conflict resolution strategies.
  - 3-Layer Safety Pipeline: Automated snapshot backup, SHA-256 integrity match, and schema validation.
  - Rollback API: One-click restore of historical configurations.

---

### Phase 5: Local MVP Deployment Hardening (Core Infrastructure Requirements)

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

### Phase 6: Desktop Application packaging

* **Desktop Mode via Wails / Native Window [ESTIMATE: 3-4h | 3 SP]**
  - Package Go backend and embedded frontend as a standalone borderless native desktop app using Wails.

* **Portable Bundle Generator [ESTIMATE: 2-3h | 2 SP]**
  - Export endpoint packaging sanitized configs, scripts, and systemd units into `.tar.gz` for cross-device migration
