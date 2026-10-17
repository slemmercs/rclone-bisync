# Task Plan: Phase 4 - Embedded Single-Binary HTTP Server & UI

## Goal
Implement single-binary Go embedded dashboard (`//go:embed`), REST API routing, CSRF protection middleware, models, telemetry compilation, and binary build.

## Tasks
- [x] Task 1: API Response & Request DTO Models (`internal/models`)
  - **Scope**: Port all REST API schema definitions matching FastAPI pydantic models (`ActionRequest`, `BrowseResponse`, `TelemetryPayload`, `TransferStats`, `RemotesResponse`, etc.).
  - **Verification**: `go build ./internal/models`
- [x] Task 2: Telemetry Aggregation & Transfer Statistics (`internal/services/telemetry.go`)
  - **Scope**:
    - Port `GetTransferStats` via Rclone RC `/core/stats`.
    - Port `GetCachedMetrics` caching lines and detecting mtime/size.
    - Port `GetRuntimeConfig` with key sanitization.
    - Port `GetTelemetryPayload` with unified metrics, logs, health, and notification state.
  - **Verification**: `go test -v ./internal/services`
- [x] Task 3: Embedded Web Server & REST Endpoints (`internal/server`)
  - **Scope**:
    - CSRF Protection middleware enforcing `X-Dashboard-Request` header and allowed loopback origin checks.
    - REST Endpoints: `/api/logs/bisync/raw`, `/api/logs/daemon/raw`, `/api/telemetry`, `/api/action`, `/api/deletions/pending`, `/api/remotes`, `/api/browse`, `/api/browse/canary`, `/api/browse/copy`, `/api/browse/delete`.
    - Automatic secret redaction using `security.RedactSecrets` on all outgoing JSON responses.
    - Single-binary static assets & SPA serving: `//go:embed dashboard/frontend/*` with `/static/`, `/favicon.ico`, and `/` SPA routing.
  - **Verification**: `go test -v ./internal/server`
- [x] Task 4: Single-Binary Compilation (`main.go` -> `bin/rclone-bisync`)
  - **Scope**: Build stand-alone Linux executable embedding the full dashboard UI.
  - **Verification**: `go build -o bin/rclone-bisync main.go && ./bin/rclone-bisync --help`
- [x] Task 5: Quality Gate Verification
  - **Scope**: Execute full pre-flight quality check (`./scripts/check.sh`).
  - **Verification**: All 8 Go packages pass tests, 77 Python tests pass, secret scanners clean.

## Status Updates
- 2026-10-17: Implemented `internal/models/api.go`.
- 2026-10-17: Implemented `internal/services/telemetry.go` and extended `client.GetCoreStats`.
- 2026-10-17: Implemented `internal/server` with CSRF protection, endpoint routing, secret sanitization, and embedded static assets.
- 2026-10-17: Built standalone native binary `bin/rclone-bisync` (11MB self-contained executable).
- 2026-10-17: Quality gate `./scripts/check.sh` 100% clean.
