package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// RemotesCacheItem stores cached remotes summary with timestamp.
type RemotesCacheItem struct {
	Timestamp time.Time
	Data      models.RemotesResponse
}

// GetRemotesSummary queries remotes and storage metrics (default cached).
func (t *TelemetryService) GetRemotesSummary(ctx context.Context) (models.RemotesResponse, error) {
	return t.GetRemotesSummaryCached(ctx, false)
}

// GetRemotesSummaryCached queries remotes and storage metrics with 60s TTL caching.
func (t *TelemetryService) GetRemotesSummaryCached(ctx context.Context, forceRefresh bool) (models.RemotesResponse, error) {
	if t.Client == nil {
		return models.RemotesResponse{}, fmt.Errorf("client not configured")
	}

	if !forceRefresh {
		t.lockMu.Lock()
		if !t.remotesCache.Timestamp.IsZero() && time.Since(t.remotesCache.Timestamp) < 60*time.Second {
			cached := t.remotesCache.Data
			t.lockMu.Unlock()
			return cached, nil
		}
		t.lockMu.Unlock()
	}

	remotes, err := t.Client.ListRemotes(ctx)
	if err != nil {
		return models.RemotesResponse{}, err
	}

	remoteData := make([]map[string]interface{}, 0, len(remotes))
	var totalCap, totalUsed, totalFree int64

	activeRemote := ""
	if t.Config != nil {
		activeRemote = strings.TrimRight(t.Config.RemotePath, ":")
	}

	for _, r := range remotes {
		cleanName := strings.TrimRight(r, ":")
		cfg := t.Client.GetRemoteConfig(ctx, cleanName)
		about := t.Client.GetRemoteAbout(ctx, cleanName)

		rtype := "unknown"
		if cfg != nil {
			if at, ok := cfg["type"].(string); ok {
				rtype = at
			}
		}

		typeLabel := strings.ToUpper(rtype)
		if lbl, ok := ProviderLabels[rtype]; ok {
			typeLabel = lbl
		}

		isActive := (cleanName == activeRemote)

		if about != nil {
			if total, ok := about["total"].(float64); ok {
				totalCap += int64(total)
			}
			if used, ok := about["used"].(float64); ok {
				totalUsed += int64(used)
			}
			if free, ok := about["free"].(float64); ok {
				totalFree += int64(free)
			}
		}

		if about == nil {
			about = make(map[string]interface{})
		}

		remoteData = append(remoteData, map[string]interface{}{
			"name":             cleanName,
			"fs":               fmt.Sprintf("%s:", cleanName),
			"type":             rtype,
			"type_label":       typeLabel,
			"is_active_target": isActive,
			"about":            about,
		})
	}

	summary := map[string]interface{}{
		"total_capacity_bytes": totalCap,
		"total_used_bytes":     totalUsed,
		"total_free_bytes":     totalFree,
		"provider_count":       len(remoteData),
	}

	var activeRemoteFormatted interface{}
	if activeRemote != "" {
		activeRemoteFormatted = fmt.Sprintf("%s:", activeRemote)
	}

	resp := models.RemotesResponse{
		Remotes:      remoteData,
		Summary:      summary,
		ActiveRemote: activeRemoteFormatted,
		Count:        len(remoteData),
	}

	t.lockMu.Lock()
	t.remotesCache = RemotesCacheItem{
		Timestamp: time.Now(),
		Data:      resp,
	}
	t.lockMu.Unlock()

	return resp, nil
}
