package client

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"
)

// MonitorJobResult encapsulates the result of monitoring a bisync job.
type MonitorJobResult struct {
	Success      bool
	ErrorType    string
	ErrorMessage string
	FilesSynced  int
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

				filesSynced := 0
				if outputData, ok := status["output"].(map[string]interface{}); ok {
					if transfers, ok := outputData["transfers"].(float64); ok {
						filesSynced = int(transfers)
					} else if stats, ok := outputData["stats"].(map[string]interface{}); ok {
						if transfers, ok := stats["transfers"].(float64); ok {
							filesSynced = int(transfers)
						}
					}
				}

				if success {
					slog.Debug("Job completed successfully", "job_id", jobID, "elapsed_sec", elapsed.Seconds(), "files_transferred", filesSynced)
					return MonitorJobResult{
						Success:     true,
						FilesSynced: filesSynced,
					}
				}

				return MonitorJobResult{
					Success:      false,
					ErrorMessage: errorMsg,
					FilesSynced:  filesSynced,
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
