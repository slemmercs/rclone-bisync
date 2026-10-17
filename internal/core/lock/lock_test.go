package lock

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"golang.org/x/sys/unix"
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

	// 1. Holder holds flock with dead PID (99999999)
	holder, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Close()

	if err := unix.Flock(int(holder.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatalf("failed to hold initial flock: %v", err)
	}
	_, _ = holder.WriteString("99999999\n")
	_ = holder.Sync()

	lm := NewLockManager(lockPath, 3600, true)
	acquired, err := lm.Acquire()
	if err != nil || !acquired {
		t.Fatalf("expected to acquire lock by recovering dead PID: %v", err)
	}
	_ = lm.Release()

	// 2. Holder holds flock with LIVE PID (os.Getpid()) and fresh timestamp
	lockPathLive := filepath.Join(tmpDir, "live.lock")
	holderLive, err := os.OpenFile(lockPathLive, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer holderLive.Close()

	if err := unix.Flock(int(holderLive.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatalf("failed to hold live flock: %v", err)
	}
	_, _ = holderLive.WriteString(strconv.Itoa(os.Getpid()) + "\n")
	_ = holderLive.Sync()

	lmLive := NewLockManager(lockPathLive, 3600, true)
	acqLive, err := lmLive.Acquire()
	if err != nil {
		t.Fatalf("unexpected error on live acquire: %v", err)
	}
	if acqLive {
		t.Fatal("expected acquire to fail when live process holds lock")
	}

	// 3. Holder holds flock with LIVE PID but stale timeout age
	lockPathAge := filepath.Join(tmpDir, "stale_age.lock")
	holderAge, err := os.OpenFile(lockPathAge, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer holderAge.Close()

	if err := unix.Flock(int(holderAge.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatalf("failed to hold age flock: %v", err)
	}
	_, _ = holderAge.WriteString(strconv.Itoa(os.Getpid()) + "\n")
	_ = holderAge.Sync()
	pastTime := time.Now().Add(-2 * time.Hour)
	_ = os.Chtimes(lockPathAge, pastTime, pastTime)

	lmAge := NewLockManager(lockPathAge, 3600, true)
	acqAge, err := lmAge.Acquire()
	if err != nil || !acqAge {
		t.Fatalf("expected lock recovery via age timeout: %v", err)
	}
	_ = lmAge.Release()
}

func TestLockEdgeCasesAndHelpers(t *testing.T) {
	// 1. CheckPIDExists
	if CheckPIDExists(0) || CheckPIDExists(-1) {
		t.Fatal("expected false for invalid PID")
	}
	if !CheckPIDExists(os.Getpid()) {
		t.Fatal("expected true for current PID")
	}
	if CheckPIDExists(99999999) {
		t.Fatal("expected false for dead PID")
	}

	// 2. NewLockManager default timeout
	lm := NewLockManager("/tmp/test.lock", 0, true)
	if lm.StaleTimeoutSeconds != 3600 {
		t.Fatalf("expected default timeout 3600, got: %d", lm.StaleTimeoutSeconds)
	}

	// 3. Release when no file held
	lmEmpty := &LockManager{}
	if err := lmEmpty.Release(); err != nil {
		t.Fatalf("expected nil for empty lock release: %v", err)
	}

	// 4. IsLocked on non-existent file and unlocked file
	tmpDir := t.TempDir()
	missingFile := filepath.Join(tmpDir, "missing.lock")
	if IsLocked(missingFile) {
		t.Fatal("expected false for missing lock file")
	}
	unlockedFile := filepath.Join(tmpDir, "unlocked.lock")
	_ = os.WriteFile(unlockedFile, []byte("123"), 0600)
	if IsLocked(unlockedFile) {
		t.Fatal("expected false for unlocked lock file")
	}

	// 5. Corrupt PID in lock file
	corruptFile := filepath.Join(tmpDir, "corrupt.lock")
	_ = os.WriteFile(corruptFile, []byte("not-an-int"), 0600)
	holderCorrupt, _ := os.OpenFile(corruptFile, os.O_RDWR, 0600)
	_ = unix.Flock(int(holderCorrupt.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	defer holderCorrupt.Close()

	lmCorrupt := NewLockManager(corruptFile, 3600, true)
	acqCorrupt, _ := lmCorrupt.Acquire()
	if acqCorrupt {
		t.Fatal("expected acquire to fail on corrupt lock PID")
	}
}

