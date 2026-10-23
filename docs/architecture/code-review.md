# Bisync Hub — Architectural Code Review & Quality Audit

**Review Date**: October 23, 2026  
**Toolchain**: Go 1.24+ | Native Single Binary | Alpine.js SPA  
**Audit Standard**: `.clinerules` Section 3 (Systematic Review, Quality Audit & Architecture Protocol)  

---

## 1. Executive Summary & Verdict

### **Verdict**: `ADVISORY` (High-Quality Baseline with Targeted Safety & Schema Hardening Needed)

Bisync Hub has achieved a robust transition to a native Go single-binary architecture with zero runtime Python/Node dependencies, embedded frontend distribution (`//go:embed`), an automated quality gate with **84.9% statement test coverage**, and strict separation of concerns across core primitives.

While the core synchronization engine and frontend UI are well-engineered, this audit identified:
1. A **stale lock timeout race condition** where live sync processes can be superseded after 1 hour (`internal/core/lock/lock.go`).
2. A **symlink evaluation gap** in path confinement checks (`internal/services/explorer_service.go`).
3. **Schema governance drift** where API routes emit untyped `map[string]interface{}` dictionaries rather than leveraging existing structs in `internal/models`.
4. Monolithic bloat in `internal/server/server.go` (523 LOC) and `internal/core/client/client.go` (551 LOC).

---

## 2. Systematic Review Checklist Audit

| Category | Checklist Item | Status | Findings / Notes |
| :--- | :--- | :---: | :--- |
| **Safety & Circuit Breakers** | Path traversal jails strictly enforced (`filepath.Rel`) | ⚠️ Partial | Lexical path jail is enforced via `filepath.Rel`, but filesystem symlinks pointing outside the sync folder are not resolved via `filepath.EvalSymlinks`. |
| | Canary files (`RCLONE_TEST`) protected from deletion | ⚠️ Partial | Direct file deletion is blocked, but deleting a parent directory containing `RCLONE_TEST` via `RemoveAll` or `PurgeDir` is not prevented. |
| | Lock files and PID checks are race-free (`flock`) | ⚠️ At Risk | If a sync run exceeds `LockStaleTimeoutSeconds` (3600s), `handleStaleLock` unlinks the active lockfile and acquires a new lock even while the original PID is alive. |
| | Rate-limit backoff flag checks respected | ✅ Pass | `RATE_LIMIT` errors trigger backoff flag; sync skips unless `--force` is provided. |
| **Architecture & Standards** | Domain logic separated into service layer (`internal/services`) | ✅ Pass | `ExplorerService`, `TelemetryService`, `DeletionService`, and `CanaryService` cleanly decouple domain logic from HTTP routing. |
| | Atomic file writes used for state/telemetry (`.tmp.<pid>` + rename) | ✅ Pass | `state.WriteJSONAtomic` correctly uses `.tmp.<pid>` + `os.Rename`. |
| | Sensitive keys redacted from logs and API payloads | ✅ Pass | `security.RedactSecrets` recursively scrubs nested maps/structs, and `writeJSON` automatically applies redaction on HTTP responses. |
| **Frontend & Event Safety** | Click events isolated on draggable elements | ✅ Pass | Draggable rows have `draggable="true"`, with child buttons explicitly configured with `draggable="false"` and `@mousedown.stop`. |
| | XSS safety maintained | ✅ Pass | `utils.js` escapes all user/config content via `escapeHtml()`; no unescaped `x-html` injections. |
| | Chart.js outside Alpine reactive proxy | ✅ Pass | `performanceChart` is held in file-scoped closure variable outside Alpine reactivity. |
| **Test & Verification** | Arrange-Act-Assert (AAA) pattern adhered to | ✅ Pass | Unit and integration test suites consistently follow AAA. |
| | External RC API requests strictly mocked | ✅ Pass | All client and service unit tests use `httptest.NewServer` or fake runners. Zero live subprocesses executed during tests. |
| | Pre-flight quality gate clean | ✅ Pass | `./scripts/check.sh` passes 8/8 checks with 84.9% test coverage. |


---

## 3. App Strengths

1. **Native Single-Binary Distribution with Zero Runtime Dependencies**:
   - Compiles down to a ~17MB self-contained executable with embedded frontend SPA (`//go:embed dashboard/frontend/*`).
   - Zero dependency on Python virtual environments (`.venv`), Node.js, npm, or system pip libraries.
2. **Deep Defense-in-Depth Secret Sanitization**:
   - `internal/core/security/security.go` provides recursive reflection-based redaction across nested maps, slices, arrays, and structs.
   - Centralized HTTP response writing (`server.go:writeJSON`) automatically routes all outgoing API payloads through `security.RedactSecrets`.
3. **Comprehensive Automated Test Suite & Quality Gate**:
   - Automated quality gate `./scripts/check.sh` enforces formatting, Go module tidiness, `go vet`, cross-platform compilation (`linux/amd64`, `linux/arm64`), and statement test coverage >80% (currently **84.9%**).
   - Clean RC mock isolation using `httptest.NewServer` throughout `client_test.go`, `service_test.go`, and `server_test.go`.
4. **Resilient Frontend Architecture**:
   - Zero-build Alpine.js + Tailwind CSS architecture.
   - Modular script structure (`charts.js`, `explorer.js`, `remotes.js`, `utils.js`, `app.js`).
   - Drag-and-drop and button events properly decoupled via `draggable="false"` and `@mousedown.stop`.
5. **Decoupled Service Architecture**:
   - Prior refactoring (PR #18) separated the monolithic `SyncService` into dedicated single-concern services: `ExplorerService`, `TelemetryService`, `DeletionService`, and `CanaryService`.

---

## 4. App Weaknesses & Risks

1. **Unilateral Lock Inode Invalidation on Long Runs (Race Condition)**:
   - In `internal/core/lock/lock.go` (`handleStaleLock`), if a sync run takes longer than `LockStaleTimeoutSeconds` (1 hour) due to large transfers, `Acquire()` unlinks the lockfile and acquires a new lock on a new inode, even though the original process PID is active. This can trigger concurrent bisync runs, corrupting Rclone listings.
2. **Lexical Jail vs. Symlink Jail Bypass**:
   - `IsLocalPathJailed` uses `filepath.Rel(baseDir, targetDir)`. While this prevents string-based traversal like `../../etc`, it does not resolve filesystem symlinks via `filepath.EvalSymlinks`. A symlink within `~/gdrive` pointing to `/` or `~/.ssh` will pass lexical validation and allow file browsing/deletion outside the jail.
3. **Incomplete Parent-Level Canary Deletion Guard**:
   - `ExplorerService.DeleteItem` blocks deleting files named `RCLONE_TEST`. However, calling `DeleteItem(..., isDir: true)` on a directory containing `RCLONE_TEST` will recursively delete the directory and the canary with it.
4. **Homelab / Remote Web Access Blocked by Hardcoded Origin Whitelist**:
   - CSRF origin checking in `server.go` (`isAllowedOrigin`) strictly allows only `localhost` and `127.0.0.1`. If deployed on a homelab server bound to `0.0.0.0:8080` or accessed via a LAN IP (e.g. `192.168.1.50`), all HTTP POST requests fail with HTTP 403.
5. **Missing Graceful Server Shutdown**:
   - In `main.go`, `runDashboard` intercepts `SIGINT`/`SIGTERM` but immediately prints a log line and exits without calling `httpServer.Shutdown(ctx)`, abruptly aborting active transfers or OAuth reconnect flows.

---

## 5. Findings by Severity

### 🔴 High (Safety / Security / Regression)

* **Finding H-1: Lock Inode Unlinking while PID is Running**
  - **Location**: `internal/core/lock/lock.go:103-111`
  - **Issue**: In `handleStaleLock`, if `lockAge > lm.StaleTimeoutSeconds`, `os.Remove(lm.LockFilePath)` is executed even when `CheckPIDExists(oldPID)` returned `true`.
  - **Risk**: Concurrent bisync execution during long initial synchronizations or multi-gigabyte transfers, causing Rclone listing corruption.
  - **Remediation**: Guard lock removal behind dead PID verification:
    ```go
    if CheckPIDExists(oldPID) {
        // Active PID is still running; never remove lock regardless of elapsed age
        _ = lm.closeHandle()
        return false, nil
    }
    // Only remove if PID is not running or stale timeout exceeded on a dead process
    ```

* **Finding H-2: Symlink Path Confinement Bypass in Explorer**
  - **Location**: `internal/services/explorer_service.go:37-55`
  - **Issue**: `IsLocalPathJailed` relies on lexical evaluation without resolving real targets via `filepath.EvalSymlinks`.
  - **Risk**: If a symlink in the local directory points to sensitive files outside the base directory, users can browse, copy, or delete them.
  - **Remediation**: Resolve the target's real path:
    ```go
    realBase, err := filepath.EvalSymlinks(baseDir)
    if err != nil { return "", "", false }
    realTarget, err := filepath.EvalSymlinks(targetDir)
    if err == nil {
        rel, err := filepath.Rel(realBase, realTarget)
        if err != nil || strings.HasPrefix(rel, "..") { return baseDir, targetDir, false }
    }
    ```

### 🟡 Medium (Reliability / Edge Case / Architecture)

* **Finding M-1: Schema Governance Drift — Untyped Loose Maps in REST Endpoints**
  - **Location**: `internal/server/server.go` & `internal/services/telemetry_service.go`
  - **Violation**: `.clinerules` Rule 3.4.2 ("Web API endpoints must never return untyped loose maps without structure").
  - **Issue**: Handlers for `/api/telemetry`, `/api/action`, `/api/remotes`, and `/api/browse/copy` return `map[string]interface{}` despite concrete structs (`models.TelemetryPayload`, `models.ActionResponse`, `models.RemotesResponse`, `models.CopyResponse`) existing in `internal/models/api.go`.
  - **Remediation**: Refactor service and handler return types to concrete Go structs.

* **Finding M-2: Homelab LAN Access Blocked by Restrictive Origin Whitelist**
  - **Location**: `internal/server/server.go:115-125` (`isAllowedOrigin`)
  - **Issue**: Rejects any request where origin is not `localhost` or `127.0.0.1`.
  - **Risk**: Breaks accessibility in homelab server and NAS setups (violating the "Local-First / Homelab" architecture principle).
  - **Remediation**: Allow origin to match the configured server host address (`s.Config.Host`) and incoming `r.Host`.

* **Finding M-3: Missing HTTP Graceful Shutdown Drain**
  - **Location**: `main.go:100-102`
  - **Issue**: Dashboard server exits on signal without calling `httpServer.Shutdown(ctx)`.
  - **Remediation**: Add `ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)` and invoke `httpServer.Shutdown(ctx)`.

* **Finding M-4: Duplicate Sanitization Logic in TelemetryService**
  - **Location**: `internal/services/telemetry_service.go:252-284, 351-376`
  - **Issue**: `GetRuntimeConfig` defines custom `toUpperSnakeCase` and `isKeySensitive` helpers instead of reusing `security.RedactSecrets(t.Config)`.
  - **Remediation**: Replace custom redaction loop with `security.RedactSecrets`.

### 🟢 Low (Polish / Dead Code / Maintainability)

* **Finding L-1: Global Environment Mutation in `config.LoadEnvFile`**
  - **Location**: `internal/config/config.go:100`
  - **Issue**: Uses `os.Setenv` to load files, mutating global process environment state.
  - **Remediation**: Return a `map[string]string` dictionary and apply to `Config` struct directly.

* **Finding L-2: Redundant Forwarding Methods on `SyncService`**
  - **Location**: `internal/services/sync_service.go:320-373`
  - **Issue**: After PR #18, `Server` now holds references to `ExplorerService`, `TelemetryService`, etc. directly. The 10+ wrapper methods on `SyncService` (`BrowsePath`, `CreateCanary`, etc.) are legacy facades.


---

## 6. Technical Debt Analysis

1. **Dual Schema Definitions**:
   `state.PendingDeletionsData` and `models.PendingDeletionsResponse` represent the same domain concept with slightly diverging field types (`[]string` vs `[]interface{}`).
2. **Redundant Log Unwrapping in Multiple Modules**:
   Log parsing and JSON unmarshaling of nested Rclone logs is duplicated between `internal/services/log_service.go`, `internal/services/sync_service.go`, and frontend `dashboard/frontend/js/utils.js`.
3. **Hardcoded Command Subprocess Calls**:
   `sync_service.go` invokes systemd via `exec.Command("systemctl", "--user", ...)` without an injectable interface or mock runner (unlike `remoteauth.CommandRunner`, which is cleanly injected).

---

## 7. Architecture Fit for Current App & Future Roadmap

### Current Application Fit: **Excellent (9/10)**
The single-binary Go model with embedded static assets, POSIX locking, and direct RC API integration aligns with the principles in `docs/architecture/ARCHITECTURE_PRINCIPLES.md`.

### Fit for Future Roadmap Features:

#### 1. Phase 4: Interactive Sync Review & Deletion Diff Visualizer
- **Current State**: `state.PendingDeletionsData` exists, and `DeletionService` parses `TOO_MANY_DELETES` logs.
- **Fit Evaluation**: **Ready**. The plan to introduce `internal/models/review.go` and `internal/services/review_service.go` fits naturally into the existing domain service architecture. The Dual-Pane Explorer already supports file manipulation endpoints (`/api/browse/copy`, `/api/browse/delete`).
- **Prerequisite**: Fix Finding H-1 (lock stability) before implementing review session journals.

#### 2. Phase 5: Multi-Profile Synchronization Hub
- **Current State**: Single `Config` instance loaded at startup; hardcoded paths in `sync_service.go` (`bisync-gdrive.service`, `bisync_gdrive.lock`).
- **Fit Evaluation**: **Needs Refactoring**. Services currently bind to a single `config.Config` singleton.
- **Architectural Adjustment**: Profile manager will require parameterizing `SyncRunner` and `SyncService` with a dynamic `ProfileID` (e.g., `bisync-<profile>.lock`).

#### 3. Phase 6: Local IPC via Unix Sockets
- **Current State**: `client.RcloneClient` connects via standard HTTP TCP (`http://localhost:5572`).
- **Fit Evaluation**: **High Alignment**. `http.Transport` in `client.go` can be easily configured with `DialContext` connecting to `unix:///run/user/$UID/rclone.sock` without breaking API contracts.

#### 4. Phase 7: Desktop Application Packaging (Wails)
- **Current State**: Standard Go HTTP server serving embedded static assets via `http.ServeMux`.
- **Fit Evaluation**: **High Alignment**. The frontend is already zero-build vanilla JS/Alpine; Wails can directly bind Go service structs to JavaScript window context.

---

## 8. Modularization Assessment: Is Modularization Needed?

Yes. Two key components exceed the recommended ~400–500 LOC threshold and show signs of mixed responsibilities:

### 1. `internal/server/server.go` (523 LOC)
- **Current Issue**: Houses routing, CSRF middleware, SPA static embedding, and 18 handler methods across 5 unrelated domains (auth, sync, telemetry, logs, file explorer).
- **Recommendation**: Split `server.go` into cohesive handler files:
  - `server.go` (~120 LOC): Server struct, constructor, middleware, route registry.
  - `handlers_telemetry.go` (~90 LOC): Health, telemetry, metrics, log endpoints.
  - `handlers_explorer.go` (~110 LOC): Browse, canary, copy, delete endpoints.
  - `handlers_auth.go` (~120 LOC): Remotes, OAuth reconnect, validate, credentials.
  - `handlers_sync.go` (~80 LOC): Sync actions and pending deletion reviews.

### 2. `internal/core/client/client.go` (551 LOC)
- **Current Issue**: Combines raw HTTP transport, job polling, remote configuration, file system operations, and OAuth lifecycle.
- **Recommendation**: Decompose into domain files under `internal/core/client/`:
  - `client.go` (~130 LOC): Struct, transport, HTTP `post()`, `CheckHealth`.
  - `jobs.go` (~90 LOC): `TriggerBisync`, `GetJobStatus`, `StopJob`, `ListJobs`.
  - `remotes.go` (~150 LOC): `ListRemotes`, `GetRemoteConfig`, `GetRemoteAbout`, `UpdateRemoteConfig`, OAuth status/stop.
  - `fs_ops.go` (~180 LOC): `ListDirectory`, `CopyFile`, `CopyDir`, `DeleteFile`, `PurgeDir`, `TouchFile`.


---

## 9. Verification Evidence

Executing `./scripts/check.sh` on the working tree:

```
====================================================
              Repository Quality Gate
====================================================
[1/8] Validating required tooling...
Required tooling available.

Runner Platform: linux/amd64

[2/8] Verifying Go modules...
all modules verified
Go modules verified.

[3/8] Validating shell scripts...
  ✓ deploy/install.sh
  ✓ scripts/check.sh
  ✓ scripts/update-tools.sh
Shell validation passed.

[4/8] Checking Go formatting...
Go formatting check passed.

[5/8] Running go vet...
Go vet passed.

[6/8] Running unit tests with coverage...
Unit tests passed across all 15 packages.
Coverage : 84.9%
Required : 80.0%
Coverage threshold met.

[7/8] Verifying native build...
Native build passed.

[8/8] Verifying cross-platform builds...
Building linux/amd64...
linux/amd64 build passed.
Building linux/arm64...
linux/arm64 build passed.

Validating executable...
CLI validation passed.

====================================================
Quality Gate Passed (Coverage: 84.9%)
====================================================
```

---

## 10. Recommended Next Steps

To align with `.clinerules` prior to proceeding with Phase 4 (`task_plan.md`):

1. **Remediate High-Priority Findings**:
   - Fix lock timeout handling in `internal/core/lock/lock.go` so running processes are never unlocked prematurely (Finding H-1).
   - Integrate `filepath.EvalSymlinks` into `internal/services/explorer_service.go` (Finding H-2).
2. **Execute Server & Client Modularization**:
   - Decompose `internal/server/server.go` into domain handler files.
   - Decompose `internal/core/client/client.go` into `jobs.go`, `remotes.go`, `fs_ops.go`.
3. **Harmonize API Schema Contracts**:
   - Update REST endpoints to return concrete Go structs from `internal/models` rather than loose maps.
4. **Resume Phase 4 Implementation**:
   - Advance with Task 1 in `docs/roadmap/task_plan.md` (Review Domain Models & Contract Schemas).

