package models

import "time"

// ReviewSessionStatus represents the lifecycle state machine of an interactive review session.
type ReviewSessionStatus string

const (
	ReviewStatusCreated       ReviewSessionStatus = "CREATED"
	ReviewStatusReviewing     ReviewSessionStatus = "REVIEWING"
	ReviewStatusApplying      ReviewSessionStatus = "APPLYING"
	ReviewStatusReady         ReviewSessionStatus = "READY"
	ReviewStatusRunningBisync ReviewSessionStatus = "RUNNING_BISYNC"
	ReviewStatusComplete      ReviewSessionStatus = "COMPLETE"
	ReviewStatusFailed        ReviewSessionStatus = "FAILED"
	ReviewStatusRollback      ReviewSessionStatus = "ROLLBACK"
	ReviewStatusDismissed     ReviewSessionStatus = "DISMISSED"
)

// CandidateCondition represents the circuit breaker trigger condition.
type CandidateCondition string

const (
	ConditionTooManyDeletes CandidateCondition = "TOO_MANY_DELETES"
	ConditionConflict       CandidateCondition = "CONFLICT"
)

// CandidateSide designates local vs remote storage origin.
type CandidateSide string

const (
	SideLocal  CandidateSide = "local"
	SideRemote CandidateSide = "remote"
)

// ActionDecision defines the chosen resolution strategy.
type ActionDecision string

const (
	DecisionKeepLocal    ActionDecision = "keep-local"
	DecisionKeepRemote   ActionDecision = "keep-remote"
	DecisionKeepBoth     ActionDecision = "keep-both"
	DecisionAcceptDelete ActionDecision = "accept-delete"
	DecisionRestoreItem  ActionDecision = "restore-item"
	DecisionSkip         ActionDecision = "skip"
)

// DeletionCandidate represents an item scheduled for deletion during an aborted run.
type DeletionCandidate struct {
	ID           string             `json:"id"`
	Path         string             `json:"path"`
	Name         string             `json:"name"`
	IsDir        bool               `json:"is_dir"`
	Source       CandidateSide      `json:"source"`
	TargetAction string             `json:"target_action"`
	Size         int64              `json:"size,omitempty"`
	ModTime      string             `json:"mod_time,omitempty"`
	Condition    CandidateCondition `json:"condition"`
	Decision     ActionDecision     `json:"decision,omitempty"`
	Resolved     bool               `json:"resolved"`
}

// ConflictCandidate represents a conflicting file altered concurrently on both sides.
type ConflictCandidate struct {
	ID            string             `json:"id"`
	Path          string             `json:"path"`
	Name          string             `json:"name"`
	LocalSize     int64              `json:"local_size"`
	LocalModTime  string             `json:"local_mod_time"`
	RemoteSize    int64              `json:"remote_size"`
	RemoteModTime string             `json:"remote_mod_time"`
	Condition     CandidateCondition `json:"condition"`
	Decision      ActionDecision     `json:"decision,omitempty"`
	Resolved      bool               `json:"resolved"`
}

// ResolutionAction logs an individual atomic operation executed or staged for execution.
type ResolutionAction struct {
	ID         string         `json:"id"`
	SessionID  string         `json:"session_id"`
	Type       ActionDecision `json:"type"`
	ItemPath   string         `json:"item_path"`
	IsDir      bool           `json:"is_dir"`
	Status     string         `json:"status"` // "pending", "completed", "failed", "rolled_back"
	BackupPath string         `json:"backup_path,omitempty"`
	BackupSide CandidateSide  `json:"backup_side,omitempty"`
	ExecutedAt *time.Time     `json:"executed_at,omitempty"`
	Error      string         `json:"error,omitempty"`
}

// ReviewSession encapsulates the entire state of an interactive sync review session.
type ReviewSession struct {
	ID          string              `json:"id"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	Status      ReviewSessionStatus `json:"status"`
	Deletions   []DeletionCandidate `json:"deletions"`
	Conflicts   []ConflictCandidate `json:"conflicts"`
	Actions     []ResolutionAction  `json:"actions"`
	OriginError string              `json:"origin_error,omitempty"`
	Message     string              `json:"message,omitempty"`
}

// WALEntry represents an append-only write-ahead log record for crash recovery.
type WALEntry struct {
	Timestamp string         `json:"timestamp"`
	SessionID string         `json:"session_id"`
	ActionID  string         `json:"action_id"`
	Step      string         `json:"step"` // "BACKUP_COMPLETED", "MUTATION_STARTED", "MUTATION_COMPLETED", "ROLLBACK_COMPLETED"
	ItemPath  string         `json:"item_path"`
	Type      ActionDecision `json:"type"`
	BackupRef string         `json:"backup_ref,omitempty"`
	Error     string         `json:"error,omitempty"`
}

// ReviewSessionResponse is the response payload for GET /api/review/session.
type ReviewSessionResponse struct {
	Session *ReviewSession `json:"session,omitempty"`
	Active  bool           `json:"active"`
	Error   string         `json:"error,omitempty"`
}

// StageDecisionRequest is the request payload for POST /api/review/action.
type StageDecisionRequest struct {
	SessionID    string         `json:"session_id,omitempty"`
	CandidateID  string         `json:"candidate_id,omitempty"`
	CandidateIDs []string       `json:"candidate_ids,omitempty"`
	Decision     ActionDecision `json:"decision"`
}

// StageDecisionResponse is the response payload for POST /api/review/action.
type StageDecisionResponse struct {
	Status  string         `json:"status"`
	Session *ReviewSession `json:"session,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// ExecuteReviewRequest is the optional request payload for POST /api/review/execute.
type ExecuteReviewRequest struct {
	SessionID string `json:"session_id,omitempty"`
}

// ExecuteReviewResponse is the response payload for POST /api/review/execute.
type ExecuteReviewResponse struct {
	Status    string             `json:"status"`
	SessionID string             `json:"session_id"`
	Applied   int                `json:"applied"`
	Failed    int                `json:"failed"`
	Actions   []ResolutionAction `json:"actions"`
	Error     string             `json:"error,omitempty"`
}

// RollbackReviewRequest is the optional request payload for POST /api/review/rollback.
type RollbackReviewRequest struct {
	SessionID string `json:"session_id,omitempty"`
}

// RollbackReviewResponse is the response payload for POST /api/review/rollback.
type RollbackReviewResponse struct {
	Status    string `json:"status"`
	SessionID string `json:"session_id"`
	Restored  int    `json:"restored"`
	Error     string `json:"error,omitempty"`
}

// DismissReviewRequest is the optional request payload for POST /api/review/dismiss.
type DismissReviewRequest struct {
	SessionID string `json:"session_id,omitempty"`
}
