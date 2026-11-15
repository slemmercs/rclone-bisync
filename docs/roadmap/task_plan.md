# Task Plan: Deduplication Edge-Cases, Smart Safety Recommendations & Staged Batch Review Engine

## Goal
Resolve Google Drive duplicate streaming collisions (HTTP 500), fix UI explorer navigation/selection desync, alert users to content divergence across same-name copies, and introduce a transactional Staged Batch Review & Resync Bridge.

## Tasks

- [x] Task 1: Backend Duplicate Stream Interception & Safe Error Mapping (`internal/models/`, `internal/services/`, `internal/server/`)
  - **Scope**: `internal/models/api_explorer.go`, `internal/services/explorer_preview.go`, `internal/services/explorer_download.go`, `internal/server/handlers_preview.go`, `internal/services/explorer_preview_test.go`
  - **Verification**: `go test -count=1 ./internal/services/... ./internal/server/... -run "Test.*Preview|Test.*Download"`

- [x] Task 2: Backend Dedupe Batch Execution Engine & Endpoints (`internal/models/`, `internal/services/`, `internal/server/`)
  - **Scope**: `internal/models/api_dedupe.go`, `internal/services/dedupe_service.go`, `internal/server/handlers_dedupe.go`, `internal/server/routes.go`, tests
  - **Verification**: `go test -count=1 ./internal/services/... ./internal/server/... -run "Test.*DedupeBatch"`

- [x] Task 3: Frontend UID Disambiguation & Explorer Target Row Highlighting (`dashboard/frontend/`)
  - **Scope**: `dashboard/frontend/js/explorer.js`, `dashboard/frontend/js/explorer_compare.js`, `dashboard/frontend/partials/view_explorer.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check-loc.sh 400`

- [x] Task 4: Dedupe Directory Explorer Resilience & Content Divergence Detection (`dashboard/frontend/`)
  - **Scope**: `dashboard/frontend/js/dedupe_workspace.js`, `dashboard/frontend/partials/modals/modal_dedupe.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check-loc.sh 400`

- [x] Task 5: Staged Batch Review UI & Post-Dedupe Resync Bridge (`dashboard/frontend/`)
  - **Scope**: `dashboard/frontend/js/dedupe_workspace.js`, `dashboard/frontend/partials/modals/modal_dedupe.html`, `dashboard/frontend/partials/modals/modal_preview.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check-loc.sh 400`

- [ ] Task 6: Local Sandbox Fixtures & Full Quality Pre-Flight (`scripts/`, `tests/`)
  - **Scope**: `scripts/run-sandbox.sh`, `tests/e2e/dashboard_smoke_test.go`, full gate
  - **Verification**: `./scripts/check.sh all && ./scripts/check-loc.sh 400`

## Status Updates

- 2026-11-15: Initialized plan from approved `implementation_plan.md` on branch `fix/dedupe-workspace-redesign-and-link-edge-case`.

- 2026-11-15: Completed Task 1 - Backend duplicate stream interception and safe error mapping implemented. Intercepted Google Drive duplicate collisions ("found more than one file" / "duplicate object found") in `ExplorerService.PreviewItem` returning 200 with `IsDuplicateCollision: true` and actionable deduplication instructions instead of 500 error; mapped duplicate download attempts in `ExplorerService.DownloadStream` and `handleExplorerDownload` to 409 Conflict. Verified with targeted unit and HTTP handler tests.
- 2026-11-15: Completed Task 2 - Backend Dedupe Batch Execution Engine & Endpoints implemented. Created `internal/models/api_dedupe.go`, `internal/services/dedupe_service.go`, and `POST /api/remotes/dedupe/batch` in `handlers_dedupe.go` and `routes.go`. Enabled transactional staging resolutions (keep, rename, delete) with canary guard, jail bounds, immutable audit ledger event recording, and optional post-dedupe resync trigger. Verified with unit and HTTP handler tests.
- 2026-11-15: Completed Task 3 - Frontend UID Disambiguation & Explorer Target Row Highlighting implemented. Added composite key generation (`getItemKey`, `getItemRowKey`), multi-selection disambiguation, `isDuplicateItem` detection and badges with click-to-resolve, context menu integration, and target row UID pulse highlighting (`this.explorer.highlightId`). Verified with template check, E2E headless chrome smoke test, and LOC inspection.

- 2026-11-15: Completed Task 4 - Dedupe Directory Explorer Resilience & Content Divergence Detection implemented. Added `switchDedupeTab`, `hasContentDivergence`, `getDedupeBreadcrumbs`, parent directory row navigation, and error retry state handling. Verified with templates check, headless Chrome smoke tests, and LOC audit.
- 2026-11-15: Completed Task 5 - Staged Batch Review UI & Post-Dedupe Resync Bridge implemented. Added `DedupeBatchModule` (`dashboard/frontend/js/dedupe_batch.js`), candidate card inline staging actions (Keep, Rename, Delete), Staged Actions Review Bar with resync queuing checkbox, and preview modal duplicate collision quick-link. Verified with templates check, headless Chrome smoke tests, and LOC audit.











