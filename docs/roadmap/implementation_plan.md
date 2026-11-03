# Implementation Plan: Safe Session Path Confinement (CodeQL #24) & MCP Server Hardening

## Proposed Changes

Address security alert CodeQL #24 (`go/path-injection`) by strictly validating session identifiers and enforcing filesystem jail confinement in review session handling, and outline the upcoming MCP server enhancements based on the agent evaluation review.

### Part 1: Safe Review Session ID Validation & Path Confinement (CodeQL Alert #24)

#### Affected Components & Packages
- `internal/engine/review/session.go`:
  - Add `IsValidSessionID(id string) bool` enforcing alphanumeric, hyphen, and underscore characters (`^[a-zA-Z0-9_-]+$`) and forbidding path separators (`/`, `\`) or traversal tokens (`..`).
- `internal/engine/review/journal.go`:
  - In `LoadActiveSession(activeFilePath string)`, validate that `filepath.Clean(activeFilePath)` contains no `..` traversal sequences before opening.
- `internal/server/handlers_review.go`:
  - Sanitize and validate `sessionID` across all review HTTP handlers (`handleReviewRollback`, `handleReviewDismiss`, `handleReviewExecute`, `handleReviewAction`, `handleReviewActionBatch`), rejecting any input containing `/`, `\`, or `..` with HTTP 400 Bad Request.
- `internal/services/review_execution.go`:
  - In `RollbackReviewSession(ctx, sessionID)`, validate `IsValidSessionID(sessionID)`, use `filepath.Base(sessionID)`, and verify relative directory containment with `filepath.Rel(cleanReviewDir, archivePath)` ensuring `!strings.HasPrefix(rel, "..")`.
- `internal/mcp/resources.go`, `internal/mcp/tools_read.go`, `internal/mcp/tools_write.go`:
  - Guard `sessionID` lookups against directory traversal using `review.IsValidSessionID(sessionID)`.

### Part 2: Native MCP Server Hardening & DX Improvements (Next Milestone)

#### Proposed Capabilities
1. **Diagnostic Error Excerpts on Aborts (`bisync_trigger_sync`)**:
   - Parse and return the last stderr error line or reason (e.g. `too many deletes (>0%)`, canary missing) in the tool response so AI agents do not need secondary log round-trips.
2. **Profile & Run ID Scoped Log Filtering (`bisync_get_logs`)**:
   - Add optional `profile_id` and `run_id` filter arguments to isolate traces on multi-profile systems.
3. **Compiler Flag Validation Tool (`bisync_validate_flags`)**:
   - New read-only MCP tool exposing `compiler.CompileFlagsToParams` to test flag parsing and `_config` dictionaries without executing sync.
4. **Explorer Pagination Metadata (`bisync_explore_path`)**:
   - Include `total_count` and `has_more: bool` in explorer responses to avoid speculative pagination queries.
5. **Dashboard Header MCP Status Indicator**:
   - Top header status pill (`MCP: ACTIVE [read/write]`) fed by the SSE connection manager or daemon state.

## Verification Plan

- `go test -v -count=1 ./internal/engine/review/... -run "TestSession|TestJournal"`
- `go test -v -count=1 ./internal/services/... -run "TestReview"`
- `go test -v -count=1 ./internal/server/... -run "TestReview"`
- `./scripts/check.sh all`

