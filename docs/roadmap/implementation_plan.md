# Implementation Plan: Dual-Pane Explorer Stability, Ad-Hoc File Comparison, Paired Dedupe & Unified Ledger Streamlining

## Proposed Changes

### Affected Components & Packages
- `internal/models/api.go`: `CompareItemRequest`, `CompareItemResult`, `FileCompareResponse`, `DualDedupeRequest`, `DualDedupeResponse`.
- `internal/services/explorer_service.go`: `CompareItems` with cross-backend hash negotiation and equivalence hash clustering for $N \ge 2$ files.
- `internal/server/`:
  - `routes.go` & `handlers_explorer.go`: `POST /api/browse/compare`.
  - `handlers_auth.go`: Paired remote + local dedupe and dry-run noise suppression.
  - `handlers_telemetry.go`: Server-side `exclude_types` filtering for dense log pagination.
- `dashboard/frontend/`:
  - `js/profiles.js`: Fix background polling navigation reset.
  - `js/explorer.js` & `view_explorer.html`: Middle-truncated breadcrumbs without horizontal scrollbars.
  - `js/explorer_compare.js` & `modal_compare.html`: Multi-file selection & hash comparison modal.
  - `js/remotes_modals.js` & `modal_dedupe.html`: Remote folder navigation, collision jump-to-file, paired dedupe, and post-dedupe 1-click `--resync` CTA.
  - `js/logs.js` & `pattern_timeline_ledger.html`: Unified Log Ledger renaming, job-level clustering, dense pagination.
  - `js/log_dock.js`: True server-side pagination.
  - `index.html`: Modern wide-screen fluid layout (`max-w-[1720px] 2xl:max-w-[1920px]`).

### Technical Strategy & Data Flow
- **Hash Negotiation**: Priority `sha256` -> `sha1` -> `md5` -> `quickxorhash`. If intersection is empty (e.g. `crypt`), return `Algorithm: "none"` and compare file size. Cluster items by matching hash.
- **Paired Deduplication**: Target folder in Remote or Local automatically maps to the paired counterpart using profile definitions. Executes pass 1 (Remote) and pass 2 (Local with jail confinement).
- **Navigation Safety**: `fetchProfiles()` skips path synchronization when explorer panes are already initialized.

### Safety & Error Handling
- Path confinement strictly enforced for all local hash and dedupe operations.
- Canary file (`RCLONE_TEST`) protected from dedupe and modification.
- Dense log pagination prevents memory bloat while ensuring complete log page views.

## Verification Plan
- Unit tests: `go test -count=1 ./internal/services -run TestCompareItems`, `go test -count=1 ./internal/server -run TestHandleBrowseCompare`
- Pre-flight scripts: `./scripts/check.sh templates`, `./scripts/check.sh loc 400`, `./scripts/check.sh all`
