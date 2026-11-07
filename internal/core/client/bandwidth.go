package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// SetBandwidthLimit sets a dynamic bandwidth limit via rclone RC core/bwlimit.
// Rate can be a size string (e.g., "10M", "500k", "10M:10M", or "off").
func (c *RcloneClient) SetBandwidthLimit(ctx context.Context, rate string) (*models.BandwidthLimitResponse, error) {
	payload := map[string]interface{}{
		"rate": rate,
	}

	resp, body, err := c.post(ctx, "core/bwlimit", payload, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to call core/bwlimit: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("core/bwlimit returned status %d: %s", resp.StatusCode, string(body))
	}

	var raw models.RcloneBwlimitRaw
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse core/bwlimit response: %w", err)
	}

	rateFormatted := raw.Rate
	if rateFormatted == "" && rate != "" {
		rateFormatted = rate
	}

	return &models.BandwidthLimitResponse{
		BytesPerSecond: raw.BytesPerSecond,
		RateFormatted:  rateFormatted,
		Status:         "ok",
	}, nil
}

// GetBandwidthLimit retrieves the current active bandwidth limit from rclone RC core/bwlimit.
func (c *RcloneClient) GetBandwidthLimit(ctx context.Context) (*models.BandwidthLimitResponse, error) {
	resp, body, err := c.post(ctx, "core/bwlimit", nil, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to query core/bwlimit: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("core/bwlimit query returned status %d: %s", resp.StatusCode, string(body))
	}

	var raw models.RcloneBwlimitRaw
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse core/bwlimit response: %w", err)
	}

	return &models.BandwidthLimitResponse{
		BytesPerSecond: raw.BytesPerSecond,
		RateFormatted:  raw.Rate,
		Status:         "ok",
	}, nil
}

// GetParsedCoreStats retrieves structured aggregate transfer metrics and active transfers from core/stats.
func (c *RcloneClient) GetParsedCoreStats(ctx context.Context) (*models.CoreStatsResponse, error) {
	resp, body, err := c.post(ctx, "core/stats", nil, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to query core/stats: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("core/stats returned status %d: %s", resp.StatusCode, string(body))
	}

	var raw models.RcloneStatsRaw
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse core/stats response: %w", err)
	}

	transfers := make([]models.ActiveTransfer, 0, len(raw.Transferring))
	for _, t := range raw.Transferring {
		var eta int64
		if t.ETA != nil {
			eta = *t.ETA
		}
		transfers = append(transfers, models.ActiveTransfer{
			Name:       t.Name,
			Size:       t.Size,
			Bytes:      t.Bytes,
			Percentage: t.Percentage,
			Speed:      t.Speed,
			SpeedAvg:   t.SpeedAvg,
			ETA:        eta,
		})
	}

	return &models.CoreStatsResponse{
		Bytes:        raw.Bytes,
		Speed:        raw.Speed,
		Transfers:    raw.Transfers,
		Checks:       raw.Checks,
		Errors:       raw.Errors,
		FatalErrors:  raw.FatalErrors,
		Transferring: transfers,
	}, nil
}
