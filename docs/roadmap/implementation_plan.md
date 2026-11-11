# Implementation Plan: Refine FullRemoteTarget, Fix Static Analysis & Deep Coverage Threshold

## Proposed Changes

Address three specific system issues identified during GitHub Actions nightly execution and MCP live testing:
1. Refine `FullRemoteTarget()` in `internal/profiles/model.go` to support local filesystem paths cleanly without prepending invalid `"local:"` remote names.
2. Bump `go.mod` to Go `1.26.11` to resolve vulnerability `GO-2026-6859` detected by `govulncheck` in CI static analysis.
3. Elevate statement test coverage across the repository to >86.0% to comfortably clear the strict 85.0% nightly quality gate threshold under race detection.

### 1. Refine `FullRemoteTarget()` (`internal/profiles/model.go`)
- When `Profile.Remote` is `"local"` or empty (case-insensitive, ignoring trailing colons):
  - Return `p.RemotePath` directly so that local-to-local sync or local destinations pass raw filesystem paths (e.g., `"/var/backups"` or `"backups"`) to Rclone instead of non-existent `"local:path"` remotes.
- When `Profile.Remote` is a configured remote (e.g., `"workdrive"`, `"gdrive"`):
  - Retain `fmt.Sprintf("%s:%s", r, sub)` (or `r + ":"` if subpath is empty).
- Update unit tests in `internal/profiles/model_test.go` and `internal/profiles/store_test.go`.

### 2. Fix Static Analysis Vulnerability (`go.mod`)
- Bump `go.mod` Go directive to `go 1.26.11` so GitHub Actions `actions/setup-go@v7` provisions `go 1.26.11`, eliminating vulnerability `GO-2026-6859`.
- Verify with `./scripts/check.sh lint`.

### 3. Elevate Unit Test Coverage for Deep Quality Gate (`MIN_COVERAGE=85.0`)
- Expand test coverage in `internal/server`, `internal/mcp`, and `internal/profiles`:
  - `internal/server/handlers_profiles_test.go`: Cover error paths for `handleDeleteProfile` (404), `handleCloneProfile` (400, 404, 409), `handleRollbackProfile` (400, 404), and `handleProfileHistory` (404).
  - `internal/server/handlers_telemetry_test.go`: Cover `handleMCPStatus` (offline and active states), `handleRawBisyncLog`, and `handleRawDaemonLog` (missing and streamed files).
  - `internal/mcp/server_test.go`: Cover `writeStatusFile` and `removeStatusFile` lifecycle with mock container config.
  - `internal/profiles/model_test.go`: Cover `MaxDeleteValue` string and JSON marshalling branches.

## Affected Components & Packages
- `internal/profiles/model.go`: Local vs remote composite path handling in `FullRemoteTarget()`.
- `internal/profiles/model_test.go` & `store_test.go`: Unit tests for `FullRemoteTarget()` and `MaxDeleteValue`.
- `go.mod`: Upgrade Go runtime version to 1.26.11.
- `internal/server/handlers_profiles_test.go`: Extended test coverage for profile endpoints.
- `internal/server/handlers_telemetry_test.go`: Extended test coverage for telemetry and raw log endpoints.
- `internal/mcp/server_test.go`: Status file write/remove tests.

## Verification Plan
1. Task 1: `go test -count=1 -v ./internal/profiles/...`
2. Task 2: `./scripts/check.sh lint`
3. Task 3: `MIN_COVERAGE=85.0 RUN_RACE=true ./scripts/check.sh test`
4. Task 4: `MIN_COVERAGE=85.0 RUN_RACE=true ./scripts/check.sh all`

