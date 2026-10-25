# Implementation Plan: Comprehensive Service Hardening & Review Modularization

## Proposed Changes

1. **Modularize `internal/services/review_service.go` (722 LOC)** into:
   - `internal/services/review_service.go` (~220 LOC): Core struct, constructor, lifecycle state, path helpers, and candidate discovery.
   - `internal/services/review_recommendations.go` (~190 LOC): Candidate recommendation enrichment (heuristics engine integration, conflict enrichment, deletion recommendation rules, and timestamp parser `parseCandidateTime`).
   - `internal/services/review_execution.go` (~310 LOC): Candidate staging, batch application, Saga execution, rollback, dismissal, and backup pruning.

2. **Systematic Service Hardening & Targeted Tests**:
   - `ReviewService` (Target: >90%): Direct unit tests for `parseCandidateTime` formats and fallbacks, `RollbackReviewSession` rollback execution & audit recording, and missing/uninitialized session error branches.
   - `CanaryService` (Target: >95%): Jail escape path traversal (`..`), directory-as-canary validation, and remote client outage / touch failure paths.
   - `LogService` (Target: >90%): `extractNestedJSONLogs` newline-delimited fallback parser and malformed JSON branches.
   - `ExplorerService` (Target: >92%): Canary deletion safety circuit breaker (`RCLONE_TEST` block) and local path traversal confinement violations.
   - `SyncService` & Handlers (Target: >92% / >95%): Active review delegation facade methods (`StageReviewDecision`, `ApplyReviewSession`, `RollbackReviewSession`), unsupported action handling, and HTTP review endpoints (`handleReviewExecute`, `handleReviewRollback`).

### Affected Components & Packages

- `internal/services/review_service.go`: Modularization of core struct and discovery.
- `internal/services/review_recommendations.go`: Modularized heuristics & rules.
- `internal/services/review_execution.go`: Modularized staging, batch, apply, rollback, and dismissal.
- `internal/services/review_service_test.go`: Added test coverage for rollback, time parsing, and edge cases.
- `internal/services/canary_service_test.go`: Added test coverage for jail escapes and remote failures.
- `internal/services/log_service_test.go`: Added test coverage for nested JSON fallbacks and scanner edges.
- `internal/services/explorer_service_test.go`: Added test coverage for canary deletion protection and jail boundaries.
- `internal/services/sync_service_test.go`: Added test coverage for review facade delegation and action fallbacks.
- `internal/server/server_test.go`: Added test coverage for review execute/rollback HTTP handlers.

## Verification Plan

- Run targeted test per service: `go test -count=1 ./internal/services -run ...`
- Run full pre-flight verification: `./scripts/check.sh`
- Checkpoint commits per service with pause for context compaction.
