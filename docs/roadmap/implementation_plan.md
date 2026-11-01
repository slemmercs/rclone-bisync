# Implementation Plan: Type Safety, Quality Hardening & Automated License Audit

## Proposed Changes

Address remaining technical debt items before the MCP server integration sprint:
1. Enforce strict type contracts on service boundaries (`ExplorerService.DeleteItem` and `CanaryService.CreateCanary`) replacing loose `map[string]interface{}` with explicit DTOs in `internal/models`.
2. Strengthen test coverage for `internal/remoteauth` to push coverage well above the 80% threshold.
3. Enhance visual caller origin identification in the Unified Ledger (`CLI`, `CRON`, `UI`, `MCP`).
4. Modularize near-ceiling frontend scripts (`remotes.js` and `app.js`) to stay strictly below the 400 LOC ceiling.
5. Implement an automated License Compliance & Copyleft Guard in `cmd/check` that verifies licenses, prevents copyleft (GPL/LGPL/AGPL/SSPL) pollution, verifies parity with `THIRD_PARTY_LICENSES.md`, and supports automated generation/updating.

### Affected Components & Packages

- `internal/models/api.go`: Add `DeleteResponse` and `CanaryResponse` models.
- `internal/services/explorer_ops.go`: Update `DeleteItem` return signature to `(models.DeleteResponse, error)`.
- `internal/services/canary_service.go`: Update `CreateCanary` return signature to `(models.CanaryResponse, error)`.
- `internal/services/sync_service.go` & `internal/server/handlers_*.go`: Update callers to consume typed responses.
- `internal/remoteauth/service_test.go`: Add unit tests for `GetAuthStatus`, `GetAuthStatusCached`, and cache invalidation.
- `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`: Render visual badges for origin (`CLI`, `CRON`, `UI`, `MCP`).
- `dashboard/frontend/js/remotes.js` & `dashboard/frontend/js/remotes_modals.js`: Modularize modals out of `remotes.js`.
- `cmd/check/licenses.go` & `cmd/check/main.go`: Add `licenses` stage to Quality Gate engine.
- `THIRD_PARTY_LICENSES.md`: Update documentation to include all compiled Go backend dependencies.

### Technical Strategy & Data Flow

- Service methods return explicit Go structs defined in `internal/models`.
- HTTP handlers use `json.Marshal(typedResponse)` rather than arbitrary map literals.
- Quality Gate license auditor queries `go list -deps .` and inspects module cache files for copyleft indicators (`GPL`, `AGPL`, `LGPL`, `SSPL`, `EUPL`) and matches against allowlist (`MIT`, `Apache-2.0`, `BSD-2-Clause`, `BSD-3-Clause`, `ISC`, `CC0`, `Unlicense`).
- Modals logic extracted cleanly without changing Alpine.js reactive state scopes.

### Safety & Error Handling

- All path traversal checks and canary protections remain intact.
- Quality gate blocks any dependency introducing copyleft licensing or undocumented modules.

## Verification Plan

- `go test -v ./internal/services/... ./internal/server/...`
- `go test -v -cover ./internal/remoteauth/...`
- `./scripts/check.sh loc 400`
- `./scripts/check.sh licenses`
- `./scripts/check.sh all`






