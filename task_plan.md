# Task Plan: Phase 3 - Go Sync Service & Explorer Architecture

## Goal
Port domain orchestration, explorer services, state tracking, and log parsing to native Go (`internal/core/state` and `internal/services`) while strictly enforcing safety circuit breakers (path traversal jails, canary protection, non-interactive flock checks).

## Tasks
- [x] Task 1: State Management & Atomic File Operations (`internal/core/state`)
  - **Scope**: Implement `WriteJSONAtomic` (.tmp + sync + rename), `PendingDeletionsData` parser/record/clear, and rate limit backoff clearance.
  - **Verification**: `go test -v ./internal/core/state`
- [x] Task 2: Sync & Explorer Service Domain Logic (`internal/services`)
  - **Scope**:
    - Dual-pane path browsing (`BrowsePath`) with path traversal jail (`filepath.Rel` escape prevention) and canary detection (`RCLONE_TEST`).
    - Canary provisioning (`CreateCanary`) on local and remote filesystems.
    - File/directory copy (`CopyItem`) and deletion (`DeleteItem`) with root deletion and canary deletion blocks.
    - Reverse log seeking file tailing (`TailFile` / `TailJSONLog`) unwrapping nested JSON.
    - Pending deletions parsing fallback from daemon log.
    - System actions (`HandleAction`: start, stop, force_resync, force_sync, dry_run, remove_lock).
    - Daemon health tracking with memory caching.
  - **Verification**: `go test -v ./internal/services`
- [x] Task 3: Quality Gate Verification
  - **Scope**: Run full pre-flight quality check (`./scripts/check.sh`).
  - **Verification**: All 7 Go packages pass tests, 77 Python tests pass, secret scanners clean.

## Status Updates
- 2026-10-17: Implemented `internal/core/state` with atomic file persistence and pending deletion models.
- 2026-10-17: Implemented `internal/services` (SyncService & explorer operations).
- 2026-10-17: Validated quality gate via `./scripts/check.sh` - 100% green across Go & Python suites.


