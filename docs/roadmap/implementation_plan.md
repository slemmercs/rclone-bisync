# Implementation Plan: MCP Transport Fix & Resilient Review Work Queue

## Proposed Changes

### 1. MCP Transport & Settings Alignment
- In `internal/mcp/server.go`, replace `server.NewStreamableHTTPServer` with standard `server.NewSSEServer(s.mcpServer)` to send standard SSE events (`event: endpoint\ndata: ...`) so SSE clients do not experience body timeout errors.
- In `/home/csaba/.cline/data/settings/cline_mcp_settings.json`, align the server definitions so that both `bisync-hub-mcp` and `rclone-bisync` exist, pointing to the binary via stdio.

### 2. Resilient Review Queue (`internal/engine/review/queue.go`)
- Introduce topological sorting:
  - Restore: directories top-down (depth ascending) before files.
  - Delete: files before directories bottom-up (depth descending).
- Introduce error classification:
  - Idempotent (404 on delete, 409 already exists on mkdir) -> marked completed.
  - Retryable (429 rate limit, network timeout) -> exponential backoff up to 3 attempts.
  - Fatal (path traversal, invalid payload) -> item marked failed, batch continues to completion.
- Introduce `ReviewQueue` worker to process items with per-item WAL updates and generate a `BatchExecutionSummary`.

## Verification Plan
- Stdio & SSE MCP connection verification.
- Unit tests: `go test -v -count=1 ./internal/engine/review/... -run "TestReviewQueue.*"`
- Full test pass: `./scripts/check.sh all`


