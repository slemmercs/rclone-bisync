package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// TailFile reads the last N whole lines from a file by seeking from EOF.
// Returns lines in chronological order (oldest to newest among the last N lines).
func TailFile(filepathStr string, linesCount int) []string {
	f, err := os.Open(filepathStr)
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

	return lines
}

// TailJSONLog reads log file, unwrapping nested JSON strings if present.
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
							entries = append(entries, string(b))
							unwrappedAny = true
						}
					}
				}
				if !unwrappedAny {
					b, _ := json.Marshal(parsed)
					entries = append(entries, string(b))
				}
			} else {
				b, _ := json.Marshal(parsed)
				entries = append(entries, string(b))
			}
		} else {
			entries = append(entries, lineStr)
		}
	}

	if len(entries) > entriesCount {
		return entries[len(entries)-entriesCount:]
	}
	return entries
}
