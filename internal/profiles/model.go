package profiles

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	ErrProfileNotFound      = errors.New("profile not found")
	ErrProfileAlreadyExists = errors.New("profile already exists")
	ErrInvalidProfileID     = errors.New("invalid profile id: must contain only alphanumeric characters, underscores, and hyphens")
	ErrInvalidLocalPath     = errors.New("invalid local path: path cannot be empty")
	ErrInvalidRemote        = errors.New("invalid remote: remote cannot be empty")
	ErrInvalidName          = errors.New("invalid name: profile name cannot be empty")
	ErrVersionNotFound      = errors.New("profile version not found")
	validIDRegex            = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// ProfileStatus represents operational status of a profile.
type ProfileStatus string

const (
	StatusHealthy                ProfileStatus = "healthy"
	StatusWarning                ProfileStatus = "warning"
	StatusAuthenticationRequired ProfileStatus = "authentication_required"
	StatusSyncFailed             ProfileStatus = "sync_failed"
	StatusPaused                 ProfileStatus = "paused"
	StatusError                  ProfileStatus = "error"
	StatusRunning                ProfileStatus = "running"
)

// ReviewMode defines when sync operations require human review.
type ReviewMode string

const (
	ReviewDisabled  ReviewMode = "disabled"
	ReviewDeletions ReviewMode = "deletions"
	ReviewConflicts ReviewMode = "conflicts"
	ReviewAll       ReviewMode = "all"
)

// NotificationPreference specifies notification filtering rules.
type NotificationPreference string

const (
	NotifyAll            NotificationPreference = "all"
	NotifyFailuresOnly   NotificationPreference = "failures-only"
	NotifyReviewRequired NotificationPreference = "review-required"
	NotifyDisabled       NotificationPreference = "disabled"
)

// SafetySettings defines thresholds that trigger safeguards.
type SafetySettings struct {
	DeleteThresholdPct float64 `json:"delete_threshold_pct"` // e.g. 10.0 for 10%
	FailureThreshold   int     `json:"failure_threshold"`    // e.g. 3 consecutive failures
	CanaryValidation   bool    `json:"canary_validation"`
}

// OperationalState captures dynamic runtime metrics and status.
type OperationalState struct {
	Status              ProfileStatus `json:"status"`
	LastSync            *time.Time    `json:"last_sync,omitempty"`
	NextSync            *time.Time    `json:"next_sync,omitempty"`
	ConsecutiveFailures int           `json:"consecutive_failures"`
	LastError           string        `json:"last_error,omitempty"`
	LastDurationSec     float64       `json:"last_duration_sec,omitempty"`
}

// Profile represents a managed synchronization workload.
type Profile struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description,omitempty"`
	Enabled          bool                   `json:"enabled"`
	Archived         bool                   `json:"archived"`
	Remote           string                 `json:"remote"`
	RemotePath       string                 `json:"remote_path,omitempty"`
	LocalPath        string                 `json:"local_path"`
	Schedule         string                 `json:"schedule"` // e.g. "15m", "1h", "daily", "manual"
	ReviewMode       ReviewMode             `json:"review_mode"`
	NotificationPref NotificationPreference `json:"notification_pref"`
	Safety           SafetySettings         `json:"safety"`
	Tags             []string               `json:"tags,omitempty"`
	Version          int                    `json:"version"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	State            OperationalState       `json:"state"`
}

// ProfileVersion stores an immutable snapshot of a profile's configuration for rollback.
type ProfileVersion struct {
	Version   int       `json:"version"`
	ProfileID string    `json:"profile_id"`
	Timestamp time.Time `json:"timestamp"`
	Reason    string    `json:"reason"`
	Snapshot  Profile   `json:"snapshot"`
}

// FullRemoteTarget returns the composite remote string (e.g. "gdrive:subfolder" or "gdrive:").
func (p *Profile) FullRemoteTarget() string {
	r := strings.TrimSuffix(p.Remote, ":")
	sub := strings.TrimPrefix(p.RemotePath, "/")
	if sub == "" {
		return r + ":"
	}
	return fmt.Sprintf("%s:%s", r, sub)
}

// LockFilePath returns the dedicated lock file path for this profile.
func (p *Profile) LockFilePath() string {
	return fmt.Sprintf("/tmp/rclone-bisync-%s.lock", p.ID)
}

// Validate checks essential profile invariants.
func (p *Profile) Validate() error {
	p.ID = strings.TrimSpace(p.ID)
	if p.ID == "" || !validIDRegex.MatchString(p.ID) {
		return ErrInvalidProfileID
	}
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return ErrInvalidName
	}
	p.Remote = strings.TrimSpace(p.Remote)
	if p.Remote == "" {
		return ErrInvalidRemote
	}
	p.LocalPath = strings.TrimSpace(p.LocalPath)
	if p.LocalPath == "" {
		return ErrInvalidLocalPath
	}
	if p.Schedule == "" {
		p.Schedule = "manual"
	}
	if p.ReviewMode == "" {
		p.ReviewMode = ReviewDisabled
	}
	if p.NotificationPref == "" {
		p.NotificationPref = NotifyFailuresOnly
	}
	if p.Safety.FailureThreshold <= 0 {
		p.Safety.FailureThreshold = 3
	}
	if p.Safety.DeleteThresholdPct <= 0 {
		p.Safety.DeleteThresholdPct = 10.0
	}
	return nil
}
