# Implementation Plan: Global Fleet Settings UI/UX Redesign, System Health Diagnostics & Ledger Noise Filtering (Tasks 34-37)

## Proposed Changes

Address operator UX friction and diagnostic ambiguity identified in the cockpit sandbox:
1. **Global Fleet Settings UI/UX Overhaul**: Redesign the modal from a cramped 720px horizontal-tabbed 2-column grid into a spacious 980px split-pane layout featuring a vertical left navigation sidebar and card/row-based settings canvas. Clean up tab nomenclature ("Telemetry & Features") and present toggles with descriptive subtitles.
2. **System Health Status Resolution & Diagnostics**: Fix the client-side health computation where `'healthy'` from Go backend was mismatched with `'ok'`. Enhance `ModalSystemHealth.vue` to display actionable diagnostic messages (connectivity details, canary verification errors, failing profiles). Update sandbox seed script so canary tokens exist across all profile paths.
3. **Unified Ledger Slog Noise Suppression**: Enhance `internal/services/filetail.go` to inspect `m["msg"]` for structured JSON logs, correctly categorizing `http_access` events so polling noise (`/api/telemetry`, `/api/queue`, `/api/status`) is suppressed under default filters.

### Affected Components & Packages

- `dashboard/frontend/src/components/ModalGlobalSettings.vue`: Split-pane dialog with 220px left navigation rail and semantic card rows.
- `dashboard/frontend/src/stores/telemetry.ts`: Accept both `'healthy'` and `'ok'` for `daemonHealthy`.
- `dashboard/frontend/src/components/ModalSystemHealth.vue`: Diagnostic status card with root-cause feedback for degraded/offline states.
- `scripts/run-sandbox.sh`: Touch `$SANDBOX_DIR/local/projects/RCLONE_TEST` to prevent false-positive canary trips on `prof_backup`.
- `internal/services/filetail.go` & `filetail_test.go`: Categorize JSON slog lines with `msg == "http_access"` as `entryType = "http_access"`.

### Technical Strategy & Data Flow

- **Split-Pane Layout**: `<div class="flex">` with fixed sidebar `<div class="w-56 shrink-0 border-r border-carbon-border ...">` and scrollable content pane `<div class="flex-1 p-5 overflow-y-auto space-y-5">`.
- **Vertical Spacing & Hierarchy**: Each section is an independent card with a crisp header, context subtitle, and comfortable vertical field spacing.
- **Log Categorization**: In `filterLogEntry`, if `m["type"]` is empty and `m["msg"] == "http_access"`, assign `entryType = "http_access"`. When `exclude_types=http_access` or default timeline ledger filters apply, HTTP polling noise is eliminated.

## Verification Plan

- Frontend verification: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`.
- Go unit test verification: `go test -v -count=1 ./internal/services/... -run TestTailJSONLogPaginatedFiltered`.
- Full quality gate: `./scripts/check.sh all`.

