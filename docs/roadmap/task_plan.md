
# Task Plan: Heuristic Conflict Recommendation Engine

## Goal
Implement a deterministic, pure-Go heuristic conflict recommendation engine that evaluates file conflict pairs, assigns confidence scores and risk levels, and presents actionable recommendations (`keep-local`, `keep-remote`, `keep-both`, `manual-review`) in the Interactive Review Modal, with one-click batch resolution for safe recommendations.

## Tasks
- [x] Task 1: Heuristic Domain Contracts & Recommendation Models
  - **Scope**: Define `RuleResult`, `ConflictRecommendation`, `BatchReviewActionRequest` in `internal/models/review.go`, enrich `ConflictCandidate`.
  - **Verification**: `go test -count=1 ./internal/models`
- [x] Task 2: Pure-Go Analyzers & Scoring Engine Core
  - **Scope**: Implement `internal/engine/heuristics/` (`types.go`, `analyzers.go`, `engine.go`) with worker pool ($G=4$) and scoring aggregation.
  - **Verification**: `go test -count=1 ./internal/engine/heuristics/...`
- [x] Task 3: Comprehensive Unit Test Suite
  - **Scope**: Add unit tests in `internal/engine/heuristics/engine_test.go` covering all analyzers, truncation overrides, and concurrency (>85% coverage).
  - **Verification**: `go test -race -count=1 ./internal/engine/heuristics/...`
- [x] Task 4: Service Facade Integration & Batch Application
  - **Scope**: Wire `heuristics.Engine` into `internal/services/review_service.go`, auto-enrich conflicts on discovery, implement `ApplyBatchRecommendations`.
  - **Verification**: `go test -count=1 ./internal/services`
- [x] Task 5: Server REST Batch Endpoint
  - **Scope**: Implement `POST /api/review/action-batch` in `internal/server/handlers_review.go` and `server.go`.
  - **Verification**: `go test -count=1 ./internal/server`
- [x] Task 6: Frontend Recommendation Badges & Batch Trigger
  - **Scope**: Update `dashboard/frontend/js/review.js` and `index.html` to display recommendation badges, reasoning tooltips, and "Apply High-Confidence Recommendations" batch button.
  - **Verification**: `./scripts/check.sh`
- [x] Task 7: Full Quality Gate & End-to-End Verification
  - **Scope**: Run full `./scripts/check.sh` (>80% coverage across all packages, clean static analysis, native and cross-platform builds).
  - **Verification**: `./scripts/check.sh`

## Next Up
- **Milestone Complete: Heuristic Conflict Recommendation Engine Ready for Review / Merge**

## Status Updates
- 2026-10-23: Initialized Heuristic Conflict Recommendation Engine plan on branch `feat/heuristic-conflict-engine`. Updated `docs/design/`, `ROADMAP.md`, and `backlog.md` with delivered milestones.
- 2026-10-23: Completed Task 1: Defined `RuleResult`, `ConflictRecommendation`, `BatchReviewActionRequest`, `BatchReviewActionResponse` in `internal/models/review.go` and enriched `ConflictCandidate`. Verified via `go test -count=1 ./internal/models`.
- 2026-10-23: Completed Task 2 & Task 3: Implemented `types.go`, `analyzers.go`, `analyzers_ext.go`, `engine.go`, and comprehensive unit tests in `engine_test.go` with 94.0% statement coverage and zero race conditions under `go test -race`.
- 2026-10-23: Completed Task 4: Wired `heuristics.Engine` into `ReviewService`, automated conflict enrichment on discovery and retrieval, and implemented `ApplyBatchRecommendations`. Verified via `go test -count=1 ./internal/services/...`.
- 2026-10-23: Completed Task 5: Implemented `POST /api/review/action-batch` endpoint in `internal/server/handlers_review.go`, registered route in `server.go`, and added tests in `server_test.go`. Verified via `go test -count=1 ./internal/server/...`.
- 2026-10-23: Completed Task 6 & Task 7: Integrated recommendation badges, confidence meters, and one-click batch trigger in `dashboard/frontend/` (`review.js` and `index.html`). Full quality gate passed with 83.2% statement coverage, clean formatting, zero vet warnings, and successful native + ARM64 compilation.
- 2026-10-23: Completed Phase 4 (Interactive Sync Review & Deletion Diff Visualizer) and maintenance milestone merged to `main`.



