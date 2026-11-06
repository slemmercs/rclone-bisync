# Task Plan: Fix Test Race Conditions & Merge Dependabot Dependencies

## Goal
Eliminate data races detected in nightly test runs by synchronizing test mock state with `sync/atomic.Bool`, and merge passing Dependabot PRs #73 and #74.

## Tasks

- [x] Task 1: Merge Dependabot PRs #73 & #74, pull main, and create branch `fix/test-race-conditions`
  - **Scope**: GitHub PRs #73 & #74, local git branch
  - **Verification**: `git status && git log -n 3 --oneline`

- [x] Task 2: Implement atomic.Bool race fixes in telemetry and client test suites
  - **Scope**: `internal/services/telemetry_service_test.go`, `internal/core/client/client_test.go`
  - **Verification**: `go test -race -v -count=1 ./internal/services/... -run TestTelemetryService_ReconnectGracePeriodAndWatchdog && go test -race -v -count=1 ./internal/core/client/... -run TestWaitForHealthyAndReconnectGrace`

- [x] Task 3: Full pre-flight quality gate and verification
  - **Scope**: Full repository
  - **Verification**: `go test -race ./internal/... && ./scripts/check.sh all`

## Status Updates

- 2026-11-06: Initialized plan from implementation_plan.md.
- 2026-11-06: Completed Task 1:
  - Squashed and merged Dependabot PR #74 (`modernc.org/sqlite` v1.61.0 -> v1.61.1).
  - Squashed and merged Dependabot PR #73 (`github.com/chromedp/chromedp` v0.21.0 -> v0.21.1).
  - Pulled latest `main` into local repository and created branch `fix/test-race-conditions`.
- 2026-11-06: Completed Task 2:
  - Replaced unsynchronized primitive bool flags with `sync/atomic.Bool` in `internal/services/telemetry_service_test.go` and `internal/core/client/client_test.go`.
  - Verified race-free execution with `go test -race` passing cleanly on both targeted test suites.
- 2026-11-06: Completed Task 3:
  - Executed full race test suite across all internal packages (`go test -race ./internal/...`), passing with zero data races.
  - Executed full pre-flight quality gate (`./scripts/check.sh all`): all 7 quality stages passed cleanly.





