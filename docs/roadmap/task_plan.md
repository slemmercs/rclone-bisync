# Task Plan: Dual-Pane Explorer Stability, Ad-Hoc File Comparison, Paired Dedupe & Unified Ledger Streamlining

## Goal
Resolve explorer navigation reset, add multi-file ad-hoc hash comparison with cross-backend negotiation, support paired dual-pass dedupe with folder browsing and post-dedupe resync, streamline the Unified Log Ledger, enable true log dock pagination, eliminate breadcrumb scrollbars, and expand widescreen responsiveness.

## Tasks

- [x] Task 1: Backend Ad-Hoc File Checksum Comparison (`internal/models/`, `internal/services/`, `internal/server/`)
  - **Scope**: `internal/models/api.go`, `internal/services/explorer_service.go`, `internal/services/explorer_service_test.go`, `internal/server/routes.go`, `internal/server/handlers_explorer.go`, `internal/server/handlers_explorer_test.go`
  - **Action**: Define compare models, implement `CompareItems` in `ExplorerService` with hash negotiation and equivalence clustering, register `POST /api/browse/compare`, and add comprehensive unit tests.
  - **Verification**: `go test -count=1 -v ./internal/services -run TestCompareItems && go test -count=1 -v ./internal/server -run TestHandleBrowseCompare`

- [x] Task 2: Paired Deduplication Engine & Suppression of Dry-Run Noise (`internal/server/handlers_auth.go`)
  - **Scope**: `internal/server/handlers_auth.go`, `internal/server/handlers_audit_test.go`
  - **Action**: Support `dedupe_both` for coordinated remote + local dedupe with jail confinement, and suppress stdout spam to `bisync.log` during dry-runs (`mode == "list"`).
  - **Verification**: `go test -count=1 -v ./internal/server -run TestHandleRemoteDedupe`

- [x] Task 3: Server-Side Log Type Filtering & Dense Pagination (`internal/server/`, `internal/services/`)
  - **Scope**: `internal/server/handlers_telemetry.go`, `internal/services/tailer.go`, `internal/server/handlers_telemetry_test.go`
  - **Action**: Add `exclude_types` query parameter to `/api/logs/bisync` to filter entries before pagination calculation.
  - **Verification**: `go test -count=1 -v ./internal/server -run TestHandleBisyncLogs`

- [x] Task 4: Explorer Navigation Stability & Middle-Truncated Breadcrumbs (`dashboard/frontend/`)
  - **Scope**: `dashboard/frontend/js/profiles.js`, `dashboard/frontend/js/explorer.js`, `dashboard/frontend/partials/view_explorer.html`
  - **Action**: Fix `fetchProfiles` background path clobbering; implement middle-truncated breadcrumbs without scrollbars.
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400`

- [x] Task 5: Frontend Multi-Select & Checksum Comparison Modal (`dashboard/frontend/`)
  - **Scope**: `dashboard/frontend/js/explorer_compare.js`, `dashboard/frontend/partials/modals/modal_compare.html`, `dashboard/frontend/partials/components/component_explorer_toolbar.html`, `dashboard/frontend/partials/view_explorer.html`, `dashboard/frontend/index.html`
  - **Action**: Implement multi-file selection state, compare modal with hash clustering, and register module in `index.html`.
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400`

- [ ] Task 6: Dedupe Modal Enhancements & Post-Dedupe Resync CTA (`dashboard/frontend/`)
  - **Scope**: `dashboard/frontend/partials/modals/modal_dedupe.html`, `dashboard/frontend/js/remotes_modals.js`
  - **Action**: Add folder picker, duplicate collision jump-to-file buttons, paired dedupe toggle (default on), and post-dedupe 1-click resync banner with mirroring advice.
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400`

- [ ] Task 7: Unified Log Ledger Streamlining & Log Dock Server-Side Pagination (`dashboard/frontend/`)
  - **Scope**: `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`, `dashboard/frontend/js/logs.js`, `dashboard/frontend/js/log_dock.js`
  - **Action**: Rename to "Unified Log Ledger", exclude `sync_file` records, add 1-row-per-job clustering, and wire true server-side pagination to `log_dock.js`.
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400`

- [ ] Task 8: Layout Expansion & Full Pre-Commit Verification (`dashboard/frontend/`, Repository-Wide)
  - **Scope**: `dashboard/frontend/index.html`, full test suite
  - **Action**: Expand widescreen fluid layout (`max-w-[1720px] 2xl:max-w-[1920px]`) and run all quality stages.
  - **Verification**: `./scripts/check.sh all`

## Status Updates
- 2026-11-14: Initialized task plan based on approved implementation plan.
- 2026-11-14: Completed Task 1 - Implemented CompareItems in ExplorerService with cross-backend hash negotiation and equivalence clustering (N >= 2), registered POST /api/browse/compare endpoint, and verified with unit tests.

- 2026-11-14: Completed Task 2 - Supported dedupe_both coordinated paired deduplication with jail confinement, suppressed stdout noise in bisync.log during dry-runs (mode == "list"), and verified with unit tests.
- 2026-11-14: Completed Task 3 - Implemented server-side log type filtering (exclude_types, type, level) with pre-pagination filtering in services.TailJSONLogPaginatedFiltered, ensuring dense pages without gaps. Verified with unit tests.
- 2026-11-14: Completed Task 4 - Prevented background fetchProfiles() path clobbering by checking uninitialized state, and implemented middle-collapsed breadcrumb navigation (root / … / parent / current) with tooltip previews and zero scrollbars. Verified with template validator, LOC check, and headless Chrome smoke tests.

- 2026-11-14: Completed Task 5 - Built Explorer multi-select checkboxes across both panes, Compare action toolbar controls, and modal_compare.html with hash equivalence clustering, verdict banners, and copyable hashes. Verified with template validator, LOC check, and headless Chrome smoke tests.



