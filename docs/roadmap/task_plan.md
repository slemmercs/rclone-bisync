# Task Plan: De-Noise Unified Ledger & Sync Milestones

## Goal
Eliminate UI noise and duplicate milestone lines so that sync and dry-run executions present a single, clean, reassuring milestone entry in the dashboard Unified Ledger.

## Tasks

- [x] Task 1: Backend Log De-Noising & Milestones
  - **Scope**: `internal/core/client/monitor.go`, `internal/engine/runner/runner_handlers.go`, `internal/engine/runner/runner_test.go`, `internal/server/server.go`, `internal/server/server_test.go`
  - **Verification**: `go test -count=1 ./internal/core/client/... ./internal/engine/runner/... ./internal/server/...` and `./scripts/check.sh loc 400`

- [ ] Task 2: Frontend Ledger De-Noising & UI Signal Isolation
  - **Scope**: `dashboard/frontend/js/logs.js`
  - **Verification**: `go test -count=1 ./tests/e2e/...` and `./scripts/check-loc.sh 400`

- [ ] Task 3: Full Quality Gate Verification & Release Preparation
  - **Scope**: Full repository quality gate verification
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-10-31: Initialized plan from implementation_plan.md to de-noise the Unified Ledger, demote routine job polling and HTTP access to debug, and eliminate duplicate sync completion logs.
- 2026-10-31: Completed Task 1: Demoted job polling milestones and successful HTTP access (status < 400) to slog.LevelDebug, eliminated duplicate log.Printf calls in runner_handlers.go, added contextual sync mode prefixes (Dry run, Resync, Sync), and added test coverage. All tests passed and LOC verified.




