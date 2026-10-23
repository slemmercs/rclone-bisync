package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/review"
)

// ReviewService orchestrates interactive sync review sessions, candidate staging, and execution.
type ReviewService struct {
	Config  *config.Config
	Client  *client.RcloneClient
	mu      sync.Mutex
	session *models.ReviewSession
}

// NewReviewService constructs a new review service and loads any existing persisted active session.
func NewReviewService(cfg *config.Config, rcClient *client.RcloneClient) *ReviewService {
	svc := &ReviewService{
		Config: cfg,
		Client: rcClient,
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
		return s.session, nil
	}

	sessionFile := s.sessionFilePath()
	if sessionFile != "" {
		if sess, err := review.LoadActiveSession(sessionFile); err == nil && sess != nil {
			s.session = sess
			return s.session, nil
		}
	}

	// If pending deletions file exists and indicates pending changes, auto-discover
	if s.Config != nil && s.Config.PendingDeletionsFile != "" {
		if _, err := os.Stat(s.Config.PendingDeletionsFile); err == nil {
			return s.discoverCandidatesLocked(ctx, "TOO_MANY_DELETES", "Safety abort: pending deletions require review")
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
	if s.Config.LogFile != "" {
		if data, err := os.ReadFile(s.Config.LogFile); err == nil {
			logLines = strings.Split(string(data), "\n")
		}
	}

	deletions, conflicts, _ := review.DiscoverCandidates(
		logLines,
		s.Config.PendingDeletionsFile,
		s.Config.LocalPath,
	)

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

// StageCandidateDecision stages a resolution decision for a specific candidate.
func (s *ReviewService) StageCandidateDecision(sessionID, candidateID string, decision models.ActionDecision) (*models.ReviewSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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

	if s.session == nil || s.session.ID != sessionID {
		return nil, fmt.Errorf("review session %s not found or inactive", sessionID)
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

	appliedSession := s.session
	s.session = nil
	return appliedSession, nil
}

// RollbackReviewSession executes reverse-journal compensation for executed actions.
func (s *ReviewService) RollbackReviewSession(ctx context.Context, sessionID string) (*models.ReviewSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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

	return s.session, err
}

// DismissReviewSession transitions the session to DISMISSED and archives it without executing mutations.
func (s *ReviewService) DismissReviewSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
