# Task Plan: Safe Session Path Confinement (CodeQL #24) & MCP Server Hardening

## Goal
Resolve CodeQL alert #24 (`go/path-injection`) by implementing strict session ID validation and path jail confinement for review session handling, and prepare next steps for MCP server hardening.

## Tasks

- [x] Task 1: Safe Review Session ID Validation & Path Confinement (CodeQL Alert #24)
  - **Scope**: `internal/engine/review/session.go`, `internal/engine/review/session_test.go`, `internal/engine/review/journal.go`, `internal/engine/review/journal_test.go`, `internal/server/handlers_review.go`, `internal/services/review_execution.go`, `internal/services/review_service_test.go`, `internal/mcp/resources.go`, `internal/mcp/tools_read.go`, `internal/mcp/tools_write.go`
  - **Verification**: `go test -v -count=1 ./internal/engine/review/... ./internal/services/... ./internal/server/... ./internal/mcp/...` and `./scripts/check.sh all`

- [x] Task 2: Diagnostic Error Excerpts in MCP Sync Trigger (`bisync_trigger_sync`)
  - **Scope**: `internal/mcp/tools_write.go`, `internal/mcp/tools_write_test.go`
  - **Verification**: `go test -v -count=1 ./internal/mcp/... -run "TestTriggerSync"`

- [x] Task 3: Profile and Run ID Scoped Log Filtering in MCP (`bisync_get_logs`)
  - **Scope**: `internal/mcp/tools_read.go`, `internal/mcp/tools_read_test.go`
  - **Verification**: `go test -v -count=1 ./internal/mcp/... -run "TestGetLogs"`

- [x] Task 4: Flag Compiler Validation Tool (`bisync_validate_flags`) & Explorer Pagination Metadata
  - **Scope**: `internal/mcp/tools_read.go`, `internal/mcp/server.go`, `internal/mcp/tools_read_test.go`
  - **Verification**: `go test -v -count=1 ./internal/mcp/... -run "TestValidateFlags"`

- [ ] Task 5: Dashboard Header MCP Server Status Indicator
  - **Scope**: `dashboard/frontend/index.html`, `dashboard/frontend/app.js`, `dashboard/frontend/js/polling.js`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-03: Initialized implementation plan and task plan on branch `fix/codeql-24-path-injection-and-mcp-plan`.
- 2026-11-03: Completed Task 1:
  - Implemented `review.IsValidSessionID` with strict character set validation (`^[a-zA-Z0-9_-]+$`).
  - Added traversal sequence checks in `review.LoadActiveSession` and `review_execution.RollbackReviewSession`.
  - Added request sanitization and HTTP 400 rejection for traversal payloads across all review API handlers.
  - Guarded MCP conflict resources and tools against directory traversal.
  - Verified with comprehensive unit regression tests and passed full `./scripts/check.sh all` gate.
- 2026-11-03: Completed Task 2:
  - Implemented `SyncRunner.RunWithError` returning explicit, classified error reasons upon failure.
  - Propagated detailed abort diagnostics through `DefaultProfileExecutor` and `Scheduler`.
  - Enhanced MCP `bisync_trigger_sync` to return exact error reasons (e.g. `TOO_MANY_DELETES`, canary missing, lock contention) and automatically append active review barrier guidance.
  - Added unit test `TestWriteTools_TriggerSyncDiagnostics` in `internal/mcp/tools_write_test.go` and verified with `./scripts/check.sh all`.
- 2026-11-03: Completed Task 3:
  - Added `profile_id` and `run_id` filter parameters to MCP `bisync_get_logs`.
  - Implemented structured JSON and plain-text fallback matching for log levels, profile IDs, and execution run IDs.
  - Decomposed `internal/mcp/tools_read.go` into cohesive `tools_read_explorer.go` to strictly satisfy <400 LOC limits.
  - Added unit tests in `internal/mcp/tools_read_opt_test.go` and verified with all 7 quality gate checks in `./scripts/check.sh all`.
- 2026-11-03: Completed Task 4:
  - Implemented `compiler.CompileFlagsToParams` returning structured `FlagCompilationResult` (params, config, filter, backend options, unrecognized flags, errors).
  - Added read-only MCP tool `bisync_validate_flags` supporting string or array flags and profile ID resolution.
  - Added `total_count` and `has_more` metadata to `bisync_explore_path` pagination responses.
  - Created `internal/mcp/tools_read_compiler.go` maintaining file sizes under 400 LOC.
  - Added unit tests `TestCompileFlagsToParams` and `TestValidateFlags` and verified with `./scripts/check.sh all`.

