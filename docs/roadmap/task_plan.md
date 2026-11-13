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

- [x] Task 3: Distinct Clustering Key & Single-Row Action Badge (`dashboard/frontend/js/logs.js`, `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`)
  - **Scope**: `dashboard/frontend/js/logs.js`, `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400`

- [x] Task 4: Auto-refresh Paginated Bisync & Audit Logs on Upload (`dashboard/frontend/js/explorer_actions.js`)
  - **Scope**: `dashboard/frontend/js/explorer_actions.js`
  - **Verification**: `./scripts/check.sh loc 400`

- [x] Task 5: End-to-End & Full Quality Gate Verification (`tests/e2e/dashboard_smoke_test.go`)
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-13: Initialized task plan from implementation_plan.md.
- 2026-11-13: Completed Task 1 - Added group stats transfer extraction from `/core/stats` and stdout log parsing fallback in `MonitorJob` (`internal/core/client/monitor.go`), verified with `go test -count=1 -v ./internal/core/client -run TestMonitorJob`.
- 2026-11-13: Completed Task 2 - Mirrored object target path into `Msg` in `RecordEvent` (`internal/services/audit_service.go`), verified with `go test -count=1 -v ./internal/services -run TestAuditService`.
- 2026-11-13: Completed Task 3 - Distinct clustering key including object/action in `clusterLogEntries` and action badges rendered on single rows and subEntries, verified with `./scripts/check.sh templates && ./scripts/check.sh loc 400`.
- 2026-11-13: Completed Task 4 - Immediate auto-refresh of paginated bisync logs and audit records on upload/delete/copy in `dashboard/frontend/js/explorer_actions.js`, verified with `./scripts/check.sh loc 400` (394 LOC).
- 2026-11-13: Completed Task 5 - Verified end-to-end headless Chrome smoke test covering audit timeline clustering, action badges, and upload refresh hooks; formatted files and verified all 7 quality gate checks via `./scripts/check.sh all` (OK: 7/7).



