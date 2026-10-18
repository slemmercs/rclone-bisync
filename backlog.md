# Rclone Bisync Enterprise Automation Hub - Architecture & Roadmap Backlog

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

## 3. Post-Go Migration Feature Roadmap

* **Branch 4 (Go): `feat/remote-oauth-reauth` [ESTIMATE: 3-4h | 3 SP]**
  - Backend: Implement `ReconnectRemote(name)` executing desktop-integrated OAuth flow (`rclone config reconnect {name}:`) preserving user session environment (`DBUS_SESSION_BUS_ADDRESS`, `DISPLAY`, `XDG_RUNTIME_DIR`).
  - Endpoint: `POST /api/remotes/{name}/reconnect`.
  - Frontend: Add "Reconnect / Re-Auth" action button to Remote Cards with reactive loading state and notification toasts.

* **Branch 6 (Go): `feat/interactive-log-cross-filtering` [COMPLETED]**
  - High-performance log stream parsing in Go via `bufio.Scanner` (`internal/services/log_service.go`).
  - Central structured JSON logger (`internal/core/logger`) recording single-line ledger records in `bisync.log`.
  - Endpoint `GET /api/logs/daemon` serving correlated time-window slices.
  - UI Component: Streamlined orchestrator feed with on-demand focused daemon log modal dialog featuring live search, level filtering, and copy-to-clipboard.

* **Multi-Profile Synchronization Hub [ESTIMATE: 6-8h | 5 SP]**
  - Single background Go daemon spawning lightweight goroutines per profile (Personal GDrive, Work OneDrive, etc.) with isolated flock files and telemetry streams.

* **Desktop Mode via Wails / Native Window [ESTIMATE: 3-4h | 3 SP]**
  - Package Go backend and embedded frontend as a standalone borderless native desktop app using Wails.

---

## 4. Completed Historical Branches (Python Reference Implementation)

* **Branch 1: `refactor/clean-env-breaking-change` [MERGED]**
* **Branch 2: `refactor/service-layer-and-rc-correlation` [MERGED]**
* **Branch 3: `feat/security-pydantic-dtos-and-redaction` [MERGED]**
* **Branch 5: `feat/cockpit-workspace-and-file-explorer` [MERGED]**
* **Branch 7: `test/backend-integration-suite` [MERGED]**

---

## 5. Future Releases & Extended Backlog

* **In-Dashboard Configuration Editor & 3-Layer Safety Pipeline [ESTIMATE: 6-8h | 5 SP]:**
  * Visual parameter adjustment for `--buffer-size`, `--tpslimit`, and conflict resolution strategies.
  * 3-Layer Safety Pipeline: Automated snapshot backup, SHA-256 integrity match, and schema validation.
  * Rollback API: One-click restore of historical configurations.

* **Side-by-Side Deletion & Conflict Diff Visualizer [ESTIMATE: 4-6h | 5 SP]:**
  * When `TOO_MANY_DELETES` aborts, automatically open File Explorer highlighting deletions in red and conflicts in amber before requesting authorization with `--force`.

* **Portable Bundle Generator [ESTIMATE: 2-3h | 2 SP]:**
  * Export endpoint packaging sanitized configs, scripts, and systemd units into `.tar.gz` for cross-device migration.

## Open for Evulation

- daemon log modal needs more structure, the log has a message field that contains json formatted log which should be extracted displayed in a structured form
- sync service needs modularization probably
- switch to svelte/vue evaluation


### Phase 1: Local MVP (Core Requirements)

#### Epic 1: Local Inter-Process Communication (Unix Domain Sockets)

##### Target A: Bare-Metal / Local Host Execution
- [ ] **Task 1.1.A.1 (Rclone Socket Config):** Update `rclone rcd` launch configuration to listen on a local Unix socket (`unix:///tmp/rclone.sock`) instead of a TCP port.
- [ ] **Task 1.1.A.2 (Go Custom HTTP Dialer):** Implement custom `net.DialContext` in the Go HTTP client to route all API calls over `/tmp/rclone.sock`.
- [ ] **Task 1.1.A.3 (Socket File Permissions):** Ensure socket file creation sets strict `0600` permissions owned by the running user.
- [ ] **Task 1.1.A.4 (Socket Cleanup):** Implement a graceful shutdown handler in Go to unlink/remove `/tmp/rclone.sock` on process termination.

##### Target B: Docker Container Deployment
- [ ] **Task 1.1.B.1 (Volume Bind-Mount):** Configure `docker-compose.yml` to share the IPC socket directory between the host and container (`/tmp/rclone-ipc:/tmp/rclone-ipc:rw`).
- [ ] **Task 1.1.B.2 (Container Path Mapping):** Pass `RCLONE_RC_ADDR=unix:///tmp/rclone-ipc/rclone.sock` via environment variables to the containerized dashboard.
- [ ] **Task 1.1.B.3 (Container Startup Order):** Add healthchecks/retries to wait until `/tmp/rclone-ipc/rclone.sock` exists before booting the Go WebUI HTTP listener.

---

#### Epic 2: Host File Permissions Alignment (`rwx` Ownership)

##### Target A: Bare-Metal / Local Host Execution
- [ ] **Task 1.2.A.1 (Systemd User Execution):** Configure Systemd service unit (`rclone-dashboard.service`) using `User=%I` and `Group=%I` so the process inherits the active user's environment.
- [ ] **Task 1.2.A.2 (Umask Enforcement):** Set explicit `UMask=0022` in the service definition to guarantee default `0755` directory and `0644` file permissions on synced data.

##### Target B: Docker Container Deployment
- [ ] **Task 1.2.B.1 (Host UID/GID Passing):** Configure `docker-compose.yml` to accept dynamic user environment variables (`user: "${UID}:${GID}"`).
- [ ] **Task 1.2.B.2 (Non-Root Docker Base):** Update `Dockerfile` to ensure runtime process executes under an unprivileged user space matching the host `UID`/`GID`.
- [ ] **Task 1.2.B.3 (Volume Permission Verification):** Add pre-flight check in container entrypoint script to verify write access on host bind-mounted sync folders without requiring `sudo chown`.

---

#### Epic 3: Auto-Generated Cryptographic Credentials (`.env`)

##### Target A: Bare-Metal / Local Host Execution
- [ ] **Task 1.3.A.1 (Installer Script):** Create `install.sh` shell script using `openssl` / `/dev/urandom` to auto-generate 32-character random credentials in `/etc/rclone-dashboard/env` with `0600` permissions.
- [ ] **Task 1.3.A.2 (Go CLI Rotation Command):** Implement `rclone-dashboard rotate-creds` sub-command using Go `crypto/rand` to overwrite `.env` with fresh credentials on demand.
- [ ] **Task 1.3.A.3 (Systemd ExecStartPre):** Configure Systemd unit with `ExecStartPre=` hook to invoke credential generation if `.env` file is missing prior to main app launch.

##### Target B: Docker Container Deployment
- [ ] **Task 1.3.B.1 (Container Entrypoint Wrapper):** Add an `entrypoint.sh` script to the Docker image that generates `/app/.env` on container first boot if the mounted file is missing or empty.
- [ ] **Task 1.3.B.2 (Docker Secret / Env File Mount):** Configure `docker-compose.yml` to bind-mount host `.env` or use Docker Secrets for credential persistence across container updates.

---

### Phase 2: Remote & Multi-Node Platform (Future / Optional)

#### Epic 4: Remote API mTLS Authentication Engine
- [ ] **Task 2.1 (Go CertManager Package):** Implement Go package using `crypto/ecdsa` and `crypto/x509` to generate internal CA, server cert/key, and client cert/key bundles.
- [ ] **Task 2.2 (Go Custom CA Client Pool):** Configure Go HTTP client to load client certificates and trust server certificates signed by internal CA (`MinVersion: tls.VersionTLS13`).
- [ ] **Task 2.3 (Transport Auto-Selector):** Add WebUI settings toggle to automatically switch between Unix Socket (local) and mTLS over TCP (remote).

#### Epic 5: Remote SSH Certificate Deployment & Rotation Engine
- [ ] **Task 3.1 (Go SSH/SFTP Client Engine):** Build Go SSH client module (`golang.org/x/crypto/ssh`) for key-authenticated remote file management.
- [ ] **Task 3.2 (Staged Dual-CA Deployment):** Implement 3-phase rotation workflow (Push new CA $\rightarrow$ Swap server/client certs $\rightarrow$ Prune old CA) to prevent remote API lockout.
- [ ] **Task 3.3 (Remote Process Reload Trigger):** Execute remote systemd or Docker restart commands over SSH after writing updated certificate bundles.
- [ ] **Task 3.4 (Verification & Rollback Safeguard):** Validate mTLS handshake post-rotation; automatically revert certificate files on remote host if connection check fails.

#### Epic 6: Edge Proxy Integration
- [ ] **Task 4.1 (Traefik / Caddy Sidecar Architectures):** Provide production configurations for running Traefik or Caddy in front of remote `rclone` API instances for hot-reloading TLS certs without restarting `rclone`.
