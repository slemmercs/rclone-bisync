# Bisync Hub — Architectural Code Review & Quality Audit

**Review Date**: November 4, 2026  
**Toolchain**: Go 1.24+ | Native Single Binary | Alpine.js SPA | Embedded MCP SSE Server  
**Audit Standard**: `.clinerules` Section 3 (Systematic Review, Quality Audit & Architecture Protocol)  

---

## 1. Executive Summary & Verdict

### **Verdict**: `APPROVED WITH STRATEGIC TESTING ADVISORY`
*(Production-Ready Baseline with Zero LOC Bloat, Embedded Single-Daemon Topology, and a Mandate for Higher-Order Testing Tiers)*

Bisync Hub has achieved substantial architecture milestones across Phases 4 & 5:
- **Phase 4 Interactive Sync Review Engine**: Full FSM state machine, candidate discovery, dual-provider backups, write-ahead journaling (WAL), and saga compensation rollback.
- **Multi-Profile Workspace Manager**: Dynamic multi-profile routing, autonomous profile schedulers, 3-tier operational flags hierarchy, and version snapshot rollbacks.
- **Native Model Context Protocol (MCP) Server**: 11 agentic tools, blast-radius TTL confirmation tickets, read-only vs. mutating access boundaries, and seamless embedding into the primary dashboard daemon.
- **Zero LOC Bloat**: All monolithic files have been decomposed; all Go source files strictly adhere to sub-400 LOC limits (`internal/server/`, `internal/core/client/`, `internal/profiles/store.go`).

However, recent feature releases exposed **testing blind spots** where high statement test coverage (>85%) failed to prevent bugs in production:
1. **Multi-process concurrency race**: Dual systemd daemons (`rclone-mcp.service` and `rclone-dashboard.service`) running duplicate schedulers and clobbering `profiles.json` due to lack of cross-process advisory file locking.
2. **Filesystem semantic mismatch**: Rclone RC `/operations/copyfile` failing with HTTP 500 when executed on directories instead of files (`Mkdir`/`PurgeDir` required).
3. **Path normalization divergence**: Trailing whitespace in directory names causing remote path normalization mismatches and 404s during restorations.
4. **Mock Drift**: Over-reliance on synthetic in-memory HTTP mocks (`httptest.NewServer`) that verify client expectations rather than upstream Rclone filesystem realities.

---

## 2. Systematic Review Checklist Audit

| Category | Checklist Item | Status | Evaluation |
| :--- | :--- | :---: | :--- |
| **Safety & Circuit Breakers** | Path traversal jails strictly enforced (`filepath.Rel` + `filepath.EvalSymlinks`) | ✅ Pass | `IsLocalPathJailed` in `ExplorerService` and review engine resolves canonical filesystem paths via `filepath.EvalSymlinks`. |
| | Canary files (`RCLONE_TEST`) protected from deletion | ✅ Pass | Canary detection blocks both direct file deletion and directory purge operations on canary folders. |
| | Advisory POSIX locking race-free (`flock`) | ✅ Pass | `profiles.Store` uses exclusive OS `syscall.Flock` on `profiles.json.flock`; `lock.Manager` prevents stale lock stealing on active PIDs. |
| | Rate-limit backoff flag checks respected | ✅ Pass | Rate limit backoff markers prevent rapid retry storms against cloud APIs. |
| **Architecture & Standards** | Domain logic separated into service layer (`internal/services/`) | ✅ Pass | Clean decoupling: HTTP routing delegates to `ProfileService`, `ReviewService`, `ExplorerService`, `TelemetryService`. |
| | Single-process daemon topology | ✅ Pass | MCP SSE server embedded directly into dashboard daemon; duplicate scheduler loops eliminated. |
| | Atomic file writes used for state/telemetry (`.tmp.<pid>` + `os.Rename`) | ✅ Pass | All disk persistence (state, profiles, active session, WAL) adheres to POSIX atomic replace. |
| | Deep recursive secret redaction | ✅ Pass | `security.RedactSecrets` scrubs nested structures prior to logging, audit storage, and API emission. |
| **Frontend & Event Safety** | Click events isolated on draggable elements | ✅ Pass | Draggable rows have child buttons isolated with `draggable="false"` and `@mousedown.stop`. |
| | DOM/HTML parser compliance | ✅ Pass | Alpine templates validated by `cmd/check` AST scanner; static `<option>` tags used inside `<select>`. |
| | External libraries unproxied | ✅ Pass | Chart.js instances stored outside Alpine reactive proxy to avoid recursive update storms. |
| **Test & Verification** | AAA pattern adhered to in unit tests | ✅ Pass | Clean Arrange-Act-Assert structure across all 50+ test files. |
| | Test coverage >80% | ✅ Pass | **85.5% average statement coverage** across all packages. |
| | Pre-flight quality gate clean | ✅ Pass | `./scripts/check.sh all` passes all 7 checks in ~16s. |


---

## 3. Higher-Order Testing Strategy: Beyond Unit Tests

To permanently eliminate mock drift and concurrency regressions, the project establishes a **5-Tier Testing Pyramid**:

```
                  ┌───────────────────────────────┐
                  │ Tier 5: E2E User Journey & UI │ (Headless Chrome + Live Backend)
                  └───────────────┬───────────────┘
                                  │
                  ┌───────────────┴───────────────┐
                  │ Tier 4: Real Rclone Sandbox   │ (t.TempDir() + Real rclone rcd)
                  └───────────────┬───────────────┘
                                  │
                  ┌───────────────┴───────────────┐
                  │ Tier 3: Stateful Chaos / Saga │ (Interrupted WAL recovery)
                  └───────────────┬───────────────┘
                                  │
                  ┌───────────────┴───────────────┐
                  │ Tier 2: Multi-Process Races   │ (Subprocess os.Exec + flock)
                  └───────────────┬───────────────┘
                                  │
                  ┌───────────────┴───────────────┐
                  │ Tier 1: Fast Unit Tests       │ (In-memory AAA business logic)
                  └───────────────────────────────┘
```

### Tier 1: Fast Unit Tests (In-Memory AAA)
- **Role**: Pure algorithmic and data-structure correctness (heuristics scoring, flag compilation, DTO translation).
- **Execution**: `go test -count=1 ./internal/...` (<5s).

### Tier 2: Multi-Process Subprocess Concurrency Tests
- **Role**: Exercise true OS-level advisory locking (`flock`), disk reloads on `mtime` drift, and atomic renames across multiple concurrent operating system processes.
- **Pattern**: `TestStore_MultiProcessSynchronizationAndNoSettingsClobber` spawns separate sub-processes hammering the same JSON file simultaneously.

### Tier 3: Stateful Chaos & Saga Compensation
- **Role**: Validates failure handling and transaction rollback in multi-step operations (Interactive Review).
- **Execution**: Simulates I/O or network interruptions mid-execution, verifying that compensating rollback actions restore disk state and WAL recovery leaves the database consistent.

### Tier 4: Real Local Rclone Sandbox Integration Suite
- **Role**: Complete elimination of mock drift without touching cloud credentials or live profiles.
- **Mechanics**:
  - Spawns `rclone rcd --rc-addr 127.0.0.1:0 --rc-no-auth --config /dev/null` on a dynamic localhost port.
  - Syncs exclusively between two temporary directories in `t.TempDir()`.
  - Exercises directory candidates, trailing whitespace, special characters, and verifies that real upstream Rclone accepts compiled RC parameters without error.

### Tier 5: Headless Chrome End-to-End User Journeys
- **Role**: Catches DOM parsing bugs, Alpine.js hydration failures, and interactive UI regressions.
- **Execution**: `tests/e2e/dashboard_smoke_test.go` running headless browser automation against live embedded server endpoints.

---

## 4. Architectural Findings by Severity

### 🔴 High (Safety / Architecture / Reliability)
* **Finding H-1: High Mock Drift Risk in Synchronization & Review Execution**
  - **Issue**: 100% of Rclone API interactions were tested against artificial `httptest.NewServer` stubs. Real filesystem semantics (directory operations, trailing spaces, actual Rclone RC parameter acceptance) were not exercised.
  - **Remediation**: Implement Tier 4 Real Local Rclone Sandbox Integration Suite (`tests/integration/rclone_sandbox_test.go`).

### 🟡 Medium (Edge Cases & Concurrency)
* **Finding M-1: Missing POSIX Advisory Locking on Review Session Disk Files**
  - **Issue**: `active_review_session.json` and `review.wal` rely on atomic rename for writes, but lack `syscall.Flock`. External CLI commands or scripts reading/writing the session concurrently with the daemon could experience partial reads or overwrite races.
  - **Remediation**: Wrap `SaveActiveSession`, `LoadActiveSession`, and WAL operations in advisory file locks on `active_review_session.json.flock`.

* **Finding M-2: Transient RC Server Disconnects Trip Immediate Workload Aborts**
  - **Issue**: If `rclone-daemon.service` is briefly restarted, scheduled syncs fail immediately with network dial errors.
  - **Remediation**: Introduce a 10–15 second reconnect backoff grace period in `TelemetryService`.

### 🟢 Low (Polish & Maintainability)
* **Finding L-1: Schema Governance Consistency in Legacy REST Handlers**
  - **Issue**: Several endpoints in `handlers_config.go` and `handlers_telemetry.go` construct ad-hoc response structs rather than referencing centralized types in `internal/models`.
  - **Remediation**: Standardize remaining endpoints on `models.ActionResponse` and concrete schema models.

---

## 5. Strategic Roadmap Alignment

Following the November 4, 2026 review and operator decisions, priorities are categorized into:

### Short-Term (Immediate Sprint):
1. **Real Local Rclone Sandbox Integration Suite (`tests/integration/`)**: Spawns real ephemeral `rclone rcd` with `--config /dev/null` in `t.TempDir()`. Zero cloud touch, complete elimination of mock drift.
2. **POSIX Advisory File Locking on Review Session & WAL**: Protect `active_review_session.json` and `review.wal` with `syscall.Flock` on `active_review_session.json.flock`.
3. **REST API Response Schema Standardization**: Eliminate loose untyped maps across all remaining endpoints, migrating to `internal/models/api.go`.
4. **Upstream Daemon Watchdog & Auto-Reconnect Grace Period**: Backoff reconnection retry before failing workloads on brief daemon restarts.

### Medium-Term (Targeted Priorities):
- **A. Event-Driven Local Filesystem Sync (`fsnotify` Debounced Watcher)**: Optional "On Change" schedule mode with 30-second debounce. Syncs immediately when files change while eliminating wasted empty polling cycles.
- **B. Server-Side Configuration Snapshot & Rollback Engine**: Versioned immutable snapshots in `config_snapshots/` with 1-click restore.
- **C. Homelab Reverse Proxy Security Hardening & Prometheus `/metrics`**: Bearer/Basic auth (`DASHBOARD_AUTH_TOKEN`), trusted proxy headers (`X-Forwarded-*`), and `/healthz` + `/metrics` for Grafana monitoring.

### Long-Term Backlog:
- Visual Text/Markdown Diffing in Review Modal (deferred).
- Complex enterprise mesh topologies and autonomous loops are deprioritized as out-of-scope for the core homelab/power user audience.

---

## 6. Modularization & Technical Debt Status

All previous modularization recommendations have been fully achieved:
- `internal/server/`: Split into `server.go`, `routes.go`, `handlers_auth.go`, `handlers_config.go`, `handlers_explorer.go`, `handlers_profiles.go`, `handlers_review.go`, `handlers_sync.go`, `handlers_telemetry.go`. (All <350 LOC).
- `internal/core/client/`: Split into `client.go`, `fs_ops.go`, `jobs.go`, `remotes.go`. (All <250 LOC).
- `internal/profiles/store.go`: Streamlined to 395 LOC with POSIX flock and atomic updates.
- Total files exceeding 400 LOC in production Go code: **0**.

---

## 7. Verification Evidence

Executing `./scripts/check.sh all` on the working tree:

```
====================================================
              Repository Quality Gate
====================================================
[1/7] Verifying Go modules...
all modules verified
Go modules verified.

[2/7] Validating shell scripts...
  ✓ deploy/install.sh
  ✓ scripts/check-loc.sh
  ✓ scripts/check.sh
  ✓ scripts/update-tools.sh
Shell validation passed.

[3/7] Checking Go formatting...
Go formatting check passed.

[4/7] Running go vet...
Go vet passed.

[5/7] Running unit tests with coverage...
Unit tests passed across all packages.
Coverage : 85.5%
Required : 80.0%
Coverage threshold met.

[6/7] Verifying static analysis (staticcheck & gosec)...
Staticcheck passed.
Gosec passed.

[7/7] Verifying line counts (sub-400 LOC limit)...
All production Go files are under 400 lines.

====================================================
Quality Gate Passed (Coverage: 85.5%)
====================================================
```


