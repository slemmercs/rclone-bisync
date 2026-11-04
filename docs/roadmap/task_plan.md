# Task Plan: Consolidate MCP into Dashboard & Secure Profile Store

## Goal
Consolidate the MCP SSE server into the primary dashboard daemon to eliminate duplicate schedulers and multi-process file overwrite races. Add POSIX flock and safe in-place state updating to `profiles.Store`. Consolidate systemd user services to two units.

## Tasks

- [x] Task 1: Add POSIX Advisory Locking & Safe In-Place Updates to Profile Store
  - **Scope**: `internal/profiles/store.go`, `internal/profiles/store_test.go`
  - **Verification**: `go test -v -count=1 ./internal/profiles/...`

- [ ] Task 2: Embed MCP SSE Server in Dashboard & Strip Scheduler from Standalone MCP
  - **Scope**: `main.go`, `internal/config/config.go`
  - **Verification**: `go test -count=1 ./internal/config/... ./internal/server/...`

- [ ] Task 3: Consolidate Systemd Services & Update Install Script
  - **Scope**: `deploy/install.sh`, `deploy/systemd/rclone-mcp.service`, `dashboard/frontend/partials/modals/modal_mcp.html`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-04: Initialized plan on branch `feat/consolidate-mcp-and-secure-profiles`.
- 2026-11-04: Completed Task 1:
  - Implemented POSIX advisory flock (`syscall.Flock`) on `profiles.json.flock` around all profile reads and writes.
  - Implemented `reloadIfModifiedLocked()` comparing file `mtime` so stores dynamically reload on disk mutations.
  - Secured `UpdateState` to reload latest on-disk options and update only dynamic telemetry (`LastSync`, `Status`, `ConsecutiveFailures`, `LastDurationSec`) without clobbering user configuration (`Schedule`, `ReviewMode`, `ExcludePatterns`, `Flags`).
  - Added multi-process test `TestStore_MultiProcessSynchronizationAndNoSettingsClobber` verifying non-destructive concurrency. All profile tests pass.






