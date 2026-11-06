# Implementation Plan: Fix Test Race Conditions & Merge Dependabot Dependencies

## Proposed Changes

Address data race warnings detected during scheduled Nightly Deep Quality & Security runs (`RUN_RACE=true`) in `internal/services/telemetry_service_test.go` and `internal/core/client/client_test.go`. In addition, merge approved patch updates from Dependabot (PR #73 for `chromedp` and PR #74 for `modernc.org/sqlite`).

### Affected Components & Packages

- `internal/services/telemetry_service_test.go`: Use `sync/atomic.Bool` for `isAlive` mock server state.
- `internal/core/client/client_test.go`: Use `sync/atomic.Bool` for `isHealthy` mock server state.
- `go.mod`, `go.sum`: Updated via merging PRs #73 and #74.

### Technical Strategy & Concurrency Safety

- In both tests, an `httptest.Server` handler runs concurrently with background asynchronous test goroutines that simulate daemon shutdown and recovery.
- Unsynchronized read/write on standard `bool` variables triggers Go's race detector.
- Converting these variables to `atomic.Bool` (`Load()` / `Store()`) guarantees thread-safe memory synchronization with zero external dependencies and zero performance penalty.

### Verification Plan

- `gh pr merge 73 --admin --squash --delete-branch` & `gh pr merge 74 --admin --squash --delete-branch`
- `go test -race -v -count=1 ./internal/services/... -run TestTelemetryService_ReconnectGracePeriodAndWatchdog`
- `go test -race -v -count=1 ./internal/core/client/... -run TestWaitForHealthyAndReconnectGrace`
- `go test -race ./internal/...`
- `./scripts/check.sh all`



