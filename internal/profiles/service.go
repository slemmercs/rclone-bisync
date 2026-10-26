package profiles

import (
	"context"
	"fmt"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/audit"
)

// RemoteChecker abstracts remote validation against Rclone RC.
type RemoteChecker interface {
	ListRemotes(ctx context.Context) ([]string, error)
}

// AuditRecorder defines audit logging capability.
type AuditRecorder interface {
	RecordEvent(profileID string, action audit.Action, actor string, details map[string]any) (*audit.AuditRecord, error)
}

// Service manages operational lifecycle, remote validation, and scheduling for profiles.
type Service struct {
	store     *Store
	remotes   RemoteChecker
	audit     AuditRecorder
	scheduler *Scheduler
}

// NewService creates a profile management service.
func NewService(store *Store, remotes RemoteChecker, auditRec AuditRecorder, scheduler *Scheduler) *Service {
	return &Service{
		store:     store,
		remotes:   remotes,
		audit:     auditRec,
		scheduler: scheduler,
	}
}

// List returns all managed profiles.
func (s *Service) List(includeArchived bool) []*Profile {
	return s.store.List(includeArchived)
}

// Get returns a single profile by ID.
func (s *Service) Get(id string) (*Profile, error) {
	return s.store.Get(id)
}

// GetHistory returns version snapshot history for a profile.
func (s *Service) GetHistory(id string) ([]ProfileVersion, error) {
	return s.store.GetHistory(id)
}

// Create validates operational parameters and persists a new profile.
func (s *Service) Create(ctx context.Context, p *Profile) (*Profile, error) {
	if p == nil {
		return nil, fmt.Errorf("profile cannot be nil")
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := s.validateRemote(ctx, p.Remote); err != nil {
		return nil, err
	}

	if _, err := s.store.Get(p.ID); err == nil {
		return nil, ErrProfileAlreadyExists
	}

	if err := s.store.Save(p, "Profile created"); err != nil {
		return nil, err
	}

	if s.scheduler != nil && p.Enabled && !p.Archived {
		s.scheduler.RestartProfile(p)
	}

	s.recordAudit(p.ID, audit.ActionProfileCreated, map[string]any{
		"name":        p.Name,
		"remote":      p.Remote,
		"local_path":  p.LocalPath,
		"schedule":    p.Schedule,
		"review_mode": p.ReviewMode,
	})

	return s.store.Get(p.ID)
}

// Update modifies configuration for an existing profile.
func (s *Service) Update(ctx context.Context, p *Profile, reason string) (*Profile, error) {
	if p == nil {
		return nil, fmt.Errorf("profile cannot be nil")
	}
	if _, err := s.store.Get(p.ID); err != nil {
		return nil, err
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := s.validateRemote(ctx, p.Remote); err != nil {
		return nil, err
	}

	if reason == "" {
		reason = "Profile updated"
	}

	if err := s.store.Save(p, reason); err != nil {
		return nil, err
	}

	if s.scheduler != nil {
		s.scheduler.RestartProfile(p)
	}

	s.recordAudit(p.ID, audit.ActionProfileUpdated, map[string]any{
		"name":        p.Name,
		"version":     p.Version,
		"reason":      reason,
		"enabled":     p.Enabled,
		"schedule":    p.Schedule,
		"review_mode": p.ReviewMode,
	})

	return s.store.Get(p.ID)
}

// Clone duplicates an existing profile into a new independent profile workload.
func (s *Service) Clone(ctx context.Context, sourceID, newID, newName string) (*Profile, error) {
	source, err := s.store.Get(sourceID)
	if err != nil {
		return nil, fmt.Errorf("source profile not found: %w", err)
	}

	cloned := *source
	cloned.ID = strings.TrimSpace(newID)
	cloned.Name = strings.TrimSpace(newName)
	cloned.State = OperationalState{Status: StatusHealthy}

	if err := cloned.Validate(); err != nil {
		return nil, err
	}

	if _, err := s.store.Get(cloned.ID); err == nil {
		return nil, ErrProfileAlreadyExists
	}

	if err := s.store.Save(&cloned, fmt.Sprintf("Cloned from %s", sourceID)); err != nil {
		return nil, err
	}

	if s.scheduler != nil && cloned.Enabled && !cloned.Archived {
		s.scheduler.RestartProfile(&cloned)
	}

	s.recordAudit(cloned.ID, audit.ActionProfileCloned, map[string]any{
		"source_id": sourceID,
		"name":      cloned.Name,
		"remote":    cloned.Remote,
	})

	return s.store.Get(cloned.ID)
}

// Delete stops execution and permanently removes a profile.
func (s *Service) Delete(ctx context.Context, id string) error {
	p, err := s.store.Get(id)
	if err != nil {
		return err
	}

	if s.scheduler != nil {
		s.scheduler.StopProfile(id)
	}

	if err := s.store.Delete(id); err != nil {
		return err
	}

	s.recordAudit(id, audit.ActionProfileDeleted, map[string]any{
		"name":   p.Name,
		"remote": p.Remote,
	})
	return nil
}

// Rollback restores a profile to an earlier version snapshot.
func (s *Service) Rollback(ctx context.Context, id string, targetVersion int) (*Profile, error) {
	restored, err := s.store.Rollback(id, targetVersion)
	if err != nil {
		return nil, err
	}

	if s.scheduler != nil {
		s.scheduler.RestartProfile(restored)
	}

	s.recordAudit(id, audit.ActionProfileRolledBack, map[string]any{
		"target_version": targetVersion,
		"new_version":    restored.Version,
	})

	return restored, nil
}

// TriggerSync prompts immediate execution of a profile sync.
func (s *Service) TriggerSync(ctx context.Context, id string, force bool) error {
	if s.scheduler == nil {
		return fmt.Errorf("scheduler is not active")
	}
	return s.scheduler.TriggerSync(ctx, id, force)
}

// ListRemotes queries available remotes from Rclone RC.
func (s *Service) ListRemotes(ctx context.Context) ([]string, error) {
	if s.remotes == nil {
		return []string{}, nil
	}
	return s.remotes.ListRemotes(ctx)
}

func (s *Service) validateRemote(ctx context.Context, remote string) error {
	if s.remotes == nil {
		return nil
	}
	remotes, err := s.remotes.ListRemotes(ctx)
	if err != nil {
		return nil
	}
	if len(remotes) == 0 {
		return nil
	}
	cleanRemote := strings.TrimSuffix(strings.TrimSpace(remote), ":")
	for _, r := range remotes {
		if strings.TrimSuffix(strings.TrimSpace(r), ":") == cleanRemote {
			return nil
		}
	}
	return fmt.Errorf("remote '%s' is not configured in Rclone", remote)
}

func (s *Service) recordAudit(profileID string, action audit.Action, details map[string]any) {
	if s.audit != nil {
		_, _ = s.audit.RecordEvent(profileID, action, "SYSTEM", details)
	}
}
