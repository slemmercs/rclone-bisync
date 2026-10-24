package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// Logger provides thread-safe structured logging with file and console sinks.
type Logger struct {
	mu            sync.Mutex
	logFile       *os.File
	filePath      string
	enableConsole bool
	slogLogger    *slog.Logger
}

// New initializes a Logger instance writing to filePath and optionally to console.
func New(filePath string, enableConsole bool) (*Logger, error) {
	var file *os.File
	var err error

	if filePath != "" {
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
		file, err = os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %q: %w", filePath, err)
		}
	}

	var writers []io.Writer
	if file != nil {
		writers = append(writers, file)
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
		logFile:       file,
		filePath:      filePath,
		enableConsole: enableConsole,
		slogLogger:    slogL,
	}, nil
}

// Close closes the underlying file handle if open.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.logFile != nil {
		err := l.logFile.Close()
		l.logFile = nil
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

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal BisyncLogEntry: %w", err)
	}
	line := append(data, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFile != nil {
		if _, err := l.logFile.Write(line); err != nil {
			return fmt.Errorf("failed to write to log file: %w", err)
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
