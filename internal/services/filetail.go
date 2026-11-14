package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/core/logger"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

// TailFile reads the last N whole lines from a file by seeking from EOF.
// Returns lines in chronological order (oldest to newest among the last N lines)
// with credentials, tokens, and secrets masked.
func TailFile(filepathStr string, linesCount int) []string {
	// #nosec G304 - reading tail of configured log target
	f, err := os.Open(filepath.Clean(filepathStr))
	if err != nil {
		return []string{fmt.Sprintf("Log file target not found at: %s", filepathStr)}
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil || stat.Size() == 0 {
		return []string{}
	}

	position := stat.Size()
	var bufferSize int64 = 8192
	var lines []string
	remainder := []byte{}

	for position > 0 && len(lines) < linesCount {
		toRead := bufferSize
		if position < toRead {
			toRead = position
		}
		position -= toRead

		_, _ = f.Seek(position, io.SeekStart)
		chunk := make([]byte, toRead)
		_, _ = io.ReadFull(f, chunk)
		chunk = append(chunk, remainder...)

		chunkLines := bytes.Split(chunk, []byte("\n"))
		remainder = chunkLines[0]

		for i := len(chunkLines) - 1; i >= 1; i-- {
			lineBytes := bytes.TrimRight(chunkLines[i], "\r")
			if len(lineBytes) > 0 {
				lines = append(lines, string(lineBytes))
				if len(lines) >= linesCount {
					break
				}
			}
		}
	}

	if len(remainder) > 0 && len(lines) < linesCount {
		trimmed := bytes.TrimRight(remainder, "\r")
		if len(trimmed) > 0 {
			lines = append(lines, string(trimmed))
		}
	}

	// Reverse to restore original chronological order
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}

	// Apply secret redaction to every line
	for i := range lines {
		lines[i] = logger.MaskLine(lines[i])
	}

	return lines
}

// TailLines is an alias for TailFile, reading the last N whole lines with secrets masked.
func TailLines(filepathStr string, linesCount int) []string {
	return TailFile(filepathStr, linesCount)
}

// TailJSONLog reads log file, unwrapping nested JSON strings if present,
// with credentials, tokens, and secrets masked.
func TailJSONLog(filepathStr string, entriesCount int) []string {
	rawLines := TailFile(filepathStr, entriesCount*2)
	entries := make([]string, 0)

	for _, line := range rawLines {
		lineStr := strings.TrimSpace(line)
		if lineStr == "" {
			continue
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(lineStr), &parsed); err == nil {
			if msg, ok := parsed["msg"].(string); ok && (strings.Contains(msg, `{"`) || strings.Contains(msg, `{'`)) {
				sublines := strings.Split(msg, "\n")
				unwrappedAny := false
				for _, sub := range sublines {
					subTrim := strings.TrimSpace(sub)
					if strings.HasPrefix(subTrim, "{") && strings.HasSuffix(subTrim, "}") {
						var inner map[string]interface{}
						if err := json.Unmarshal([]byte(subTrim), &inner); err == nil {
							b, _ := json.Marshal(inner)
							entries = append(entries, logger.MaskLine(string(b)))
							unwrappedAny = true
						}
					}
				}
				if !unwrappedAny {
					b, _ := json.Marshal(parsed)
					entries = append(entries, logger.MaskLine(string(b)))
				}
			} else {
				b, _ := json.Marshal(parsed)
				entries = append(entries, logger.MaskLine(string(b)))
			}
		} else {
			entries = append(entries, logger.MaskLine(lineStr))
		}
	}

	if len(entries) > entriesCount {
		return entries[len(entries)-entriesCount:]
	}
	return entries
}

// LogFilter defines criteria for filtering log entries before pagination calculation.
type LogFilter struct {
	Types        []string
	ExcludeTypes []string
	Levels       []string
}

// filterLogEntry checks if an entry matches the filter criteria.
func filterLogEntry(entryStr string, filter LogFilter) bool {
	if len(filter.Types) == 0 && len(filter.ExcludeTypes) == 0 && len(filter.Levels) == 0 {
		return true
	}

	var m map[string]interface{}
	entryType := "sync"
	entryLevel := ""
	if err := json.Unmarshal([]byte(entryStr), &m); err == nil {
		if t, ok := m["type"].(string); ok && t != "" {
			entryType = strings.ToLower(t)
		} else if orig, ok := m["origin"].(string); ok && orig != "" {
			entryType = strings.ToLower(orig)
		} else if msg, ok := m["msg"].(string); ok && msg == "http_access" {
			entryType = "http_access"
		} else if _, hasMethod := m["method"]; hasMethod {
			entryType = "http_access"
		}
		if lvl, ok := m["level"].(string); ok {
			entryLevel = strings.ToLower(strings.TrimSpace(lvl))
		}
	} else {
		if strings.Contains(entryStr, "GET /api/") || strings.Contains(entryStr, "HEAD /api/") {
			entryType = "http_access"
		}
	}

	if len(filter.ExcludeTypes) > 0 {
		for _, ex := range filter.ExcludeTypes {
			if strings.EqualFold(entryType, ex) {
				return false
			}
		}
	}

	if len(filter.Types) > 0 {
		matched := false
		for _, inc := range filter.Types {
			if strings.EqualFold(entryType, inc) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if len(filter.Levels) > 0 {
		matched := false
		for _, inc := range filter.Levels {
			if strings.EqualFold(entryLevel, inc) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// TailJSONLogPaginated reads log file, unwrapping nested JSON strings and masking secrets,
// and returns paginated entries along with total count and page metadata.
// Page 1 contains the most recent entries in chronological order.
func TailJSONLogPaginated(filepathStr string, page, limit int) models.PaginatedLogsResponse {
	return TailJSONLogPaginatedFiltered(filepathStr, page, limit, LogFilter{})
}

// TailJSONLogPaginatedFiltered filters log entries by type and level before calculating pagination,
// ensuring completely dense pages without gaps.
func TailJSONLogPaginatedFiltered(filepathStr string, page, limit int, filter LogFilter) models.PaginatedLogsResponse {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if page < 1 {
		page = 1
	}

	if filepathStr == "" {
		return models.PaginatedLogsResponse{
			Page:       1,
			Limit:      limit,
			Total:      0,
			TotalPages: 1,
			Logs:       []string{},
		}
	}
	if _, err := os.Stat(filepathStr); os.IsNotExist(err) {
		return models.PaginatedLogsResponse{
			Page:       1,
			Limit:      limit,
			Total:      0,
			TotalPages: 1,
			Logs:       []string{},
		}
	}

	allEntries := TailJSONLog(filepathStr, 5000)
	filtered := make([]string, 0, len(allEntries))
	for _, entry := range allEntries {
		if filterLogEntry(entry, filter) {
			filtered = append(filtered, entry)
		}
	}

	total := len(filtered)
	if total == 0 {
		return models.PaginatedLogsResponse{
			Page:       1,
			Limit:      limit,
			Total:      0,
			TotalPages: 1,
			Logs:       []string{},
		}
	}

	totalPages := (total + limit - 1) / limit
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	// Page 1 has the newest items: [total - limit : total]
	// Page 2 has older items: [total - 2*limit : total - limit]
	end := total - (page-1)*limit
	start := end - limit
	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = 0
	}

	sliced := filtered[start:end]
	return models.PaginatedLogsResponse{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		Logs:       sliced,
	}
}
