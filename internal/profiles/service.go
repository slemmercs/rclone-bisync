package profiles

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RemoteChecker abstracts remote validation against Rclone RC.
type RemoteChecker interface {
	ListRemotes(ctx context.Context) ([]string, error)
}

// Service manages operational lifecycle, remote validation, and scheduling for profiles.
type Service struct {
	store     *Store
	remotes   RemoteChecker
	scheduler *Scheduler
}

// NewService creates a profile management service.
func NewService(store *Store, remotes RemoteChecker, scheduler *Scheduler) *Service {
	return &Service{
		store:     store,
		remotes:   remotes,
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
	if strings.TrimSpace(p.ID) == "" {
		for {
			cand := GenerateProfileID()
			if _, err := s.store.Get(cand); err != nil {
				p.ID = cand
				break
			}
		}
	} else {
		if _, err := s.store.Get(p.ID); err == nil {
			return nil, ErrProfileAlreadyExists
		}
	}

	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := s.validateRemote(ctx, p.Remote); err != nil {
		return nil, err
	}

	if strings.TrimSpace(p.LocalPath) != "" {
		resolved, err := resolveAndEnsureLocalPath(p.LocalPath)
		if err != nil {
			return nil, err
		}
		p.LocalPath = resolved
	}

	if err := s.store.Save(p, "Profile created"); err != nil {
		return nil, err
	}

	if s.scheduler != nil && p.Enabled && !p.Archived {
		s.scheduler.RestartProfile(p)
	}

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

	if strings.TrimSpace(p.LocalPath) != "" {
		resolved, err := resolveAndEnsureLocalPath(p.LocalPath)
		if err != nil {
			return nil, err
		}
		p.LocalPath = resolved
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
	if cloned.ID == "" {
		for {
			cand := GenerateProfileID()
			if _, err := s.store.Get(cand); err != nil {
				cloned.ID = cand
				break
			}
		}
	} else {
		if _, err := s.store.Get(cloned.ID); err == nil {
			return nil, ErrProfileAlreadyExists
		}
	}

	cloned.Name = strings.TrimSpace(newName)
	cloned.State = OperationalState{Status: StatusHealthy}

	if err := cloned.Validate(); err != nil {
		return nil, err
	}

	if strings.TrimSpace(cloned.LocalPath) != "" {
		resolved, err := resolveAndEnsureLocalPath(cloned.LocalPath)
		if err != nil {
			return nil, err
		}
		cloned.LocalPath = resolved
	}

	if err := s.store.Save(&cloned, fmt.Sprintf("Cloned from %s", sourceID)); err != nil {
		return nil, err
	}

	if s.scheduler != nil && cloned.Enabled && !cloned.Archived {
		s.scheduler.RestartProfile(&cloned)
	}

	return s.store.Get(cloned.ID)
}

// Delete stops execution and permanently removes a profile.
func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.store.Get(id); err != nil {
		return err
	}

	if s.scheduler != nil {
		s.scheduler.StopProfile(id)
	}

	return s.store.Delete(id)
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

	return restored, nil
}

// TriggerSync prompts immediate execution of a profile sync.
func (s *Service) TriggerSync(ctx context.Context, id string, force bool) error {
	return s.TriggerSyncWithOptions(ctx, id, SyncOptions{Force: force})
}

// TriggerSyncWithOptions prompts immediate execution of a profile sync with custom options.
func (s *Service) TriggerSyncWithOptions(ctx context.Context, id string, opts SyncOptions) error {
	if s.scheduler == nil {
		return fmt.Errorf("scheduler is not active")
	}
	return s.scheduler.TriggerSyncWithOptions(ctx, id, opts)
}

// IsAnySyncing reports whether any profile worker is actively executing a synchronization cycle.
func (s *Service) IsAnySyncing() bool {
	if s.scheduler == nil {
		return false
	}
	return s.scheduler.IsAnyRunning()
}

// IsProfileSyncing reports whether the specified profile is actively executing a synchronization cycle.
func (s *Service) IsProfileSyncing(id string) bool {
	if s.scheduler == nil {
		return false
	}
	return s.scheduler.IsProfileRunning(id)
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

func resolveAndEnsureLocalPath(localPath string) (string, error) {
	cleanPath := strings.TrimSpace(localPath)
	if cleanPath == "" {
		return "", nil
	}
	if strings.Contains(cleanPath, "..") {
		return "", errors.New("relative path traversal '..' is forbidden")
	}

	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		if cleanPath == "~" {
			cleanPath = homeDir
		} else if strings.HasPrefix(cleanPath, "~/") {
			cleanPath = filepath.Join(homeDir, strings.TrimPrefix(cleanPath, "~/"))
		}
	}

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("invalid local path %q: %w", cleanPath, err)
	}
	cleanPath = filepath.Clean(absPath)

	// Collect safe roots for auto-provisioning (user home directory, temp directories, standard mount points).
	var safeRoots []string
	if homeDir != "" {
		h := filepath.Clean(homeDir)
		safeRoots = append(safeRoots, h)
		if realH, err := filepath.EvalSymlinks(h); err == nil && realH != h {
			safeRoots = append(safeRoots, realH)
		}
	}
	if tmp := os.TempDir(); tmp != "" {
		t := filepath.Clean(tmp)
		safeRoots = append(safeRoots, t)
		if realT, err := filepath.EvalSymlinks(t); err == nil && realT != t {
			safeRoots = append(safeRoots, realT)
		}
	}
	for _, mnt := range []string{"/mnt", "/media", "/Volumes"} {
		if fi, err := os.Stat(mnt); err == nil && fi.IsDir() {
			safeRoots = append(safeRoots, mnt)
		}
	}

	var matchedRoot string
	for _, root := range safeRoots {
		rootPrefix := root + string(filepath.Separator)
		if cleanPath == root || strings.HasPrefix(cleanPath, rootPrefix) {
			matchedRoot = root
			break
		}
	}
	if matchedRoot == "" {
		return "", fmt.Errorf("local path %q must reside within your home directory, /tmp, /mnt, or /media", cleanPath)
	}

	rootPrefix := matchedRoot + string(filepath.Separator)
	rel, err := filepath.Rel(matchedRoot, cleanPath)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("path traversal forbidden: %q", cleanPath)
	}
	safePath := filepath.Join(matchedRoot, rel)
	if !strings.HasPrefix(safePath, rootPrefix) && safePath != matchedRoot {
		return "", fmt.Errorf("path traversal forbidden: %q", cleanPath)
	}

	if err := os.MkdirAll(safePath, 0750); err != nil {
		return "", fmt.Errorf("failed to create local path %q: %w", safePath, err)
	}
	return safePath, nil
}
