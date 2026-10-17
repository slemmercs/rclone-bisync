
# Task Plan: Central JSON Logger & Interactive Log Cross-Filtering (Branch 6)

## Goal
Establish a centralized JSON logging package (`internal/core/logger`), condense bisync run entries in `bisync.log` to a single high-signal line, implement time-window daemon log slicing (`/api/logs/daemon`), and replace the dual-terminal layout with a unified orchestrator feed and on-demand focused daemon log modal dialog.

## Tasks
- [x] Task 1: Models & Central Logger Package (`internal/models/api.go`, `internal/core/logger/logger.go`, `internal/core/logger/logger_test.go`)
  - **Scope**: Define `BisyncLogEntry`, `DaemonLogsQuery`, and `DaemonLogsResponse` DTOs. Implement `internal/core/logger` standard library `log/slog` wrapper supporting multi-output dispatch (rotating JSON file and console).
  - **Verification**: `go test -count=1 -cover ./internal/core/logger` (achieved 92.2%).
- [x] Task 2: Log Slice Scanner Service (`internal/services/log_service.go`, `internal/services/log_service_test.go`)
  - **Scope**: Implement `LogService` with `FilterDaemonLogs` using `bufio.Scanner` to extract entries bounded by execution timestamp windows `[startTime - 1s, endTime + 1s]`.
  - **Verification**: `go test -count=1 -cover ./internal/services` (achieved 83.7%).
- [x] Task 3: Single-Line Bisync Run Logging in Runner (`internal/engine/runner/runner.go`, `internal/engine/runner/runner_test.go`, `internal/core/client/monitor.go`)
  - **Scope**: Refactor `SyncRunner` to record execution start/end times and emit exactly 1 high-density JSON ledger line to `bisync.log` via central logger. Downgrade verbose polling logs in `monitor.go` to debug.
  - **Verification**: `go test -count=1 -cover ./internal/engine/runner ./internal/core/client` (runner: 90.2%, client: 85.5%).
- [x] Task 4: Server Correlated Daemon Logs Route (`internal/server/server.go`, `internal/server/server_test.go`)
  - **Scope**: Add `GET /api/logs/daemon` endpoint accepting `start_time`, `end_time`, `run_id`, `job_id`, `limit` and serving `DaemonLogsResponse`.
  - **Verification**: `go test -count=1 -cover ./internal/server` (achieved 86.2%).
- [x] Task 5: Frontend Streamlined Feed & Focused Daemon Log Modal (`dashboard/frontend/index.html`, `dashboard/frontend/app.js`, `dashboard/frontend/js/utils.js`)
  - **Scope**: Remove 640px stacked daemon terminal, add "Daemon Logs" detail button per row, embed focused modal dialog with search, level filters, copy button, and JSON preview.
  - **Verification**: Verified DOM structure, event handlers, and data bindings.
- [x] Task 6: Pre-Flight Verification & Quality Gate (`./scripts/check.sh`, `go test -cover ./...`)
  - **Scope**: Run full pre-flight quality gate, verify statement coverage >80% across all packages, and compile single binary.
  - **Verification**: `./scripts/check.sh` and `go build -o bin/rclone-bisync main.go` passed cleanly.

## Status Updates
- 2026-10-17: Initialized Branch 6 plan for central JSON logger and interactive log cross-filtering.
- 2026-10-17: Completed Task 1 (Models & `internal/core/logger` with 92.2% coverage).
- 2026-10-17: Completed Task 2 (`internal/services/log_service.go` with 83.7% coverage).
- 2026-10-17: Completed Task 3 (Single-line structured ledger logging in `SyncRunner`, runner coverage 90.2%).
- 2026-10-17: Completed Task 4 (`GET /api/logs/daemon` endpoint in `internal/server`, coverage 86.2%).
- 2026-10-17: Completed Task 5 (Streamlined orchestrator terminal, 640px stacked terminal removed, focused daemon modal implemented).
- 2026-10-17: Completed Task 6 (All packages verified >80% statement coverage, quality gate passed, binary compiled).
