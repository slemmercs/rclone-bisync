# Task Plan: Architecture Hardening & Real Rclone Sandbox Suite

## Goal
Implement architectural review recommendations: document November 4, 2026 audit and 5-tier testing strategy, harden review sessions with POSIX flock, standardize API responses, implement upstream RC daemon auto-reconnect, and implement the real local Rclone sandbox test suite.

## Tasks

- [x] Task 1: Document November 4, 2026 Code Review & Testing Strategy
  - **Scope**: `docs/architecture/code-review.md`, `docs/roadmap/ROADMAP.md`
  - **Verification**: `test -f docs/architecture/code-review.md && ./scripts/check.sh loc`

- [x] Task 2: POSIX Advisory File Locking on Review Session & WAL
  - **Scope**: `internal/engine/review/journal.go`, `internal/services/review_service.go`, `internal/engine/review/journal_test.go`
  - **Verification**: `go test -v -count=1 ./internal/engine/review/... ./internal/services/... -run "TestJournal|TestReview"`

- [x] Task 3: Standardize REST API Schemas with Concrete Models
  - **Scope**: `internal/server/handlers_config.go`, `internal/server/handlers_telemetry.go`, `internal/models/api.go`
  - **Verification**: `go test -v -count=1 ./internal/server/...`

- [x] Task 4: Implement Daemon Watchdog & Auto-Reconnect Grace Period
  - **Scope**: `internal/services/telemetry_service.go`, `internal/core/client/client.go`
  - **Verification**: `go test -v -count=1 ./internal/services/... -run "TestTelemetry"`

- [x] Task 5: Real Local Rclone Sandbox Integration Test Suite
  - **Scope**: `tests/integration/rclone_sandbox_test.go`
  - **Verification**: `go test -v -count=1 ./tests/integration/...`

- [x] Task 6: Full Pre-Flight Validation & Quality Gate
  - **Scope**: Full repository
  - **Verification**: `./scripts/check.sh all`
- [x] Task 7: Real Local Rclone Sandbox Regression Tests (Trailing Whitespace, Idempotent 404, Mixed Directory/File Review Execution)
  - **Scope**: `tests/integration/rclone_sandbox_edgecases_test.go`
  - **Verification**: `go test -v -count=1 ./tests/integration/... -run "TestRcloneSandbox_"`


## Status Updates

- 2026-11-04: Initialized plan from implementation_plan.md on branch `feat/consolidate-mcp-and-secure-profiles`.
- 2026-11-04: Completed Task 1:
  - Documented root causes of recent bug clusters (concurrency, directory semantics, path quirks, RC parameter mapping) in `docs/architecture/code-review.md`.
  - Formulated the 5-tier testing architecture (Unit -> Subprocess Concurrency -> Real Local Rclone Sandbox -> Stateful Chaos/Saga -> E2E Chrome).
  - Aligned short, medium, and long-term roadmap priorities in `docs/roadmap/ROADMAP.md` and `docs/architecture/code-review.md`.
  - Verification passed (`test -f docs/architecture/code-review.md && ./scripts/check.sh loc`).
- 2026-11-04: Completed Task 2:
  - Added POSIX advisory file locking (`syscall.Flock`) to review session and WAL disk operations (`SaveActiveSession`, `LoadActiveSession`, `ArchiveSession`, `AppendWAL`, `ReadWAL`, `DismissActiveSessionFile`, `RemoveWAL`).
  - Integrated `DismissActiveSessionFile` and `RemoveWAL` into `ReviewService.ClearSession`, `ExecuteSession`, and `DismissSession`.
  - Added concurrency unit test `TestJournal_POSIXFlockConcurrency` verifying race-free concurrent session and WAL operations across 20 workers.
  - Verification passed (`go test -v -count=1 ./internal/engine/review/... ./internal/services/... -run "TestJournal|TestReview"`).
- 2026-11-04: Completed Task 3:
  - Centralized REST API models in `internal/models/api.go` (`GlobalDefaults`, `ConfigDaemonSettings`, `ConfigNotificationSettings`, `GlobalConfigResponse`, `UpdateConfigRequest`, `TelemetryResponse`).
  - Aliased `config.GlobalDefaults = models.GlobalDefaults` to retain 100% backward compatibility without cyclic dependencies.
  - Replaced unstructured map returns in `internal/server/handlers_telemetry.go` with typed `models.TelemetryResponse`.
  - Added schema validation tests in `internal/models/api_test.go`.
  - Verification passed (`go test -v -count=1 ./internal/server/...`).





- 2026-11-04: Completed Task 4:
  - Added auto-reconnect grace period (`ReconnectGracePeriod`) and progressive retry backoff on transient connection drops in `RcloneClient`.
  - Added `WaitForHealthy` to `RcloneClient` and `WaitForDaemon` watchdog in `TelemetryService`.
  - Added reconnecting state (`status: "reconnecting"`) within grace period before declaring daemon offline in `TelemetryService.GetDaemonHealth`.
  - Modularized watchdog implementation into `internal/services/telemetry_watchdog.go` maintaining strictly bounded files (<400 LOC).
  - Verification passed (`go test -v -count=1 ./internal/services/... -run "TestTelemetry"`).


- 2026-11-04: Completed Task 5:
  - Created Tier 4 Real Local Rclone Sandbox Integration Test Suite in `tests/integration/rclone_sandbox_test.go`.
  - Implemented dynamic-port `rclone rcd` harness using `t.TempDir()`, isolated cache, and `--config /dev/null`.
  - Validated live daemon health checks, real core stats, file/directory lifecycle (`Mkdir`, `TouchFile`, `ListDirectory`, `Canary`, `CopyFile`, `DeleteFile`, `PurgeDir`), and full two-way `sync/bisync` with job monitoring.
  - Verification passed (`go test -v -count=1 ./tests/integration/...`).

- 2026-11-04: Completed Task 6:
  - Ran full pre-flight quality check suite (`./scripts/check.sh all`).
  - All 7 stages (gofmt, shellcheck, loc under 400 lines, go vet, staticcheck/gosec, unit/integration test coverage, cross-platform build) passed cleanly in 15.686s.
  - All tasks in task plan are now complete.


- 2026-11-04: Completed Task 7:
  - Added Tier 4 regression tests in `tests/integration/rclone_sandbox_edgecases_test.go` targeting recent bug clusters.
  - Verified trailing-space directory reconciliation on disk (`Alpha `) with real Rclone review execution (`TestRcloneSandbox_TrailingSpaceDirectoryReconciliation`).
  - Verified idempotent 404/not-found handling when deleting absent files/directories against real Rclone RC (`TestRcloneSandbox_Idempotent404Handling`).
  - Verified mixed directory and file restoration without HTTP 500 (`TestRcloneSandbox_MixedDirectoryAndFileReviewExecution`).
  - All 6 sandbox tests passed cleanly in 1.4s. Full quality gate passed in 14.3s.



