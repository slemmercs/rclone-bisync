package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// Rotator implements a thread-safe io.WriteCloser that automatically rotates
// files when they exceed a size threshold, keeping up to MaxBackups archive files.
type Rotator struct {
	mu         sync.Mutex
	filePath   string
	maxSize    int64
	maxBackups int
	size       int64
	file       *os.File
}

// Ensure Rotator implements io.WriteCloser.
var _ io.WriteCloser = (*Rotator)(nil)

// NewRotator creates a Rotator for filePath.
// maxSize is the file size threshold in bytes. If maxSize <= 0, default to 10 MB (10 * 1024 * 1024).
// maxBackups is the number of rotated backups retained. If maxBackups <= 0, default to 5.
func NewRotator(filePath string, maxSize int64, maxBackups int) (*Rotator, error) {
	if filePath == "" {
		return nil, fmt.Errorf("rotator file path cannot be empty")
	}
	if maxSize <= 0 {
		maxSize = 10 * 1024 * 1024
	}
	if maxBackups <= 0 {
		maxBackups = 5
	}

	r := &Rotator{
		filePath:   filePath,
		maxSize:    maxSize,
		maxBackups: maxBackups,
	}

	if err := r.openFile(); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Rotator) openFile() error {
	dir := filepath.Dir(r.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for log: %w", err)
	}

	f, err := os.OpenFile(r.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", r.filePath, err)
	}

	fi, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to stat log file %s: %w", r.filePath, err)
	}

	r.file = f
	r.size = fi.Size()
	return nil
}

// Write writes bytes to the active log file, triggering rotation if the threshold is exceeded.
func (r *Rotator) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	writeLen := int64(len(p))
	if r.file == nil {
		if err := r.openFile(); err != nil {
			return 0, err
		}
	}

	if r.size+writeLen > r.maxSize && r.size > 0 {
		if err := r.rotate(); err != nil {
			return 0, err
		}
	}

	n, err = r.file.Write(p)
	r.size += int64(n)
	return n, err
}

// rotate closes the current log file, cascades backup archives, and reopens a clean primary file.
func (r *Rotator) rotate() error {
	if r.file != nil {
		_ = r.file.Sync()
		_ = r.file.Close()
		r.file = nil
	}

	// Remove the oldest backup if it exceeds maxBackups
	oldest := fmt.Sprintf("%s.%d", r.filePath, r.maxBackups)
	_ = os.Remove(oldest)

	// Cascade backups from maxBackups-1 down to 1
	for i := r.maxBackups - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", r.filePath, i)
		dst := fmt.Sprintf("%s.%d", r.filePath, i+1)
		if _, err := os.Stat(src); err == nil {
			_ = os.Rename(src, dst)
		}
	}

	// Move active file to .1
	dst := fmt.Sprintf("%s.1", r.filePath)
	if _, err := os.Stat(r.filePath); err == nil {
		if err := os.Rename(r.filePath, dst); err != nil {
			return fmt.Errorf("failed to rotate active file %s: %w", r.filePath, err)
		}
	}

	r.size = 0
	return r.openFile()
}

// Close flushes and closes the active file handle.
func (r *Rotator) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file != nil {
		err := r.file.Close()
		r.file = nil
		return err
	}
	return nil
}
