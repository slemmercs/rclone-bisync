package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/engine/heuristics"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/review"
)

// ReviewService orchestrates interactive sync review sessions, candidate staging, and execution.
type ReviewService struct {
	Config           *config.Config
	Client           *client.RcloneClient
	HeuristicsEngine *heuristics.Engine
	Audit            *AuditService
	mu               sync.Mutex
	session          *models.ReviewSession
}

// NewReviewService constructs a new review service and loads any existing persisted active session.
func NewReviewService(cfg *config.Config, rcClient *client.RcloneClient) *ReviewService {
	svc := &ReviewService{
		Config:           cfg,
		Client:           rcClient,
		HeuristicsEngine: heuristics.NewEngine(),
	}

	sessionFile := svc.sessionFilePath()
	if sessionFile != "" {
		if sess, err := review.LoadActiveSession(sessionFile); err == nil && sess != nil {
			walFile := svc.walFilePath()
			if walFile != "" {
				if entries, err := review.ReadWAL(walFile); err == nil && len(entries) > 0 {
					sess = review.ReconcileSessionWithWAL(sess, entries)
				}
			}
			svc.session = sess
		}
	}

	return svc
}

func (s *ReviewService) sessionFilePath() string {
	if s.Config == nil {
		return ""
	}
	if s.Config.ActiveReviewFile != "" {
		return s.Config.ActiveReviewFile
	}
	if s.Config.BaseDir != "" {
		return filepath.Join(s.Config.BaseDir, "active_review_session.json")
	}
	return ""
}

func (s *ReviewService) walFilePath() string {
	if s.Config == nil {
		return ""
	}
	if s.Config.ReviewDir != "" {
		return filepath.Join(s.Config.ReviewDir, "review.wal")
	}
	if s.Config.BaseDir != "" {
		return filepath.Join(s.Config.BaseDir, "review.wal")
	}
	return ""
}

func (s *ReviewService) backupsDirPath() string {
	if s.Config == nil {
		return ""
	}
	if s.Config.ReviewBackupsDir != "" {
		return s.Config.ReviewBackupsDir
	}
	if s.Config.BaseDir != "" {
		return filepath.Join(s.Config.BaseDir, "review_backups")
	}
	return ""
}

func (s *ReviewService) reviewDirPath() string {
	if s.Config == nil {
		return ""
	}
	if s.Config.ReviewDir != "" {
		return s.Config.ReviewDir
	}
	if s.Config.BaseDir != "" {
		return filepath.Join(s.Config.BaseDir, "review_sessions")
	}
	return ""
}

// HasBlockingReview returns true if an unresolved review session exists.
func (s *ReviewService) HasBlockingReview() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.session == nil {
		sessionFile := s.sessionFilePath()
		if sessionFile != "" {
			if sess, err := review.LoadActiveSession(sessionFile); err == nil && sess != nil {
				s.session = sess
			}
		}
	}

	if s.session == nil {
		return false
	}

	switch s.session.Status {
	case models.ReviewStatusCreated, models.ReviewStatusReviewing, models.ReviewStatusApplying, models.ReviewStatusFailed:
		return true
	default:
		return false
	}
}

// GetActiveSession returns the active session, attempting auto-discovery from pending deletions if none exists.
func (s *ReviewService) GetActiveSession(ctx context.Context) (*models.ReviewSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.session != nil {
		s.session.Conflicts = s.enrichConflictRecommendationsLocked(ctx, s.session.Conflicts)
		s.session.Deletions = s.enrichDeletionRecommendationsLocked(ctx, s.session.Deletions)
		return s.session, nil
	}

	sessionFile := s.sessionFilePath()
	if sessionFile != "" {
		if sess, err := review.LoadActiveSession(sessionFile); err == nil && sess != nil {
			s.session = sess
			s.session.Conflicts = s.enrichConflictRecommendationsLocked(ctx, s.session.Conflicts)
			s.session.Deletions = s.enrichDeletionRecommendationsLocked(ctx, s.session.Deletions)
			return s.session, nil
		}
	}

	// If pending deletions file exists and indicates pending changes, auto-discover
	if s.Config != nil && s.Config.PendingDeletionsFile != "" {
		if _, err := os.Stat(s.Config.PendingDeletionsFile); err == nil {
			sess, err := s.discoverCandidatesLocked(ctx, "TOO_MANY_DELETES", "Safety abort: pending deletions require review")
			if err == nil && sess != nil {
				if len(sess.Deletions) == 0 && len(sess.Conflicts) == 0 {
					// Clean up empty pending deletions file and empty session so user is not stuck
					_ = os.Remove(s.Config.PendingDeletionsFile)
					if sessionFile != "" {
						_ = os.Remove(sessionFile)
					}
					s.session = nil
					return nil, nil
				}
				return sess, nil
			}
		}
	}

	return nil, nil
}

// DiscoverSession forces a candidate discovery pass and creates an active review session.
func (s *ReviewService) DiscoverSession(ctx context.Context, errType, errMessage string) (*models.ReviewSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.discoverCandidatesLocked(ctx, errType, errMessage)
}

func (s *ReviewService) discoverCandidatesLocked(ctx context.Context, errType, errMessage string) (*models.ReviewSession, error) {
	if s.Config == nil {
		return nil, fmt.Errorf("review service config is nil")
	}

	var logLines []string
	if s.Config.BaseDir != "" {
		daemonLog := filepath.Join(s.Config.BaseDir, "rclone-daemon.log")
		if data, err := os.ReadFile(daemonLog); err == nil {
			logLines = append(logLines, strings.Split(string(data), "\n")...)
		}
	}
	if s.Config.LogFile != "" {
		if data, err := os.ReadFile(s.Config.LogFile); err == nil {
			logLines = append(logLines, strings.Split(string(data), "\n")...)
		}
	}

	deletions, conflicts, _ := review.DiscoverCandidates(
		logLines,
		s.Config.PendingDeletionsFile,
		s.Config.LocalPath,
	)

	conflicts = s.enrichConflictRecommendationsLocked(ctx, conflicts)
	deletions = s.enrichDeletionRecommendationsLocked(ctx, deletions)

	sess := review.NewSession(errType, errMessage, deletions, conflicts)
	_ = review.Transition(sess, models.ReviewStatusReviewing)

	sessionFile := s.sessionFilePath()
	if sessionFile != "" {
		if err := review.SaveActiveSession(sessionFile, sess); err != nil {
			return nil, fmt.Errorf("failed to persist active session: %w", err)
		}
	}

	s.session = sess
	return s.session, nil
}

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
