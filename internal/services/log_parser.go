package services

import (
	"encoding/json"
	"io"
	"strings"
	"time"
)

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
