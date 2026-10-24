package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

func TestRotator_BasicAndThreshold(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	// Limit to 80 bytes per file, max 3 backups (.1, .2, .3)
	rot, err := NewRotator(logPath, 80, 3)
	if err != nil {
		t.Fatalf("failed to create rotator: %v", err)
	}
	defer rot.Close()

	// Write small chunks that trigger rotations
	for i := 0; i < 10; i++ {
		msg := fmt.Sprintf("line %02d: this is a log message that exceeds size\n", i)
		n, err := rot.Write([]byte(msg))
		if err != nil || n != len(msg) {
			t.Fatalf("write failed on step %d: n=%d, err=%v", i, n, err)
		}
	}

	// Verify main file exists
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("main log file does not exist: %v", err)
	}

	// Verify backup .1 exists
	b1 := logPath + ".1"
	if _, err := os.Stat(b1); err != nil {
		t.Fatalf("backup log .1 does not exist: %v", err)
	}

	// Verify backup .2 exists
	b2 := logPath + ".2"
	if _, err := os.Stat(b2); err != nil {
		t.Fatalf("backup log .2 does not exist: %v", err)
	}

	// Verify backup .3 exists
	b3 := logPath + ".3"
	if _, err := os.Stat(b3); err != nil {
		t.Fatalf("backup log .3 does not exist: %v", err)
	}

	// Verify backup .4 does NOT exist (maxBackups = 3)
	b4 := logPath + ".4"
	if _, err := os.Stat(b4); !os.IsNotExist(err) {
		t.Fatalf("backup log .4 should not exist, but stat returned: %v", err)
	}
}

func TestLogger_RecordAuditEventAndRotation(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "bisync.log")

	l, err := NewWithRotation(logPath, false, 1, 2)
	if err != nil {
		t.Fatalf("failed to create logger with rotation: %v", err)
	}
	defer l.Close()

	auditEntry := models.AuditLogRecord{
		SeqID:     1,
		ProfileID: "gdrive",
		Action:    "CONFLICT_RESOLVED",
		Actor:     "USER",
		Hash:      "abcdef123456",
		PrevHash:  "000000000000",
		Msg:       "Resolved conflict for test.txt",
		Details:   json.RawMessage(`{"path":"test.txt"}`),
	}

	if err := l.RecordAuditEvent(auditEntry); err != nil {
		t.Fatalf("RecordAuditEvent failed: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read bisync.log: %v", err)
	}

	var parsed models.AuditLogRecord
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal AuditLogRecord: %v", err)
	}

	if parsed.Type != "audit" {
		t.Fatalf("expected Type 'audit', got %q", parsed.Type)
	}
	if parsed.Action != "CONFLICT_RESOLVED" || parsed.SeqID != 1 {
		t.Fatalf("unexpected parsed record: %+v", parsed)
	}
}

func TestRotator_Concurrent(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "concurrent.log")

	rot, err := NewRotator(logPath, 200, 3)
	if err != nil {
		t.Fatalf("failed to create rotator: %v", err)
	}
	defer rot.Close()

	var wg sync.WaitGroup
	workers := 10
	iterations := 20

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				line := []byte(fmt.Sprintf("w=%d i=%d message\n", workerID, i))
				_, _ = rot.Write(line)
			}
		}(w)
	}

	wg.Wait()

	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("log file should exist after concurrent writes: %v", err)
	}
}
