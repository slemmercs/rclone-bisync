package lock

import (
"fmt"
"io"
"os"
"path/filepath"
"strconv"
"strings"
"time"

"golang.org/x/sys/unix"
)

// CheckPIDExists checks if a process with the given PID is currently active.
func CheckPIDExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := unix.Kill(pid, 0)
	if err == nil {
		return true
	}
	// EPERM means process exists but we don't have permission to signal it
return err == unix.EPERM
}

// LockManager manages an exclusive process lockfile using POSIX flock.
type LockManager struct {
LockFilePath        string
StaleTimeoutSeconds int
CheckStale          bool
file                *os.File
}

// NewLockManager creates a new LockManager instance.
func NewLockManager(lockFilePath string, staleTimeoutSeconds int, checkStale bool) *LockManager {
if staleTimeoutSeconds <= 0 {
staleTimeoutSeconds = 3600
}
return &LockManager{
LockFilePath:        lockFilePath,
StaleTimeoutSeconds: staleTimeoutSeconds,
CheckStale:          checkStale,
}
}

// Acquire acquires an exclusive lock without truncation until acquired.
func (lm *LockManager) Acquire() (bool, error) {
if err := os.MkdirAll(filepath.Dir(lm.LockFilePath), 0700); err != nil {
return false, fmt.Errorf("failed to create lock directory: %w", err)
}

f, err := os.OpenFile(lm.LockFilePath, os.O_RDWR|os.O_CREATE, 0600)
if err != nil {
return false, fmt.Errorf("failed to open lock file: %w", err)
}
lm.file = f

// Attempt non-blocking exclusive lock
err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
if err == nil {
// Acquired lock: safe to truncate and write PID
if err := f.Truncate(0); err != nil {
_ = lm.closeHandle()
return false, err
}
if _, err := f.Seek(0, io.SeekStart); err != nil {
_ = lm.closeHandle()
return false, err
}
pidStr := strconv.Itoa(os.Getpid())
if _, err := f.WriteString(pidStr); err != nil {
_ = lm.closeHandle()
return false, err
}
_ = f.Sync()
return true, nil
}

// Could not acquire immediately (lock held)
if !lm.CheckStale {
_ = lm.closeHandle()
return false, nil
}

return lm.handleStaleLock()
}

func (lm *LockManager) handleStaleLock() (bool, error) {
contentBytes, err := os.ReadFile(lm.LockFilePath)
if err == nil {
content := strings.TrimSpace(string(contentBytes))
if content != "" {
if oldPID, err := strconv.Atoi(content); err == nil {
if !CheckPIDExists(oldPID) {
// Stale PID: not running
_ = lm.closeHandle()
_ = os.Remove(lm.LockFilePath)
return lm.Acquire()
}

if fi, err := os.Stat(lm.LockFilePath); err == nil {
lockAge := time.Since(fi.ModTime()).Seconds()
if lockAge > float64(lm.StaleTimeoutSeconds) {
// Lock age exceeded timeout
_ = lm.closeHandle()
_ = os.Remove(lm.LockFilePath)
return lm.Acquire()
}
}
}
}
}

_ = lm.closeHandle()
return false, nil
}

// Release unlocks and closes the lockfile.
func (lm *LockManager) Release() error {
if lm.file == nil {
return nil
}
defer lm.closeHandle()
return unix.Flock(int(lm.file.Fd()), unix.LOCK_UN)
}

func (lm *LockManager) closeHandle() error {
if lm.file != nil {
err := lm.file.Close()
lm.file = nil
return err
}
return nil
}

// IsLocked checks whether the given lockfile is actively held.
func IsLocked(lockFilePath string) bool {
if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
return false
}

f, err := os.Open(lockFilePath)
if err != nil {
return false
}
defer f.Close()

// Try non-blocking shared lock
err = unix.Flock(int(f.Fd()), unix.LOCK_SH|unix.LOCK_NB)
if err == nil {
// Could get shared lock -> not exclusively locked
_ = unix.Flock(int(f.Fd()), unix.LOCK_UN)
return false
}
return true
}
