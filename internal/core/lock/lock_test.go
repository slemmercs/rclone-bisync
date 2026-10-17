package lock

import (
"os"
"path/filepath"
"strconv"
"testing"
)

func TestLockManagerAcquireRelease(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	lm := NewLockManager(lockPath, 3600, true)

	acquired, err := lm.Acquire()
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}
	if !acquired {
		t.Fatal("expected lock to be acquired")
	}

	if !IsLocked(lockPath) {
		t.Fatal("expected IsLocked to return true while lock held")
	}

	// Verify PID was recorded
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("failed to read lock file: %v", err)
	}
	if string(data) != strconv.Itoa(os.Getpid()) {
		t.Fatalf("expected PID %d, got %s", os.Getpid(), string(data))
	}

	// Second acquire should fail while held
	lm2 := NewLockManager(lockPath, 3600, false)
	acq2, err := lm2.Acquire()
	if err != nil {
		t.Fatalf("unexpected error on second acquire: %v", err)
	}
	if acq2 {
		t.Fatal("expected second acquire to fail while lock is held")
	}

	// Release
	if err := lm.Release(); err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}

	// Now second lock manager should be able to acquire
	acq3, err := lm2.Acquire()
	if err != nil {
		t.Fatalf("failed to acquire lock after release: %v", err)
	}
	if !acq3 {
		t.Fatal("expected second manager to acquire released lock")
	}
	_ = lm2.Release()
}

func TestStalePIDLockRecovery(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "stale.lock")

	// Write an invalid/dead PID to lockfile (e.g. 99999999)
	deadPID := 99999999
	if err := os.WriteFile(lockPath, []byte(strconv.Itoa(deadPID)), 0600); err != nil {
		t.Fatalf("failed to write dead lock file: %v", err)
	}

	lm := NewLockManager(lockPath, 3600, true)
	acquired, err := lm.Acquire()
	if err != nil {
		t.Fatalf("unexpected error recovering stale lock: %v", err)
	}
	if !acquired {
		t.Fatal("expected to acquire lock by recovering dead PID")
	}

	// Read content, should now be current PID
	data, _ := os.ReadFile(lockPath)
	if string(data) != strconv.Itoa(os.Getpid()) {
		t.Fatalf("expected recovered lock to contain PID %d, got %s", os.Getpid(), string(data))
	}
	_ = lm.Release()
}
