# Task Plan: Decommission Legacy Google Drive Systemd Timer & Generalize Deployment

## Goal
Retire redundant `bisync-gdrive.service` and `bisync-gdrive.timer`, upgrade `deploy/install.sh` with safe decommissioning of existing timer units, and generalize transient systemd units in `sync_actions.go`.

## Tasks

- [x] Task 1: Retire `deploy/systemd/bisync-gdrive.*` and Update `deploy/install.sh`
  - **Scope**: `deploy/systemd/bisync-gdrive.service`, `deploy/systemd/bisync-gdrive.timer`, `deploy/install.sh`
  - **Verification**: `bash -n deploy/install.sh`

- [x] Task 2: Generalize Transient Unit Naming & Identifiers in `internal/services/sync_actions.go`
  - **Scope**: `internal/services/sync_actions.go`, `internal/services/sync_service_test.go`
  - **Verification**: `go test -count=1 ./internal/services/...`

- [x] Task 3: Quality Gate Verification & Pre-Flight Checks
  - **Scope**: Entire repository test suite, linters, templates, and LOC budgets
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-10-31: Initialized plan from implementation_plan.md to retire legacy Google Drive systemd timer and align system deployment with multi-profile scheduler.
- 2026-10-31: Completed Task 1: Deleted obsolete `bisync-gdrive.service` and `bisync-gdrive.timer` from `deploy/systemd/`. Updated `deploy/install.sh` to remove them from installation targets and added migration logic to stop, disable, and clean up lingering units on target systems. Verified with `bash -n`.
- 2026-10-31: Completed Task 2: Generalized transient unit names from `bisync-gdrive-*` to `bisync-sync-*` and updated `SyslogIdentifier` to `bisync-hub` in `internal/services/sync_actions.go`. Retained legacy stop patterns in `stop_sync` for backward compatibility. Verified with `go test -count=1 ./internal/services/...`.
- 2026-10-31: Completed Task 3: Executed full 6-stage pre-flight quality gate (`./scripts/check.sh all`). All build, test, vet, leak check, template validation, and LOC budget stages passed cleanly.


