package review

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// SaveActiveSession writes the current ReviewSession to disk atomically.
func SaveActiveSession(activeFilePath string, session *models.ReviewSession) error {
	if activeFilePath == "" {
		return fmt.Errorf("active review session file path is empty")
	}
	if session == nil {
		return fmt.Errorf("session is nil")
	}

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
	f, err := os.OpenFile(tmpFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
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
}

// LoadActiveSession loads the active ReviewSession from disk, or returns nil if none exists.
func LoadActiveSession(activeFilePath string) (*models.ReviewSession, error) {
	if activeFilePath == "" {
		return nil, nil
	}

	data, err := os.ReadFile(activeFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read active review session file: %w", err)
	}

	var session models.ReviewSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal active review session: %w", err)
	}

	return &session, nil
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
		_ = os.Remove(activeFilePath)
	}

	return nil
}

// AppendWAL appends a write-ahead log entry to the session's WAL file with fsync durability.
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

	f, err := os.OpenFile(walPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
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
}

// ReadWAL reads all recorded entries from a session's write-ahead log.
func ReadWAL(walPath string) ([]models.WALEntry, error) {
	if walPath == "" {
		return nil, nil
	}

	f, err := os.Open(walPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open WAL file: %w", err)
	}
	defer f.Close()

	var entries []models.WALEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry models.WALEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			return nil, fmt.Errorf("corrupt WAL line: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning WAL file: %w", err)
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

	if session.Status == models.ReviewStatusApplying {
		if hasFailed {
			session.Status = models.ReviewStatusFailed
		} else if allCompleted && len(session.Actions) > 0 {
			session.Status = models.ReviewStatusReady
		}
	}

	return session
}
