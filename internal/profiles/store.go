package profiles

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Store handles atomic filesystem persistence for profiles and version history.
type Store struct {
	mu          sync.RWMutex
	storePath   string
	historyPath string
	lockPath    string
	profiles    map[string]*Profile
	history     map[string][]ProfileVersion
	lastModTime time.Time
}

// NewStore initializes a profile store from disk.
func NewStore(storePath, historyPath string) (*Store, error) {
	if storePath == "" {
		return nil, fmt.Errorf("storePath cannot be empty")
	}
	if historyPath == "" {
		historyPath = filepath.Join(filepath.Dir(storePath), "profiles_history.json")
	}

	s := &Store{
		storePath:   storePath,
		historyPath: historyPath,
		lockPath:    storePath + ".flock",
		profiles:    make(map[string]*Profile),
		history:     make(map[string][]ProfileVersion),
	}

	if err := s.load(); err != nil {
		return nil, fmt.Errorf("failed to load profiles store: %w", err)
	}

	return s, nil
}

func (s *Store) withFileLock(fn func() error) error {
	if err := os.MkdirAll(filepath.Dir(s.lockPath), 0750); err != nil {
		return fmt.Errorf("failed to create directory for profile lock: %w", err)
	}
	lockFile, err := os.OpenFile(s.lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("failed to open profile lock file: %w", err)
	}
	defer lockFile.Close()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to acquire profile file lock: %w", err)
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

	return fn()
}

func (s *Store) load() error {
	return s.withFileLock(func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.loadLocked()
	})
}

func (s *Store) loadLocked() error {
	if fi, err := os.Stat(s.storePath); err == nil {
		data, err := os.ReadFile(s.storePath)
		if err != nil {
			return err
		}
		var list []*Profile
		if err := json.Unmarshal(data, &list); err != nil {
			return err
		}
		s.profiles = make(map[string]*Profile, len(list))
		for _, p := range list {
			s.profiles[p.ID] = p
		}
		s.lastModTime = fi.ModTime()
	}

	if data, err := os.ReadFile(s.historyPath); err == nil {
		var hist map[string][]ProfileVersion
		if err := json.Unmarshal(data, &hist); err == nil {
			s.history = hist
		}
	}
	return nil
}

func (s *Store) reloadIfModifiedLocked() error {
	fi, err := os.Stat(s.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if fi.ModTime().After(s.lastModTime) {
		return s.loadLocked()
	}
	return nil
}

// List returns all profiles, optionally filtering out archived ones.
func (s *Store) List(includeArchived bool) []*Profile {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.reloadIfModifiedLocked()

	var result []*Profile
	for _, p := range s.profiles {
		if !includeArchived && p.Archived {
			continue
		}
		copyP := *p
		result = append(result, &copyP)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].Name < result[j].Name
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result
}

// Get retrieves a profile by ID.
func (s *Store) Get(id string) (*Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.reloadIfModifiedLocked()

	p, exists := s.profiles[id]
	if !exists {
		return nil, ErrProfileNotFound
	}
	copyP := *p
	return &copyP, nil
}

// Save stores or updates a profile with atomic disk writes and version tracking.
func (s *Store) Save(p *Profile, reason string) error {
	if err := p.Validate(); err != nil {
		return err
	}

	return s.withFileLock(func() error {
		s.mu.Lock()
		defer s.mu.Unlock()

		_ = s.reloadIfModifiedLocked()

		now := time.Now().UTC()
		existing, exists := s.profiles[p.ID]
		if !exists {
			p.Version = 1
			p.CreatedAt = now
			p.UpdatedAt = now
			if p.State.Status == "" {
				p.State.Status = StatusHealthy
			}
		} else {
			p.Version = existing.Version + 1
			p.CreatedAt = existing.CreatedAt
			p.UpdatedAt = now
		}

		action := "UPDATED"
		if !exists {
			action = "CREATED"
		}
		if strings.HasPrefix(reason, "Cloned from") {
			action = "CLONED"
		}
		if strings.HasPrefix(reason, "Rollback") {
			action = "ROLLED_BACK"
		}

		snapshot := ProfileVersion{
			Version:   p.Version,
			ProfileID: p.ID,
			Action:    action,
			Timestamp: now,
			Reason:    reason,
			Snapshot:  *p,
		}
		s.history[p.ID] = append(s.history[p.ID], snapshot)

		copyP := *p
		s.profiles[p.ID] = &copyP

		return s.persistLocked()
	})
}

// UpdateState updates dynamic operational state without bumping configuration version.
// It acquires a POSIX file lock, reloads on-disk configuration if changed, updates
// strictly the operational state fields, and writes atomically to protect user settings.
func (s *Store) UpdateState(id string, state OperationalState) error {
	return s.withFileLock(func() error {
		s.mu.Lock()
		defer s.mu.Unlock()

		if err := s.reloadIfModifiedLocked(); err != nil {
			return fmt.Errorf("failed to check profiles disk state: %w", err)
		}

		p, exists := s.profiles[id]
		if !exists {
			return ErrProfileNotFound
		}
		p.State = state
		p.UpdatedAt = time.Now().UTC()

		return s.persistLocked()
	})
}

// Delete removes a profile from memory and disk.
func (s *Store) Delete(id string) error {
	return s.withFileLock(func() error {
		s.mu.Lock()
		defer s.mu.Unlock()

		_ = s.reloadIfModifiedLocked()

		if _, exists := s.profiles[id]; !exists {
			return ErrProfileNotFound
		}

		delete(s.profiles, id)
		return s.persistLocked()
	})
}


// BootstrapDefault creates a "default" profile if the store is empty.
func (s *Store) BootstrapDefault(localPath, remote string) (*Profile, error) {
	s.mu.Lock()
	if len(s.profiles) > 0 {
		s.mu.Unlock()
		return nil, nil
	}
	s.mu.Unlock()

	defaultProf := &Profile{
		ID:               "default",
		Name:             "Default Workload",
		Description:      "Initial synchronization profile auto-generated from bisync.env",
		Enabled:          true,
		Archived:         false,
		Remote:           remote,
		LocalPath:        localPath,
		Schedule:         "15m",
		ReviewMode:       ReviewDisabled,
		NotificationPref: NotifyFailuresOnly,
		Safety: SafetySettings{
			DeleteThresholdPct: 10.0,
			FailureThreshold:   3,
			CanaryValidation:   true,
		},
		Tags: []string{"default"},
	}

	if err := s.Save(defaultProf, "Genesis profile bootstrap"); err != nil {
		return nil, err
	}
	return defaultProf, nil
}

func (s *Store) persistLocked() error {
	dir := filepath.Dir(s.storePath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory for profile store: %w", err)
	}

	var list []*Profile
	for _, p := range s.profiles {
		list = append(list, p)
	}
	if err := atomicWriteJSON(s.storePath, list); err != nil {
		return fmt.Errorf("failed to persist profiles: %w", err)
	}

	if fi, err := os.Stat(s.storePath); err == nil {
		s.lastModTime = fi.ModTime()
	}

	if err := atomicWriteJSON(s.historyPath, s.history); err != nil {
		return fmt.Errorf("failed to persist profile history: %w", err)
	}

	return nil
}

func atomicWriteJSON(filePath string, data any) error {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := fmt.Sprintf("%s.tmp.%d", filePath, os.Getpid())
	if err := os.WriteFile(tmpPath, bytes, 0600); err != nil {
		return err
	}

	return os.Rename(tmpPath, filePath)
}
