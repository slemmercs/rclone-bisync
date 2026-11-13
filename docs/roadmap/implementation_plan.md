# Implementation Plan: Fix Bisync Transfer Counting & Unified Ledger Visibility

## Overview
Resolve two critical synchronization and observability defects:
1. **Accurate Transfer Counting**: In Rclone `/sync/bisync`, `/job/status` does not populate `output.transfers` or `output.stats`. The authoritative transfer stats are recorded under the job's group in `/core/stats`, with fallback parsing of `output.output` log text for `"Copied (new)"` and `"Copied (replaced)"`. `MonitorJob` in `internal/core/client/monitor.go` must query group stats from `/core/stats` or parse output log text when `filesSynced == 0`.
2. **Unified Ledger File Upload Visibility**:
   - In `internal/services/audit_service.go`, mirror audit events into `bisync.log` with target object paths embedded in `Msg` (`[profile] ACTION: path by actor`) so searches for the filename succeed.
   - In `dashboard/frontend/js/logs.js`, ensure `clusterLogEntries` incorporates `item.object` into the cluster key to prevent dissimilar file uploads from collapsing into a single folded row.
   - In `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`, display the `action` badge on unclustered rows.
   - In `dashboard/frontend/js/explorer_actions.js`, trigger `fetchBisyncLogs(1, true)` and `fetchAuditRecords()` upon successful upload so paginated ledgers update immediately.

---

## Proposed Changes

### Phase 1: Transfer Counting in `MonitorJob` (`internal/core/client`)
- In `internal/core/client/monitor.go`:
  - When `status["finished"] == true` and `filesSynced == 0`:
    - If `status["group"]` is a non-empty string, call `c.getGroupTransfers(ctx, group)` via `/core/stats`.
    - If still 0, parse `outputData["output"]` string counting occurrences of `"msg":"Copied (new)"`, `"msg":"Copied (replaced)"`, `Copied (new)`, and `Copied (replaced)`.
- In `internal/core/client/monitor_test.go`:
  - Unit tests verifying transfer extraction from `/core/stats` group and from `output["output"]`.

### Phase 2: Audit Event Mirroring with Object Path (`internal/services`)
- In `internal/services/audit_service.go`:
  - Format `logEntry.Msg` with `objectTarget` when present:
    `fmt.Sprintf("[%s] %s: %s by %s", record.ProfileID, record.Action, objectTarget, record.Actor)`.
  - Fallback to `fmt.Sprintf("[%s] %s by %s", record.ProfileID, record.Action, record.Actor)` when `objectTarget` is empty.
- In `internal/services/audit_service_test.go`:
  - Unit tests verifying mirrored audit log line includes target path in message.

### Phase 3: Non-Collapsing File Ops Clusters & Unified Ledger Action Badges
- In `dashboard/frontend/js/logs.js`:
  - In `clusterLogEntries`, include `item.object || ""` in the clustering `key`. This ensures different uploaded or modified files are never lumped into an ambiguous folded cluster.
  - Keep `dashboard/frontend/js/logs.js` strictly under 400 LOC.
- In `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`:
  - In single entry row (`!cluster.isCluster`), render `<span class="px-1 py-0.2 rounded bg-purple-950/60 text-purple-300 text-[9px] font-semibold uppercase shrink-0" x-text="cluster.entry.action"></span>` when `cluster.entry.action` is present.

### Phase 4: Paginated Log Invalidation on Explorer Upload
- In `dashboard/frontend/js/explorer_actions.js`:
  - After upload loop completes successfully (`okCount > 0`):
    - Trigger `if (typeof this.fetchBisyncLogs === "function") this.fetchBisyncLogs(1, true);`
    - Trigger `if (typeof this.fetchAuditRecords === "function") this.fetchAuditRecords();`
  - Keep `dashboard/frontend/js/explorer_actions.js` strictly under 400 LOC.

---

## Safety & Error Handling

- **Non-blocking Transfer Probe**: The `/core/stats` lookup in `MonitorJob` must never fail the job if `/core/stats` is inaccessible; it only logs a debug warning and falls back to string parsing.
- **Strict LOC Limit**: All files must strictly remain under 400 lines (`./scripts/check.sh loc 400`).
- **Canary & Jail Integrity**: Canary files and jail confinement remain unmodified.

---

## Verification Plan

- `go test -count=1 -v ./internal/core/client -run TestMonitorJob`
- `go test -count=1 -v ./internal/services -run TestAuditService`
- `./scripts/check.sh templates`
- `./scripts/check.sh loc 400`
- `go test -count=1 ./tests/e2e -run TestDashboard_HeadlessChromeSmoke`
- `./scripts/check.sh all`


