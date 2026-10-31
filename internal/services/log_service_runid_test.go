package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogService_FilterLogsByRunID(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "bisync.log")
	svc := NewLogService()

	// 1. Missing file returns empty slice and nil error
	res, err := svc.FilterLogsByRunID(filepath.Join(tmpDir, "missing.log"), "dedupe-123", 50)
	if err != nil || len(res) != 0 {
		t.Fatalf("expected empty result for missing file, got %v, err: %v", res, err)
	}

	// 2. Empty runID returns empty slice and nil error
	res, err = svc.FilterLogsByRunID(logPath, "", 50)
	if err != nil || len(res) != 0 {
		t.Fatalf("expected empty result for empty runID, got %v, err: %v", res, err)
	}

	// 3. Populate test log file with different run_ids
	testData := []string{
		`{"time":"2026-10-31T07:10:00Z","level":"info","type":"dedupe","msg":"[gdrive] Deduplication started","run_id":"dedupe-1001","origin":"dedupe"}`,
		`{"time":"2026-10-31T07:10:01Z","level":"notice","type":"dedupe","msg":"[gdrive] Deleted duplicate file1.txt with token ya29.secret_password","run_id":"dedupe-1001","origin":"dedupe"}`,
		`{"time":"2026-10-31T07:11:00Z","level":"info","type":"sync","msg":"Sync running","run_id":"sync_2002","job_id":99}`,
		`{"time":"2026-10-31T07:12:00Z","level":"notice","type":"audit","msg":"[default] DEDUPE_EXECUTED","details":{"run_id":"dedupe-1001","collisions":1,"deleted":1}}`,
		`{"time":"2026-10-31T07:15:00Z","level":"info","type":"dedupe","msg":"[onedrive] No duplicates","run_id":"dedupe-3003","origin":"dedupe"}`,
	}
	content := strings.Join(testData, "\n") + "\n"
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test log file: %v", err)
	}

	// 4. Query dedupe-1001: should match direct run_id lines and audit details.run_id line
	matched, err := svc.FilterLogsByRunID(logPath, "dedupe-1001", 50)
	if err != nil {
		t.Fatalf("unexpected error filtering logs by run_id: %v", err)
	}
	if len(matched) != 3 {
		t.Fatalf("expected 3 matched lines for dedupe-1001, got %d: %+v", len(matched), matched)
	}

	// Verify secrets were masked
	if strings.Contains(matched[1], "secret_password") {
		t.Errorf("expected secret token to be redacted in output, got: %s", matched[1])
	}

	// 5. Query non-existent run_id: should return empty
	matchedNone, err := svc.FilterLogsByRunID(logPath, "dedupe-9999", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matchedNone) != 0 {
		t.Fatalf("expected 0 matches for non-existent run_id, got %d", len(matchedNone))
	}

	// 6. Test limit clamping
	matchedLimited, err := svc.FilterLogsByRunID(logPath, "dedupe-1001", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matchedLimited) != 2 {
		t.Fatalf("expected 2 lines clamped by limit, got %d", len(matchedLimited))
	}
}

func TestLogService_GetRunTimeWindow(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "bisync.log")
	svc := NewLogService()

	// 1. Missing file or empty runID
	_, _, found := svc.GetRunTimeWindow(filepath.Join(tmpDir, "missing.log"), "sync_1")
	if found {
		t.Fatal("expected found=false for missing file")
	}
	_, _, found = svc.GetRunTimeWindow(logPath, "")
	if found {
		t.Fatal("expected found=false for empty runID")
	}

	// 2. Populate log with sync run entry containing start_time and end_time
	startStr := "2026-10-31T07:20:00Z"
	endStr := "2026-10-31T07:21:15Z"
	lines := []string{
		fmt.Sprintf(`{"time":"2026-10-31T07:21:15Z","level":"info","type":"sync","msg":"Sync done","run_id":"sync_abc","start_time":"%s","end_time":"%s"}`, startStr, endStr),
		`{"time":"2026-10-31T07:25:00Z","level":"error","type":"sync","msg":"Aborted","run_id":"sync_xyz"}`,
	}
	if err := os.WriteFile(logPath, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatalf("failed to write test log file: %v", err)
	}

	// 3. Find sync_abc
	st, et, found := svc.GetRunTimeWindow(logPath, "sync_abc")
	if !found {
		t.Fatalf("expected found=true for sync_abc")
	}
	expectedStart, _ := time.Parse(time.RFC3339, startStr)
	expectedEnd, _ := time.Parse(time.RFC3339, endStr)
	if !st.Equal(expectedStart) || !et.Equal(expectedEnd) {
		t.Fatalf("unexpected window: got %v → %v, want %v → %v", st, et, expectedStart, expectedEnd)
	}

	// 4. Find sync_xyz (only time field present)
	stXyz, _, foundXyz := svc.GetRunTimeWindow(logPath, "sync_xyz")
	if !foundXyz || stXyz.IsZero() {
		t.Fatalf("expected found=true and non-zero start time for sync_xyz, got %v, %v", foundXyz, stXyz)
	}
}
