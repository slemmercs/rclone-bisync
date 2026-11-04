# Implementation Plan: Consolidate MCP into Dashboard & Secure Profile Store

## Proposed Changes

### 1. Profile Store POSIX Advisory Locking & Safe In-Place Updates (`internal/profiles/store.go`)
- Wrap file read/write operations on `profiles.json` with POSIX advisory locking (`syscall.Flock`) on a dedicated lock file (`profiles.json.flock`).
- In `Store.UpdateState`, perform in-place operational state updating without blindly overwriting configuration fields (`schedule`, `review_mode`, `exclude_patterns`, `flags`) if disk contents have diverged.
- Check file modification timestamp (`mtime`) on disk to detect changes and reload before mutations.

### 2. Embedded MCP SSE Server within Dashboard (`main.go`, `internal/config/config.go`)
- Add `MCPEnabled` (default: `true`) and `MCPPort` (default: `8081`) to `config.Config` and CLI flags in `dashboard`.
- In `main.go:runDashboard`, launch the MCP SSE server concurrently as a managed background listener sharing the same `appContainer` as the Web UI.
- Handle unified graceful shutdown: when `SIGTERM` / `SIGINT` is received, both the Dashboard HTTP server (`:8080`) and the MCP SSE server (`:8081`) terminate gracefully without orphan processes.
- Strip `appContainer.Start()` from standalone `runMCP` so external CLI or stdio invocations of `rclone-bisync mcp` never spawn background scheduler loops.

### 3. Service Consolidation & Cleanup (`deploy/install.sh`, `deploy/systemd/`)
- Retire `deploy/systemd/rclone-mcp.service`.
- In `deploy/install.sh`, stop, disable, and clean up any existing `rclone-mcp.service` on the user system.
- Manage exactly two systemd services:
  1. `rclone-daemon.service` (upstream Rclone RC engine)
  2. `rclone-dashboard.service` (dashboard web UI + embedded MCP)
- Update UI modal (`dashboard/frontend/partials/modals/modal_mcp.html`) to reference `rclone-dashboard.service`.

## Verification Plan

- Unit tests:
  - `go test -count=1 ./internal/profiles/...`
  - `go test -count=1 ./internal/server/...`
  - `go test -count=1 ./internal/mcp/...`
- Quality gate pre-flight check:
  - `./scripts/check.sh all`


