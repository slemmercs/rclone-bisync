package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// Logger provides thread-safe structured logging with file and console sinks and automatic rotation.
type Logger struct {
	mu            sync.Mutex
	logWriter     io.WriteCloser
	filePath      string
	enableConsole bool
	slogLogger    *slog.Logger
}

// New initializes a Logger instance with default rotation thresholds (10 MB, 5 backups).
func New(filePath string, enableConsole bool) (*Logger, error) {
	return NewWithRotation(filePath, enableConsole, 10, 5)
}

// NewWithRotation initializes a Logger instance with custom rotation parameters.
// maxSizeMB is the file size limit in megabytes.
// maxBackups is the maximum number of retained rotated archive files.
func NewWithRotation(filePath string, enableConsole bool, maxSizeMB int, maxBackups int) (*Logger, error) {
	var writer io.WriteCloser

	if filePath != "" {
		maxBytes := int64(maxSizeMB) * 1024 * 1024
		if maxBytes <= 0 {
			maxBytes = 10 * 1024 * 1024
		}
		rotator, err := NewRotator(filePath, maxBytes, maxBackups)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize rotator for %q: %w", filePath, err)
		}
		writer = rotator
	}

	var writers []io.Writer
	if writer != nil {
		writers = append(writers, writer)
	}
	if enableConsole {
		writers = append(writers, os.Stderr)
	}

	var combinedWriter io.Writer
	if len(writers) == 0 {
		combinedWriter = io.Discard
	} else if len(writers) == 1 {
		combinedWriter = writers[0]
	} else {
		combinedWriter = io.MultiWriter(writers...)
	}

	handler := slog.NewJSONHandler(combinedWriter, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slogL := slog.New(handler)

	return &Logger{
		logWriter:     writer,
		filePath:      filePath,
		enableConsole: enableConsole,
		slogLogger:    slogL,
	}, nil
}

// Close closes the underlying writer handle if open.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.logWriter != nil {
		err := l.logWriter.Close()
		l.logWriter = nil
		return err
	}
	return nil
}

// RecordSyncRun appends a single-line JSON ledger entry to the log file and console.
func (l *Logger) RecordSyncRun(entry models.BisyncLogEntry) error {
	if entry.Time == "" {
		entry.Time = time.Now().Format(time.RFC3339)
	}
	if entry.Level == "" {
		entry.Level = "info"
	}
	if entry.Type == "" {
		entry.Type = "sync"
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal BisyncLogEntry: %w", err)
	}
	line := append(data, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logWriter != nil {
		if _, err := l.logWriter.Write(line); err != nil {
			return fmt.Errorf("failed to write to log file: %w", err)
		}
	}
	if l.enableConsole {
		_, _ = os.Stderr.Write(line)
	}
	return nil
}

// RecordAuditEvent appends a cryptographically verified audit record to the log stream.
func (l *Logger) RecordAuditEvent(entry models.AuditLogRecord) error {
	if entry.Time == "" {
		entry.Time = time.Now().Format(time.RFC3339)
	}
	if entry.Level == "" {
		entry.Level = "info"
	}
	if entry.Type == "" {
		entry.Type = "audit"
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal AuditLogRecord: %w", err)
	}
	line := append(data, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logWriter != nil {
		if _, err := l.logWriter.Write(line); err != nil {
			return fmt.Errorf("failed to write audit record to log: %w", err)
		}
	}
	if l.enableConsole {
		_, _ = os.Stderr.Write(line)
	}
	return nil
}

// Info logs an informational message via slog.
func (l *Logger) Info(msg string, args ...any) {
	l.slogLogger.Info(msg, args...)
}

// Warn logs a warning message via slog.
func (l *Logger) Warn(msg string, args ...any) {
	l.slogLogger.Warn(msg, args...)
}

// Error logs an error message via slog.
func (l *Logger) Error(msg string, args ...any) {
	l.slogLogger.Error(msg, args...)
}

// Debug logs a debug message via slog.
func (l *Logger) Debug(msg string, args ...any) {
	l.slogLogger.Debug(msg, args...)
}

// Slog returns the underlying *slog.Logger.
func (l *Logger) Slog() *slog.Logger {
	return l.slogLogger
}
