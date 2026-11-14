package profiles

import (
	"fmt"
	"time"
)

// GetHistory returns the version history for a profile.
func (s *Store) GetHistory(id string) ([]ProfileVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, exists := s.profiles[id]; !exists {
		return nil, ErrProfileNotFound
	}

	hist, ok := s.history[id]
	if !ok || len(hist) == 0 {
		return []ProfileVersion{}, nil
	}

	out := make([]ProfileVersion, len(hist))
	copy(out, hist)
	return out, nil
}

// Rollback restores a profile configuration to an earlier version.
func (s *Store) Rollback(id string, targetVersion int) (*Profile, error) {
	var ret Profile
	err := s.withFileLock(func() error {
		s.mu.Lock()
		defer s.mu.Unlock()

		_ = s.reloadIfModifiedLocked()

		p, exists := s.profiles[id]
		if !exists {
			return ErrProfileNotFound
		}

		var targetSnapshot *Profile
		for _, v := range s.history[id] {
			if v.Version == targetVersion {
				snap := v.Snapshot
				targetSnapshot = &snap
				break
			}
		}
		if targetSnapshot == nil {
			return ErrVersionNotFound
		}

		now := time.Now().UTC()
		newVersion := p.Version + 1

		restored := *targetSnapshot
		restored.Version = newVersion
		restored.UpdatedAt = now
		restored.State = p.State

		snapshot := ProfileVersion{
			Version:   newVersion,
			ProfileID: id,
			Timestamp: now,
			Reason:    fmt.Sprintf("Rollback to version %d", targetVersion),
			Snapshot:  restored,
		}
		s.history[id] = append(s.history[id], snapshot)
		s.profiles[id] = &restored

		if err := s.persistLocked(); err != nil {
			return err
		}

		ret = restored
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &ret, nil
}
