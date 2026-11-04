# Implementation Plan: Architecture Hardening & Real Rclone Local Sandbox Suite

## Proposed Changes

Address testing blind spots, race conditions, and schema drift uncovered during recent feature development (Phases 4 & 5, MCP server). Transition from purely synthetic unit-test mocks to a multi-tiered testing model featuring real local Rclone sandbox testing.

### Affected Components & Packages

- `docs/architecture/code-review.md`: Comprehensive periodic review documenting November 4, 2026 audit, bug root causes, and 5-tier testing strategy.
- `internal/engine/review/journal.go`: Wrap active review session and WAL disk operations in POSIX advisory file locking (`syscall.Flock`).
- `internal/services/review_service.go`: Synchronize review session discovery and dismissal with advisory file lock.
- `internal/server/handlers_config.go`, `internal/server/handlers_telemetry.go`, `internal/models/api.go`: Standardize API responses to explicit Go structs (`models.ActionResponse`, etc.).
- `internal/services/telemetry_service.go`, `internal/core/client/client.go`: Upstream RC daemon health watchdog with auto-reconnect backoff.
- `tests/integration/rclone_sandbox_test.go`: Ephemeral local Rclone integration sandbox using `t.TempDir()`, fake source/dest folders, and dynamic-port `rclone rcd --config /dev/null`.

### Technical Strategy & Data Flow

1. **Advisory POSIX File Locking for Review Sessions**:
   - Create `active_review_session.json.flock`.
   - Use `syscall.Flock(fd, syscall.LOCK_EX)` during writes (`SaveActiveSession`, `AppendWAL`) and `syscall.Flock(fd, syscall.LOCK_SH)` during reads (`LoadActiveSession`).
   - Prevents race conditions with external CLI commands or scripts reading/writing session files simultaneously.

2. **Schema Governance**:
   - Replace any remaining untyped `map[string]any` returns in `server` handlers with typed models.

3. **Real Local Rclone Integration Sandbox**:
   - Safe, isolated test harness that spawns a real `rclone rcd` child process on `127.0.0.1:0` with `--config /dev/null`.
   - Uses `t.TempDir()` subdirectories as source and destination. User Google Drive and profile configurations are strictly untouched.
   - Tests directory vs. file semantics, trailing whitespace, and RC parameter mapping directly against the real Rclone engine.

### Safety & Error Handling

- **Canary & Jail Protection**: Sandbox strictly confines all disk mutations inside Go's `t.TempDir()`.
- **Zero Cloud Impact**: Tests pass `--config /dev/null` to guarantee no external cloud remotes are loaded or contacted.
- **Process Cleanup**: Ephemeral `rclone rcd` processes are guaranteed terminated via `t.Cleanup()`.

## Verification Plan

- `test -f docs/architecture/code-review.md`
- `go test -v -count=1 ./internal/engine/review/... ./internal/services/...`
- `go test -v -count=1 ./internal/server/...`
- `go test -v -count=1 ./tests/integration/...`
- `./scripts/check.sh all`


