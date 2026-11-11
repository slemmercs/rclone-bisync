package profiles

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	ErrProfileNotFound         = errors.New("profile not found")
	ErrProfileAlreadyExists     = errors.New("profile already exists")
	ErrInvalidProfileID        = errors.New("invalid profile id: must contain only alphanumeric characters, underscores, and hyphens")
	ErrInvalidLocalPath        = errors.New("invalid local path: path cannot be empty")
	ErrInvalidRemote           = errors.New("invalid remote: remote cannot be empty")
	ErrInvalidName             = errors.New("invalid name: profile name cannot be empty")
	ErrVersionNotFound         = errors.New("profile version not found")
	ErrInvalidSchedule         = errors.New("invalid schedule: must be one of 'manual', '15m', '30m', '1h', '4h', 'daily'")
	ErrInvalidReviewMode       = errors.New("invalid review mode: must be one of 'disabled', 'deletions', 'conflicts', 'all'")
	ErrInvalidConflictMode     = errors.New("invalid conflict mode: must be one of 'path1', 'path2', 'newer', 'older', 'larger', 'smaller'")
	ErrInvalidCompareMode      = errors.New("invalid compare mode: must be one of 'size,modtime', 'size,modtime,checksum', 'checksum', 'modtime'")
	ErrInvalidNotificationPref = errors.New("invalid notification preference: must be one of 'all', 'failures-only', 'review-required', 'disabled'")
	validIDRegex               = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// ValidSchedules enumerates all valid synchronization schedule intervals supported by Bisync Hub UI and Scheduler.
var ValidSchedules = []string{"manual", "15m", "30m", "1h", "4h", "daily"}

// ValidConflictModes enumerates all valid conflict resolution strategies.
var ValidConflictModes = []string{"path1", "path2", "newer", "older", "larger", "smaller"}

// ValidCompareModes enumerates all valid comparison detection strategies.
var ValidCompareModes = []string{"size,modtime", "size,modtime,checksum", "checksum", "modtime"}

// ValidReviewModes enumerates all valid review mode strategies.
var ValidReviewModes = []ReviewMode{ReviewDisabled, ReviewDeletions, ReviewConflicts, ReviewAll}

// ValidNotificationPrefs enumerates all valid notification preferences.
var ValidNotificationPrefs = []NotificationPreference{NotifyAll, NotifyFailuresOnly, NotifyReviewRequired, NotifyDisabled}

// IsValidSchedule checks if the schedule is in the supported set.
func IsValidSchedule(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	for _, v := range ValidSchedules {
		if s == v {
			return true
		}
	}
	return false
}

// IsValidReviewMode checks if the review mode is supported.
func IsValidReviewMode(m ReviewMode) bool {
	for _, v := range ValidReviewModes {
		if m == v {
			return true
		}
	}
	return false
}

// IsValidConflictMode checks if the conflict resolution mode is supported.
func IsValidConflictMode(c string) bool {
	c = strings.ToLower(strings.TrimSpace(c))
	for _, v := range ValidConflictModes {
		if c == v {
			return true
		}
	}
	return false
}

// IsValidCompareMode checks if the comparison mode is supported.
func IsValidCompareMode(m string) bool {
	m = strings.ToLower(strings.TrimSpace(m))
	for _, v := range ValidCompareModes {
		if m == v {
			return true
		}
	}
	return false
}

// IsValidNotificationPref checks if the notification preference is supported.
func IsValidNotificationPref(n NotificationPreference) bool {
	for _, v := range ValidNotificationPrefs {
		if n == v {
			return true
		}
	}
	return false
}

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

// MaxDeleteValue represents a max-delete threshold as either an absolute file count (e.g. "10")
// or a percentage ratio (e.g. "10%").
type MaxDeleteValue string

func (m *MaxDeleteValue) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "null" || trimmed == `""` || trimmed == "" {
		*m = ""
		return nil
	}
	if strings.HasPrefix(trimmed, `"`) && strings.HasSuffix(trimmed, `"`) {
		*m = MaxDeleteValue(strings.Trim(trimmed, `"`))
		return nil
	}
	*m = MaxDeleteValue(trimmed)
	return nil
}

func (m MaxDeleteValue) MarshalJSON() ([]byte, error) {
	s := string(m)
	if s == "" {
		return []byte("null"), nil
	}
	if strings.HasSuffix(s, "%") {
		return json.Marshal(s)
	}
	if n, err := strconv.Atoi(s); err == nil {
		return json.Marshal(n)
	}
	return json.Marshal(s)
}

func (m MaxDeleteValue) String() string {
	return string(m)
}

// ProfileFlags defines profile-level operational flag overrides.
// Pointer fields represent explicit overrides; nil indicates inheritance from GlobalDefaults.
type ProfileFlags struct {
	ConflictResolve  *string           `json:"conflict_resolve,omitempty"`
	MaxDelete        *MaxDeleteValue   `json:"max_delete,omitempty"`
	CanaryValidation *bool             `json:"canary_validation,omitempty"`
	CompareMode      *string           `json:"compare_mode,omitempty"`
	RemoveEmptyDirs  *bool             `json:"remove_empty_dirs,omitempty"`
	CreateEmptyDirs  *bool             `json:"create_empty_dirs,omitempty"`
	FiltersFilePath  string            `json:"filters_file_path,omitempty"`
	CustomFlags      string            `json:"custom_flags,omitempty"`
	ProviderOptions  map[string]string `json:"provider_options,omitempty"`
}

// ResolvedFlags represents effective execution settings after evaluating overrides and globals.
type ResolvedFlags struct {
	DryRun           bool              `json:"dry_run"`
	Resync           bool              `json:"resync"`
	Force            bool              `json:"force"`
	ConflictResolve  string            `json:"conflict_resolve"`
	MaxDelete        MaxDeleteValue    `json:"max_delete"`
	CanaryValidation bool              `json:"canary_validation"`
	CompareMode      string            `json:"compare_mode"`
	RemoveEmptyDirs  bool              `json:"remove_empty_dirs"`
	CreateEmptyDirs  bool              `json:"create_empty_dirs"`
	ExcludePatterns  []string          `json:"exclude_patterns,omitempty"`
	FiltersFilePath  string            `json:"filters_file_path,omitempty"`
	CustomFlags      string            `json:"custom_flags,omitempty"`
	ProviderOptions  map[string]string `json:"provider_options,omitempty"`
}

// ExecutionOverrides captures single-run execution toggles triggered via CLI or dashboard UI.
type ExecutionOverrides struct {
	DryRun *bool `json:"dry_run,omitempty"`
	Resync *bool `json:"resync,omitempty"`
	Force  *bool `json:"force,omitempty"`
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
	Flags            ProfileFlags           `json:"flags"`
	ExcludePatterns  []string               `json:"exclude_patterns,omitempty"`
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
	Action    string    `json:"action,omitempty"`
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

// GenerateProfileID creates a collision-resistant unique profile identifier prefixed with "prof_".
func GenerateProfileID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("prof_%x", time.Now().UnixNano())
	}
	return fmt.Sprintf("prof_%x", b)
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
	p.Remote = strings.TrimRight(p.Remote, ":")
	if p.Remote == "" {
		return ErrInvalidRemote
	}
	p.LocalPath = strings.TrimSpace(p.LocalPath)
	if p.LocalPath == "" {
		return ErrInvalidLocalPath
	}
	p.Schedule = strings.ToLower(strings.TrimSpace(p.Schedule))
	if p.Schedule == "" {
		p.Schedule = "manual"
	}
	if !IsValidSchedule(p.Schedule) {
		return fmt.Errorf("%w: %q", ErrInvalidSchedule, p.Schedule)
	}
	if p.ReviewMode == "" {
		p.ReviewMode = ReviewDisabled
	}
	if !IsValidReviewMode(p.ReviewMode) {
		return fmt.Errorf("%w: %q", ErrInvalidReviewMode, p.ReviewMode)
	}
	if p.NotificationPref == "" {
		p.NotificationPref = NotifyFailuresOnly
	}
	if !IsValidNotificationPref(p.NotificationPref) {
		return fmt.Errorf("%w: %q", ErrInvalidNotificationPref, p.NotificationPref)
	}

	if p.Flags.ConflictResolve != nil {
		cr := strings.ToLower(strings.TrimSpace(*p.Flags.ConflictResolve))
		if cr != "" {
			if !IsValidConflictMode(cr) {
				return fmt.Errorf("%w: %q", ErrInvalidConflictMode, *p.Flags.ConflictResolve)
			}
			p.Flags.ConflictResolve = &cr
		} else {
			p.Flags.ConflictResolve = nil
		}
	}

	if p.Flags.CompareMode != nil {
		cm := strings.ToLower(strings.TrimSpace(*p.Flags.CompareMode))
		if cm != "" {
			if !IsValidCompareMode(cm) {
				return fmt.Errorf("%w: %q", ErrInvalidCompareMode, *p.Flags.CompareMode)
			}
			p.Flags.CompareMode = &cm
		} else {
			p.Flags.CompareMode = nil
		}
	}

	if p.Safety.FailureThreshold <= 0 {
		p.Safety.FailureThreshold = 3
	}
	if p.Safety.DeleteThresholdPct <= 0 {
		p.Safety.DeleteThresholdPct = 10.0
	}
	return nil
}
