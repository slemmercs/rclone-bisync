package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWriteAndRead(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "state-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	file := filepath.Join(tmpDir, "test.json")
	data := map[string]interface{}{"status": "healthy", "count": float64(10)}

	if err := WriteJSONAtomic(file, data); err != nil {
		t.Fatalf("WriteJSONAtomic failed: %v", err)
	}

	var readData map[string]interface{}
	if err := ReadJSONFile(file, &readData); err != nil {
		t.Fatalf("ReadJSONFile failed: %v", err)
	}

	if readData["status"] != "healthy" || readData["count"] != float64(10) {
		t.Fatalf("unexpected content read back: %v", readData)
	}
}

func TestPendingDeletionsLifecycle(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pending-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	file := filepath.Join(tmpDir, "pending.json")

	// Initially missing
	pending, err := ReadPendingDeletions(file)
	if err != nil || pending != nil {
		t.Fatalf("expected nil when file missing, got: %v, err: %v", pending, err)
	}

	// Write
	record := PendingDeletionsData{
		HasDeletions: true,
		TotalCount:   2,
		Source:       "local",
		TargetAction: "delete from remote",
		Message:      "Too many deletes",
		Files:        []string{"a.txt", "b.txt"},
	}
	if err := RecordPendingDeletions(file, record); err != nil {
		t.Fatalf("RecordPendingDeletions failed: %v", err)
	}

	// Read
	readPending, err := ReadPendingDeletions(file)
	if err != nil || readPending == nil {
		t.Fatalf("expected pending deletions data, got nil or err: %v", err)
	}
	if len(readPending.Files) != 2 || readPending.Files[0] != "a.txt" {
		t.Fatalf("unexpected files in pending deletion: %v", readPending.Files)
	}

	// Clear
	if !ClearPendingDeletions(file) {
		t.Fatal("expected ClearPendingDeletions to return true")
	}

	// Read again
	afterClear, _ := ReadPendingDeletions(file)
	if afterClear != nil {
		t.Fatal("expected nil after clear")
	}
}

func TestRateLimitBackoffLifecycle(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ratelimit-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	flagPath := filepath.Join(tmpDir, "rate_limit.flag")

	if IsRateLimitActive(flagPath) {
		t.Fatal("expected rate limit backoff to be inactive initially")
	}
	if IsRateLimitActive("") {
		t.Fatal("expected empty path to return inactive")
	}

	if err := SetRateLimitBackoff(flagPath, 3600); err != nil {
		t.Fatalf("SetRateLimitBackoff failed: %v", err)
	}
	if err := SetRateLimitBackoff("", 3600); err != nil {
		t.Fatalf("expected nil for empty path: %v", err)
	}

	if !IsRateLimitActive(flagPath) {
		t.Fatal("expected rate limit backoff to be active after setting")
	}

	if ClearRateLimitBackoff("") {
		t.Fatal("expected false clearing empty path")
	}
	if ClearRateLimitBackoff(filepath.Join(tmpDir, "missing.flag")) {
		t.Fatal("expected false clearing missing flag")
	}
	if !ClearRateLimitBackoff(flagPath) {
		t.Fatal("expected ClearRateLimitBackoff to return true")
	}
	if IsRateLimitActive(flagPath) {
		t.Fatal("expected rate limit to be inactive after clearing")
	}
}

func TestNotificationStateLifecycle(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "notif-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	notifPath := filepath.Join(tmpDir, "notification_state.json")

	sEmpty := LoadNotificationState("")
	if sEmpty.LastSuccessTimestamp != 0 || sEmpty.LastErrorType != "" {
		t.Fatalf("expected empty default state, got: %v", sEmpty)
	}
	sMissing := LoadNotificationState(notifPath)
	if sMissing.LastSuccessTimestamp != 0 {
		t.Fatalf("expected empty state on missing file, got: %v", sMissing)
	}

	initState := NotificationState{
		LastSuccessTimestamp: 1729000000.5,
		LastErrorType:        "RATE_LIMIT",
		LastErrorTimestamp:   1729001000.0,
	}
	if err := WriteNotificationState("", initState); err != nil {
		t.Fatalf("expected nil for empty path write: %v", err)
	}
	if err := WriteNotificationState(notifPath, initState); err != nil {
		t.Fatalf("WriteNotificationState failed: %v", err)
	}

	loaded := LoadNotificationState(notifPath)
	if loaded.LastSuccessTimestamp != initState.LastSuccessTimestamp ||
		loaded.LastErrorType != initState.LastErrorType ||
		loaded.LastErrorTimestamp != initState.LastErrorTimestamp {
		t.Fatalf("loaded state does not match: got %v, expected %v", loaded, initState)
	}
}

func TestMetricsEntryLifecycle(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "metrics-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	metricsPath := filepath.Join(tmpDir, "metrics.jsonl")

	if err := AppendMetricsEntry("", MetricsEntry{}); err != nil {
		t.Fatalf("expected nil for empty path: %v", err)
	}

	entry1 := MetricsEntry{
		Timestamp:       100.0,
		Datetime:        "2026-10-17T20:00:00Z",
		RunID:           "run-1",
		JobID:           1,
		Origin:          "timer",
		Success:         true,
		DurationSeconds: 1.25,
		FilesSynced:     5,
	}
	entry2 := MetricsEntry{
		Timestamp:       200.0,
		Datetime:        "2026-10-17T21:00:00Z",
		RunID:           "run-2",
		JobID:           2,
		Origin:          "cli",
		Success:         false,
		DurationSeconds: 0.5,
		ErrorType:       "RATE_LIMIT",
		FilesSynced:     0,
	}

	if err := AppendMetricsEntry(metricsPath, entry1); err != nil {
		t.Fatalf("failed to append entry 1: %v", err)
	}
	if err := AppendMetricsEntry(metricsPath, entry2); err != nil {
		t.Fatalf("failed to append entry 2: %v", err)
	}

	data, err := os.ReadFile(metricsPath)
	if err != nil || len(data) == 0 {
		t.Fatalf("failed to read metrics file: %v", err)
	}
}

func TestStateEdgeCases(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "state-edge-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 1. ReadPendingDeletions with empty path
	p, err := ReadPendingDeletions("")
	if err != nil || p != nil {
		t.Fatalf("expected nil for empty path: %v, %v", p, err)
	}

	// 2. Corrupt JSON in ReadPendingDeletions
	corruptFile := filepath.Join(tmpDir, "corrupt.json")
	_ = os.WriteFile(corruptFile, []byte("{bad-json"), 0644)
	if _, err := ReadPendingDeletions(corruptFile); err == nil {
		t.Fatal("expected error reading corrupt pending deletions")
	}

	// 3. ClearPendingDeletions on empty path and missing file
	if ClearPendingDeletions("") {
		t.Fatal("expected false for ClearPendingDeletions with empty path")
	}
	if ClearPendingDeletions(filepath.Join(tmpDir, "missing.json")) {
		t.Fatal("expected false for ClearPendingDeletions with missing file")
	}

	// 4. WriteJSONAtomic error on unmarshallable type (e.g. channel)
	unmarshallable := map[string]interface{}{"ch": make(chan int)}
	if err := WriteJSONAtomic(filepath.Join(tmpDir, "unmarshal.json"), unmarshallable); err == nil {
		t.Fatal("expected error marshalling unmarshallable type")
	}
}
