
# Task Plan: Cryptographic Immutable Audit Ledger & Unified Log Architecture

## Goal
Implement the Cryptographic Immutable Audit Ledger (`internal/audit`) with pure-Go SQLite (`modernc.org/sqlite`) and migration management (`pressly/goose/v3`), SHA-256 hash chaining, database append-only triggers, secret sanitization, thread-safe in-process POSIX log rotator (`internal/core/logger/rotator.go`), REST endpoints (`/api/audit/records`, `/api/audit/verify`, `/api/audit/export`), and frontend integration (`partials/view_audit.html`, navigation tab, and telemetry filter).

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

## Status Updates
- 2026-10-24: Initialized task plan for Immutable Audit Ledger on branch `feat/immutable-audit-ledger`.
- 2026-10-24: Completed Task 1: Implemented DB path resolver, event taxonomy, goose schema migration with triggers and WAL mode, pure-Go SQLite initialization, and unit tests in `internal/audit`.
- 2026-10-24: Completed Task 2: Implemented SHA-256 hash chaining, secret-sanitizing `Append`, `QueryRecords` with filtering/pagination, `VerifyLedger` cryptographic validator, and tests with 82.0% coverage.
- 2026-10-24: Completed Task 3: Implemented in-process POSIX log rotator (`rotator.go`), dual-logging `RecordAuditEvent` with `"type": "audit"` in `logger.go`, and concurrency tests with 84.7% coverage.
- 2026-10-24: Completed Task 4: Implemented `AuditService` with secret-sanitizing event recorder, JSONL/CSV export, and verification methods; wired into `ReviewService` and `SyncService`; added REST endpoints `GET /api/audit/records`, `GET /api/audit/verify`, and `GET /api/audit/export`; verified with 80.8% services coverage and 81.4% server coverage.
- 2026-10-24: Completed Task 5: Implemented dedicated Audit Ledger view (`view_audit.html`), added 4th navigation tab, created Alpine module (`audit.js`) with verification and filter state, and upgraded multi-stream log terminal with segment filter pills and purple/cyan audit badges.
- 2026-10-24: Completed Task 6: Successfully ran full pre-flight quality checks (`./scripts/check.sh`), validating format, vet, staticcheck, govulncheck, betterleaks, esbuild, tidy, 83.0% test coverage, and binary compilation.
















