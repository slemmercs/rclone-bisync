# Implementation Plan: Repository Quality Hardening, Security Jails & Modularization (Maintenance Milestone)

## Overview
Execute a comprehensive repository maintenance and quality hardening milestone that resolves all High-, Medium-, and Low-severity findings identified in the architectural code review (`docs/architecture/code-review.md`) and decomposes monolithic packages (`internal/server/` and `internal/core/client/`) into cohesive domain components. This directly aligns the codebase with Bisync Hub's core principles in `README.md`—**"Visibility Before Action"**, **"Safety Before Automation"**, and **"Recovery Before Rebuild"**—providing the rock-solid, modular foundation required to implement Bisync Hub's unique selling point: the **Interactive Sync Review & Resolution Engine ("Review. Decide. Sync.")**.

---

## User Review Required

> [!IMPORTANT]
> This maintenance sprint strengthens runtime safety, eliminates race conditions on long-running synchronization locks, blocks directory deletions targeting safety canaries, expands homelab LAN origin support, and cleanly breaks up monolithic files (`client.go` and `server.go`). No public API routes or CLI interfaces are broken.

---

## Proposed Changes

### 1. High-Severity Security & Concurrency Fixes (H-1, H-2, H-3)
- **`internal/core/lock/lock.go` (H-1)**:
  - Update `handleStaleLock()` so that when a lockfile's age exceeds `LockStaleTimeoutSeconds` (3600s), it checks if the PID is still alive (`CheckPIDExists(oldPID)`). If the process is alive, the lock must **never** be removed or stolen.
- **`internal/services/explorer_service.go` (H-2, H-3)**:
  - In `IsLocalPathJailed()`, evaluate real target paths using `filepath.EvalSymlinks()` to prevent symlink traversal outside the sync boundaries.
  - In `DeleteItem()`, verify if a directory targeted for deletion (`RemoveAll` or `PurgeDir`) contains `RCLONE_TEST` and block deletion if present.

### 2. Medium & Low-Severity Hardening (M-2, M-3, M-4, L-1, L-2)
- **`internal/server/server.go` (M-2)**:
  - Expand `isAllowedOrigin()` to permit private RFC 1918 subnets (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`), `.local` mDNS domains, and the configured host.
- **`main.go` (M-3)**:
  - Add graceful HTTP server drain on `SIGINT`/`SIGTERM` via `httpServer.Shutdown(ctx)` with a 5-second graceful shutdown timeout.
- **`internal/services/telemetry_service.go` (M-4)**:
  - Replace redundant custom sanitization methods (`toUpperSnakeCase`, `isKeySensitive`) in `GetRuntimeConfig()` with direct delegation to `security.RedactSecrets(t.Config)`.
- **`internal/config/config.go` (L-1)**:
  - Add `ParseEnvFile(path string) (map[string]string, error)` to safely parse `.env` files into maps without globally mutating `os.Environ` via `os.Setenv`.
- **`internal/services/sync_service.go` (L-2)**:
  - Standardize and document service facade delegator methods.

### 3. Monolithic Package Modularization & Schema Contract Hardening (M-1)
- **`internal/core/client/client.go` (Decomposition)**:
  - Split `client.go` (551 LOC) into:
    - `client.go`: HTTP client pooling, RcloneClient struct, constructor, POST helper, health check (~120 LOC).
    - `jobs.go`: Rclone async job triggers and polling (`TriggerBisync`, `GetJobStatus`, `StopJob`, `ListJobs`).
    - `remotes.go`: Remote configs and OAuth lifecycle (`ListRemotes`, `GetRemoteConfig`, `GetRemoteAbout`, `UpdateRemoteConfig`, `OAuthStatus`, `OAuthStop`).
    - `fs_ops.go`: Directory listings and file operations (`ListDirectory`, `CopyFile`, `CopyDir`, `DeleteFile`, `PurgeDir`, `TouchFile`).
- **`internal/server/server.go` (Decomposition & Typed Contracts M-1)**:
  - Define concrete response structs in `internal/models/api.go` (`ActionResponse`, `RemotesResponse`, `RemoteSummaryItem`, `DaemonHealthResponse`, `GenericSuccessResponse`).
  - Split `server.go` (523 LOC) into:
    - `server.go`: Server struct, constructor, routes registration, writeJSON, origin checking (~130 LOC).
    - `handlers_telemetry.go`: `/api/health`, `/api/telemetry`, `/api/metrics`, `/api/logs/daemon`.
    - `handlers_explorer.go`: `/api/browse`, `/api/canary`, `/api/copy`, `/api/delete`.
    - `handlers_auth.go`: `/api/remotes`, `/api/remotes/{name}/reconnect`, `/api/remotes/{name}/credentials`, `/api/remotes/auth`, etc.
    - `handlers_sync.go`: `/api/action`, `/api/pending-deletions`.

---

## Verification Plan

### Automated Tests
1. `go test -count=1 ./internal/core/lock -run TestLockManager_ActivePIDPastTimeoutNotStale`
2. `go test -count=1 ./internal/services -run TestExplorerService_SymlinkJailEscape`
3. `go test -count=1 ./internal/services -run TestExplorerService_CanaryInDirProtected`
4. `go test -count=1 ./internal/server -run TestAllowedOrigins_PrivateLAN`
5. `go test -count=1 ./internal/config -run TestParseEnvFile`
6. `go test -count=1 ./...`
7. `./scripts/check.sh`


