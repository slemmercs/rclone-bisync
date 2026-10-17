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
