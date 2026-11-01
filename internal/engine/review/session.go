package review

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// GenerateSessionID creates a unique, cryptographically random session identifier.
func GenerateSessionID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("rev-%d", time.Now().UnixNano())
	}
	return "rev-" + hex.EncodeToString(b)
}

// NewSession initializes a new ReviewSession in CREATED state.
func NewSession(originError, message string, deletions []models.DeletionCandidate, conflicts []models.ConflictCandidate) *models.ReviewSession {
	now := time.Now().UTC()
	return &models.ReviewSession{
		ID:          GenerateSessionID(),
		CreatedAt:   now,
		UpdatedAt:   now,
		Status:      models.ReviewStatusCreated,
		Deletions:   deletions,
		Conflicts:   conflicts,
		Actions:     make([]models.ResolutionAction, 0),
		OriginError: originError,
		Message:     message,
	}
}

// CanTransition validates whether the FSM transition from current to target is allowed.
func CanTransition(current, target models.ReviewSessionStatus) bool {
	if current == target {
		return true
	}

	switch current {
	case models.ReviewStatusCreated:
		return target == models.ReviewStatusReviewing || target == models.ReviewStatusDismissed

	case models.ReviewStatusReviewing:
		return target == models.ReviewStatusApplying || target == models.ReviewStatusDismissed

	case models.ReviewStatusApplying:
		return target == models.ReviewStatusReady || target == models.ReviewStatusFailed || target == models.ReviewStatusRollback

	case models.ReviewStatusReady:
		return target == models.ReviewStatusRunningBisync || target == models.ReviewStatusRollback || target == models.ReviewStatusDismissed || target == models.ReviewStatusReviewing

	case models.ReviewStatusRunningBisync:
		return target == models.ReviewStatusComplete || target == models.ReviewStatusFailed

	case models.ReviewStatusFailed:
		return target == models.ReviewStatusApplying || target == models.ReviewStatusRollback || target == models.ReviewStatusReviewing || target == models.ReviewStatusDismissed

	case models.ReviewStatusRollback:
		return target == models.ReviewStatusReviewing || target == models.ReviewStatusDismissed

	case models.ReviewStatusComplete, models.ReviewStatusDismissed:
		return false

	default:
		return false
	}
}

// Transition performs the state transition on session, updating UpdatedAt or returning an error.
func Transition(session *models.ReviewSession, target models.ReviewSessionStatus) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	if !CanTransition(session.Status, target) {
		return fmt.Errorf("invalid review session transition: cannot transition from %s to %s", session.Status, target)
	}

	session.Status = target
	session.UpdatedAt = time.Now().UTC()
	return nil
}

// NormalizeDecision maps UI aliases to standard canonical decisions.
func NormalizeDecision(d models.ActionDecision) models.ActionDecision {
	val := strings.ToLower(strings.ReplaceAll(string(d), "_", "-"))
	switch val {
	case "restore-local", "restore-remote", "preserve", "restore":
		return models.DecisionRestoreItem
	case "skip", "ignore":
		return models.DecisionSkip
	case "keep-local", "keep-remote", "keep-both", "accept-delete", "restore-item":
		return models.ActionDecision(val)
	default:
		return d
	}
}

// IsValidDecision checks if decision is a supported ActionDecision.
func IsValidDecision(d models.ActionDecision) bool {
	norm := NormalizeDecision(d)
	switch norm {
	case models.DecisionKeepLocal, models.DecisionKeepRemote, models.DecisionKeepBoth,
		models.DecisionAcceptDelete, models.DecisionRestoreItem, models.DecisionSkip:
		return true
	default:
		return false
	}
}

// StageDecision records a user's resolution decision on a specific candidate item.
func StageDecision(session *models.ReviewSession, candidateID string, decision models.ActionDecision) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	if session.Status != models.ReviewStatusCreated && session.Status != models.ReviewStatusReviewing && session.Status != models.ReviewStatusReady && session.Status != models.ReviewStatusFailed {
		return fmt.Errorf("cannot stage decisions while session is in %s state", session.Status)
	}

	normDecision := NormalizeDecision(decision)
	if !IsValidDecision(normDecision) {
		return fmt.Errorf("unsupported resolution decision: %s", decision)
	}

	// Try matching deletion candidate
	for i := range session.Deletions {
		if session.Deletions[i].ID == candidateID {
			if normDecision != models.DecisionAcceptDelete && normDecision != models.DecisionRestoreItem && normDecision != models.DecisionSkip {
				return fmt.Errorf("invalid decision %s for deletion candidate; expected accept-delete, restore-item, or skip", decision)
			}
			session.Deletions[i].Decision = normDecision
			session.Deletions[i].Resolved = true
			session.UpdatedAt = time.Now().UTC()
			if session.Status == models.ReviewStatusCreated || session.Status == models.ReviewStatusFailed {
				session.Status = models.ReviewStatusReviewing
			}
			return nil
		}
	}

	// Try matching conflict candidate
	for i := range session.Conflicts {
		if session.Conflicts[i].ID == candidateID {
			if normDecision != models.DecisionKeepLocal && normDecision != models.DecisionKeepRemote && normDecision != models.DecisionKeepBoth && normDecision != models.DecisionSkip {
				return fmt.Errorf("invalid decision %s for conflict candidate; expected keep-local, keep-remote, keep-both, or skip", decision)
			}
			session.Conflicts[i].Decision = normDecision
			session.Conflicts[i].Resolved = true
			session.UpdatedAt = time.Now().UTC()
			if session.Status == models.ReviewStatusCreated || session.Status == models.ReviewStatusFailed {
				session.Status = models.ReviewStatusReviewing
			}
			return nil
		}
	}

	return fmt.Errorf("candidate with ID %q not found in session", candidateID)
}

// AllResolved reports whether every candidate in the session has been resolved.
func AllResolved(session *models.ReviewSession) bool {
	if session == nil {
		return false
	}
	if len(session.Deletions) == 0 && len(session.Conflicts) == 0 {
		return true
	}

	for _, d := range session.Deletions {
		if !d.Resolved || d.Decision == "" {
			return false
		}
	}

	for _, c := range session.Conflicts {
		if !c.Resolved || c.Decision == "" {
			return false
		}
	}

	return true
}

// PendingCount returns the number of candidates awaiting resolution decision.
func PendingCount(session *models.ReviewSession) int {
	if session == nil {
		return 0
	}
	pending := 0
	for _, d := range session.Deletions {
		if !d.Resolved || d.Decision == "" {
			pending++
		}
	}
	for _, c := range session.Conflicts {
		if !c.Resolved || c.Decision == "" {
			pending++
		}
	}
	return pending
}

// TotalCandidateCount returns the combined count of deletion and conflict candidates.
func TotalCandidateCount(session *models.ReviewSession) int {
	if session == nil {
		return 0
	}
	return len(session.Deletions) + len(session.Conflicts)
}

// ActionCandidateSummary returns a short formatted summary of candidate statuses.
func ActionCandidateSummary(session *models.ReviewSession) string {
	if session == nil {
		return "no session"
	}
	resolved := TotalCandidateCount(session) - PendingCount(session)
	return fmt.Sprintf("%d/%d items resolved (%s)", resolved, TotalCandidateCount(session), strings.ToLower(string(session.Status)))
}
