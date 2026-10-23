# Implementation Plan - Heuristic Conflict Recommendation Engine

[Overview]
Implement a deterministic pure-Go conflict recommendation engine that enriches Interactive Sync Review sessions with confidence-scored resolution advice and enables safe one-click batch application.

When `rclone bisync` aborts due to bidirectional changes, Bisync Hub traps the abort and generates a `ReviewSession`. Previously, conflict candidates only displayed raw modification timestamps and byte sizes, leaving users to guess the safest action. The Heuristic Conflict Recommendation Engine evaluates candidate file pairs concurrently across a chain of deterministic pure-Go analyzers (temporary locks, content hashes, 0-byte/truncation checks, clock skew/DST drift, size divergence, and timestamp recency). It generates normalized confidence scores and risk tiers (`LOW`, `MEDIUM`, `HIGH`, `CRITICAL`), providing users with clear operational rationale and enabling high-confidence recommendations to be batch-applied safely via `POST /api/review/action-batch`.

[Types]
Type system changes in `internal/models/review.go`:
```go
// RuleResult captures the evaluation output of a single heuristic rule.
type RuleResult struct {
	RuleName string  `json:"rule_name"`
	Action   string  `json:"action"` // keep-local, keep-remote, keep-both, manual-review
	Score    float64 `json:"score"`  // Range: -1.0 (Local) to +1.0 (Remote)
	Weight   float64 `json:"weight"`
	Risk     string  `json:"risk"`   // LOW, MEDIUM, HIGH, CRITICAL
	Reason   string  `json:"reason"`
	Matched  bool    `json:"matched"`
}

// ConflictRecommendation aggregates analyzer evaluations for a conflict candidate.
type ConflictRecommendation struct {
	SuggestedAction string       `json:"suggested_action"`
	Confidence      float64      `json:"confidence"` // Normalized 0.00 to 1.00
	Risk            string       `json:"risk"`       // LOW, MEDIUM, HIGH, CRITICAL
	PrimaryReason   string       `json:"primary_reason"`
	EvaluatedRules  []RuleResult `json:"evaluated_rules"`
}

// BatchReviewActionRequest defines criteria for auto-staging recommendations.
type BatchReviewActionRequest struct {
	MinConfidence  float64 `json:"min_confidence"`
	MaxAllowedRisk string  `json:"max_allowed_risk"` // LOW, MEDIUM
}

// BatchReviewActionResponse summarizes staged batch actions.
type BatchReviewActionResponse struct {
	StagedCount int    `json:"staged_count"`
	Message     string `json:"message"`
}
```


[Files]
New files:
- `/home/csaba/repos/rclone-bisync/internal/engine/heuristics/types.go`: Domain models, action/risk enums, and `Analyzer` interface.
- `/home/csaba/repos/rclone-bisync/internal/engine/heuristics/analyzers.go`: Concrete analyzer implementations (`TempLockAnalyzer`, `HashMatchAnalyzer`, `TruncationAnalyzer`, `ClockSkewAnalyzer`, `SizeDivergenceAnalyzer`, `TimestampRecencyAnalyzer`).
- `/home/csaba/repos/rclone-bisync/internal/engine/heuristics/engine.go`: Engine coordinator, worker pool ($G=4$), and weighted scoring aggregator.
- `/home/csaba/repos/rclone-bisync/internal/engine/heuristics/engine_test.go`: Unit test suite verifying analyzers and concurrency.

Modified files:
- `/home/csaba/repos/rclone-bisync/internal/models/review.go`: Add recommendation DTOs and batch action structs, update `ConflictCandidate`.
- `/home/csaba/repos/rclone-bisync/internal/services/review_service.go`: Wire heuristics into discovery and add `ApplyBatchRecommendations`.
- `/home/csaba/repos/rclone-bisync/internal/server/handlers_review.go`: Implement `POST /api/review/action-batch`.
- `/home/csaba/repos/rclone-bisync/internal/server/server.go`: Mount `/api/review/action-batch` route.
- `/home/csaba/repos/rclone-bisync/dashboard/frontend/js/review.js`: Recommendation badges and batch button.
- `/home/csaba/repos/rclone-bisync/dashboard/frontend/index.html`: Review modal template updates.

[Functions]
New functions:
- `NewEngine(analyzers ...Analyzer) *Engine` (`internal/engine/heuristics/engine.go`): Constructor with default or custom analyzer pipeline.
- `(*Engine).Evaluate(ctx context.Context, c Candidate) models.ConflictRecommendation` (`internal/engine/heuristics/engine.go`): Runs analyzer chain, applies weighted scoring formula, and evaluates circuit breakers.
- `(*Engine).EvaluateBatch(ctx context.Context, candidates []Candidate) map[string]models.ConflictRecommendation` (`internal/engine/heuristics/engine.go`): Concurrent worker pool execution over candidate slices.
- `(*ReviewService).ApplyBatchRecommendations(ctx context.Context, minConfidence float64, maxRisk string) (int, error)` (`internal/services/review_service.go`): Finds all unresolved conflicts matching confidence and risk criteria and stages decisions.
- `(*Server).handleReviewActionBatch(w http.ResponseWriter, r *http.Request)` (`internal/server/handlers_review.go`): HTTP handler parsing `BatchReviewActionRequest` and returning `BatchReviewActionResponse`.

Modified functions:
- `(*ReviewService).DiscoverCandidates(ctx context.Context, output string, profile string)` (`internal/services/review_service.go`): Enrich discovered conflict candidates with recommendations.
- `(*ReviewService).GetActiveSession(ctx context.Context)` (`internal/services/review_service.go`): Ensure recommendations are present on active session conflicts.

[Classes]
(In Go, structs with methods):
- `Engine` (`internal/engine/heuristics/engine.go`): Orchestrator managing analyzer pipelines and worker pool.
- `TempLockAnalyzer` (`internal/engine/heuristics/analyzers.go`): Evaluates lock and temporary files.
- `HashMatchAnalyzer` (`internal/engine/heuristics/analyzers.go`): Detects identical content hashes across endpoints.
- `TruncationAnalyzer` (`internal/engine/heuristics/analyzers.go`): Flags 0-byte or >90% size drops and enforces `RiskCritical`.
- `ClockSkewAnalyzer` (`internal/engine/heuristics/analyzers.go`): Identifies exact 1h/2h DST time jumps and $\le 2\text{s}$ filesystem drift.
- `SizeDivergenceAnalyzer` (`internal/engine/heuristics/analyzers.go`): Flags >20% binary file size growth/shrinkage.
- `TimestampRecencyAnalyzer` (`internal/engine/heuristics/analyzers.go`): Computes recency score from modification timestamp deltas.

[Dependencies]
- Pure Go standard library: `context`, `math`, `path/filepath`, `strings`, `sync`, `time`.
- No new external packages; zero CGO; single-binary distribution preserved.

[Testing]
- `internal/engine/heuristics/engine_test.go`:
  - `TestEngine_TruncationCriticalRisk`: Ensures 0-byte or >90% dropped size forces `RiskCritical` and overrides timestamp recency.
  - `TestEngine_TempLock`: Ensures temporary lockfiles recommend keeping the primary document.
  - `TestEngine_HashMatch`: Ensures identical content hashes result in high confidence `keep-local`.
  - `TestEngine_ClockSkewDST`: Checks that 3600s/7200s timestamps are handled cleanly.
  - `TestEngine_SizeDivergence`: Validates binary assets with diverging sizes require manual review.
  - `TestEngine_Concurrency`: Concurrently evaluates 100 conflict pairs with `go test -race`.
- `internal/server/handlers_review_test.go`:
  - Tests `POST /api/review/action-batch` with valid and invalid confidence/risk filters.
- Repository Quality Gate:
  - `./scripts/check.sh` verifying formatting, `go vet`, and >80% statement coverage across all packages.

[Implementation Order]
1. Update `internal/models/review.go` with recommendation models and batch request/response DTOs.
2. Implement `internal/engine/heuristics/types.go`, `analyzers.go`, and `engine.go`.
3. Create unit tests in `internal/engine/heuristics/engine_test.go` and verify with `go test -race -count=1 ./internal/engine/heuristics/...`.
4. Integrate `heuristics.Engine` into `internal/services/review_service.go` and implement `ApplyBatchRecommendations`.
5. Add `POST /api/review/action-batch` endpoint in `internal/server/handlers_review.go` and register route in `server.go`.
6. Update frontend `dashboard/frontend/js/review.js` and `dashboard/frontend/index.html` with recommendation badges and batch trigger.
7. Run `./scripts/check.sh` to ensure full linting, cross-compilation, and >80% test coverage.



