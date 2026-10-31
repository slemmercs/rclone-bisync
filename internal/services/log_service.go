package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/core/logger"
)

// LogService provides high-performance log parsing, filtering, and cross-correlation.
type LogService struct{}

// NewLogService creates a new LogService instance.
func NewLogService() *LogService {
	return &LogService{}
}

const maxDaemonLogLineBytes = 32 * 1024 * 1024 // 32MB safety ceiling

// FilterDaemonLogs reads the daemon JSON log and returns lines occurring within [startTime-1s, endTime+1s].
// If startTime is zero, it falls back to returning the tail of the log up to limit entries.
// If a daemon log line wraps nested JSON logs in its msg field (e.g. from bisync/rc.go),
// it unwraps them into individual, normalized JSON entries.
func (s *LogService) FilterDaemonLogs(logPath string, startTime, endTime time.Time, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 2000 {
		limit = 2000
	}

	// #nosec G304 - reading configured daemon log file
	file, err := os.Open(filepath.Clean(logPath))
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to open daemon log %q: %w", logPath, err)
	}
	defer file.Close()

	var windowStart, windowEnd time.Time
	useWindow := !startTime.IsZero()
	if useWindow {
		windowStart = startTime.Add(-1 * time.Second)
		if endTime.IsZero() {
			windowEnd = time.Now().Add(1 * time.Second)
		} else {
			windowEnd = endTime.Add(1 * time.Second)
		}
	}

	reader := bufio.NewReaderSize(file, 64*1024)
	var matchedLines []string

	for {
		lineBytes, err := reader.ReadBytes('\n')
		if len(lineBytes) > maxDaemonLogLineBytes {
			return matchedLines, fmt.Errorf("line in %q exceeds maximum limit of 32MB", logPath)
		}
		if len(lineBytes) > 0 {
			line := strings.TrimSpace(string(lineBytes))
			if line != "" {
				s.filterLine(line, useWindow, windowStart, windowEnd, limit, &matchedLines)
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return matchedLines, fmt.Errorf("error reading daemon log %q: %w", logPath, err)
		}
	}

	return matchedLines, nil
}

// FilterLogsByRunID reads the given log file (e.g. bisync.log) and returns all lines matching the given runID.
// Returned lines are in chronological order with credentials masked.
func (s *LogService) FilterLogsByRunID(logPath string, runID string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 2000 {
		limit = 2000
	}
	cleanRunID := strings.TrimSpace(runID)
	if cleanRunID == "" {
		return []string{}, nil
	}

	// #nosec G304 - reading configured log file
	file, err := os.Open(filepath.Clean(logPath))
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to open log %q: %w", logPath, err)
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, 64*1024)
	var matchedLines []string

	for {
		lineBytes, err := reader.ReadBytes('\n')
		if len(lineBytes) > maxDaemonLogLineBytes {
			return matchedLines, fmt.Errorf("line in %q exceeds maximum limit of 32MB", logPath)
		}
		if len(lineBytes) > 0 {
			line := strings.TrimSpace(string(lineBytes))
			if line != "" && strings.Contains(line, cleanRunID) {
				var rie runIDMatch
				if jsonErr := json.Unmarshal([]byte(line), &rie); jsonErr == nil {
					if rie.RunID == cleanRunID || rie.Details.RunID == cleanRunID {
						appendWithLimit(&matchedLines, logger.MaskLine(line), limit)
					}
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return matchedLines, fmt.Errorf("error reading log %q: %w", logPath, err)
		}
	}

	return matchedLines, nil
}

// GetRunTimeWindow scans logPath for an entry matching runID that contains start_time and/or end_time.
func (s *LogService) GetRunTimeWindow(logPath string, runID string) (time.Time, time.Time, bool) {
	cleanRunID := strings.TrimSpace(runID)
	if cleanRunID == "" {
		return time.Time{}, time.Time{}, false
	}

	// #nosec G304 - reading configured log file
	file, err := os.Open(filepath.Clean(logPath))
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, 64*1024)
	var startTime, endTime time.Time
	var found bool

	for {
		lineBytes, err := reader.ReadBytes('\n')
		if len(lineBytes) > 0 {
			line := strings.TrimSpace(string(lineBytes))
			if line != "" && strings.Contains(line, cleanRunID) {
				var entry runIDMatch
				if jsonErr := json.Unmarshal([]byte(line), &entry); jsonErr == nil {
					if entry.RunID == cleanRunID || entry.Details.RunID == cleanRunID {
						found = true
						startStr := entry.StartTime
						if startStr == "" {
							startStr = entry.Details.StartTime
						}
						endStr := entry.EndTime
						if endStr == "" {
							endStr = entry.Details.EndTime
						}

						if t, ok := parseFlexibleTime(startStr); ok {
							startTime = t
						}
						if t, ok := parseFlexibleTime(endStr); ok {
							endTime = t
						}
						if startTime.IsZero() {
							if t, ok := parseFlexibleTime(entry.Time); ok {
								startTime = t
							}
						}
					}
				}
			}
		}
		if err != nil {
			break
		}
	}

	return startTime, endTime, found
}

func (s *LogService) filterLine(line string, useWindow bool, windowStart, windowEnd time.Time, limit int, matchedLines *[]string) {
	wrapTime, wrapOk := parseLogTime(line)
	// Optimization: if this is a wrapper line with a timestamp before windowStart,
	// all nested events inside this bisync run occurred before windowStart, so skip.
	if useWindow && wrapOk && wrapTime.Before(windowStart) {
		return
	}

	// Check if line wraps nested JSON events inside msg (e.g. bisync RC sub-logs)
	subEntries, isWrapper := extractNestedJSONLogs(line)
	if isWrapper && len(subEntries) > 0 {
		for _, subEntry := range subEntries {
			if !useWindow {
				appendWithLimit(matchedLines, subEntry, limit)
				continue
			}

			subTime, ok := parseLogTime(subEntry)
			if !ok {
				// Fall back to outer wrapper line timestamp if subEntry has no time
				if wrapOk && (wrapTime.Equal(windowStart) || wrapTime.After(windowStart)) &&
					(wrapTime.Equal(windowEnd) || wrapTime.Before(windowEnd)) {
					appendWithLimit(matchedLines, subEntry, limit)
				}
				continue
			}

			if (subTime.Equal(windowStart) || subTime.After(windowStart)) &&
				(subTime.Equal(windowEnd) || subTime.Before(windowEnd)) {
				appendWithLimit(matchedLines, subEntry, limit)
			}
		}
		return
	}

	if !useWindow {
		appendWithLimit(matchedLines, line, limit)
		return
	}

	if !wrapOk {
		return
	}

	if (wrapTime.Equal(windowStart) || wrapTime.After(windowStart)) &&
		(wrapTime.Equal(windowEnd) || wrapTime.Before(windowEnd)) {
		appendWithLimit(matchedLines, line, limit)
	}
}
