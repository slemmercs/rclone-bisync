package logger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

func TestLogger_NewAndClose(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	l, err := New(logPath, false)
	if err != nil {
		t.Fatalf("unexpected error creating logger: %v", err)
	}

	if l.Slog() == nil {
		t.Fatal("expected slog logger to not be nil")
	}

	if err := l.Close(); err != nil {
		t.Fatalf("unexpected error closing logger: %v", err)
	}

	// Multiple close should be safe
	if err := l.Close(); err != nil {
		t.Fatalf("subsequent close returned error: %v", err)
	}

	// Empty path
	emptyL, err := New("", false)
	if err != nil {
		t.Fatalf("unexpected error with empty path: %v", err)
	}
	_ = emptyL.Close()
}

func TestLogger_RecordSyncRun(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "bisync.log")

	l, err := New(logPath, true)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer l.Close()

	entry := models.BisyncLogEntry{
		Msg:         "Sync completed successfully",
		RunID:       "run_12345",
		JobID:       42,
		Origin:      "timer",
		Status:      "SUCCESS",
		DurationSec: 12.5,
		FilesSynced: 3,
		StartTime:   time.Now().Add(-12 * time.Second).Format(time.RFC3339),
		EndTime:     time.Now().Format(time.RFC3339),
	}

	if err := l.RecordSyncRun(entry); err != nil {
		t.Fatalf("failed to record sync run: %v", err)
	}

	// Read and verify JSON record
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var parsed models.BisyncLogEntry
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal recorded log: %v", err)
	}

	if parsed.RunID != "run_12345" || parsed.JobID != 42 || parsed.FilesSynced != 3 {
		t.Fatalf("unexpected parsed entry: %+v", parsed)
	}
	if parsed.Level != "info" || parsed.Time == "" {
		t.Fatalf("expected defaults populated for level and time: %+v", parsed)
	}
}

func TestLogger_SlogMethods(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "slog.log")

	l, err := New(logPath, false)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer l.Close()

	l.Info("info message", "key", "val")
	l.Warn("warn message")
	l.Error("error message")
	l.Debug("debug message")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected logged output in file")
	}
}

func TestLogger_ConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "concurrent.log")

	l, err := New(logPath, false)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer l.Close()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_ = l.RecordSyncRun(models.BisyncLogEntry{
				RunID:       "run_concurrent",
				JobID:       idx,
				Status:      "SUCCESS",
				DurationSec: 1.0,
			})
			l.Info("concurrent log", "idx", idx)
		}(i)
	}
	wg.Wait()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected content in concurrent log")
	}
}
