# Task Plan: Technical Debt Remediation & Quality Hardening

## Goal
Remediate key technical debt items: clean up workspace binaries, modularize near-ceiling files (`profiles.js`, `review.js`, `log_service.go`), strengthen test coverage in `internal/core/security` and `internal/engine/runner` above 80%, and verify full repository quality gates.

## Tasks

- [x] Task 1: Workspace Hygiene & Local Artifact Cleanliness
  - **Scope**: Root directory binaries (`bisync-hub`, `bisync-hub-new`)
  - **Verification**: `ls bisync-hub bisync-hub-new 2>&1` (verify absent) and `git status` clean

- [x] Task 2: Frontend High-LOC Module Decomposition (`profiles.js` & `review.js`)
  - **Scope**: `dashboard/frontend/js/profiles.js`, `dashboard/frontend/js/profiles_modal.js`, `dashboard/frontend/js/review.js`, `dashboard/frontend/js/review_actions.js`, `dashboard/frontend/index.html`
  - **Verification**: `./scripts/check.sh loc 400` and `go test -count=1 ./tests/e2e/...`

- [x] Task 3: Backend High-LOC Module Decomposition (`log_service.go`)
  - **Scope**: `internal/services/log_service.go`, `internal/services/log_parser.go`
  - **Verification**: `./scripts/check.sh loc 400` and `go test -count=1 ./internal/services/...`

- [ ] Task 4: Security & Redaction Test Coverage Hardening (>80%)
  - **Scope**: `internal/core/security/redact_test.go`
  - **Verification**: `go test -cover ./internal/core/security/...` (verify >80%)

- [ ] Task 5: Engine Runner Test Coverage Hardening (>80%)
  - **Scope**: `internal/engine/runner/runner_test.go`
  - **Verification**: `go test -cover ./internal/engine/runner/...` (verify >80%)

- [ ] Task 6: Schema Uniformity for Error Responses
  - **Scope**: `internal/models/api.go`, `internal/server/server.go`
  - **Verification**: `go test -count=1 ./internal/server/...`

- [ ] Task 7: Full Quality Gate Verification & Release
  - **Scope**: Full repository quality gate verification
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-10-31: Initialized technical debt remediation plan from implementation_plan.md covering LOC ceilings, test coverage gaps, and workspace cleanliness.
- 2026-10-31: Completed Task 1: Cleaned up root workspace binary artifacts (bisync-hub, bisync-hub-new), eliminating index bloat and search scanner interference.
- 2026-10-31: Completed Task 2: Modularized profiles.js (down to 154 LOC with profiles_modal.js at 256 LOC) and review.js (down to 196 LOC with review_actions.js at 205 LOC). Headless Chrome E2E test and LOC checks passed.
- 2026-10-31: Completed Task 3: Decomposed log_service.go (down to 246 LOC) by extracting parsing helpers into log_parser.go (152 LOC). All services tests and LOC checks passed.







