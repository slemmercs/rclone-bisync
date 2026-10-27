package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

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
		// #nosec G304 - reading configured daemon log
		if data, err := os.ReadFile(filepath.Clean(daemonLog)); err == nil {
			logLines = append(logLines, strings.Split(string(data), "\n")...)
		}
	}
	if s.Config.LogFile != "" {
		// #nosec G304 - reading configured log file
		if data, err := os.ReadFile(filepath.Clean(s.Config.LogFile)); err == nil {
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
