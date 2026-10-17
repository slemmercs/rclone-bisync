package state

import (
"encoding/json"
"fmt"
"os"
"path/filepath"
)

// WriteJSONAtomic writes data serialized as JSON atomically using a .tmp file and rename.
func WriteJSONAtomic(filename string, data interface{}) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	tmpFile := fmt.Sprintf("%s.tmp.%d", filename, os.Getpid())
	if err := os.WriteFile(tmpFile, bytes, 0644); err != nil {
		return fmt.Errorf("failed to write tmp file: %w", err)
	}

	// Sync to disk if possible
	if f, err := os.Open(tmpFile); err == nil {
		_ = f.Sync()
		_ = f.Close()
	}

	if err := os.Rename(tmpFile, filename); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to atomic rename state file: %w", err)
	}

	return nil
}

// ReadJSONFile reads and parses a JSON file into target interface.
func ReadJSONFile(filename string, target interface{}) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// PendingDeletionsData represents structured pending deletion events.
type PendingDeletionsData struct {
	HasDeletions bool     `json:"has_deletions"`
	TotalCount   int      `json:"total_count"`
	Source       string   `json:"source"`
	TargetAction string   `json:"target_action"`
	Message      string   `json:"message"`
	Files        []string `json:"files"`
	Timestamp    float64  `json:"timestamp,omitempty"`
}

// ReadPendingDeletions reads pending deletions state if file exists.
func ReadPendingDeletions(path string) (*PendingDeletionsData, error) {
	if path == "" {
		return nil, nil
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}

	var data PendingDeletionsData
	if err := ReadJSONFile(path, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// RecordPendingDeletions writes pending deletions state atomically.
func RecordPendingDeletions(path string, data PendingDeletionsData) error {
	return WriteJSONAtomic(path, data)
}

// ClearPendingDeletions removes the pending deletions state file.
func ClearPendingDeletions(path string) bool {
	if path == "" {
		return false
	}
	if _, err := os.Stat(path); err == nil {
		return os.Remove(path) == nil
	}
	return false
}

// ClearRateLimitBackoff removes the rate limit backoff flag file.
func ClearRateLimitBackoff(path string) bool {
	if path == "" {
		return false
	}
	if _, err := os.Stat(path); err == nil {
		return os.Remove(path) == nil
	}
	return false
}

// NotificationState tracks recent notification history.
type NotificationState struct {
	LastSuccessTimestamp float64 `json:"last_success_timestamp"`
	LastErrorType        string  `json:"last_error_type"`
	LastErrorTimestamp   float64 `json:"last_error_timestamp"`
}

// LoadNotificationState loads notification state or returns default empty state.
func LoadNotificationState(path string) NotificationState {
	var s NotificationState
	if path == "" {
		return s
	}
	_ = ReadJSONFile(path, &s)
	return s
}

// WriteNotificationState atomically saves notification state.
func WriteNotificationState(path string, s NotificationState) error {
	if path == "" {
		return nil
	}
	return WriteJSONAtomic(path, s)
}

// IsRateLimitActive checks if rate limit backoff is currently active.
func IsRateLimitActive(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) || err != nil {
		return false
	}
	_ = info
	return true
}

// SetRateLimitBackoff creates the rate limit backoff flag file.
func SetRateLimitBackoff(path string, ttlSeconds int) error {
	if path == "" {
		return nil
	}
	data := map[string]interface{}{
		"timestamp": os.Getpid(), // simple token
		"ttl":       ttlSeconds,
	}
	return WriteJSONAtomic(path, data)
}

// MetricsEntry represents a logged sync operation metric.
type MetricsEntry struct {
	Timestamp       float64 `json:"timestamp"`
	Datetime        string  `json:"datetime"`
	RunID           string  `json:"run_id"`
	JobID           int     `json:"job_id"`
	Origin          string  `json:"origin"`
	Success         bool    `json:"success"`
	DurationSeconds float64 `json:"duration_seconds"`
	ErrorType       string  `json:"error_type,omitempty"`
	FilesSynced     int     `json:"files_synced"`
}

// AppendMetricsEntry appends a single metric line to the jsonl file.
func AppendMetricsEntry(path string, entry MetricsEntry) error {
	if path == "" {
		return nil
	}
	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}
