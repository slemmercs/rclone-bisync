package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/engine/heuristics"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

func (s *ReviewService) enrichConflictRecommendationsLocked(ctx context.Context, conflicts []models.ConflictCandidate) []models.ConflictCandidate {
	if s.HeuristicsEngine == nil || len(conflicts) == 0 {
		return conflicts
	}

	for i := range conflicts {
		if conflicts[i].Recommendation != nil {
			continue
		}

		c := heuristics.Candidate{
			Path: conflicts[i].Path,
			Local: heuristics.FileMeta{
				Path:    conflicts[i].Path,
				Size:    conflicts[i].LocalSize,
				ModTime: parseCandidateTime(conflicts[i].LocalModTime),
			},
			Remote: heuristics.FileMeta{
				Path:    conflicts[i].Path,
				Size:    conflicts[i].RemoteSize,
				ModTime: parseCandidateTime(conflicts[i].RemoteModTime),
			},
		}

		rec := s.HeuristicsEngine.Evaluate(ctx, c)
		conflicts[i].Recommendation = &rec
	}

	return conflicts
}

func (s *ReviewService) enrichDeletionRecommendationsLocked(ctx context.Context, deletions []models.DeletionCandidate) []models.DeletionCandidate {
	for i := range deletions {
		// Clean up any legacy dirty paths from past sessions
		p := deletions[i].Path
		if idx := strings.Index(p, `","`); idx != -1 {
			p = p[:idx]
		}
		if idx := strings.Index(p, `", "`); idx != -1 {
			p = p[:idx]
		}
		if idx := strings.Index(p, `","source"`); idx != -1 {
			p = p[:idx]
		}
		if idx := strings.Index(p, `", "source"`); idx != -1 {
			p = p[:idx]
		}
		p = strings.Trim(p, `"',{}\`)
		deletions[i].Path = filepath.Clean(strings.TrimSpace(p))
		deletions[i].Name = filepath.Base(deletions[i].Path)

		if s.Config != nil && s.Config.LocalPath != "" && deletions[i].Size == 0 {
			cleanBase := filepath.Clean(s.Config.LocalPath)
			basePrefix := cleanBase + string(filepath.Separator)
			fullPath := filepath.Clean(filepath.Join(cleanBase, deletions[i].Path))
			if fullPath == cleanBase || strings.HasPrefix(fullPath, basePrefix) {
				// #nosec G703 - fullPath is strictly verified to reside within cleanBase
				if fi, err := os.Stat(fullPath); err == nil {
					deletions[i].Size = fi.Size()
					deletions[i].ModTime = fi.ModTime().UTC().Format(time.RFC3339)
					deletions[i].IsDir = fi.IsDir()
				}
			}
		}

		if deletions[i].Recommendation != nil {
			continue
		}
		rec := s.evaluateDeletionRecommendation(deletions[i])
		deletions[i].Recommendation = &rec
	}
	return deletions
}

func (s *ReviewService) evaluateDeletionRecommendation(d models.DeletionCandidate) models.ConflictRecommendation {
	// Rule 1: Canary Protection
	if d.Name == "RCLONE_TEST" {
		return models.ConflictRecommendation{
			SuggestedAction: string(models.DecisionRestoreItem),
			Confidence:      1.0,
			Risk:            string(heuristics.RiskCritical),
			PrimaryReason:   "Canary integrity file; deletion strictly prohibited.",
			EvaluatedRules: []models.RuleResult{
				{
					RuleName: "CanaryProtection",
					Action:   string(models.DecisionRestoreItem),
					Risk:     string(heuristics.RiskCritical),
					Score:    1.0,
					Weight:   10.0,
					Reason:   "Canary file missing; circuit breaker protection.",
					Matched:  true,
				},
			},
		}
	}

	lowerName := strings.ToLower(d.Name)
	// Rule 2: Office temp / lock / transient files
	if strings.HasPrefix(d.Name, "~$") || strings.HasPrefix(d.Name, ".~") ||
		strings.HasSuffix(lowerName, ".tmp") || strings.HasSuffix(lowerName, ".swp") ||
		lowerName == "thumbs.db" || lowerName == ".ds_store" {
		return models.ConflictRecommendation{
			SuggestedAction: string(models.DecisionAcceptDelete),
			Confidence:      0.95,
			Risk:            string(heuristics.RiskLow),
			PrimaryReason:   "Transient system/lock file; safe to accept deletion.",
			EvaluatedRules: []models.RuleResult{
				{
					RuleName: "TempLockDeletion",
					Action:   string(models.DecisionAcceptDelete),
					Risk:     string(heuristics.RiskLow),
					Score:    0.95,
					Weight:   2.0,
					Reason:   "Transient system or lock artifact.",
					Matched:  true,
				},
			},
		}
	}

	// Rule 3: Deletion initiated on Local by user (propagating to remote)
	if d.Source == models.SideLocal {
		return models.ConflictRecommendation{
			SuggestedAction: string(models.DecisionAcceptDelete),
			Confidence:      0.85,
			Risk:            string(heuristics.RiskLow),
			PrimaryReason:   "Deleted locally by user; safe to propagate to remote.",
			EvaluatedRules: []models.RuleResult{
				{
					RuleName: "UserLocalDeletion",
					Action:   string(models.DecisionAcceptDelete),
					Risk:     string(heuristics.RiskLow),
					Score:    0.85,
					Weight:   1.5,
					Reason:   "Local origin deletion intended by user.",
					Matched:  true,
				},
			},
		}
	}

	// Rule 4: Deletion originated on Remote (targeting local)
	if d.Size == 0 {
		return models.ConflictRecommendation{
			SuggestedAction: string(models.DecisionAcceptDelete),
			Confidence:      0.80,
			Risk:            string(heuristics.RiskLow),
			PrimaryReason:   "Empty (0-byte) file deleted on remote; safe to remove.",
			EvaluatedRules: []models.RuleResult{
				{
					RuleName: "EmptyRemoteDeletion",
					Action:   string(models.DecisionAcceptDelete),
					Risk:     string(heuristics.RiskLow),
					Score:    0.80,
					Weight:   1.2,
					Reason:   "Zero-byte file removed on remote.",
					Matched:  true,
				},
			},
		}
	}

	return models.ConflictRecommendation{
		SuggestedAction: string(models.DecisionRestoreItem),
		Confidence:      0.75,
		Risk:            string(heuristics.RiskMedium),
		PrimaryReason:   "Remote deletion of local file; preserve local copy unless deletion intended.",
		EvaluatedRules: []models.RuleResult{
			{
				RuleName: "RemoteDeletionGuard",
				Action:   string(models.DecisionRestoreItem),
				Risk:     string(heuristics.RiskMedium),
				Score:    0.75,
				Weight:   1.5,
				Reason:   "Unexpected remote deletion; preserving local file is safest default.",
				Matched:  true,
			},
		},
	}
}

func parseCandidateTime(ts string) time.Time {
	if ts == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02 15:04:05", ts); err == nil {
		return t
	}
	return time.Time{}
}
