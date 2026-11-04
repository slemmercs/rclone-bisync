# Task Plan: MCP Transport Fix & Resilient Review Work Queue

## Goal
Resolve MCP client connection issues (SSE timeout and setting alias alignment) and refactor review resolution into a resilient work queue with topological ordering, transient retries, and partial success isolation.

## Tasks

- [x] Task 1: Fix MCP SSE Server Handshake & Client Settings Alignment
  - **Scope**: `internal/mcp/server.go`, `/home/csaba/.cline/data/settings/cline_mcp_settings.json`
  - **Verification**: `curl -N -s http://127.0.0.1:8081/sse | head -n 3` and stdio initialization check

- [x] Task 2: Implement Review Work Queue & Error Classifier
  - **Scope**: `internal/engine/review/queue.go`, `internal/engine/review/queue_test.go`
  - **Verification**: `go test -v -count=1 ./internal/engine/review/... -run "TestReviewQueue.*"`

- [x] Task 3: Integrate Queue with Resolution Executor & Session Reporting
  - **Scope**: `internal/engine/review/executor.go`, `internal/engine/review/executor_test.go`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-04: Initialized plan on branch `feat/mcp-sse-fix-and-review-queue`.
- 2026-11-04: Completed Task 1:
  - Replaced `server.NewStreamableHTTPServer` with `server.NewSSEServer` in `internal/mcp/server.go`.
  - Verified standard SSE initial endpoint broadcast (`event: endpoint\ndata: /message?sessionId=...`).
  - Aligned `/home/csaba/.cline/data/settings/cline_mcp_settings.json` to configure `bisync-hub-mcp` with SSE transport.
  - Successfully tested end-to-end SSE JSON-RPC initialization and tool discovery via script.
- 2026-11-04: Completed Task 2:
  - Created `internal/engine/review/queue.go` with `ErrorClass` classifier (idempotent, retryable, fatal).
  - Implemented `SortWorkItems` with topological directory-first restores (top-down) and leaf-first deletions (bottom-up).
  - Implemented `ExecuteWithRetry` with exponential backoff on transient errors and instant success on idempotent 404/409 errors.
  - Implemented `ReviewQueue.Process` with per-candidate error isolation and `BatchSummary` generation.
  - Added unit test coverage in `internal/engine/review/queue_test.go` (100% pass).
- 2026-11-04: Completed Task 3:
  - Integrated `ReviewQueue` into `ResolutionExecutor.ExecuteSession` in `internal/engine/review/executor.go`.
  - Added atomic in-place action recording and per-item error isolation.
  - Added `TestExecutor_QueueTopologicalOrderAndErrorIsolation` to verify parent dirs are restored before files.
  - Verified all 7 quality gate checks pass via `./scripts/check.sh all`.






