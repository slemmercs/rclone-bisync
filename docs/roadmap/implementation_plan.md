# Implementation Plan: Technical Debt Remediation & Quality Hardening

## Proposed Changes

Remediate identified technical debt across the codebase to ensure long-term maintainability, eliminate LOC ceiling hazards (<400 LOC), strengthen test coverage across core engines, and enhance workspace hygiene.

### Affected Components & Packages

- **Workspace**: Root directory binaries (`bisync-hub`, `bisync-hub-new`)
- **Frontend Modules**:
  - `dashboard/frontend/js/profiles.js` (399 LOC) & `dashboard/frontend/js/profiles_modal.js`
  - `dashboard/frontend/js/review.js` (393 LOC) & `dashboard/frontend/js/review_diff.js`
- **Backend Services**:
  - `internal/services/log_service.go` (389 LOC) & `internal/services/log_parser.go`
- **Backend Test Coverage**:
  - `internal/core/security/redact_test.go` (increase package coverage from 76.9% to >80%)
  - `internal/engine/runner/runner_test.go` (increase package coverage from 74.2% to >80%)
- **API Models & Error Contracts**:
  - `internal/models/api.go` and `internal/server/server.go` (standardize JSON error responses)

### Technical Strategy & Data Flow

1. **Workspace Hygiene**:
   - Clean up untracked local binaries (`bisync-hub`, `bisync-hub-new`) from the repo root to prevent indexing bloat and search pollution.
2. **Frontend Modularization**:
   - Decompose `profiles.js` into core listing/state and `profiles_modal.js` (modal forms, schedule presets, input sanitization).
   - Decompose `review.js` into core review listing and `review_diff.js` (diff modal, side-by-side view, tree rendering).
   - Maintain Alpine.js zero-build modularity and test with E2E suite.
3. **Backend Modularization**:
   - Decompose `internal/services/log_service.go` by extracting parsing routines (`parseBisyncLogLine`, regex extraction, unmarshaling) into `internal/services/log_parser.go`.
4. **Coverage Hardening**:
   - Add deep recursive sanitization test cases to `internal/core/security/redact_test.go` covering edge-case slices, complex structs, and custom types.
   - Add targeted error handling, timeout, and cancellation test cases to `internal/engine/runner/runner_test.go`.
5. **Schema Governance**:
   - Ensure uniform JSON error envelopes across HTTP routes using a standardized model.

### Safety & Error Handling

- All file operations adhere to path confinement jails.
- Zero breaking changes to public Web API endpoints or dashboard Alpine state signatures.
- All files strictly verified against `<400 LOC` (`./scripts/check.sh loc 400`).

## Verification Plan

- Unit tests: `go test -count=1 ./internal/...`
- E2E smoke tests: `go test -count=1 ./tests/e2e/...`
- LOC audit: `./scripts/check.sh loc 400`
- Full quality gate: `./scripts/check.sh all` (all 6 stages passing clean).





