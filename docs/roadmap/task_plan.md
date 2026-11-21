# Task Plan: Scheduler Active Jobs Race Condition & Test Teardown

## Goal
Eliminate race condition in `Scheduler.runJob()` where completion signaling precedes `s.activeJobs` de-registration, causing intermittent `ErrSyncInProgress` errors during rapid sequential sync invocations. Clean up test scheduler lifecycles.

## Tasks

- [x] Task 1: Reorder state cleanup in `Scheduler.runJob` and add test teardowns
  - **Scope**: `internal/profiles/scheduler.go`, `internal/mcp/tools_write_test.go`, `internal/profiles/scheduler_test.go`
  - **Verification**: `go test -race -shuffle=1795249090223818792 -count=5 ./internal/mcp` and `go test -race -shuffle=on -count=20 ./internal/mcp -run TestWriteTools_TriggerSyncConfirmation`

- [x] Task 2: Comprehensive validation and quality gate check
  - **Scope**: Repository-wide test suite and quality gates
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-20: Completed Task 2. Ran full quality gates via `./scripts/check.sh all` (all 7 stages: code formatting, template safety, license check, go vet, all unit/integration tests with -race, LOC audit, release binary build). All 7 stages passed cleanly.
- 2026-11-20: Completed Task 1. Reordered `runJob` to de-register `s.activeJobs` and reset worker state before signaling `item.ErrCh <- err`. Added missing `defer scheduler.Stop()` across `internal/mcp/tools_write_test.go` and `internal/profiles/scheduler_test.go`. Verified with `-race`, `-shuffle=on`, and exact seed `1795249090223818792`.
- 2026-11-20: Initialized plan from implementation_plan.md on branch `fix/scheduler-active-jobs-race-condition`.

