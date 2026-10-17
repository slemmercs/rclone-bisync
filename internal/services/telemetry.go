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

	"github.com/slemmercs/rclone-bisync/internal/models"
)

func (s *SyncService) GetTransferStats(ctx context.Context) *models.TransferStats {
	resp, err := s.Client.GetCoreStats(ctx)
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

func (s *SyncService) GetCachedMetrics(linesCount int) []map[string]interface{} {
	metricsPath := s.Config.MetricsFile
	fi, err := os.Stat(metricsPath)
	if err != nil {
		return []map[string]interface{}{}
	}

	s.lockMu.Lock()
	if cached, ok := s.fileCache[metricsPath]; ok {
		if cached.MTime == fi.ModTime().UnixNano() && cached.Size == fi.Size() {
			if res, ok := cached.Data.([]map[string]interface{}); ok {
				s.lockMu.Unlock()
				return res
			}
		}
	}
	s.lockMu.Unlock()

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

	s.lockMu.Lock()
	s.fileCache[metricsPath] = FileCacheItem{
		MTime: fi.ModTime().UnixNano(),
		Size:  fi.Size(),
		Data:  parsed,
	}
	s.lockMu.Unlock()

	return parsed
}

func (s *SyncService) GetRuntimeConfig() map[string]interface{} {
	res := make(map[string]interface{})
	val := reflect.ValueOf(s.Config).Elem()
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		keyUpper := strings.ToUpper(f.Name)
		v := val.Field(i).Interface()

		if isKeySensitive(keyUpper) {
			if strVal, ok := v.(string); ok && strVal != "" {
				res[keyUpper] = "********"
			} else if v != nil {
				res[keyUpper] = "********"
			} else {
				res[keyUpper] = v
			}
		} else {
			res[keyUpper] = v
		}
	}
	return res
}

func isKeySensitive(key string) bool {
	u := strings.ToUpper(key)
	sensitivePatterns := []string{"PASS", "SECRET", "TOKEN", "KEY", "CREDENTIAL", "AUTH", "WEBHOOK", "PRIVATE"}
	for _, pat := range sensitivePatterns {
		if strings.Contains(u, pat) {
			return true
		}
	}
	return false
}

func (s *SyncService) GetTelemetryPayload(ctx context.Context, limit int) map[string]interface{} {
	daemonData := s.GetDaemonHealth(ctx)
	isSync := s.IsSyncing()
	var transferStats *models.TransferStats
	if isSync {
		transferStats = s.GetTransferStats(ctx)
	}

	notifState := map[string]interface{}{
		"last_success_timestamp": nil,
		"last_error_type":        nil,
		"last_error_timestamp":   nil,
	}
	if data, err := os.ReadFile(s.Config.NotificationStateFile); err == nil {
		_ = json.Unmarshal(data, &notifState)
	}

	metricsData := s.GetCachedMetrics(30)
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

	bisyncLogs := TailJSONLog(s.Config.LogFile, limit)
	daemonLogPath := filepath.Join(s.Config.BaseDir, "rclone-daemon.log")
	daemonLogs := TailJSONLog(daemonLogPath, limit)

	return map[string]interface{}{
		"daemon_health":      daemonData,
		"notification_state": notifState,
		"metrics":           metricsData,
		"bisync_logs":        bisyncLogs,
		"daemon_logs":        daemonLogs,
		"config":             s.GetRuntimeConfig(),
		"is_syncing":         isSync,
		"transfer_stats":     transferStats,
	}
}
