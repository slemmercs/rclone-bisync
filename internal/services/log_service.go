package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
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

// FilterDaemonLogs reads the daemon JSON log and returns lines occurring within [startTime-1s, endTime+1s].
// If startTime is zero, it falls back to returning the tail of the log up to limit entries.
func (s *LogService) FilterDaemonLogs(logPath string, startTime, endTime time.Time, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 2000 {
		limit = 2000
	}

	file, err := os.Open(logPath)
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

	scanner := bufio.NewScanner(file)
	// Buffer up to 512KB for wide JSON log lines
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 512*1024)

	var matchedLines []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if !useWindow {
			matchedLines = append(matchedLines, line)
			if len(matchedLines) > limit {
				matchedLines = matchedLines[1:]
			}
			continue
		}

		logTime, ok := parseLogTime(line)
		if !ok {
			// If JSON without time or legacy format, check if fallback time matches
			continue
		}

		if (logTime.Equal(windowStart) || logTime.After(windowStart)) &&
			(logTime.Equal(windowEnd) || logTime.Before(windowEnd)) {
			matchedLines = append(matchedLines, line)
			if len(matchedLines) > limit {
				matchedLines = matchedLines[1:]
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return matchedLines, fmt.Errorf("error reading daemon log %q: %w", logPath, err)
	}

	return matchedLines, nil
}

// parseLogTime attempts to extract and parse the timestamp from a JSON or legacy log line.
func parseLogTime(line string) (time.Time, bool) {
	if strings.HasPrefix(line, "{") {
		var te timeExtractor
		if err := json.Unmarshal([]byte(line), &te); err == nil && te.Time != "" {
			if t, err := time.Parse(time.RFC3339Nano, te.Time); err == nil {
				return t, true
			}
			if t, err := time.Parse(time.RFC3339, te.Time); err == nil {
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
