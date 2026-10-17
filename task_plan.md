
# Task Plan: Go Test Coverage Uplift & Quality Gate (>80%)

## Goal
Resolve root package covdata toolchain error under `go test -cover ./...`, modernize engineering rules, fix UI frontend issues, and uplift automated test coverage across all packages to >80%.

## Tasks
- [x] Task 1: UI Bug Fixes & Rules Modernization (`.clinerules`, `utils.js`, `charts.js`, `remotes.js`, `app.js`, `telemetry.go`, `index.html`)
  - **Scope**:
    - Fixed duplicate `performanceChart` declaration in global scope (`utils.js` / `charts.js`).
    - Fixed double comma syntax errors in `remotes.js` and `app.js`.
    - Modernized `.clinerules` to Go toolchain standards.
    - Fixed active local topology empty tiles by converting Config struct fields to upper snake_case (`LOCAL_PATH`, `REMOTE_PATH`, `BASE_DIR`).
  - **Verification**: `./scripts/check.sh` and browser console error-free.
- [x] Task 2: Fix Root Package covdata Toolchain Issue (`main_test.go`)
  - **Scope**: Add `main_test.go` covering CLI flag parsing and subcommands (`dashboard`, `sync`, `canary`).
  - **Verification**: `go test -cover ./...` succeeds without `covdata` error.
- [x] Task 3: Uplift `internal/core/state` Test Coverage (>90%)
  - **Scope**: Add unit tests for rate-limit backoff flag lifecycle, notification state persistence, metrics JSONL appending, and pending deletions edge cases (`internal/core/state/state_test.go`).
  - **Verification**: `go test -cover ./internal/core/state` (achieved 91.3%).
- [x] Task 4: Uplift `internal/core/client` Test Coverage (>85%)
  - **Scope**: Add mock HTTP tests for `TouchFile`, `CopyFile`, `CopyDir`, `DeleteFile`, `PurgeDir`, `ListRemotes`, `GetRemoteAbout`, `GetCoreStats`, and retry backoff (`internal/core/client/client_test.go`, `monitor_test.go`).
  - **Verification**: `go test -cover ./internal/core/client` (achieved 85.5%).
- [x] Task 5: Uplift `internal/services` Test Coverage (>80%)
  - **Scope**: Add `internal/services/telemetry_test.go` (stats, cached metrics, secret redaction, telemetry payload) and expand `internal/services/sync_service_test.go` (`CopyItem`, log parsing, `GetRemotesSummary`, action handling, remote deletions).
  - **Verification**: `go test -cover ./internal/services` (achieved 82.9%).
- [x] Task 6: Uplift `internal/engine/runner` & `internal/server` Test Coverage (>85%)
  - **Scope**: Add tests for lock contention, rate-limit aborts, and job failure error classification (`runner_test.go`). Add tests for logs, actions, pending deletions, remotes, browse, copy, and delete routes (`server_test.go`).
  - **Verification**: `go test -cover ./internal/engine/runner ./internal/server` (runner: 88.4%, server: 87.2%).
- [x] Task 7: Final Quality Gate & End-to-End Build Verification
  - **Scope**: Run full `./scripts/check.sh`, `go test -cover ./...` verifying >80% statement coverage across all packages.
  - **Verification**: `./scripts/check.sh` passed cleanly with all packages >80%.

## Status Updates
- 2026-10-17: Completed Task 1 (UI syntax fixes, topology snake_case mapping, .clinerules modernization).
- 2026-10-17: Completed Task 2 (Fixed root package `covdata` issue by adding `main_test.go`).
- 2026-10-17: Completed Task 3 (`internal/core/state` uplifted to 91.3%).
- 2026-10-17: Completed Task 4 (`internal/core/client` uplifted to 85.5%).
- 2026-10-17: Completed Task 5 (`internal/services` uplifted to 82.9%).
- 2026-10-17: Completed Task 6 (`internal/engine/runner` to 88.4%, `internal/server` to 87.2%).
- 2026-10-17: Completed Task 7 (`internal/config`: 92.6%, `internal/core/compiler`: 90.2%, `internal/core/notifier`: 89.4%, `internal/core/lock`: 85.7%). Entire repo passes quality gate.
