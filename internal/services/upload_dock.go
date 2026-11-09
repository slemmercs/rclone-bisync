package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// SyncStatusChecker defines the contract to inspect active synchronization runs.
type SyncStatusChecker interface {
	IsProfileSyncing(profileID string) bool
	IsAnySyncing() bool
}

// UploadDock manages an in-memory queue of pending and active uploads,
// coordinating with the sync engine to defer transfers when sync cycles are running.
type UploadDock struct {
	mu      sync.RWMutex
	checker SyncStatusChecker
	items   []models.UploadQueueItem
}

// NewUploadDock creates a new UploadDock.
func NewUploadDock(checker SyncStatusChecker) *UploadDock {
	return &UploadDock{
		checker: checker,
		items:   make([]models.UploadQueueItem, 0),
	}
}

// Enqueue adds an upload item to the dock. If a sync cycle is currently active for
// the profile, its status is set to "waiting_for_sync"; otherwise it defaults to "queued".
func (d *UploadDock) Enqueue(item models.UploadQueueItem) (models.UploadQueueItem, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if item.FileName == "" {
		return models.UploadQueueItem{}, fmt.Errorf("file name is required")
	}

	if item.ID == "" {
		b := make([]byte, 4)
		_, _ = rand.Read(b)
		item.ID = fmt.Sprintf("upl-%d-%s", time.Now().UnixNano(), hex.EncodeToString(b))
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}

	isSyncing := d.isSyncingLocked(item.ProfileID)
	if isSyncing {
		item.Status = "waiting_for_sync"
	} else if item.Status == "" {
		item.Status = "queued"
	}

	d.items = append(d.items, item)
	return item, nil
}

// GetStatus returns the current dock telemetry and items, optionally filtered by profileID.
func (d *UploadDock) GetStatus(profileID string) models.UploadDockStatus {
	d.mu.Lock()
	defer d.mu.Unlock()

	isSyncing := d.isSyncingLocked(profileID)

	var filtered []models.UploadQueueItem
	pendingCount := 0

	for i := range d.items {
		// If sync has released, transition waiting items to queued
		if !isSyncing && d.items[i].Status == "waiting_for_sync" {
			if profileID == "" || d.items[i].ProfileID == profileID {
				d.items[i].Status = "queued"
			}
		}

		if profileID != "" && d.items[i].ProfileID != profileID {
			continue
		}

		status := d.items[i].Status
		if status == "queued" || status == "waiting_for_sync" || status == "uploading" {
			pendingCount++
		}
		filtered = append(filtered, d.items[i])
	}

	if filtered == nil {
		filtered = []models.UploadQueueItem{}
	}

	return models.UploadDockStatus{
		PendingCount: pendingCount,
		IsSyncActive: isSyncing,
		Items:        filtered,
	}
}

// UpdateItemStatus updates the execution status and optional error message of a queued item.
func (d *UploadDock) UpdateItemStatus(id string, status string, errStr string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	for i := range d.items {
		if d.items[i].ID == id {
			d.items[i].Status = status
			d.items[i].Error = errStr
			return true
		}
	}
	return false
}

// GetItem retrieves a queued item by its unique ID.
func (d *UploadDock) GetItem(id string) (models.UploadQueueItem, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, item := range d.items {
		if item.ID == id {
			return item, true
		}
	}
	return models.UploadQueueItem{}, false
}

// RemoveItem removes an item from the dock by ID.
func (d *UploadDock) RemoveItem(id string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	for i, item := range d.items {
		if item.ID == id {
			d.items = append(d.items[:i], d.items[i+1:]...)
			return true
		}
	}
	return false
}

// ClearCompleted purges all completed and failed uploads from memory.
func (d *UploadDock) ClearCompleted(profileID string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var retained []models.UploadQueueItem
	for _, item := range d.items {
		if (item.Status == "completed" || item.Status == "failed") && (profileID == "" || item.ProfileID == profileID) {
			continue
		}
		retained = append(retained, item)
	}
	d.items = retained
}

// ActiveUploadsCount returns the number of uploads currently in-flight.
func (d *UploadDock) ActiveUploadsCount(profileID string) int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	count := 0
	for _, item := range d.items {
		if item.Status == "uploading" && (profileID == "" || item.ProfileID == profileID) {
			count++
		}
	}
	return count
}

// HasActiveUploads checks whether any upload is currently in-flight.
func (d *UploadDock) HasActiveUploads(profileID string) bool {
	return d.ActiveUploadsCount(profileID) > 0
}

func (d *UploadDock) isSyncingLocked(profileID string) bool {
	if d.checker == nil {
		return false
	}
	if profileID != "" {
		return d.checker.IsProfileSyncing(profileID)
	}
	return d.checker.IsAnySyncing()
}
