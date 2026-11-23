# Implementation Plan: Default Exclusion and Remote Pruning for Review Backups

## Proposed Changes

Bisync Hub uses an interactive Write-Ahead Log (WAL) review engine that stages pre-mutation backups. For remote file mutations, it creates a zero-bandwidth server-side copy on the cloud remote under `.review_backups/<session_id>/`.
However, because `.review_backups/**` was not automatically injected as an exclusion pattern in the bisync argument builder, `rclone bisync` treats this internal staging folder as user data and attempts to synchronize it back and forth between the local filesystem and remote endpoints. Additionally, while local backups are pruned by TTL, remote backups currently have no pruning mechanism and linger on cloud remotes indefinitely.

This plan addresses both issues:
1. Injects `--exclude ".review_backups/**"` into the profile command builder and flag compiler by default so bidirectional synchronization completely ignores review backup directories.
2. Introduces remote review backup pruning (`PruneExpiredRemoteBackups`) into `internal/engine/review/backup.go` and hooks it into `ReviewService.PruneBackups` to garbage-collect expired session backups from the remote storage target.

### Affected Components & Packages

- `internal/profiles/builder.go`: Add default `--exclude ".review_backups/**"` rule in `BuildCLIArgs` and `BuildBisyncArgs` if not already present.
- `internal/profiles/builder_test.go`: Update argument builder unit tests to reflect the default exclusion rule.
- `internal/engine/review/backup.go`: Add `RemoteListingClient` interface or extend `RemoteClient` to support listing and purging remote backup sessions, and add `PruneExpiredRemoteBackups`.
- `internal/engine/review/backup_test.go`: Add unit tests for `PruneExpiredRemoteBackups`.
- `internal/services/review_execution.go`: Update `ReviewService.PruneBackups` to invoke remote pruning when a remote client and remote path are configured.

### Technical Strategy & Data Flow

1. **Default Exclusion**:
   - In `internal/profiles/builder.go`, before custom excludes or file filters are added, ensure that `".review_backups/**"` is excluded.
   - Guard against duplicate `--exclude ".review_backups/**"` if the user already specified it in `ExcludePatterns`.

2. **Remote Pruning Lifecycle**:
   - `ReviewService.PruneBackups` runs periodically or on demand.
   - For remote paths, it inspects `.review_backups/` via `client.ListDirectory(ctx, remoteFs, ".review_backups")`.
   - Any session directory whose modification timestamp is older than `ReviewBackupTTLHours` (or matching archived/dismissed sessions) is purged via `client.PurgeDir(ctx, remoteFs, ".review_backups/" + dirName)`.

### Safety & Error Handling

- **Canary Protection**: Remote purge operations must never touch or traverse paths containing `RCLONE_TEST`.
- **Path Confinement**: Target paths for remote pruning are strictly restricted to `.review_backups/<session_id>` without relative path traversals (`..`).
- **Graceful Failure**: If a remote backend is offline or inaccessible during pruning, remote pruning failures must not halt local pruning or trigger a fatal error.

## Verification Plan

- Unit tests in `internal/profiles/builder_test.go` verifying that `--exclude ".review_backups/**"` is always present in generated bisync commands.
- Unit tests in `internal/engine/review/backup_test.go` verifying remote backup pruning logic with mock remote clients.
- Integration verification via `go test -v -count=1 ./internal/profiles/... ./internal/engine/review/... ./internal/services/...`.
