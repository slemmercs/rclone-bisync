# Task Plan: Architecture Hardening & Real Rclone Sandbox Suite

## Goal
Implement architectural review recommendations: document November 4, 2026 audit and 5-tier testing strategy, harden review sessions with POSIX flock, standardize API responses, implement upstream RC daemon auto-reconnect, and implement the real local Rclone sandbox test suite.

## Tasks

- [x] Task 1: Document November 4, 2026 Code Review & Testing Strategy
  - **Scope**: `docs/architecture/code-review.md`, `docs/roadmap/ROADMAP.md`
  - **Verification**: `test -f docs/architecture/code-review.md && ./scripts/check.sh loc`

- [ ] Task 2: POSIX Advisory File Locking on Review Session & WAL
  - **Scope**: `internal/engine/review/journal.go`, `internal/services/review_service.go`, `internal/engine/review/journal_test.go`
  - **Verification**: `go test -v -count=1 ./internal/engine/review/... ./internal/services/... -run "TestJournal|TestReview"`

- [ ] Task 3: Standardize REST API Schemas with Concrete Models
  - **Scope**: `internal/server/handlers_config.go`, `internal/server/handlers_telemetry.go`, `internal/models/api.go`
  - **Verification**: `go test -v -count=1 ./internal/server/...`

- [ ] Task 4: Implement Daemon Watchdog & Auto-Reconnect Grace Period
  - **Scope**: `internal/services/telemetry_service.go`, `internal/core/client/client.go`
  - **Verification**: `go test -v -count=1 ./internal/services/... -run "TestTelemetry"`

- [ ] Task 5: Real Local Rclone Sandbox Integration Test Suite
  - **Scope**: `tests/integration/rclone_sandbox_test.go`
  - **Verification**: `go test -v -count=1 ./tests/integration/...`

- [ ] Task 6: Full Pre-Flight Validation & Quality Gate
  - **Scope**: Full repository
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-04: Initialized plan from implementation_plan.md on branch `feat/consolidate-mcp-and-secure-profiles`.
- 2026-11-04: Completed Task 1:
  - Documented root causes of recent bug clusters (concurrency, directory semantics, path quirks, RC parameter mapping) in `docs/architecture/code-review.md`.
  - Formulated the 5-tier testing architecture (Unit -> Subprocess Concurrency -> Real Local Rclone Sandbox -> Stateful Chaos/Saga -> E2E Chrome).
  - Aligned short, medium, and long-term roadmap priorities in `docs/roadmap/ROADMAP.md` and `docs/architecture/code-review.md`.
  - Verification passed (`test -f docs/architecture/code-review.md && ./scripts/check.sh loc`).









