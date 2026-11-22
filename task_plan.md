# Task Plan: Unified Virtual Overlays (Hasher & Crypt) & Dynamic Provider Form Engine

## Goal
Implement a unified virtual overlay and schema-driven remote configuration engine across Bisync Hub, combining Rclone Hasher checksum acceleration, client-side Crypt remote encryption, and a universal dynamic provider form engine.

## Tasks

- [x] Task 1: Merge Branches & Unify Implementation Plans
  - **Scope**: Merge `feat/hasher-backend-integration` into `feat/dynamic-provider-form`, resolve conflicts, unify `implementation_plan.md` and `task_plan.md`.
  - **Verification**: `go test -count=1 ./internal/...`

- [x] Task 2: Core RC Obscure Client & Crypt Models
  - **Scope**: `internal/core/client/remotes.go`, `internal/core/client/client_test.go`, `internal/models/crypt.go`, `internal/models/crypt_test.go`, `internal/models/provider.go`
  - **Verification**: `go test -count=1 ./internal/models/... ./internal/core/client/...`

- [x] Task 3: Hasher and Crypt Domain Services
  - **Scope**: `internal/services/hasher_service.go`, `internal/services/hasher_service_test.go`, `internal/services/crypt_service.go`, `internal/services/crypt_service_test.go`, `internal/services/remote_wizard_service.go`, `internal/app/container.go`
  - **Verification**: `go test -count=1 ./internal/services/...`

- [x] Task 4: Virtual Remotes HTTP Handlers & MCP Tools
  - **Scope**: `internal/server/handlers_virtual_remotes.go`, `internal/server/handlers_virtual_remotes_test.go`, `internal/server/routes.go`, `internal/mcp/tools_remote.go`, `internal/mcp/tools_remote_test.go`
  - **Verification**: `go test -count=1 ./internal/server/... ./internal/mcp/...`

- [x] Task 5: Cache Persistence Configuration in Deployment Scripts
  - **Scope**: `deploy/install.sh`, `deploy/systemd/rclone-daemon.service`, `bisync.env.example`
  - **Verification**: `bash -n deploy/install.sh`

- [x] Task 6: Frontend TypeScript Contracts & Pinia Store Actions
  - **Scope**: `dashboard/frontend/src/types/hasher.ts`, `dashboard/frontend/src/types/crypt.ts`, `dashboard/frontend/src/types/dedupe.ts`, `dashboard/frontend/src/stores/remotes.ts`, `dashboard/frontend/src/stores/dedupe.ts`
  - **Verification**: `npm --prefix dashboard/frontend run type-check`

- [ ] Task 7: Build DynamicProviderForm Component
  - **Scope**: `dashboard/frontend/src/components/DynamicProviderForm.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check`

- [ ] Task 8: Integrate DynamicProviderForm & Crypt Auto-Wrap in ModalRemoteWizard
  - **Scope**: `dashboard/frontend/src/components/ModalRemoteWizard.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check`

- [ ] Task 9: Hasher Manager & Crypt Wrap Modals and ViewRemotes Quick Actions
  - **Scope**: `dashboard/frontend/src/components/ModalHasherManager.vue`, `dashboard/frontend/src/components/ModalCryptWrap.vue`, `dashboard/frontend/src/views/ViewRemotes.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [ ] Task 10: Full Verification & Quality Gates
  - **Scope**: Run all 8 quality stages across backend and frontend
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-21: Completed Task 1. Merged `feat/hasher-backend-integration` into `feat/dynamic-provider-form`. Resolved plan conflicts by unifying implementation plan and task execution plan covering Hasher overlay, Crypt overlay, and Dynamic Provider form. Verified all internal Go tests pass with `go test -count=1 ./internal/...`.

- 2026-11-21: Completed Task 2. Implemented `RcloneClient.Obscure` method with unit tests in `internal/core/client/`, added `CryptWrapRequest` and `CryptWrapResponse` in `internal/models/crypt.go` with unit tests, and extended `CreateRemoteBridgeRequest` in `internal/models/provider.go`. Verified with `go test -count=1 ./internal/models/... ./internal/core/client/...`.
- 2026-11-21: Completed Task 3. Implemented `HasherService` and `CryptService` in `internal/services/` with unit tests, wired encrypted overlay support and `"hasher"` categorization into `RemoteWizardService`, and registered both services into `internal/app/container.go`. Verified with `go test -count=1 ./internal/services/... ./internal/app/...`.
- 2026-11-21: Completed Task 4. Implemented HTTP endpoints for `/api/remotes/hasher/wrap`, `/api/remotes/hasher/command`, and `/api/remotes/crypt/wrap` in `internal/server/handlers_virtual_remotes.go`. Implemented MCP tools `bisync_hasher_command` and `bisync_create_crypt_remote` in `internal/mcp/tools_remote.go`. Verified with `go test -count=1 ./internal/server/... ./internal/mcp/...`.
- 2026-11-21: Completed Task 5. Verified persistent cache directory creation in `deploy/install.sh`, verified `--cache-dir %h/.cache/rclone` in `deploy/systemd/rclone-daemon.service`, and documented `RCLONE_CACHE_DIR` in `config/bisync.env.example` and `bisync.env`. Verified with `bash -n deploy/install.sh`.

- 2026-11-21: Completed Task 6. Created `dashboard/frontend/src/types/hasher.ts` and `dashboard/frontend/src/types/crypt.ts`, re-exported in `dashboard/frontend/src/types/index.ts`, extended `dashboard/frontend/src/types/dedupe.ts` with `hasher_accelerated` and encrypted overlay fields, refreshed `dashboard/frontend/src/types/models.gen.ts`, and implemented `wrapHasher`, `executeHasherCommand`, and `wrapCrypt` in `dashboard/frontend/src/stores/remotes.ts`. Verified with `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`.



