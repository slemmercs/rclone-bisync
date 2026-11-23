# Task Plan: Default Exclusion and Remote Pruning for Review Backups

## Goal
Exclude `.review_backups/**` by default from all sync profiles in the CLI builder, and implement remote review backup lifecycle pruning so internal pre-mutation backups do not sync back and forth or linger indefinitely on cloud storage.

## Tasks

- [x] Task 1: Add default exclusion for .review_backups/** in profile CLI builder
  - **Scope**: `internal/profiles/builder.go`, `internal/profiles/builder_test.go`
  - **Verification**: `go test -count=1 ./internal/profiles/...`

- [ ] Task 2: Implement PruneExpiredRemoteBackups in review engine
  - **Scope**: `internal/engine/review/backup.go`, `internal/engine/review/backup_test.go`
  - **Verification**: `go test -count=1 ./internal/engine/review/...`

- [ ] Task 3: Hook remote review pruning into ReviewService and verify full lifecycle
  - **Scope**: `internal/services/review_execution.go`, `internal/services/review_service_test.go`
  - **Verification**: `go test -count=1 ./internal/services/...`

- [ ] Task 4: Full pre-flight quality verification & check suite
  - **Scope**: Entire codebase
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-23: Completed Task 1. Injected default `--exclude .review_backups/**` into profile CLI argument builder with duplicate prevention, and verified with updated unit tests in `internal/profiles`.
- 2026-11-23: Initialized implementation plan and task plan.
