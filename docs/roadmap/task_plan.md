# Task Plan: Resolution Executor Directory Handling

## Goal
Fix HTTP 500 failure when restoring or deleting directory candidates in review sessions by introducing directory-aware operations (`Mkdir`, `PurgeDir`, `os.RemoveAll`, `os.MkdirAll`) across the review engine and Rclone RC client.

## Tasks

- [x] Task 1: RcloneClient Mkdir & Review Engine RemoteClient Interface Expansion
  - **Scope**: `internal/core/client/fs_ops.go`, `internal/core/client/client_test.go`, `internal/engine/review/backup.go`
  - **Verification**: `go test -v -count=1 ./internal/core/client/... -run "TestClientMkdir"` and `go test -v -count=1 ./internal/engine/review/...`

- [x] Task 2: Directory-Aware Resolution Execution & Local Directory Backups
  - **Scope**: `internal/engine/review/backup.go`, `internal/engine/review/executor.go`, `internal/engine/review/executor_test.go`, `internal/services/review_service_test.go`
  - **Verification**: `go test -v -count=1 ./internal/engine/review/... -run "TestExecutor_Directory"` and `./scripts/check.sh all`

## Status Updates

- 2026-11-04: Initialized task plan on branch `fix/review-directory-restore-http-500` following root-cause analysis of HTTP 500 failure on session `rev-0b6b9ffb861c7e84`.
- 2026-11-04: Completed Task 1:
  - Added `Mkdir(ctx, fs, remote)` method to `*RcloneClient` invoking RC `/operations/mkdir`.
  - Added `Mkdir` and `PurgeDir` to `review.RemoteClient` interface.
  - Added unit test `TestClientMkdir` in `internal/core/client/client_test.go`.
  - Updated review test mocks to implement the expanded interface.
- 2026-11-04: Completed Task 2:
  - Updated `LocalBackup` and `RestoreLocalBackup` to handle directories with `os.MkdirAll` without attempting file copy/hardlinks.
  - Updated `executeDeletion` in `internal/engine/review/executor.go` to handle `isDir`: using `e.Client.Mkdir` on remote or `os.MkdirAll` on local for `restore-item`, and `os.RemoveAll` on local or `e.Client.PurgeDir` on remote for `accept-delete`.
  - Added unit test `TestExecutor_Directory` in `internal/engine/review/executor_test.go`.
  - Verified with `go test -count=1 ./...` and `./scripts/check.sh all` passing all 7 gates.



