
# Task Plan: Structured Daemon Log Unwrapping & Modal Refinement

## Goal
Unwrap nested JSON logs embedded within daemon wrapper messages (specifically `bisync/rc.go`), slice events accurately by sub-event timestamps, and render structured, syntax-highlighted daemon logs with individual line counts, level filters, object badges, and formatted settings.

## Tasks
- [x] Task 1: Backend Daemon Log Unwrapper & Time Window Slicing (`internal/services/log_service.go`, `internal/services/log_service_test.go`)
  - **Scope**: Update `LogService.FilterDaemonLogs` to detect `msg` containing newline-delimited JSON sub-events, unpack each sub-event, evaluate its timestamp against `[startTime - 1s, endTime + 1s]`, and append individual JSON events.
  - **Verification**: `go test -count=1 -cover ./internal/services` (achieved 81.7%).
- [x] Task 2: Frontend Log Parser & Structured Metadata Extraction (`dashboard/frontend/js/utils.js`, `dashboard/frontend/app.js`)
  - **Scope**: Update `parseJsonLogLine` to extract `objectType` and `source`, parse embedded JSON payloads (e.g. comparison settings) into structured details, and add client-side unwrapping in `app.js`.
  - **Verification**: Verified parser handles sub-event records, source, objectType, and structured details.
- [x] Task 3: Modal UI Template & Filter Enhancements (`dashboard/frontend/index.html`)
  - **Scope**: Render individual unwrapped events with timestamp, level pill, message, object badge, and source tag. Add formatted key-value view for comparison settings.
  - **Verification**: Verified DOM structure, bindings, and cache-busting tags.
- [x] Task 4: Quality Gate & Checkpoint Verification (`./scripts/check.sh`, `go test -cover ./...`)
  - **Scope**: Run pre-flight checks, verify statement coverage >80% across all packages, compile binary.
  - **Verification**: `./scripts/check.sh` and `go build -o bin/rclone-bisync main.go` passed cleanly.

## Status Updates
- 2026-10-18: Initialized task plan on branch `feat/structured-daemon-logs`.
- 2026-10-18: Completed Task 1 (Backend nested JSON log unwrapping and event-level time window slicing in `LogService`, 81.7% coverage).
- 2026-10-18: Completed Task 2 (Frontend log parser enhanced with `objectType`, `source`, `details`, and defense-in-depth unwrapper).
- 2026-10-18: Completed Task 3 (Modal UI enhanced with individual lines, object & type tags, source badges, and comparison settings grid).
- 2026-10-18: Completed Task 4 (Full quality gate `./scripts/check.sh` passed, all packages >80% statement coverage, single binary compiled).

