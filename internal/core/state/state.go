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
