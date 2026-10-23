package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// TriggerBisync calls /sync/bisync?_async=true and returns the job ID integer.
func (c *RcloneClient) TriggerBisync(ctx context.Context, params map[string]interface{}, group string) (int, error) {
	endpoint := "sync/bisync?_async=true"
	if group != "" {
		endpoint += fmt.Sprintf("&_group=%s", url.QueryEscape(group))
	}

	resp, body, err := c.post(ctx, endpoint, params, 30*time.Second)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("trigger bisync failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res struct {
		JobID int `json:"jobid"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return 0, fmt.Errorf("failed to parse jobid: %w", err)
	}

	if res.JobID == 0 {
		return 0, fmt.Errorf("rclone RC returned 0 or missing jobid: %s", string(body))
	}

	return res.JobID, nil
}

// GetJobStatus queries /job/status for a specific job.
func (c *RcloneClient) GetJobStatus(ctx context.Context, jobID int) (map[string]interface{}, error) {
	resp, body, err := c.post(ctx, "job/status", map[string]interface{}{"jobid": jobID}, 10*time.Second)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("query job status failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res map[string]interface{}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}
	return res, nil
}

// StopJob stops a running job via /job/stop.
func (c *RcloneClient) StopJob(ctx context.Context, jobID int) bool {
	resp, _, err := c.post(ctx, "job/stop", map[string]interface{}{"jobid": jobID}, 10*time.Second)
	if err != nil {
		return false
	}
	return resp.StatusCode == http.StatusOK
}

// ListJobs queries /job/list and returns slice of running job IDs.
func (c *RcloneClient) ListJobs(ctx context.Context) []int {
	resp, body, err := c.post(ctx, "job/list", nil, 5*time.Second)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}

	var res struct {
		JobIDs []int `json:"jobids"`
	}
	_ = json.Unmarshal(body, &res)
	return res.JobIDs
}
