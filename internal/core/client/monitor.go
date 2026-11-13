package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// MonitorJobResult encapsulates the result of monitoring a bisync job.
type MonitorJobResult struct {
	Success          bool
	ErrorType        string
	ErrorMessage     string
	FilesSynced      int
	TransferredFiles []models.SyncFileOp
}

// MonitorJob polls /job/status with exponential backoff and transient failure tolerance.
func (c *RcloneClient) MonitorJob(
	ctx context.Context,
	jobID int,
	initialSleep time.Duration,
	maxSleep time.Duration,
	backoffMultiplier float64,
	maxConsecutivePollErrors int,
	timeout time.Duration,
) MonitorJobResult {
	startTime := time.Now()
	sleepDuration := initialSleep
	consecutiveErrors := 0

	slog.Debug("Monitoring bisync job", "job_id", jobID)

	for {
		elapsed := time.Since(startTime)
		if elapsed > timeout {
			slog.Error("Job exceeded timeout", "job_id", jobID, "timeout", timeout)
			c.StopJob(ctx, jobID)
			return MonitorJobResult{
				Success:      false,
				ErrorType:    "TIMEOUT",
				ErrorMessage: fmt.Sprintf("Job exceeded %v timeout", timeout),
				FilesSynced:  0,
			}
		}

		status, err := c.GetJobStatus(ctx, jobID)
		if err != nil {
			consecutiveErrors++
			slog.Warn("Temporary error polling job", "job_id", jobID, "attempt", consecutiveErrors, "max", maxConsecutivePollErrors, "error", err)
			if consecutiveErrors >= maxConsecutivePollErrors {
				slog.Error("Job polling failed after maximum consecutive errors", "job_id", jobID, "consecutive_errors", consecutiveErrors)
				return MonitorJobResult{
					Success:      false,
					ErrorType:    "MONITOR_ERROR",
					ErrorMessage: err.Error(),
					FilesSynced:  0,
				}
			}
		} else {
			consecutiveErrors = 0
			if finished, ok := status["finished"].(bool); ok && finished {
				success, _ := status["success"].(bool)
				errorMsg, _ := status["error"].(string)

				filesSynced := c.extractTransfers(ctx, status)
				var transferredFiles []models.SyncFileOp
				if outputData, ok := status["output"].(map[string]interface{}); ok && outputData != nil {
					if outStr, ok := outputData["output"].(string); ok && outStr != "" {
						transferredFiles = ExtractTransferredFiles(outStr, 50)
					}
				}
				if filesSynced == 0 && len(transferredFiles) > 0 {
					filesSynced = len(transferredFiles)
				}

				if success {
					slog.Debug("Job completed successfully", "job_id", jobID, "elapsed_sec", elapsed.Seconds(), "files_transferred", filesSynced)
					return MonitorJobResult{
						Success:          true,
						FilesSynced:      filesSynced,
						TransferredFiles: transferredFiles,
					}
				}

				return MonitorJobResult{
					Success:          false,
					ErrorMessage:     errorMsg,
					FilesSynced:      filesSynced,
					TransferredFiles: transferredFiles,
				}
			}
		}

		select {
		case <-ctx.Done():
			return MonitorJobResult{
				Success:      false,
				ErrorType:    "CANCELLED",
				ErrorMessage: "Context cancelled",
			}
		case <-time.After(sleepDuration):
		}

		nextSleep := float64(sleepDuration) * backoffMultiplier
		if nextSleep > float64(maxSleep) {
			sleepDuration = maxSleep
		} else {
			sleepDuration = time.Duration(math.Round(nextSleep))
		}
	}
}

// extractTransfers determines the number of transferred files by checking output data,
// querying /core/stats with the job's group, or parsing the output log text.
func (c *RcloneClient) extractTransfers(ctx context.Context, status map[string]interface{}) int {
	outputData, _ := status["output"].(map[string]interface{})
	if outputData != nil {
		if transfers, ok := outputData["transfers"].(float64); ok && transfers > 0 {
			return int(transfers)
		}
		if stats, ok := outputData["stats"].(map[string]interface{}); ok {
			if transfers, ok := stats["transfers"].(float64); ok && transfers > 0 {
				return int(transfers)
			}
		}
	}

	// 1. Authoritative: query /core/stats for the job's group if available
	if group, ok := status["group"].(string); ok && group != "" {
		if transfers := c.queryGroupTransfers(ctx, group); transfers > 0 {
			return transfers
		}
	}

	// 2. Fallback: parse copy operation occurrences in stdout log
	if outputData != nil {
		if outStr, ok := outputData["output"].(string); ok && outStr != "" {
			count := strings.Count(outStr, "Copied (new)") + strings.Count(outStr, "Copied (replaced)")
			if count > 0 {
				return count
			}
		}
	}

	return 0
}

func (c *RcloneClient) queryGroupTransfers(ctx context.Context, group string) int {
	resp, body, err := c.post(ctx, "core/stats", map[string]interface{}{"group": group}, 2*time.Second)
	if err != nil || resp == nil || resp.StatusCode != http.StatusOK {
		return 0
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return 0
	}
	if transfers, ok := raw["transfers"].(float64); ok && transfers > 0 {
		return int(transfers)
	}
	if total, ok := raw["totalTransfers"].(float64); ok && total > 0 {
		return int(total)
	}
	return 0
}

var (
	textTransferRegex   = regexp.MustCompile(`(?i)^(?:-\s*)?(?:[a-zA-Z0-9_.-]+(?:\{[^}]+\})?:[/\\]?)?(.+?):\s*(Copied \(new\)|Copied \(replaced\)|Deleted|File was deleted|Moved|Renamed)`)
	timestampStripRegex = regexp.MustCompile(`^(?:\d{4}/\d{2}/\d{2}\s+\d{2}:\d{2}:\d{2}\s+)?(?:(?:INFO|NOTICE|ERROR|WARN|DEBUG)\s*:\s*)?`)
	remotePrefixRegex   = regexp.MustCompile(`^[a-zA-Z0-9_.-]+(?:\{[^}]+\})?:[/\\]?`)
)

// ExtractTransferredFiles parses rclone stdout/log output and extracts transferred, replaced,
// or deleted file operations, returning up to maxEntries.
func ExtractTransferredFiles(outputStr string, maxEntries int) []models.SyncFileOp {
	if outputStr == "" || maxEntries <= 0 {
		return nil
	}
	lines := strings.Split(outputStr, "\n")
	results := make([]models.SyncFileOp, 0, len(lines))
	seen := make(map[string]bool)

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		var path, action, msg string

		if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
			var jsonEntry struct {
				Msg    string `json:"msg"`
				Object string `json:"object"`
			}
			if err := json.Unmarshal([]byte(line), &jsonEntry); err == nil && jsonEntry.Object != "" {
				action = mapActionFromMsg(jsonEntry.Msg)
				if action != "" {
					path = cleanFilePath(jsonEntry.Object)
					msg = fmt.Sprintf("%s: %s", path, jsonEntry.Msg)
				}
			}
		}

		if path == "" {
			cleaned := timestampStripRegex.ReplaceAllString(line, "")
			if m := textTransferRegex.FindStringSubmatch(cleaned); len(m) >= 3 {
				path = cleanFilePath(m[1])
				action = mapActionFromMsg(m[2])
				msg = fmt.Sprintf("%s: %s", path, m[2])
			}
		}

		if path != "" && action != "" {
			dedupeKey := path + "::" + action
			if !seen[dedupeKey] {
				seen[dedupeKey] = true
				results = append(results, models.SyncFileOp{
					Path:   path,
					Action: action,
					Msg:    msg,
				})
				if len(results) >= maxEntries {
					break
				}
			}
		}
	}

	return results
}

func mapActionFromMsg(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "copied (new)"):
		return "COPIED"
	case strings.Contains(lower, "copied (replaced)"):
		return "REPLACED"
	case strings.Contains(lower, "deleted") || strings.Contains(lower, "file was deleted"):
		return "DELETED"
	case strings.Contains(lower, "moved") || strings.Contains(lower, "renamed"):
		return "MOVED"
	default:
		return ""
	}
}

func cleanFilePath(p string) string {
	p = strings.Trim(p, "\"'` \t\r\n")
	p = remotePrefixRegex.ReplaceAllString(p, "")
	p = strings.ReplaceAll(p, "\\", "/")
	return strings.Trim(p, "/")
}
