# Implementation Plan

[Overview]
Establish a centralized JSON logging package (`internal/core/logger`), eliminate multiline log clutter by condensing each bisync execution into a single comprehensive ledger entry in `bisync.log`, and implement interactive log cross-filtering (Branch 6) with a focused modal dialog for correlated daemon transfer logs.

Historically, bisync runs emitted 7+ verbose lines with ASCII banners (`===`), start announcements, and polling heartbeats into the orchestrator log. This cluttered the UI and buried actionable telemetry. Furthermore, the dashboard displayed two disconnected stacked terminals (`bisync.log` at 320px and `rclone-daemon.log` at 640px), consuming ~960px of vertical space with no correlation between sync runs and low-level daemon events.

This plan addresses both issues:
1. Introduces `internal/core/logger`, a centralized standard-library `log/slog` module producing clean JSON records with file and console dispatch.
2. Refactors `SyncRunner` so that each sync iteration writes **exactly one high-density JSON log entry** to `bisync.log` summarizing status, duration, file counts, run ID, job ID, and execution time bounds.
3. Implements high-performance time-window daemon log slicing (`/api/logs/daemon`) and replaces the stacked terminal layout with a compact orchestrator feed featuring an on-demand **Focused Modal Dialog** for inspecting correlated daemon transfers.

[Types]
Type system changes:
1. `internal/models/api.go`:
   - `BisyncLogEntry`: Schema for single-line sync ledger records:
     ```go
     type BisyncLogEntry struct {
         Time        string  `json:"time"`
         Level       string  `json:"level"`
         Msg         string  `json:"msg"`
         RunID       string  `json:"run_id"`
         JobID       int     `json:"job_id,omitempty"`
         Origin      string  `json:"origin,omitempty"`
         Status      string  `json:"status"`               // SUCCESS, FAULT, ABORTED
         ErrorType   string  `json:"error_type,omitempty"` // TOO_MANY_DELETES, RATE_LIMIT, etc.
         DurationSec float64 `json:"duration_sec"`
         FilesSynced int     `json:"files_synced"`
         StartTime   string  `json:"start_time"`           // RFC3339
         EndTime     string  `json:"end_time"`             // RFC3339
     }
     ```
   - `DaemonLogsQuery`: Request parameters for correlated log retrieval:
     ```go
     type DaemonLogsQuery struct {
         RunID     string  `json:"run_id"`
         JobID     int     `json:"job_id"`
         StartTime string  `json:"start_time"` // RFC3339
         EndTime   string  `json:"end_time"`   // RFC3339
         Limit     int     `json:"limit"`
     }
     ```
   - `DaemonLogsResponse`: Response DTO for sliced daemon logs:
     ```go
     type DaemonLogsResponse struct {
         RunID      string   `json:"run_id"`
         JobID      int      `json:"job_id"`
         Count      int      `json:"count"`
         Logs       []string `json:"logs"`
         Correlated bool     `json:"correlated"`
     }
     ```

[Files]
File modifications:
1. New files:
   - `internal/core/logger/logger.go`: central JSON logger wrapping Go's `log/slog`, supporting file append to `bisync.log`, console teeing, and `RecordSyncRun`.
   - `internal/core/logger/logger_test.go`: unit tests verifying atomic JSON output, multi-writer dispatch, and concurrency.
   - `internal/services/log_service.go`: high-performance scanner using `bufio.Scanner` to filter `rclone-daemon.log` entries matching `[startTime - 1s, endTime + 1s]`.
   - `internal/services/log_service_test.go`: unit tests for log window filtering, corrupted JSON handling, and limits.
2. Existing files to modify:
   - `internal/engine/runner/runner.go`: consume `internal/core/logger`; remove verbose multiline banner logging; emit exactly 1 `BisyncLogEntry` per run.
   - `internal/core/client/monitor.go`: downgrade polling logs to debug level to prevent polluting orchestrator output.
   - `internal/server/server.go`: register `GET /api/logs/daemon` endpoint returning `DaemonLogsResponse`.
   - `internal/server/server_test.go`: add integration tests for `/api/logs/daemon`.
   - `dashboard/frontend/index.html`: replace 640px stacked daemon terminal with a single unified stream and embed the focused daemon log modal dialog.
   - `dashboard/frontend/app.js`: modal state management, search query filtering, and level pill filtering.
   - `dashboard/frontend/js/utils.js`: update `parseJsonLogLine` to handle new `BisyncLogEntry` fields (`job_id`, `duration_sec`, `files_synced`, `status`, `start_time`, `end_time`).
   - `task_plan.md`: document implementation sprint.
   - `backlog.md`: mark Branch 6 as active/completed.

[Functions]
Function modifications:
1. `internal/core/logger/logger.go`:
   - `New(filePath string, console bool) (*Logger, error)`: constructs logger with JSON file handler and optional console writer.
   - `RecordSyncRun(entry models.BisyncLogEntry)`: writes atomic JSON line to log file.
2. `internal/services/log_service.go`:
   - `FilterDaemonLogs(logPath string, startTime, endTime time.Time, limit int) ([]string, error)`: scans `rclone-daemon.log` and extracts entries within execution timestamp window.
3. `internal/engine/runner/runner.go`:
   - `Run(...) bool`: refactored to record `startTime`, collect outcome/jobID, and write a single `models.BisyncLogEntry` at run completion or abort.
4. `internal/server/server.go`:
   - `handleCorrelatedDaemonLogs(w http.ResponseWriter, r *http.Request)`: parses `start_time`, `end_time`, and `run_id`, querying `LogService`.
5. `dashboard/frontend/app.js`:
   - `openDaemonLogsModal(entry)`: sets active run, opens modal, and queries `/api/logs/daemon`.
   - `closeDaemonLogsModal()`: hides modal overlay.

[Classes]
Class modifications:
1. `Logger` (`internal/core/logger/logger.go`): central structured logging facade.
2. `LogService` (`internal/services/log_service.go`): domain service for log stream analysis.
3. `SyncRunner` (`internal/engine/runner/runner.go`): integrated with `Logger`.

[Dependencies]
Dependency modifications:
- Pure Go standard library: `log/slog`, `bufio`, `encoding/json`, `os`, `time`, `net/http`. Zero third-party packages or npm dependencies.

[Testing]
Testing approach:
- `internal/core/logger/logger_test.go`: test single-line JSON formatting, concurrent writes, file append.
- `internal/services/log_service_test.go`: test time-window filtering across multiline daemon logs with valid and invalid JSON.
- `internal/engine/runner/runner_test.go`: assert that `Run()` writes exactly 1 JSON line to `bisync.log` with expected metadata.
- `internal/server/server_test.go`: test `GET /api/logs/daemon` parameter validation and HTTP 200 responses.
- Full quality gate: `./scripts/check.sh` ensuring all packages pass `go test -count=1 ./...` with statement coverage >80%.

[Implementation Order]
1. `internal/core/logger`: Implement central logger module and unit tests.
2. `internal/services/log_service`: Implement high-performance daemon log slice scanner and tests.
3. `internal/engine/runner`: Refactor runner to emit single-line JSON records using central logger.
4. `internal/server`: Register and test `GET /api/logs/daemon`.
5. `dashboard/frontend`: Update `utils.js`, `index.html`, and `app.js` with single-stream UI and focused daemon modal.
6. Verification & Quality Gate: Run `./scripts/check.sh` and compile binary.
