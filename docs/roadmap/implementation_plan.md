# Implementation Plan: Resolution Executor Directory Handling in Review Sessions

## Proposed Changes

Fix the HTTP 500 error encountered when restoring or accepting deletions on review sessions containing directory candidates (such as `rev-0b6b9ffb861c7e84`).

### Root Cause
During sync conflict reviews (e.g. `TOO_MANY_DELETES`), deletion candidates can represent directories (`cand.IsDir == true`).
When executing resolutions:
1. `DecisionRestoreItem` called `e.Client.CopyFile` (`operations/copyfile`) on directories. Rclone's RC server rejected this with `HTTP 500: {"error": "is a directory not a file", "status": 500}`.
2. `LocalBackup` attempted to create a file hardlink and copy data on directories using `os.OpenFile(..., O_WRONLY)`, which fails with `EISDIR`.
3. `DecisionAcceptDelete` used `os.Remove` which fails on non-empty directories, and `DeleteFile` (`operations/deletefile`) on remote directories which fails in Rclone RC.

### Solution Architecture
1. **Core Client Enhancement (`internal/core/client/fs_ops.go`)**:
   - Add `Mkdir(ctx context.Context, fs, remote string) (bool, error)` calling Rclone RC `/operations/mkdir`.
2. **Review Engine RemoteClient Interface (`internal/engine/review/backup.go`)**:
   - Add `Mkdir(ctx context.Context, fs, remote string) (bool, error)` and `PurgeDir(ctx context.Context, fs, remote string) (bool, error)` to `RemoteClient`.
3. **Local Directory Backup & Restore (`internal/engine/review/backup.go`)**:
   - In `LocalBackup`, check `fi.IsDir()`: create backup directory via `os.MkdirAll(destPath, 0700)` and return without attempting file copy/hardlinks.
   - In `RestoreLocalBackup`, check `fi.IsDir()`: recreate directory via `os.MkdirAll(targetPath, 0755)`.
4. **Directory-Aware Resolution Execution (`internal/engine/review/executor.go`)**:
   - For `DecisionRestoreItem`:
     - If directory: call `e.Client.Mkdir` on remote or `os.MkdirAll` on local.
     - If file: call `e.Client.CopyFile`.
   - For `DecisionAcceptDelete`:
     - If directory: call `os.RemoveAll` on local or `e.Client.PurgeDir` on remote.
     - If file: call `os.Remove` on local or `e.Client.DeleteFile` on remote.

### Affected Components & Packages

- `internal/core/client/fs_ops.go`: Add `Mkdir` method.
- `internal/engine/review/backup.go`: Expand `RemoteClient` interface and handle directory in `LocalBackup`/`RestoreLocalBackup`.
- `internal/engine/review/executor.go`: Branch on `isDir` in `executeDeletion`.
- `internal/engine/review/executor_test.go`: Add unit tests for directory restore and deletion decisions.
- `internal/services/review_service_test.go`: Update mock client to implement `Mkdir`.

### Safety & Error Handling

- Jail confinement: All paths remain strictly verified via `ResolveSafePath` before any filesystem operations.
- Canary files: Canary files (`RCLONE_TEST`) remain forbidden from modification or deletion.

## Verification Plan

- `go test -v -count=1 ./internal/engine/review/... -run "TestExecutor_Directory"`
- `go test -v -count=1 ./internal/core/client/... -run "TestClientMkdir"`
- `./scripts/check.sh all`


