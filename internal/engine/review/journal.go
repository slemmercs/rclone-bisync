package review

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

func withFileLock(targetPath string, exclusive bool, fn func() error) error {
	if targetPath == "" {
		return fn()
	}
	cleanPath := filepath.Clean(targetPath)
	lockPath := cleanPath + ".flock"
	dir := filepath.Dir(lockPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory for lock: %w", err)
	}

	// #nosec G304 - advisory process flock
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("failed to open flock file %s: %w", lockPath, err)
	}
	defer f.Close()

	how := syscall.LOCK_SH
	if exclusive {
		how = syscall.LOCK_EX
	}
	if err := syscall.Flock(int(f.Fd()), how); err != nil {
		return fmt.Errorf("failed to acquire flock on %s: %w", lockPath, err)
	}
	defer func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	}()

	return fn()
}

// SaveActiveSession writes the current ReviewSession to disk atomically under exclusive flock.
func SaveActiveSession(activeFilePath string, session *models.ReviewSession) error {
	if activeFilePath == "" {
		return fmt.Errorf("active review session file path is empty")
	}
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	return withFileLock(activeFilePath, true, func() error {
		session.UpdatedAt = time.Now().UTC()
		data, err := json.MarshalIndent(session, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal review session: %w", err)
		}

		dir := filepath.Dir(activeFilePath)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		tmpFile := fmt.Sprintf("%s.tmp.%d", activeFilePath, os.Getpid())
		// #nosec G304 - atomic session file creation
		f, err := os.OpenFile(filepath.Clean(tmpFile), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}

		if _, err := f.Write(data); err != nil {
			_ = f.Close()
			_ = os.Remove(tmpFile)
			return fmt.Errorf("failed to write temp file: %w", err)
		}

		if err := f.Sync(); err != nil {
			_ = f.Close()
			_ = os.Remove(tmpFile)
			return fmt.Errorf("failed to sync temp file: %w", err)
		}

		if err := f.Close(); err != nil {
			_ = os.Remove(tmpFile)
			return fmt.Errorf("failed to close temp file: %w", err)
		}

		if err := os.Rename(tmpFile, activeFilePath); err != nil {
			_ = os.Remove(tmpFile)
			return fmt.Errorf("failed to atomically rename %s to %s: %w", tmpFile, activeFilePath, err)
		}

		return nil
	})
}

// LoadActiveSession loads the active ReviewSession from disk under shared flock, or returns nil if none exists.
func LoadActiveSession(activeFilePath string) (*models.ReviewSession, error) {
	if activeFilePath == "" {
		return nil, nil
	}

	cleanPath := filepath.Clean(activeFilePath)
	if strings.Contains(cleanPath, "..") {
		return nil, fmt.Errorf("invalid review session path: directory traversal not allowed")
	}

	var session *models.ReviewSession
	err := withFileLock(activeFilePath, false, func() error {
		// #nosec G304 - reading verified active review session file
		data, err := os.ReadFile(cleanPath)
		if err != nil {
			if os.IsNotExist(err) {
				session = nil
				return nil
			}
			return fmt.Errorf("failed to read active review session file: %w", err)
		}

		var s models.ReviewSession
		if err := json.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("failed to unmarshal active review session: %w", err)
		}
		session = &s
		return nil
	})
	if err != nil {
		return nil, err
	}
	return session, nil
}

// DismissActiveSessionFile removes the active session file under exclusive flock.
func DismissActiveSessionFile(activeFilePath string) error {
	if activeFilePath == "" {
		return nil
	}
	return withFileLock(activeFilePath, true, func() error {
		cleanPath := filepath.Clean(activeFilePath)
		if err := os.Remove(cleanPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove active session file: %w", err)
		}
		return nil
	})
}

// ArchiveSession moves an active session file into the archived sessions directory.
func ArchiveSession(reviewDir, activeFilePath string, session *models.ReviewSession) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}
	if reviewDir == "" {
		return fmt.Errorf("reviewDir is empty")
	}

	if err := os.MkdirAll(reviewDir, 0700); err != nil {
		return fmt.Errorf("failed to create review archive directory: %w", err)
	}

	archivePath := filepath.Join(reviewDir, fmt.Sprintf("%s.json", session.ID))
	if err := SaveActiveSession(archivePath, session); err != nil {
		return fmt.Errorf("failed to archive session: %w", err)
	}

	if activeFilePath != "" {
		return DismissActiveSessionFile(activeFilePath)
	}

	return nil
}

// AppendWAL appends a write-ahead log entry to the session's WAL file with fsync durability and exclusive flock.
func AppendWAL(walPath string, entry models.WALEntry) error {
	if walPath == "" {
		return fmt.Errorf("walPath is empty")
	}

	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal WAL entry: %w", err)
	}

	dir := filepath.Dir(walPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory for WAL: %w", err)
	}

	return withFileLock(walPath, true, func() error {
		// #nosec G304 - appending to session WAL
		f, err := os.OpenFile(filepath.Clean(walPath), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return fmt.Errorf("failed to open WAL file: %w", err)
		}
		defer f.Close()

		if _, err := f.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("failed to append to WAL file: %w", err)
		}

		if err := f.Sync(); err != nil {
			return fmt.Errorf("failed to fsync WAL file: %w", err)
		}

		return nil
	})
}

// RemoveWAL removes the WAL file and its flock under exclusive lock.
func RemoveWAL(walPath string) error {
	if walPath == "" {
		return nil
	}
	return withFileLock(walPath, true, func() error {
		cleanPath := filepath.Clean(walPath)
		if err := os.Remove(cleanPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove WAL file: %w", err)
		}
		return nil
	})
}

// ReadWAL reads all recorded entries from a session's write-ahead log under shared flock.
func ReadWAL(walPath string) ([]models.WALEntry, error) {
	if walPath == "" {
		return nil, nil
	}

	cleanPath := filepath.Clean(walPath)
	if strings.Contains(cleanPath, "..") {
		return nil, fmt.Errorf("invalid wal path: directory traversal not allowed")
	}

	var entries []models.WALEntry
	err := withFileLock(walPath, false, func() error {
		// #nosec G304 - reading session WAL
		f, err := os.Open(cleanPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("failed to open WAL file: %w", err)
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			var entry models.WALEntry
			if err := json.Unmarshal(line, &entry); err != nil {
				return fmt.Errorf("corrupt WAL line: %w", err)
			}
			entries = append(entries, entry)
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("error scanning WAL file: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// ReconcileSessionWithWAL reconciles an interrupted session using its WAL entries.
func ReconcileSessionWithWAL(session *models.ReviewSession, walEntries []models.WALEntry) *models.ReviewSession {
	if session == nil {
		return nil
	}
	if len(walEntries) == 0 {
		return session
	}

	latestStep := make(map[string]models.WALEntry)
	for _, entry := range walEntries {
		if entry.ActionID != "" {
			latestStep[entry.ActionID] = entry
		}
	}

	hasFailed := false
	allCompleted := true

	for i := range session.Actions {
		action := &session.Actions[i]
		if entry, exists := latestStep[action.ID]; exists {
			switch entry.Step {
			case "MUTATION_COMPLETED":
				action.Status = "completed"
			case "ROLLBACK_COMPLETED":
				action.Status = "rolled_back"
			case "MUTATION_STARTED":
				action.Status = "interrupted"
				hasFailed = true
				allCompleted = false
			default:
				if entry.Error != "" {
					action.Status = "failed"
					action.Error = entry.Error
					hasFailed = true
					allCompleted = false
				}
			}
		} else if action.Status == "pending" {
			allCompleted = false
		}
	}

	knownActionIDs := make(map[string]bool, len(session.Actions))
	knownPaths := make(map[string]bool, len(session.Actions))
	for _, a := range session.Actions {
		knownActionIDs[a.ID] = true
		if a.ItemPath != "" {
			knownPaths[a.ItemPath] = true
		}
	}
	for _, entry := range walEntries {
		if entry.Step == "MUTATION_COMPLETED" && entry.ItemPath != "" && !knownActionIDs[entry.ActionID] && !knownPaths[entry.ItemPath] {
			knownActionIDs[entry.ActionID] = true
			knownPaths[entry.ItemPath] = true
			session.Actions = append(session.Actions, models.ResolutionAction{
				ID:       entry.ActionID,
				Type:     entry.Type,
				ItemPath: entry.ItemPath,
				Status:   "completed",
			})
		}
	}

	if session.Status == models.ReviewStatusApplying {
		if hasFailed {
			session.Status = models.ReviewStatusFailed
		} else if allCompleted && len(session.Actions) > 0 {
			session.Status = models.ReviewStatusReady
		}
	}

	return session
}
