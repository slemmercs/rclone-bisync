package heuristics

import (
	"context"
	"math"
	"sync"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// DefaultConcurrency is the bounded worker pool size for offline candidate evaluations.
const DefaultConcurrency = 4

// Engine orchestrates heuristic analysis over conflict candidates.
type Engine struct {
	analyzers   []Analyzer
	concurrency int
}

// DefaultAnalyzers returns the standard pipeline of deterministic heuristic analyzers.
func DefaultAnalyzers() []Analyzer {
	return []Analyzer{
		&TruncationAnalyzer{},
		&TempLockAnalyzer{},
		&HashMatchAnalyzer{},
		&ClockSkewAnalyzer{},
		&SizeDivergenceAnalyzer{},
		&TimestampRecencyAnalyzer{},
	}
}

// NewEngine constructs an Engine with the provided or default analyzers.
func NewEngine(analyzers ...Analyzer) *Engine {
	if len(analyzers) == 0 {
		analyzers = DefaultAnalyzers()
	}
	return &Engine{
		analyzers:   analyzers,
		concurrency: DefaultConcurrency,
	}
}

// Evaluate runs the analyzer pipeline on a single conflict candidate and produces an aggregated recommendation.
func (e *Engine) Evaluate(ctx context.Context, c Candidate) models.ConflictRecommendation {
	var evaluatedRules []models.RuleResult
	var totalWeightedScore float64
	var totalWeight float64

	hasCritical := false
	maxRisk := RiskLow
	var primaryReason string
	var highestWeightRule models.RuleResult

	for _, a := range e.analyzers {
		if ctx.Err() != nil {
			break
		}

		res, err := a.Analyze(ctx, c)
		if err != nil || !res.Matched {
			continue
		}

		evaluatedRules = append(evaluatedRules, res)
		totalWeightedScore += res.Score * res.Weight
		totalWeight += res.Weight

		// Track maximum risk
		if res.Risk == string(RiskCritical) {
			hasCritical = true
			maxRisk = RiskCritical
			primaryReason = res.Reason
		} else if res.Risk == string(RiskHigh) && maxRisk != RiskCritical {
			maxRisk = RiskHigh
		} else if res.Risk == string(RiskMedium) && maxRisk != RiskCritical && maxRisk != RiskHigh {
			maxRisk = RiskMedium
		}

		if res.Weight > highestWeightRule.Weight && !hasCritical {
			highestWeightRule = res
			primaryReason = res.Reason
		}
	}

	// Specialized high-weight low-risk rule (e.g. TempLock, HashMatch, ClockSkew) overrides generic recency Medium risk
	if maxRisk == RiskMedium && highestWeightRule.Weight > 1.0 && highestWeightRule.Risk == string(RiskLow) {
		maxRisk = RiskLow
	}

	if primaryReason == "" {
		if len(evaluatedRules) > 0 {
			primaryReason = evaluatedRules[0].Reason
		} else {
			primaryReason = "Both copies altered; manual inspection recommended."
		}
	}

	// 1. Critical Risk Circuit Breaker: override and force manual review
	if hasCritical {
		return models.ConflictRecommendation{
			SuggestedAction: string(ActionManual),
			Confidence:      0.95,
			Risk:            string(RiskCritical),
			PrimaryReason:   primaryReason,
			EvaluatedRules:  evaluatedRules,
		}
	}

	// 2. Compute normalized score and confidence
	var normalizedScore float64
	var confidence float64
	if totalWeight > 0 {
		normalizedScore = totalWeightedScore / totalWeight
		// Clamp to [-1.0, +1.0]
		if normalizedScore < -1.0 {
			normalizedScore = -1.0
		} else if normalizedScore > 1.0 {
			normalizedScore = 1.0
		}
		confidence = math.Abs(normalizedScore)
	}

	// 3. Round confidence to 2 decimal places
	confidence = math.Round(confidence*100) / 100

	// 4. Derive recommended action based on normalized score threshold
	var suggestedAction string
	if confidence < 0.40 {
		suggestedAction = string(ActionKeepBoth)
		if primaryReason == "" {
			primaryReason = "Low confidence divergence; keeping both versions recommended."
		}
	} else if normalizedScore <= -0.40 {
		suggestedAction = string(ActionKeepLocal)
	} else if normalizedScore >= 0.40 {
		suggestedAction = string(ActionKeepRemote)
	} else {
		suggestedAction = string(ActionKeepBoth)
	}

	return models.ConflictRecommendation{
		SuggestedAction: suggestedAction,
		Confidence:      confidence,
		Risk:            string(maxRisk),
		PrimaryReason:   primaryReason,
		EvaluatedRules:  evaluatedRules,
	}
}

// EvaluateBatch evaluates multiple conflict candidates concurrently across a bounded worker pool.
func (e *Engine) EvaluateBatch(ctx context.Context, candidates []Candidate) map[string]models.ConflictRecommendation {
	results := make(map[string]models.ConflictRecommendation, len(candidates))
	if len(candidates) == 0 {
		return results
	}

	concurrency := e.concurrency
	if concurrency <= 0 {
		concurrency = DefaultConcurrency
	}

	var mu sync.Mutex
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for _, cand := range candidates {
		wg.Add(1)
		go func(c Candidate) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			rec := e.Evaluate(ctx, c)

			mu.Lock()
			results[c.Path] = rec
			mu.Unlock()
		}(cand)
	}

	wg.Wait()
	return results
}
