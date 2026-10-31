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

type timeExtractor struct {
	Time string `json:"time"`
}

type runIDMatch struct {
	RunID     string `json:"run_id"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Time      string `json:"time"`
	Details   struct {
		RunID     string `json:"run_id"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
	} `json:"details"`
}

func appendWithLimit(list *[]string, item string, limit int) {
	*list = append(*list, item)
	if len(*list) > limit {
		*list = (*list)[1:]
	}
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

// extractNestedJSONLogs checks if a JSON log entry wraps nested JSON logs in its msg field,
// and extracts each nested JSON object as a normalized JSON string.
func extractNestedJSONLogs(line string) ([]string, bool) {
	if !strings.HasPrefix(line, "{") {
		return nil, false
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(line), &parsed); err != nil {
		return nil, false
	}
	msg, ok := parsed["msg"].(string)
	if !ok || (!strings.Contains(msg, `{"`) && !strings.Contains(msg, `{'`)) {
		return nil, false
	}

	var entries []string
	dec := json.NewDecoder(strings.NewReader(msg))
	for {
		var inner map[string]interface{}
		err := dec.Decode(&inner)
		if err == io.EOF {
			break
		}
		if err == nil && len(inner) > 0 {
			b, err := json.Marshal(inner)
			if err == nil {
				entries = append(entries, string(b))
			}
		} else {
			break
		}
	}

	if len(entries) > 0 {
		return entries, true
	}

	// Line-by-line fallback
	sublines := strings.Split(msg, "\n")
	for _, sub := range sublines {
		subTrim := strings.TrimSpace(sub)
		if strings.HasPrefix(subTrim, "{") && strings.HasSuffix(subTrim, "}") {
			var inner map[string]interface{}
			if err := json.Unmarshal([]byte(subTrim), &inner); err == nil && len(inner) > 0 {
				b, err := json.Marshal(inner)
				if err == nil {
					entries = append(entries, string(b))
				}
			}
		}
	}

	if len(entries) > 0 {
		return entries, true
	}

	return nil, false
}

func extractTimeFast(line string) string {
	maxCheck := len(line)
	if maxCheck > 256 {
		maxCheck = 256
	}
	prefix := line[:maxCheck]
	needle := "\"time\":\""
	idx := strings.Index(prefix, needle)
	if idx == -1 {
		return ""
	}
	start := idx + len(needle)
	end := strings.IndexByte(prefix[start:], '"')
	if end == -1 {
		return ""
	}
	return prefix[start : start+end]
}

func parseFlexibleTime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	return time.Time{}, false
}

// parseLogTime attempts to extract and parse the timestamp from a JSON or legacy log line.
func parseLogTime(line string) (time.Time, bool) {
	if strings.HasPrefix(line, "{") {
		if fastTime := extractTimeFast(line); fastTime != "" {
			if t, ok := parseFlexibleTime(fastTime); ok {
				return t, true
			}
		}
		var te timeExtractor
		if err := json.Unmarshal([]byte(line), &te); err == nil {
			if t, ok := parseFlexibleTime(te.Time); ok {
				return t, true
			}
		}
	}

	// Legacy fallback: "2006/01/02 15:04:05" or "2006-01-02 15:04:05"
	if len(line) >= 19 {
		prefix := line[:19]
		if t, err := time.Parse("2006/01/02 15:04:05", prefix); err == nil {
			return t, true
		}
		if t, err := time.Parse("2006-01-02 15:04:05", prefix); err == nil {
			return t, true
		}
	}

	return time.Time{}, false
}
