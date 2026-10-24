
# Task Plan: Cryptographic Immutable Audit Ledger & Unified Log Architecture

## Goal
Implement the Cryptographic Immutable Audit Ledger (`internal/audit`) with pure-Go SQLite (`modernc.org/sqlite`) and migration management (`pressly/goose/v3`), SHA-256 hash chaining, database append-only triggers, secret sanitization, thread-safe in-process POSIX log rotator (`internal/core/logger/rotator.go`), REST endpoints (`/api/audit/records`, `/api/audit/verify`, `/api/audit/export`), frontend integration (`partials/view_audit.html`, navigation tab, and telemetry filter), UI polish (Lucide SVGs, dynamic version header, removal of Enterprise branding), and dual-logging with engine execution hooks (`SYNC_EXECUTED`, `SYNC_ABORTED`).

## Tasks
- [x] Task 1: Audit Package Core (DB Path, Schema Migration & Pure-Go SQLite Driver)
  - **Scope**: `internal/audit/path.go`, `internal/audit/taxonomy.go`, `internal/audit/migrations/00001_initial_audit_schema.sql`, `internal/audit/migrations.go`, `internal/audit/db.go`.
  - **Verification**: `go test -count=1 ./internal/audit -run TestMigrationsAndDBOpen`
- [x] Task 2: Cryptographic Hash Chaining, Ledger Append & Verification Engine
  - **Scope**: `internal/audit/ledger.go`, `internal/audit/ledger_test.go` (implementing `ComputeHash`, `Append`, `VerifyLedger`, triggers test proving UPDATE and DELETE are blocked by SQLite).
  - **Verification**: `go test -count=1 -v ./internal/audit -run TestAppendAndVerify`
- [x] Task 3: POSIX Log Rotator & Dual-Logging in Logger
  - **Scope**: `internal/core/logger/rotator.go`, update `internal/core/logger/logger.go`, `internal/core/logger/rotator_test.go`.
  - **Verification**: `go test -count=1 -v ./internal/core/logger -run TestRotator`
- [x] Task 4: Audit REST API Endpoints & Service Integration
  - **Scope**: `internal/services/audit_service.go`, `internal/server/handlers_audit.go`, register routes in `internal/server/server.go` (`GET /api/audit/records`, `GET /api/audit/verify`, `GET /api/audit/export`), with tests.
  - **Verification**: `go test -count=1 -v ./internal/server -run TestAuditEndpoints`
- [x] Task 5: Frontend UI (Audit Tab, Record Explorer, Verification Badge & Dual-Stream Telemetry)
  - **Scope**: `dashboard/frontend/partials/view_audit.html`, `dashboard/frontend/index.html` (adding the 4th top tab "Audit Ledger"), `dashboard/frontend/js/audit.js`, `dashboard/frontend/app.js`.
  - **Verification**: `go test -count=1 -v ./internal/server -run TestServer_RootHandler_RealFileSystem`
- [x] Task 6: Full Pre-Flight Quality Gate & Validation
  - **Scope**: Complete repository quality checks across all 8 stages.
  - **Verification**: `./scripts/check.sh`
- [x] Task 7: UI Polish (Lucide SVG Icons, Dynamic Version Header & Branding Cleanup)
  - **Scope**: `dashboard/frontend/partials/view_audit.html` (replace emojis with Lucide SVGs), `dashboard/frontend/index.html` (remove "Enterprise", bind `{{ .Version }}`), `internal/server/server.go` (dynamic `vMajor.Minor` injection).
  - **Verification**: `go test -count=1 -v ./internal/server -run TestServer_RootHandler_RealFileSystem`
- [x] Task 8: Dual-Stream Logger Wiring & Engine Audit Event Integration (`SYNC_EXECUTED`, `SYNC_ABORTED`)
  - **Scope**: `internal/services/sync_service.go` (pass `runLogger` to `NewAuditService`), `internal/engine/runner/runner.go` (record `SYNC_EXECUTED` on transfers > 0 or manual triggers, `SYNC_ABORTED` on failures).
  - **Verification**: `go test -count=1 -v ./internal/engine/runner -run TestSyncRunner`
- [x] Task 9: End-to-End Verification & Quality Gate
  - **Scope**: Complete repository quality checks across all 8 stages.
  - **Verification**: `./scripts/check.sh`
- [x] Task 10: Fix Deletion Candidate Parsing, Daemon Log Discovery, and Root Canary Scope
  - **Scope**: `internal/review/discovery.go`, `internal/services/review_service.go`, `internal/engine/runner/runner.go`, `internal/services/explorer_service.go`, `internal/server/handlers_sync.go`, `dashboard/frontend/app.js`, `dashboard/frontend/js/explorer.js`.
  - **Verification**: `./scripts/check.sh`
- [ ] Task 11: Google Drive Live Simulation Walkthrough
  - **Scope**: Live manual sync, file mutation, and conflict/resolution testing with Google Drive.
  - **Verification**: Verify new entries in `audit.db` and `bisync.log`.

## Status Updates
- 2026-10-24: Initialized task plan for Immutable Audit Ledger on branch `feat/immutable-audit-ledger`.
- 2026-10-24: Completed Task 1: Implemented DB path resolver, event taxonomy, goose schema migration with triggers and WAL mode, pure-Go SQLite initialization, and unit tests in `internal/audit`.
- 2026-10-24: Completed Task 2: Implemented SHA-256 hash chaining, secret-sanitizing `Append`, `QueryRecords` with filtering/pagination, `VerifyLedger` cryptographic validator, and tests with 82.0% coverage.
- 2026-10-24: Completed Task 3: Implemented in-process POSIX log rotator (`rotator.go`), dual-logging `RecordAuditEvent` with `"type": "audit"` in `logger.go`, and concurrency tests with 84.7% coverage.
- 2026-10-24: Completed Task 4: Implemented `AuditService` with secret-sanitizing event recorder, JSONL/CSV export, and verification methods; wired into `ReviewService` and `SyncService`; added REST endpoints `GET /api/audit/records`, `GET /api/audit/verify`, and `GET /api/audit/export`; verified with 80.8% services coverage and 81.4% server coverage.
- 2026-10-24: Completed Task 5: Implemented dedicated Audit Ledger view (`view_audit.html`), added 4th navigation tab, created Alpine module (`audit.js`) with verification and filter state, and upgraded multi-stream log terminal with segment filter pills and purple/cyan audit badges.
- 2026-10-24: Completed Task 6: Successfully ran full pre-flight quality checks (`./scripts/check.sh`), validating format, vet, staticcheck, govulncheck, betterleaks, esbuild, tidy, 83.0% test coverage, and binary compilation.
- 2026-10-24: Completed Task 7: Replaced raw copy emojis and text buttons with Lucide SVGs in `view_audit.html`, removed "Enterprise" branding, and implemented dynamic `vMajor.Minor` injection in `index.html` and `server.go`.
- 2026-10-24: Completed Task 8: Enabled dual-stream logging into `bisync.log` via `AuditService`, wired `SYNC_EXECUTED` (transfers > 0 and manual triggers) and `SYNC_ABORTED` in `SyncRunner`, and made rotator file opening lazy.
- 2026-10-24: Completed Task 9: Successfully executed full pre-flight quality gate (`./scripts/check.sh`), all stages passed with 83.3% test coverage and clean static analysis.
- 2026-10-24: Completed Task 10: Resolved all 3 Google Drive live issues: (1) Preserved root canary status when navigating into subfolders in explorer, (2) Fixed candidate parsing in `review/discovery.go` to support filenames with spaces without truncation (`fields[0]` bug), (3) Directed `ReviewService` and `SyncRunner` candidate discovery to read `rclone-daemon.log` where rclone outputs `File was deleted` lines and populate `pending_deletions.json`, and (4) Added detailed error reporting to `/api/action` and frontend `triggerAction` to eliminate generic `Request failed` alerts. Verified with 83.4% coverage and full `./scripts/check.sh` green pass.

















