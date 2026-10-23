
# Task Plan: Repository Quality Hardening, Security Jails & Modularization (Maintenance Milestone)

## Goal
Execute a comprehensive repository maintenance and quality hardening milestone that resolves all High-, Medium-, and Low-severity findings identified in the architectural code review (`docs/architecture/code-review.md`) and decomposes monolithic packages (`internal/server/` and `internal/core/client/`) into cohesive domain components. This aligns the codebase with Bisync Hub's core principles in `README.md`—**"Visibility Before Action"**, **"Safety Before Automation"**, and **"Recovery Before Rebuild"**—providing the robust, modular foundation needed for Bisync Hub's main differentiator: the **Interactive Sync Review & Resolution Engine ("Review. Decide. Sync.")**.

## Tasks
- [x] Task 1: Stale Lock Race Condition Remediation (Finding H-1)
  - **Scope**: Update `handleStaleLock()` in `internal/core/lock/lock.go` so locks held by an active PID past `LockStaleTimeoutSeconds` (3600s) are never unlinked or stolen; add unit tests in `internal/core/lock/lock_test.go`.
  - **Verification**: `go test -count=1 ./internal/core/lock`
- [x] Task 2: Path Confinement Symlink Resolution & Recursive Canary Safeguard (Findings H-2, H-3)
  - **Scope**: Update `IsLocalPathJailed()` in `internal/services/explorer_service.go` to evaluate real filesystem targets using `filepath.EvalSymlinks()`; update `DeleteItem()` to check if directories contain `RCLONE_TEST` before deletion. Add unit tests in `internal/services/explorer_service_test.go`.
  - **Verification**: `go test -count=1 ./internal/services -run TestExplorer`
- [ ] Task 3: Homelab LAN Access & Graceful Server Drain (Findings M-2, M-3)
  - **Scope**: Update `isAllowedOrigin()` in `internal/server/server.go` to permit private RFC 1918 subnets (`192.168.x.x`, `10.x.x.x`, `172.16.x.x`), `.local` domains, and the configured host. Add graceful `httpServer.Shutdown(ctx)` on `SIGINT`/`SIGTERM` in `main.go`.
  - **Verification**: `go test -count=1 ./internal/server` && `go test -count=1 .`
- [ ] Task 4: Duplicate Secret Sanitization Cleanup & Safe Env Parser (Findings M-4, L-1)
  - **Scope**: Streamline `GetRuntimeConfig()` in `internal/services/telemetry_service.go` to delegate directly to `security.RedactSecrets()`; add non-mutating `ParseEnvFile()` in `internal/config/config.go`.
  - **Verification**: `go test -count=1 ./internal/services ./internal/config`
- [ ] Task 5: Core Client Modularization
  - **Scope**: Decompose `internal/core/client/client.go` (551 LOC) into cohesive domain components: `jobs.go` (job tracking), `remotes.go` (remote configurations & OAuth), and `fs_ops.go` (directory listings & file operations).
  - **Verification**: `go test -count=1 ./internal/core/client`
- [ ] Task 6: Server Route Modularization & Schema Contract Hardening (Finding M-1)
  - **Scope**: Define concrete DTOs in `internal/models/api.go` for `/api/action`, `/api/health`, `/api/remotes`; decompose `internal/server/server.go` (523 LOC) into domain handlers: `handlers_telemetry.go`, `handlers_explorer.go`, `handlers_auth.go`, and `handlers_sync.go`.
  - **Verification**: `go test -count=1 ./internal/server ./internal/models`
- [ ] Task 7: Full Quality Gate & Pre-Flight Verification
  - **Scope**: Run full `./scripts/check.sh` quality gate, verify statement test coverage >= 85%, verify cross-platform builds (`linux/amd64`, `linux/arm64`), and commit cleanly.
  - **Verification**: `./scripts/check.sh`

## Next Up
- **Phase 4: Interactive Sync Review & Deletion Diff Visualizer** (Tasks 1–7 in `docs/design/interactive-sync-review.md`).

## Status Updates
- 2026-10-23: Phase 3 (OAuth Reconnect, SyncService modularization PR #18, Rebrand to Bisync Hub PR #20) completed and merged into `main`.
- 2026-10-23: Initialized Maintenance Milestone to address all architectural audit findings and complete core modularization prior to Phase 4.
- 2026-10-23: Completed Task 1 (Finding H-1): Protected locks held by active PIDs from eviction or unlinking regardless of elapsed age; verified via `go test -count=1 ./internal/core/lock`.
- 2026-10-23: Completed Task 2 (Findings H-2, H-3): Implemented symlink realpath confinement via `filepath.EvalSymlinks` and recursive directory deletion canary protection in `internal/services/`; verified via `go test -count=1 ./internal/services -run TestExplorer` and `./scripts/check.sh`.













