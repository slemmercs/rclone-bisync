# Task Plan: Audit Ledger Reset, Telemetry Refinement & Defensive Defaults

## Goal
Enforce defensive defaults (`max-delete: 0`, `lock_explorer_remote: true`), quiet background HTTP polling noise to debug level, fix dashboard `bisync.log` pagination, isolate tests from live RC port, deduplicate audit profile selector, and provide an immutable-safe audit ledger reset with backup rotation.

## Tasks

- [x] Task 1: Defensive Defaults & Profile Builder Hardening
  - **Scope**: `internal/config/dto.go`, `internal/config/dto_test.go`, `internal/profiles/builder.go`, `internal/profiles/builder_test.go`, `internal/server/handlers_profiles.go`, `internal/server/handlers_profiles_test.go`, `dashboard/frontend/js/settings.js`, `dashboard/frontend/js/profiles.js`
  - **Verification**: `go test -count=1 ./internal/config/... ./internal/profiles/... ./internal/server/...`

- [x] Task 2: HTTP Access Logging Refinement & UI Formatting
  - **Scope**: `internal/server/server.go`, `internal/server/server_test.go`, `dashboard/frontend/js/utils.js`, `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`
  - **Verification**: `go test -count=1 ./internal/server/...`

- [ ] Task 3: Bisync Log Pagination Wiring in UI
  - **Scope**: `dashboard/frontend/app.js`, `dashboard/frontend/js/logs.js`
  - **Verification**: `go test -count=1 ./internal/server/...`

- [ ] Task 4: Strict RC API Mocking in Test Suites
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`, `internal/services/review_service_test.go`
  - **Verification**: `go test -count=1 ./internal/services/... ./tests/e2e/...`

- [ ] Task 5: Audit Ledger Cleanup, Reset Endpoint & Modal
  - **Scope**: `dashboard/frontend/partials/view_audit.html`, `internal/audit/`, `internal/server/handlers_audit.go`, `internal/server/handlers_audit_test.go`, `dashboard/frontend/js/audit.js`
  - **Verification**: `go test -count=1 ./internal/audit/... ./internal/server/...`

- [ ] Task 6: Pre-Flight Verification & Quality Gate
  - **Scope**: Entire repository
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-10-31: Initialized plan from implementation_plan.md to address telemetry logging noise, log pagination, test port isolation, duplicate profile selectors, defensive defaults, and audit ledger reset.
- 2026-10-31: Completed Task 1: Enforced `max_delete: 0` and `lock_explorer_remote: true` across `DefaultGlobalDefaults()`, profile builder, and frontend settings/profiles modules. Fixed builder and handler conditions (`< 0`) to treat `0` as an explicit defensive deletion threshold. Verified with `go test -count=1 ./internal/config/... ./internal/profiles/... ./internal/server/...`.
- 2026-10-31: Completed Task 2: Demoted routine HTTP polling endpoints (`/api/telemetry`, `/api/health`, `/api/profiles/summary`, `/api/logs/*`) to `slog.Debug` level in server middleware. Formatted `http_access` lines in `utils.js` as `METHOD PATH → STATUS (Xms)` and restricted the Daemon modal button in `pattern_timeline_ledger.html` to correlated records possessing `run_id` or `job_id`. Verified with `go test -count=1 ./internal/server/...`.


