# Implementation Plan: Hasher Backend Integration (Checksum Acceleration & Emulation)

## Proposed Changes

Implement comprehensive Hasher backend support across Bisync Hub's backend services, Cockpit UI, and MCP agent suite to enable high-performance checksum caching, emulated hash generation for non-checksum remotes, and instant file comparisons in the explorer.

By wrapping remotes or local filesystem paths in Rclone's overlay `hasher` backend, Bisync Hub unlocks `--compare size,modtime,checksum` synchronization for backends without native hash support (SFTP, WebDAV, SMB, local non-hash filesystems), accelerates dual-pane Explorer file equivalence verification via cached BoltDB lookups, and supports cache pre-seeding via Rclone `backend/command` (`import` / `stickyimport`).

### Affected Components & Packages

- `internal/core/client/`:
  - `backend.go`: New file with `BackendCommand(ctx, fs, command, args, opt)` calling Rclone RC `POST /backend/command`.
  - `backend_test.go`: Unit tests for `BackendCommand`.
- `internal/models/`:
  - `hasher.go`: DTO definitions for Hasher wrap request, command request, and command response.
- `internal/services/`:
  - `remote_wizard_service.go`: Update `CategorizeProvider` to include `"hasher"` under `models.CategoryVirtual`.
  - `hasher_service.go`: New service handling remote wrapping (`WrapRemoteInHasher`) and command dispatch (`ExecuteHasherCommand`).
  - `hasher_service_test.go`: Unit tests for `HasherService`.
- `internal/app/`:
  - `container.go`: Register and initialize `HasherService`.
- `internal/server/`:
  - `handlers_hasher.go`: Route handlers for `/api/remotes/hasher/wrap` and `/api/remotes/hasher/command`.
  - `handlers_hasher_test.go`: Unit tests for Hasher endpoints.
  - `routes.go`: Register Hasher endpoints on HTTP mux.
- `internal/mcp/`:
  - `tools_remote.go`: Add `bisync_hasher_command` tool.
  - `tools_remote_test.go`: Unit tests for `bisync_hasher_command`.
- `deploy/`:
  - `install.sh`: Ensure `$REAL_HOME/.cache/rclone` persistent cache directory exists.
  - `systemd/rclone-daemon.service`: Verify persistent `--cache-dir %h/.cache/rclone`.
  - `bisync.env.example`: Document `RCLONE_CACHE_DIR` persistence notes.
- `dashboard/frontend/src/`:
  - `types/hasher.ts`: TypeScript contracts for Hasher wrapping and backend commands.
  - `components/ModalHasherManager.vue`: Cockpit modal for managing Hasher cache (dump, drop, pre-seed import).
  - `views/ViewRemotes.vue`: Add "Wrap in Hasher" quick action button on remotes lacking native checksums, and "Hasher Cache" management button on remotes of type `hasher`.
  - `components/ModalRemoteWizard.vue`: Enhance wizard UX when configuring `hasher` remotes.

### Technical Strategy & Data Flow

1. **RC Backend Commands**:
   - `RcloneClient.BackendCommand` posts to `POST /backend/command` with `{ "command": command, "fs": fs, "arg": args, "opt": opt }`.
   - Returns unmarshaled `result` payload or an error.
2. **Wrapping Existing Remotes**:
   - Operator or agent invokes `/api/remotes/hasher/wrap` or `bisync_create_remote` with `type: "hasher"`.
   - `HasherService.WrapRemoteInHasher` validates base remote existence via `ListRemotes`, sets parameters (`remote: <base>`, `hashes: <hashes>`, `max_age: <max_age>`), and creates the remote via `RemoteWizardService.CreateRemote`.
3. **Hasher Cache Maintenance & Pre-Seeding**:
   - Operators can execute `dump` (view cached hashes count / entries), `drop` (flush cache), or `import`/`stickyimport` (warm cache from sum files).
   - Executed through `HasherService.ExecuteHasherCommand` and exposed via `/api/remotes/hasher/command` and `bisync_hasher_command`.
4. **Persistent Cache Directory**:
   - BoltDB databases are placed by Rclone under `$RCLONE_CACHE_DIR/kv/<BaseRemote>~hasher.bolt`.
   - `install.sh` ensures `~/.cache/rclone` is created and owned by `$REAL_USER`.

### Safety & Error Handling

- **Confirmation Safeguard**: Dropping the Hasher cache requires explicit confirmation parameter to prevent accidental cache invalidation.
- **Remote Type Validation**: Commands to `/api/remotes/hasher/command` verify that the target remote is indeed of type `hasher`.
- **Read-Only MCP Mode**: In MCP read-only mode, destructive commands (`drop`, `import`, `stickyimport`) are rejected.

## Verification Plan

- Core RC client tests: `go test -count=1 ./internal/core/client -run TestBackendCommand`
- Service tests: `go test -count=1 ./internal/services -run TestHasher`
- Server route tests: `go test -count=1 ./internal/server -run TestHasher`
- MCP tool tests: `go test -count=1 ./internal/mcp -run TestHasher`
- Frontend type checking and build: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`
- Full pre-flight quality check: `./scripts/check.sh all`





