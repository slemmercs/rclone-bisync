# Task Plan: Replace Non-Existent Rclone RC operations/touch with core/command

## Goal
Implement `/core/command` invocation for touching files in `RcloneClient.TouchFile`, update the frontend to pass current view path when provisioning canaries in subdirectories, and update tests across client and service suites.

## Tasks

- [x] Task 1: Update `RcloneClient.TouchFile` to use `/core/command`
  - **Scope**: `internal/core/client/fs_ops.go`, `internal/core/client/client_test.go`
  - **Verification**: `go test -count=1 ./internal/core/client -run TestTouchFile`

- [x] Task 2: Update mock servers in service tests
  - **Scope**: `internal/services/canary_service_test.go`, `internal/services/sync_service_test.go`
  - **Verification**: `go test -count=1 ./internal/services -run "TestCanaryService|TestSyncService"`

- [x] Task 3: Update explorer frontend `provisionCanary` to pass current subpath
  - **Scope**: `dashboard/frontend/js/explorer.js`
  - **Verification**: `./scripts/check-loc.sh 400`

- [x] Task 4: Full verification, compilation, and live Rclone RC check
  - **Scope**: Build `bisync-hub`, run all quality gates, live test with local `rclone rcd`
  - **Verification**: `./scripts/check.sh && ./scripts/check-loc.sh 400`

## Status Updates

- 2026-10-25: Initialized plan from implementation_plan.md on branch `fix/rclone-rc-touch-canary`.
- 2026-10-25: Refactored `TouchFile` in `internal/core/client/fs_ops.go` to invoke `/core/command` with `command: "touch"` and `arg: [target]`.
- 2026-10-25: Updated unit tests in `client_test.go`, `canary_service_test.go`, and `sync_service_test.go` to match the `/core/command` contract.
- 2026-10-25: Updated `explorer.js` `provisionCanary` to pass `pane.path || ""` for proper subdirectory canary provisioning.
- 2026-10-25: Built `./bisync-hub` single binary and verified full quality pipeline (88.2% test coverage, 0 files > 400 LOC).
- 2026-10-25: Live-tested `core/command` touch against local authenticated `rclone rcd` instance; confirmed successful 200 response and zero-byte file creation.



