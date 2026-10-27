package remoteauth

import (
	"sync"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/core/security"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

const DefaultMaxAuditEvents = 100

// AuditLogger manages an in-memory thread-safe ring buffer of OAuth lifecycle events.
type AuditLogger struct {
	mu      sync.RWMutex
	events  []models.OAuthAuditEvent
	maxSize int
}

// NewAuditLogger initializes an AuditLogger with a given maximum capacity.
func NewAuditLogger(maxSize int) *AuditLogger {
	if maxSize <= 0 {
		maxSize = DefaultMaxAuditEvents
	}
	return &AuditLogger{
		events:  make([]models.OAuthAuditEvent, 0, maxSize),
		maxSize: maxSize,
	}
}

// LogEvent records a lifecycle event ensuring all details are redacted of secrets.
func (a *AuditLogger) LogEvent(event, remote, status, details string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Redact any potential secret signatures or sensitive tokens
	safeDetails := details
	if security.IsSensitiveValue(safeDetails) {
		safeDetails = security.RedactedString
	}

	record := models.OAuthAuditEvent{
		Event:     event,
		Remote:    remote,
		Status:    status,
		Details:   safeDetails,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if len(a.events) >= a.maxSize {
		// Evict oldest item
		a.events = append(a.events[1:], record)
	} else {
		a.events = append(a.events, record)
	}
}

// GetEvents returns a slice of recorded events ordered from newest to oldest.
func (a *AuditLogger) GetEvents(limit int) []models.OAuthAuditEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()

	total := len(a.events)
	if total == 0 {
		return []models.OAuthAuditEvent{}
	}

	const maxAllowedLimit = 1000
	if limit <= 0 || limit > total {
		limit = total
	}
	if limit > maxAllowedLimit {
		limit = maxAllowedLimit
	}
	if a.maxSize > 0 && limit > a.maxSize {
		limit = a.maxSize
	}
	if limit > maxAllowedLimit {
		limit = maxAllowedLimit
	}
	if limit <= 0 {
		limit = 1
	}

	out := make([]models.OAuthAuditEvent, 0, limit)
	for i := total - 1; i >= total-limit; i-- {
		out = append(out, a.events[i])
	}
	return out
}

// Clear flushes all stored events.
func (a *AuditLogger) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = make([]models.OAuthAuditEvent, 0, a.maxSize)
}
