package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/core/security"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

// FileCacheItem stores cached file content with mtime/size headers.
type FileCacheItem struct {
	MTime int64
	Size  int64
	Data  interface{}
}

// LiveHealthCache stores daemon health check with timestamp.
type LiveHealthCache struct {
	Time float64
	Data map[string]interface{}
}

// TelemetryService manages daemon health, remotes summary, transfer statistics, and metrics caching.
type TelemetryService struct {
	Config          *config.Config
	Client          *client.RcloneClient
	lockMu          sync.Mutex
	fileCache       map[string]FileCacheItem
	liveHealthCache LiveHealthCache
}

// NewTelemetryService creates a new TelemetryService instance.
func NewTelemetryService(cfg *config.Config, rcClient *client.RcloneClient) *TelemetryService {
	return &TelemetryService{
		Config:    cfg,
		Client:    rcClient,
		fileCache: make(map[string]FileCacheItem),
	}
}

// GetDaemonHealth retrieves daemon health with caching.
func (t *TelemetryService) GetDaemonHealth(ctx context.Context) map[string]interface{} {
	now := float64(time.Now().Unix())

	t.lockMu.Lock()
	if t.liveHealthCache.Data != nil && (now-t.liveHealthCache.Time) <= 5 {
		data := t.liveHealthCache.Data
		t.lockMu.Unlock()
		return data
	}
	t.lockMu.Unlock()

	if t.Client == nil {
		return map[string]interface{}{
			"status":           "offline",
			"response_time_ms": 0.0,
			"last_check":       now,
		}
	}

	healthy, latency, _ := t.Client.CheckHealth(ctx, 1, 0)
	statusStr := "offline"
	if healthy {
		statusStr = "healthy"
	} else {
		latency = 0.0
	}

	health := map[string]interface{}{
		"status":           statusStr,
		"response_time_ms": latency,
		"last_check":       now,
	}

	t.lockMu.Lock()
	t.liveHealthCache.Time = now
	t.liveHealthCache.Data = health
	t.lockMu.Unlock()

	return health
}

// GetRemotesSummary queries remotes and storage metrics.
func (t *TelemetryService) GetRemotesSummary(ctx context.Context) (models.RemotesResponse, error) {
	if t.Client == nil {
		return models.RemotesResponse{}, fmt.Errorf("client not configured")
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

	return models.RemotesResponse{
		Remotes:      remoteData,
		Summary:      summary,
		ActiveRemote: activeRemoteFormatted,
		Count:        len(remoteData),
	}, nil
}

// GetTransferStats queries core/stats from Rclone RC.
func (t *TelemetryService) GetTransferStats(ctx context.Context) *models.TransferStats {
	if t.Client == nil {
		return nil
	}
	resp, err := t.Client.GetCoreStats(ctx)
	if err != nil || resp == nil {
		return nil
	}

	var data struct {
		Bytes        int64                    `json:"bytes"`
		TotalBytes   int64                    `json:"totalBytes"`
		Speed        float64                  `json:"speed"`
		ETA          *int                     `json:"eta"`
		Transferring []map[string]interface{} `json:"transferring"`
	}
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil
	}

	speedMB := math.Round((data.Speed/1024.0/1024.0)*100) / 100
	var pct float64
	if data.TotalBytes > 0 {
		pct = math.Round((float64(data.Bytes)/float64(data.TotalBytes)*100.0)*10) / 10
	}
	if data.Transferring == nil {
		data.Transferring = []map[string]interface{}{}
	}

	return &models.TransferStats{
		Bytes:        data.Bytes,
		TotalBytes:   data.TotalBytes,
		Speed:        data.Speed,
		SpeedStr:     fmt.Sprintf("%.2f MB/s", speedMB),
		ETA:          data.ETA,
		Percentage:   pct,
		Transferring: data.Transferring,
	}
}

// GetCachedMetrics returns parsed and cached metrics lines.
func (t *TelemetryService) GetCachedMetrics(linesCount int) []map[string]interface{} {
	if t.Config == nil || t.Config.MetricsFile == "" {
		return []map[string]interface{}{}
	}
	metricsPath := t.Config.MetricsFile
	fi, err := os.Stat(metricsPath)
	if err != nil {
		return []map[string]interface{}{}
	}

	t.lockMu.Lock()
	if cached, ok := t.fileCache[metricsPath]; ok {
		if cached.MTime == fi.ModTime().UnixNano() && cached.Size == fi.Size() {
			if res, ok := cached.Data.([]map[string]interface{}); ok {
				t.lockMu.Unlock()
				return res
			}
		}
	}
	t.lockMu.Unlock()

	rawLines := TailFile(metricsPath, linesCount)
	parsed := make([]map[string]interface{}, 0, len(rawLines))
	for _, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(trimmed), &entry); err == nil {
			parsed = append(parsed, entry)
		}
	}

	t.lockMu.Lock()
	t.fileCache[metricsPath] = FileCacheItem{
		MTime: fi.ModTime().UnixNano(),
		Size:  fi.Size(),
		Data:  parsed,
	}
	t.lockMu.Unlock()

	return parsed
}

// GetRuntimeConfig returns sanitized configuration map with masked credentials.
func (t *TelemetryService) GetRuntimeConfig() map[string]interface{} {
	if t.Config == nil {
		return make(map[string]interface{})
	}
	res := make(map[string]interface{})
	val := reflect.ValueOf(t.Config).Elem()
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		keySnake := toUpperSnakeCase(f.Name)
		keyUpper := strings.ToUpper(f.Name)
		v := val.Field(i).Interface()

		var finalVal interface{} = v
		if isKeySensitive(keySnake) || isKeySensitive(keyUpper) {
			if strVal, ok := v.(string); ok && strVal != "" {
				finalVal = "********"
			} else if v != nil {
				finalVal = "********"
			}
		}

		res[keySnake] = finalVal
		if keyUpper != keySnake {
			res[keyUpper] = finalVal
		}
	}
	return res
}

// GetTelemetryPayload aggregates daemon health, logs, metrics, config, and transfer stats.
func (t *TelemetryService) GetTelemetryPayload(ctx context.Context, isSync bool, limit int) map[string]interface{} {
	daemonData := t.GetDaemonHealth(ctx)
	var transferStats interface{}
	if isSync {
		transferStats = t.GetTransferStats(ctx)
	}

	notifState := map[string]interface{}{
		"last_success_timestamp": nil,
		"last_error_type":        nil,
		"last_error_timestamp":   nil,
	}
	if t.Config != nil && t.Config.NotificationStateFile != "" {
		if data, err := os.ReadFile(t.Config.NotificationStateFile); err == nil {
			_ = json.Unmarshal(data, &notifState)
		}
	}

	metricsData := t.GetCachedMetrics(30)
	if len(metricsData) > 0 {
		var latestSuccess float64
		for _, m := range metricsData {
			if succ, ok := m["success"].(bool); ok && succ {
				if ts, ok := m["timestamp"].(float64); ok {
					if ts > latestSuccess {
						latestSuccess = ts
					}
				}
			}
		}
		if latestSuccess > 0 {
			var currentNotifSuccess float64
			if curr, ok := notifState["last_success_timestamp"].(float64); ok {
				currentNotifSuccess = curr
			}
			if latestSuccess > currentNotifSuccess {
				notifState["last_success_timestamp"] = latestSuccess
			}
		}
	}

	bisyncLogs := []string{}
	if t.Config != nil && t.Config.LogFile != "" {
		bisyncLogs = TailJSONLog(t.Config.LogFile, limit)
	}

	daemonLogs := []string{}
	if t.Config != nil && t.Config.BaseDir != "" {
		daemonLogPath := filepath.Join(t.Config.BaseDir, "rclone-daemon.log")
		daemonLogs = TailJSONLog(daemonLogPath, limit)
	}

	return map[string]interface{}{
		"daemon_health":      daemonData,
		"notification_state": notifState,
		"metrics":            metricsData,
		"bisync_logs":        bisyncLogs,
		"daemon_logs":        daemonLogs,
		"config":             t.GetRuntimeConfig(),
		"is_syncing":         isSync,
		"transfer_stats":     transferStats,
	}
}

func toUpperSnakeCase(s string) string {
	var result []rune
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			if unicode.IsLower(prev) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]) && unicode.IsUpper(prev)) {
				result = append(result, '_')
			}
		}
		result = append(result, unicode.ToUpper(r))
	}
	return string(result)
}

func isKeySensitive(key string) bool {
	return security.IsSensitiveKeyName(key)
}
