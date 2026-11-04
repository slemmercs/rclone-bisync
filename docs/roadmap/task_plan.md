# Task Plan: MCP Transport Fix & Resilient Review Work Queue

## Goal
Resolve MCP client connection issues (SSE timeout and setting alias alignment) and refactor review resolution into a resilient work queue with topological ordering, transient retries, and partial success isolation.

## Tasks

- [x] Task 1: Fix MCP SSE Server Handshake & Client Settings Alignment
  - **Scope**: `internal/mcp/server.go`, `/home/csaba/.cline/data/settings/cline_mcp_settings.json`
  - **Verification**: `curl -N -s http://127.0.0.1:8081/sse | head -n 3` and stdio initialization check

- [ ] Task 2: Implement Review Work Queue & Error Classifier
  - **Scope**: `internal/engine/review/queue.go`, `internal/engine/review/queue_test.go`
  - **Verification**: `go test -v -count=1 ./internal/engine/review/... -run "TestReviewQueue.*"`

- [ ] Task 3: Integrate Queue with Resolution Executor & Session Reporting
  - **Scope**: `internal/engine/review/executor.go`, `internal/engine/review/executor_test.go`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-04: Initialized plan on branch `feat/mcp-sse-fix-and-review-queue`.
- 2026-11-04: Completed Task 1:
  - Replaced `server.NewStreamableHTTPServer` with `server.NewSSEServer` in `internal/mcp/server.go`.
  - Verified standard SSE initial endpoint broadcast (`event: endpoint\ndata: /message?sessionId=...`).
  - Aligned `/home/csaba/.cline/data/settings/cline_mcp_settings.json` to configure `bisync-hub-mcp` with SSE transport.
  - Successfully tested end-to-end SSE JSON-RPC initialization and tool discovery via script.




