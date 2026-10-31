# Task Plan: Profile-Scoped Logs, Unified Ledger Filtering & Real-Time Polling

## Goal
Add profile names to sync execution log lines and JSON models, enable profile filtering and cluster isolation in the dashboard unified ledger, and fix real-time polling so logs appear immediately without requiring manual page reloads.

## Tasks

- [x] Task 1: Backend Model Enrichment & Profile Scoped Log Formatting
  - **Scope**: `internal/models/api.go`, `internal/engine/runner/runner.go`, `internal/engine/runner/runner_handlers.go`, `internal/engine/runner/runner_test.go`
  - **Verification**: `go test -count=1 ./internal/engine/runner/...` and `./scripts/check.sh loc 400`

- [x] Task 2: Frontend Profile Filtering, Cluster Isolation & Real-Time Polling Fix
  - **Scope**: `dashboard/frontend/js/utils.js`, `dashboard/frontend/js/logs.js`, `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`, `dashboard/frontend/js/polling.js`, `dashboard/frontend/js/profiles_history.js`
  - **Verification**: `go test -count=1 ./tests/e2e/...` and `./scripts/check.sh loc 400`

- [x] Task 3: Full Quality Gate Verification & Release
  - **Scope**: Entire repository
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-10-31: Initialized plan from implementation_plan.md to add profile name scoping to logs, enable ledger profile filtering, and restore real-time polling.
- 2026-10-31: Completed Task 1: Added ProfileID and ProfileName to BisyncLogEntry model, centralized profile label formatting in SyncRunner, prefixed console log.Printf and log Msg with [<ProfileName>] for custom profiles, and populated fields across all RecordSyncRun invocations. Verified with runner unit tests and loc check.
- 2026-10-31: Completed Task 2: Extracted profile identification in parseJsonLogLine, isolated folded clusters by profile key, added profile dropdown filtering to the Unified Ledger view, and fixed real-time log polling via silent page 1 refreshes on sync finish and during background polling. Verified with e2e smoke tests and loc check.
- 2026-10-31: Completed Task 3: Full quality gate passed with 6/6 stages passing clean (tests, loc checks, vet, lint). Ready for PR and release.


