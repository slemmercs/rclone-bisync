package models

import "time"

// UploadQueueItem represents an upload tracked or deferred by the in-memory dock.
type UploadQueueItem struct {
	ID        string    `json:"id"`
	ProfileID string    `json:"profile_id"`
	Remote    string    `json:"remote"`
	Path      string    `json:"path"`
	FileName  string    `json:"file_name"`
	Size      int64     `json:"size"`
	Status    string    `json:"status"` // "queued", "waiting_for_sync", "uploading", "completed", "failed"
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// UploadDockStatus provides current telemetry for mid-sync uploads.
type UploadDockStatus struct {
	PendingCount int               `json:"pending_count"`
	IsSyncActive bool              `json:"is_sync_active"`
	Items        []UploadQueueItem `json:"items"`
}
