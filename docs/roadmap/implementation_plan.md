# Implementation Plan: System Health Modal Log Loading & Pinned Log Dock Normalization

## Overview
Fix empty log stream in the System Health Center modal and normalize structured log ingestion (`bisync_logs` / `daemon_logs`), whitespace-padded log levels (`"INFO   "` vs `"INFO"`), and mutation classification in the Pinned Log Audit Dock.

## Proposed Changes

### Affected Components & Packages

- `dashboard/frontend/js/logs.js`:
  - Define `fetchDaemonLogs(entry = null)` on the Alpine.js state machine so `openSystemHealthModal` and `setSystemHealthTab('daemon')` can load daemon logs on demand.
  - Refactor `openDaemonModal(entry)` to delegate to `fetchDaemonLogs(entry)` while maintaining modal toggle and filter reset ergonomics.

- `dashboard/frontend/partials/modals/modal_system_health.html`:
  - Add empty-state indicator when daemon logs are not present or filtered out.
  - Support deep path navigation chips (`jumpToExplorerPath`) on entries containing detected file paths with automatic modal dismissal.

- `dashboard/frontend/js/log_dock.js`:
  - Normalize log ingestion in `getFilteredDockLogs()` to consume parsed structured entries from `this.bisyncLogs` or `this.parsedBisyncLogs`, falling back to `unwrapAndParseLogLines(this.telemetry.bisync_logs)`.
  - Strip trailing padding from log levels with `.trim().toUpperCase()` to ensure `"ERROR  "` and `"NOTICE "` match filter predicates.
  - Expand mutation detection to include entries where `object` is detected or file operations are mentioned.
  - Update `getDockLogLevelClass` and `formatDockLogTime` to handle padded levels and both `time` and `timestamp` fields.

- `dashboard/frontend/partials/components/component_log_dock.html`:
  - Bind `log.time || formatDockLogTime(log.timestamp)` for robust timestamps.
  - Bind `(log.level || 'INFO').trim()` for clean badge rendering.

### Safety & Error Handling

- Null-safe checks on `this.bisyncLogs`, `this.telemetry`, and log array structures.
- Graceful degradation when daemon endpoint returns 404 or empty list.
- Prevent unhandled exceptions during log level normalization.

## Verification Plan

- `go test -count=1 -v ./tests/e2e -run TestDashboard_HeadlessChromeSmoke`
- `./scripts/check.sh templates`
- `./scripts/check.sh loc 400`
- `./scripts/check.sh all`
