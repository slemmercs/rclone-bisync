# Task Plan: Fix Bisync Transfer Counting & Unified Ledger Visibility

## Goal
Fix zero-transfer counting in bisync job completion by pulling group stats from `/core/stats` (with log text fallback), enrich mirrored audit records with file paths, prevent identical file uploads from folding into single clusters in the Unified Ledger, and auto-refresh paginated logs on Explorer uploads.

## Tasks

- [x] Task 1: Accurate Transfer Counting in `MonitorJob` (`internal/core/client/monitor.go`, `monitor_test.go`)
  - **Scope**: `internal/core/client/monitor.go`, `internal/core/client/monitor_test.go`
  - **Verification**: `go test -count=1 -v ./internal/core/client -run TestMonitorJob`

- [x] Task 2: Mirror Object Path in Audit Log Message (`internal/services/audit_service.go`, `audit_service_test.go`)
  - **Scope**: `internal/services/audit_service.go`, `internal/services/audit_service_test.go`
  - **Verification**: `go test -count=1 -v ./internal/services -run TestAuditService`

- [ ] Task 3: Distinct Clustering Key & Single-Row Action Badge (`dashboard/frontend/js/logs.js`, `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`)
  - **Scope**: `dashboard/frontend/js/logs.js`, `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400`

- [ ] Task 4: Auto-refresh Paginated Bisync & Audit Logs on Upload (`dashboard/frontend/js/explorer_actions.js`)
  - **Scope**: `dashboard/frontend/js/explorer_actions.js`
  - **Verification**: `./scripts/check.sh loc 400`

- [ ] Task 5: End-to-End & Full Quality Gate Verification (`tests/e2e/dashboard_smoke_test.go`)
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-13: Initialized task plan from implementation_plan.md.
- 2026-11-13: Completed Task 1 - Added group stats transfer extraction from `/core/stats` and stdout log parsing fallback in `MonitorJob` (`internal/core/client/monitor.go`), verified with `go test -count=1 -v ./internal/core/client -run TestMonitorJob`.
- 2026-11-13: Completed Task 2 - Mirrored object target path into `Msg` in `RecordEvent` (`internal/services/audit_service.go`), verified with `go test -count=1 -v ./internal/services -run TestAuditService`.



