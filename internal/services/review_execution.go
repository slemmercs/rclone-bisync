package services

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/engine/heuristics"
	"github.com/slemmercs/rclone-bisync/internal/engine/review"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

// ApplyBatchRecommendations auto-stages resolutions for conflict candidates matching confidence and risk criteria.
func (s *ReviewService) ApplyBatchRecommendations(ctx context.Context, sessionID string, minConfidence float64, maxRisk string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureSessionLoadedLocked()
	if s.session == nil {
		return 0, fmt.Errorf("no active review session found")
	}
	if sessionID != "" && s.session.ID != sessionID {
		return 0, fmt.Errorf("review session %s not found or inactive", sessionID)
	}

	if minConfidence <= 0 {
		minConfidence = 0.70
	}
	if maxRisk == "" {
		maxRisk = "MEDIUM"
	}
	maxRiskUpper := strings.ToUpper(maxRisk)

	s.session.Conflicts = s.enrichConflictRecommendationsLocked(ctx, s.session.Conflicts)
	s.session.Deletions = s.enrichDeletionRecommendationsLocked(ctx, s.session.Deletions)

	stagedCount := 0
	for i := range s.session.Conflicts {
		cand := &s.session.Conflicts[i]
		if cand.Resolved || cand.Recommendation == nil {
			continue
		}

		rec := cand.Recommendation
		if rec.Risk == string(heuristics.RiskCritical) || rec.SuggestedAction == string(heuristics.ActionManual) {
			continue
		}

		if maxRiskUpper == string(heuristics.RiskLow) && rec.Risk != string(heuristics.RiskLow) {
			continue
		}
		if maxRiskUpper == string(heuristics.RiskMedium) && rec.Risk != string(heuristics.RiskLow) && rec.Risk != string(heuristics.RiskMedium) {
			continue
		}

		if rec.Confidence < minConfidence {
			continue
		}

		decision := models.ActionDecision(rec.SuggestedAction)
		if err := review.StageDecision(s.session, cand.ID, decision); err == nil {
			stagedCount++
		}
	}

	for i := range s.session.Deletions {
		cand := &s.session.Deletions[i]
		if cand.Resolved || cand.Recommendation == nil {
			continue
		}

		rec := cand.Recommendation
		if rec.Risk == string(heuristics.RiskCritical) || rec.SuggestedAction == string(heuristics.ActionManual) {
			continue
		}

		if maxRiskUpper == string(heuristics.RiskLow) && rec.Risk != string(heuristics.RiskLow) {
			continue
		}
		if maxRiskUpper == string(heuristics.RiskMedium) && rec.Risk != string(heuristics.RiskLow) && rec.Risk != string(heuristics.RiskMedium) {
			continue
		}

		if rec.Confidence < minConfidence {
			continue
		}

		decision := models.ActionDecision(rec.SuggestedAction)
		if err := review.StageDecision(s.session, cand.ID, decision); err == nil {
			stagedCount++
		}
	}

	if stagedCount > 0 {
		sessionFile := s.sessionFilePath()
		if sessionFile != "" {
			if err := review.SaveActiveSession(sessionFile, s.session); err != nil {
				return stagedCount, fmt.Errorf("failed to persist active session after batch staging: %w", err)
			}
		}
	}

	return stagedCount, nil
}

func (s *ReviewService) ensureSessionLoadedLocked() {
	if s.session != nil {
		return
	}
	sessionFile := s.sessionFilePath()
	if sessionFile != "" {
		if sess, err := review.LoadActiveSession(sessionFile); err == nil && sess != nil {
			s.session = sess
		}
	}
}

// StageCandidateDecision stages a resolution decision for a specific candidate.
func (s *ReviewService) StageCandidateDecision(sessionID, candidateID string, decision models.ActionDecision) (*models.ReviewSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureSessionLoadedLocked()
	if s.session == nil || s.session.ID != sessionID {
		return nil, fmt.Errorf("review session %s not found or inactive", sessionID)
	}

	if err := review.StageDecision(s.session, candidateID, decision); err != nil {
		return nil, err
	}

	sessionFile := s.sessionFilePath()
	if sessionFile != "" {
		if err := review.SaveActiveSession(sessionFile, s.session); err != nil {
			return nil, fmt.Errorf("failed to persist active session: %w", err)
		}
	}

	return s.session, nil
}

// ApplyReviewSession applies staged candidate decisions using the Saga resolution executor.
func (s *ReviewService) ApplyReviewSession(ctx context.Context, sessionID string) (*models.ReviewSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureSessionLoadedLocked()
	if s.session == nil || s.session.ID != sessionID {
		return nil, fmt.Errorf("review session %s not found or inactive", sessionID)
	}

	// Auto-mark any remaining un-decided candidates as 'skip' when applying resolutions
	for i := range s.session.Deletions {
		if !s.session.Deletions[i].Resolved || s.session.Deletions[i].Decision == "" {
			s.session.Deletions[i].Decision = models.DecisionSkip
			s.session.Deletions[i].Resolved = true
		}
	}
	for i := range s.session.Conflicts {
		if !s.session.Conflicts[i].Resolved || s.session.Conflicts[i].Decision == "" {
			s.session.Conflicts[i].Decision = models.DecisionSkip
			s.session.Conflicts[i].Resolved = true
		}
	}

	if !review.AllResolved(s.session) {
		return nil, fmt.Errorf("cannot apply review session: unresolved candidates remain")
	}

	executor := review.NewResolutionExecutor(
		s.Config.LocalPath,
		s.Config.RemotePath,
		s.backupsDirPath(),
		s.walFilePath(),
		s.Client,
	)

	execErr := executor.ExecuteSession(ctx, s.session)

	sessionFile := s.sessionFilePath()
	if sessionFile != "" {
		_ = review.SaveActiveSession(sessionFile, s.session)
	}

	if execErr != nil {
		return s.session, fmt.Errorf("session execution failed: %w", execErr)
	}

	// Execution succeeded: archive session and clean up pending state
	reviewDir := s.reviewDirPath()
	if reviewDir != "" && sessionFile != "" {
		_ = review.ArchiveSession(reviewDir, sessionFile, s.session)
	}

	if s.Config.PendingDeletionsFile != "" {
		_ = os.Remove(s.Config.PendingDeletionsFile)
	}
	walFile := s.walFilePath()
	if walFile != "" {
		_ = os.Remove(walFile)
	}

	if s.Audit != nil && s.session != nil {
		profileID := "default"
		if s.Config != nil && s.Config.RemotePath != "" {
			profileID = s.Config.RemotePath
		}
		for _, a := range s.session.Actions {
			if a.Status == "completed" {
				act := audit.ActionConflictResolved
				if a.Type == models.DecisionAcceptDelete {
					act = audit.ActionDeletionAccepted
				} else if a.Type == models.DecisionRestoreItem {
					act = audit.ActionDeletionRestored
				}
				_, _ = s.Audit.RecordEvent(profileID, act, "USER", map[string]any{
					"item_path":  a.ItemPath,
					"decision":   string(a.Type),
					"session_id": s.session.ID,
					"is_dir":     a.IsDir,
				})
			}
		}
	}

	appliedSession := s.session
	s.session = nil
	return appliedSession, nil
}

// RollbackReviewSession executes reverse-journal compensation for executed actions.
func (s *ReviewService) RollbackReviewSession(ctx context.Context, sessionID string) (*models.ReviewSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureSessionLoadedLocked()
	if s.session == nil || s.session.ID != sessionID {
		return nil, fmt.Errorf("review session %s not found or inactive", sessionID)
	}

	executor := review.NewResolutionExecutor(
		s.Config.LocalPath,
		s.Config.RemotePath,
		s.backupsDirPath(),
		s.walFilePath(),
		s.Client,
	)

	err := review.RollbackSession(ctx, s.session, executor)

	sessionFile := s.sessionFilePath()
	if sessionFile != "" {
		_ = review.SaveActiveSession(sessionFile, s.session)
	}

	if err == nil && s.Audit != nil && s.session != nil {
		profileID := "default"
		if s.Config != nil && s.Config.RemotePath != "" {
			profileID = s.Config.RemotePath
		}
		_, _ = s.Audit.RecordEvent(profileID, audit.ActionRollbackExecuted, "USER", map[string]any{
			"session_id": s.session.ID,
		})
	}

	return s.session, err
}

// DismissReviewSession transitions the session to DISMISSED and archives it without executing mutations.
func (s *ReviewService) DismissReviewSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureSessionLoadedLocked()
	if s.session == nil || s.session.ID != sessionID {
		return fmt.Errorf("review session %s not found or inactive", sessionID)
	}

	if err := review.Transition(s.session, models.ReviewStatusDismissed); err != nil {
		return err
	}

	sessionFile := s.sessionFilePath()
	reviewDir := s.reviewDirPath()
	if reviewDir != "" && sessionFile != "" {
		_ = review.ArchiveSession(reviewDir, sessionFile, s.session)
	}

	walFile := s.walFilePath()
	if walFile != "" {
		_ = os.Remove(walFile)
	}

	if s.Audit != nil && s.session != nil {
		profileID := "default"
		if s.Config != nil && s.Config.RemotePath != "" {
			profileID = s.Config.RemotePath
		}
		_, _ = s.Audit.RecordEvent(profileID, audit.ActionReviewDismissed, "USER", map[string]any{
			"session_id": s.session.ID,
		})
	}

	s.session = nil
	return nil
}

// PruneBackups removes review backups older than configured TTL.
func (s *ReviewService) PruneBackups() (int, error) {
	backupsDir := s.backupsDirPath()
	if backupsDir == "" || s.Config == nil {
		return 0, nil
	}

	ttlHours := s.Config.ReviewBackupTTLHours
	if ttlHours <= 0 {
		ttlHours = 48.0
	}

	return review.PruneExpiredBackups(backupsDir, time.Duration(ttlHours*float64(time.Hour)))
}
