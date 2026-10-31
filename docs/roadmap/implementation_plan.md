# Implementation Plan: Profile-Scoped Logs, Unified Ledger Filtering & Real-Time Polling

## Proposed Changes

Add profile identification (`ProfileID` and `ProfileName`) to all sync run execution logs, prefix the log message with `[<ProfileName>]` in raw and JSON logs, enable profile-based filtering in the Unified Ledger UI, prevent different profiles from clustering together into the same folded group, and resolve the real-time polling regression so logs update immediately without requiring manual page reloads.

### Affected Components & Packages

- `internal/models/api.go`:
  - Add `ProfileID` and `ProfileName` fields to `BisyncLogEntry`.
- `internal/engine/runner/runner.go` & `internal/engine/runner/runner_handlers.go`:
  - Centralize `profileLabel()` and `formatLogMsg()` helper methods.
  - Prefix console `log.Printf` output and `BisyncLogEntry.Msg` with `[<ProfileName>]` for non-default profiles.
  - Populate `ProfileID` and `ProfileName` on all `RecordSyncRun` calls (success, fault, abort, canary guard).
- `internal/engine/runner/runner_test.go`:
  - Unit tests verifying profile prefixing and field enrichment in `RecordSyncRun`.
- `dashboard/frontend/js/utils.js`:
  - Extract `profile_id` and `profile_name` in `parseJsonLogLine`.
- `dashboard/frontend/js/logs.js`:
  - Add `telemetryProfileFilter: "all"` state.
  - Update `clusterLogEntries` key to incorporate profile identity, ensuring different profiles never fold into the same cluster.
  - Filter by `telemetryProfileFilter` in `filteredBisyncLogs`.
  - Add `silent` option to `fetchBisyncLogs(page, silent = false)` to refresh in background without UI spinner flicker.
- `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`:
  - Add profile filter `<select>` dropdown next to the `All | Sync | Audit` pills.
- `dashboard/frontend/js/polling.js`:
  - Automatically call `fetchBisyncLogs(1, true)` immediately when `syncJustFinished` and periodically during active polling on page 1 of telemetry tab.
- `dashboard/frontend/js/profiles_history.js`:
  - Refresh logs when triggering a profile sync.

### Technical Strategy & Data Flow

1. **Backend Profile Scoping**:
   - When `SyncRunner.Run()` executes, `r.Config.ProfileName` and `r.Config.ProfileID` are inspected.
   - If a friendly name or custom ID is set, logs format as `[<ProfileName>] Sync succeeded in 5.0s (0 file(s) transferred)`.
   - The JSON entry contains `"profile_id": "...", "profile_name": "..."`.
2. **Frontend Filtering & Clustering**:
   - `parseJsonLogLine` extracts `profile_id` and `profile_name`.
   - `clusterLogEntries` uses `key = level + "::" + type + "::" + profile + "::" + norm`.
   - `filteredBisyncLogs` filters by selected profile.
3. **Real-Time Log Ingestion**:
   - `poll()` in `polling.js` triggers `fetchBisyncLogs(1, true)` so log updates appear dynamically without reload.

### Safety & Error Handling

- Respect `<400` LOC limit on all modified Go and JS files.
- Backward compatibility: older entries without profile fields default gracefully to clean display.
- Default profile remains clean without redundant `[default]` noise.

## Verification Plan

- Unit tests: `go test -count=1 ./internal/engine/runner/...`
- File size check: `./scripts/check.sh loc 400`
- End-to-end smoke tests: `go test -count=1 ./tests/e2e/...`
- Full quality check: `./scripts/check.sh all`



