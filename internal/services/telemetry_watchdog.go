package services

import (
	"context"
	"time"
)

// SetReconnectGracePeriod configures the grace period before a disconnected daemon is declared offline.
func (t *TelemetryService) SetReconnectGracePeriod(d time.Duration) {
	t.lockMu.Lock()
	defer t.lockMu.Unlock()
	t.gracePeriod = d
}

// ResetDaemonHealthCache invalidates cached live health data.
func (t *TelemetryService) ResetDaemonHealthCache() {
	t.lockMu.Lock()
	defer t.lockMu.Unlock()
	t.liveHealthCache = LiveHealthCache{}
}

// InReconnectGracePeriod checks if the daemon is disconnected but still within the grace period.
func (t *TelemetryService) InReconnectGracePeriod() bool {
	t.lockMu.Lock()
	defer t.lockMu.Unlock()
	grace := t.gracePeriod
	if grace <= 0 {
		grace = 15 * time.Second
	}
	return !t.lastHealthyTime.IsZero() && time.Since(t.lastHealthyTime) <= grace
}

// WaitForDaemon blocks until the upstream RC daemon is healthy or timeout expires.
func (t *TelemetryService) WaitForDaemon(ctx context.Context, timeout time.Duration) bool {
	if t.Client == nil {
		return false
	}
	healthy, _ := t.Client.WaitForHealthy(ctx, timeout)
	if healthy {
		t.lockMu.Lock()
		t.lastHealthyTime = time.Now()
		t.liveHealthCache = LiveHealthCache{}
		t.lockMu.Unlock()
	}
	return healthy
}

// GetDaemonHealth retrieves daemon health with caching and auto-reconnect grace period.
func (t *TelemetryService) GetDaemonHealth(ctx context.Context) map[string]interface{} {
	now := float64(time.Now().Unix())

	t.lockMu.Lock()
	if t.liveHealthCache.Data != nil && (now-t.liveHealthCache.Time) <= 5 {
		data := t.liveHealthCache.Data
		t.lockMu.Unlock()
		return data
	}
	grace := t.gracePeriod
	if grace <= 0 {
		grace = 15 * time.Second
	}
	lastHealthy := t.lastHealthyTime
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
		if !lastHealthy.IsZero() && time.Since(lastHealthy) <= grace {
			statusStr = "reconnecting"
		}
	}

	health := map[string]interface{}{
		"status":           statusStr,
		"response_time_ms": latency,
		"last_check":       now,
	}

	t.lockMu.Lock()
	if healthy {
		t.lastHealthyTime = time.Now()
	}
	t.liveHealthCache.Time = now
	t.liveHealthCache.Data = health
	t.lockMu.Unlock()

	return health
}
