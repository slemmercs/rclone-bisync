# Task Plan: Type Safety, Quality Hardening & Automated License Audit

## Goal
Implement remaining technical debt items: enforce type safety for `DeleteItem` and `CreateCanary`, harden `remoteauth` coverage (>80%), add visual caller origin badging in the Unified Ledger, modularize `remotes.js` and `app.js` (<400 LOC), and establish an automated License Compliance & Copyleft Guard.

## Tasks

- [x] Task 1: Type Safety & Contract Governance for Delete and Canary Services
  - **Scope**: `internal/models/api.go`, `internal/services/explorer_ops.go`, `internal/services/canary_service.go`, `internal/services/sync_service.go`, `internal/server/handlers_explorer.go`, `internal/server/server.go`
  - **Verification**: `go test -v ./internal/services/... ./internal/server/...`

- [x] Task 2: RemoteAuth Test Coverage Hardening (>80%)
  - **Scope**: `internal/remoteauth/service_test.go`
  - **Verification**: `go test -v -cover ./internal/remoteauth/...`

- [x] Task 3: Caller Origin Badging in Unified Ledger (CLI, CRON, UI, MCP)
  - **Scope**: `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`, `dashboard/frontend/js/utils.js`
  - **Verification**: `go test -v ./tests/e2e/...`

- [x] Task 4: Frontend Modularization & LOC Ceilings (remotes.js & app.js)
  - **Scope**: `dashboard/frontend/js/remotes.js`, `dashboard/frontend/js/remotes_modals.js`, `dashboard/frontend/index.html`, `dashboard/frontend/app.js`
  - **Verification**: `./scripts/check.sh loc 400` and `go test -v ./tests/e2e/...`

- [x] Task 5: Automated Quality Gate License Audit & THIRD_PARTY_LICENSES.md
  - **Scope**: `cmd/check/licenses.go`, `cmd/check/main.go`, `THIRD_PARTY_LICENSES.md`
  - **Verification**: `./scripts/check.sh licenses` and `./scripts/check.sh all`

## Status Updates

- 2026-11-01: Initialized task plan from implementation_plan.md on branch refactor/type-safety-and-license-guard.
- 2026-11-01: Completed Task 1: Standardized DeleteItem and CreateCanary to return typed models (models.DeleteResponse and models.CanaryResponse). Updated ExplorerService, CanaryService, SyncService, CLI canary provisioner, and test suite. Verified via unit and E2E tests.
- 2026-11-01: Completed Task 2: Hardened remoteauth package statement coverage to 92.0% (up from 79.3%), adding service_cache_test.go covering GetAuthStatus, caching, InvalidateAuthCache, DefaultCommandRunner, and nil client checks.
- 2026-11-01: Completed Task 3: Added caller origin badging in Unified Ledger (CLI, CRON, UI, MCP) in pattern_timeline_ledger.html and utils.js with getOriginBadge resolver. Verified via headless Chrome E2E tests.
- 2026-11-01: Completed Task 4: Modularized remotes.js (down to 203 LOC) by extracting remotes_modals.js (212 LOC) and cleaning up app.js (down to 351 LOC). Verified via check-loc and headless Chrome E2E smoke tests.
- 2026-11-01: Completed Task 5: Implemented native Go license compliance and copyleft guard in cmd/check/licenses.go, verified all 34 direct/indirect dependencies, and updated THIRD_PARTY_LICENSES.md. Verified via ./scripts/check.sh licenses and ./scripts/check.sh all (all 7 quality stages passed).









