# Implementation Plan: Replace Non-Existent Rclone RC operations/touch with core/command

## Proposed Changes

Address the HTTP 404 error when clicking "+ Touch Canary" on remote storage (such as `onedrive:`, `gdrive:`) caused by requesting non-existent endpoint `/operations/touch`. Replace with the official Rclone Remote Control method `/core/command` passing `command: "touch"` and `arg: ["<remote>:<path>"]`. Ensure the explorer UI passes the active directory path when provisioning canaries in subfolders.

### Affected Components & Packages

- `internal/core/client/fs_ops.go`: Refactor `TouchFile` to call `core/command` with `{ command: "touch", arg: [target] }`, parsing Rclone's `{ error: bool, result: string }` response format.
- `dashboard/frontend/js/explorer.js`: In `provisionCanary`, pass `pane.path || ""` instead of hardcoded `""` so canaries can be provisioned in nested subdirectories when navigating.
- `internal/core/client/client_test.go`: Update `TestTouchFile` mock server to handle `/core/command`, validating command arguments, success payloads, and error outputs.
- `internal/services/canary_service_test.go`: Update test mock servers from `/operations/touch` to `/core/command`.
- `internal/services/sync_service_test.go`: Update test mock servers from `/operations/touch` to `/core/command`.

### Technical Strategy & Data Flow

- Remote target formatting:
  - If `fs` ends with `:` (e.g., `onedrive:`), `target` is `fs + cleanPath`.
  - If local path (e.g., `/home/user`), `target` is `filepath.Join(fs, cleanPath)`.
- Rclone `core/command` contract:
  - Request: `POST /core/command` with `{"command": "touch", "arg": [target]}`.
  - Success response: HTTP 200 with `{"error": false, "result": ""}`.
  - Command error response: HTTP 200 with `{"error": true, "result": "<error message>"}` or HTTP >= 400.
  - `TouchFile` validates HTTP status, parses `cmdResp.Error`, and returns a detailed error message from `cmdResp.Result`.

### Safety & Error Handling

- Safe path handling: Preserves `NormalizeFs` and trims leading slashes.
- File size constraints: Ensure all modified files stay well below the 400 LOC ceiling.
- Regression safety: Run full test suite and quality gate to verify zero regressions.

## Verification Plan

- `go test -count=1 ./internal/core/client -run TestTouchFile`
- `go test -count=1 ./internal/services -run TestCanaryService`
- `go test -count=1 ./internal/services -run TestSyncService`
- `go test -count=1 ./...`
- `./scripts/check-loc.sh 400`
- `./scripts/check.sh`
- Binary build & live rclone RC verification: verify `core/command` touch against real `rclone rcd`.


