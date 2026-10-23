# Implementation Plan: Sync Service Modularization

## Overview

Decompose the monolithic `SyncService` God Object (~1,089 LOC across `internal/services/sync_service.go` and `internal/services/telemetry.go`) into focused, single-responsibility domain services:

1. `TelemetryService`: Daemon health polling, Core stats caching, metrics JSONL reading, and secret-redacted runtime config generation.
2. `CanaryService`: Provisioning (`CreateCanary`), presence checking (`CanaryExists`), and deletion safeguarding (`SafeguardCanary`) for `RCLONE_TEST`.
3. `DeletionService`: Reverse log scanning for `TOO_MANY_DELETES` aborts and fallback reading of structured pending deletions JSON.
4. `ExplorerService`: Dual-pane directory listing, path traversal jail enforcement (`filepath.Rel`), file/directory copying, and safe deletion.
5. `SyncService`: Focused on bisync systemd unit actions (`start_sync`, `stop_sync`, `force_resync`) and POSIX flock coordination, with backward-compatible delegators.

This unblocks **Interactive Sync Review** (`docs/design/interactive-sync-review.md`) and **Profile Manager** (`docs/design/profile-manager.md`), ensuring file mutations and deletion parsing are accessible via clean domain boundaries.


## Domain Architecture & Service Boundaries

```
internal/services/
├── canary_service.go         <-- Canary provisioning, presence checks & delete safeguards
├── canary_service_test.go
├── deletion_service.go       <-- Daemon log reverse scanner & pending deletion reader
├── deletion_service_test.go
├── explorer_service.go       <-- Path jail validation, browse, copy & delete
├── explorer_service_test.go
├── telemetry_service.go      <-- Daemon health, core stats, JSONL metrics & runtime config
├── telemetry_service_test.go
├── sync_service.go           <-- Systemd unit action lifecycle & backward compat facade
├── sync_service_test.go
├── log_service.go            <-- Sliced daemon log parser & nested JSON unpacker
└── log_service_test.go
```

---

## Service Specifications

### 1. CanaryService (`internal/services/canary_service.go`)

- `CanaryExists(ctx context.Context, remote, path string) bool`
- `CreateCanary(ctx context.Context, remote, path string) (map[string]interface{}, error)`
- `SafeguardCanary(cleanPath string) error` (blocks deletion of `RCLONE_TEST`)

### 2. DeletionService (`internal/services/deletion_service.go`)

- `ParsePendingDeletionsFromLog() state.PendingDeletionsData`
- `GetPendingDeletions() state.PendingDeletionsData`


### 3. TelemetryService (`internal/services/telemetry_service.go`)

- `GetDaemonHealth(ctx context.Context) map[string]interface{}`
- `GetRemotesSummary(ctx context.Context) (map[string]interface{}, error)`
- `GetTransferStats(ctx context.Context) *models.TransferStats`
- `GetCachedMetrics(linesCount int) []map[string]interface{}`
- `GetRuntimeConfig() map[string]interface{}`
- `GetTelemetryPayload(ctx context.Context, isSyncing bool, limit int) map[string]interface{}`

### 4. ExplorerService (`internal/services/explorer_service.go`)

- `BrowsePath(ctx context.Context, remote, path string) client.DirectoryListing`
- `CopyItem(ctx context.Context, srcRemote, srcPath, dstRemote, dstPath string, isDir bool) (map[string]interface{}, error)`
- `DeleteItem(ctx context.Context, remote, path string, isDir bool) (map[string]interface{}, error)`

### 5. SyncService (`internal/services/sync_service.go`)

- `IsSyncing() bool`
- `HandleAction(ctx context.Context, action string) (map[string]interface{}, error)`
- Composes and delegates to `Canary`, `Deletion`, `Explorer`, `Telemetry` for full backward compatibility.

---

## Testing Strategy

- Standard Go `testing` package with Arrange-Act-Assert structure.
- All RC API calls mocked using `httptest.NewServer`.
- Quality gate: `./scripts/check.sh` ensuring statement coverage >80% across all packages.
