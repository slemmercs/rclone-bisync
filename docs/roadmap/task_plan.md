
# Task Plan: Sync Service Modularization & Code Cleanup

## Goal
Decompose the monolithic `SyncService` God Object (~1,089 LOC across `internal/services/sync_service.go` and `internal/services/telemetry.go`) into focused, single-responsibility domain services (`TelemetryService`, `CanaryService`, `DeletionService`, `ExplorerService`, and `SyncService`), maintaining strict path jail safety, backward compatibility, and statement coverage >80%.

## Tasks
- [x] Task 1: Extract CanaryService (`canary_service.go` & `canary_service_test.go`)
  - **Scope**: Extract canary creation (`CreateCanary`), canary presence checks (`CanaryExists`), and deletion safeguard (`SafeguardCanary`).
  - **Verification**: `go test -count=1 ./internal/services -run TestCanary`
- [ ] Task 2: Extract DeletionService (`deletion_service.go` & `deletion_service_test.go`)
  - **Scope**: Extract `ParsePendingDeletionsFromLog` and `GetPendingDeletions` with reverse seeking and regex abort parsing.
  - **Verification**: `go test -count=1 ./internal/services -run TestDeletion`
- [ ] Task 3: Extract TelemetryService (`telemetry_service.go` & `telemetry_service_test.go`)
  - **Scope**: Decouple `telemetry.go` into `TelemetryService`, covering daemon health caching, transfer stats, JSONL metrics cache, and runtime config.
  - **Verification**: `go test -count=1 ./internal/services -run TestTelemetry`
- [ ] Task 4: Extract ExplorerService (`explorer_service.go` & `explorer_service_test.go`)
  - **Scope**: Extract `BrowsePath`, `CopyItem`, `DeleteItem`, and `isLocalPathJailed` with strict `filepath.Rel` path traversal checks.
  - **Verification**: `go test -count=1 ./internal/services -run TestExplorer`
- [ ] Task 5: Slim Down SyncService & Server Wiring
  - **Scope**: Refactor `internal/services/sync_service.go` to focus exclusively on systemd actions and flock locking; wire decomposed services into `internal/server/server.go` and `main.go`.
  - **Verification**: `go test -count=1 ./internal/services ./internal/server`
- [ ] Task 6: Pre-flight Verification & Quality Gate
  - **Scope**: Full check suite (`./scripts/check.sh`), statement coverage check (>80%), binary build (`go build -o bin/rclone-bisync main.go`).
  - **Verification**: `./scripts/check.sh`

## Status Updates
- 2026-10-22: Initialized plan on branch `refactor/sync-service-modularization`.
- 2026-10-22: Completed Task 1 (Extracted `CanaryService` in `canary_service.go` with `CanaryExists`, `CreateCanary`, `SafeguardCanary`, wired into `SyncService`, and verified with `canary_service_test.go`).










