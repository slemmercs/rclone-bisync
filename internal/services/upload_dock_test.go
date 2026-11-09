package services

import (
	"sync"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

type mockSyncChecker struct {
	mu            sync.RWMutex
	globalSyncing bool
	profileStates map[string]bool
}

func newMockSyncChecker() *mockSyncChecker {
	return &mockSyncChecker{
		profileStates: make(map[string]bool),
	}
}

func (m *mockSyncChecker) IsProfileSyncing(profileID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.profileStates[profileID] || m.globalSyncing
}

func (m *mockSyncChecker) IsAnySyncing() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.globalSyncing {
		return true
	}
	for _, syncing := range m.profileStates {
		if syncing {
			return true
		}
	}
	return false
}

func (m *mockSyncChecker) setProfileSyncing(profileID string, syncing bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profileStates[profileID] = syncing
}

func TestUploadDock_EnqueueAndStatus_Idle(t *testing.T) {
	// Arrange
	checker := newMockSyncChecker()
	dock := NewUploadDock(checker)

	// Act
	item, err := dock.Enqueue(models.UploadQueueItem{
		ProfileID: "prof_1",
		Remote:    "onedrive:",
		Path:      "docs",
		FileName:  "report.pdf",
		Size:      1024,
	})

	// Assert
	if err != nil {
		t.Fatalf("unexpected enqueue error: %v", err)
	}
	if item.Status != "queued" {
		t.Fatalf("expected status queued when idle, got %s", item.Status)
	}
	if item.ID == "" {
		t.Fatal("expected generated item ID")
	}

	status := dock.GetStatus("prof_1")
	if status.PendingCount != 1 {
		t.Fatalf("expected 1 pending item, got %d", status.PendingCount)
	}
	if status.IsSyncActive {
		t.Fatal("expected IsSyncActive to be false")
	}
	if len(status.Items) != 1 {
		t.Fatalf("expected 1 item returned, got %d", len(status.Items))
	}
}

func TestUploadDock_EnqueueAndStatus_Syncing(t *testing.T) {
	// Arrange
	checker := newMockSyncChecker()
	checker.setProfileSyncing("prof_active", true)
	dock := NewUploadDock(checker)

	// Act
	item, err := dock.Enqueue(models.UploadQueueItem{
		ProfileID: "prof_active",
		Remote:    "gdrive:",
		Path:      "sheets",
		FileName:  "finance.xlsx",
		Size:      2048,
	})

	// Assert
	if err != nil {
		t.Fatalf("unexpected enqueue error: %v", err)
	}
	if item.Status != "waiting_for_sync" {
		t.Fatalf("expected status waiting_for_sync when sync is active, got %s", item.Status)
	}

	status := dock.GetStatus("prof_active")
	if !status.IsSyncActive {
		t.Fatal("expected IsSyncActive to be true")
	}
	if status.PendingCount != 1 {
		t.Fatalf("expected 1 pending item while waiting, got %d", status.PendingCount)
	}

	// Act 2: Transition when sync finishes
	checker.setProfileSyncing("prof_active", false)
	statusAfter := dock.GetStatus("prof_active")
	if statusAfter.IsSyncActive {
		t.Fatal("expected IsSyncActive to be false after sync finishes")
	}
	if len(statusAfter.Items) != 1 || statusAfter.Items[0].Status != "queued" {
		t.Fatalf("expected waiting item to transition to queued, got: %+v", statusAfter.Items)
	}
}

func TestUploadDock_LifecycleAndActiveUploads(t *testing.T) {
	// Arrange
	checker := newMockSyncChecker()
	dock := NewUploadDock(checker)

	item, _ := dock.Enqueue(models.UploadQueueItem{
		ProfileID: "prof_life",
		Remote:    "local",
		Path:      "test",
		FileName:  "life.txt",
		Size:      50,
	})

	// Act 1: Transition to uploading
	dock.UpdateItemStatus(item.ID, "uploading", "")
	if !dock.HasActiveUploads("prof_life") {
		t.Fatal("expected active uploads to be true")
	}
	if dock.ActiveUploadsCount("prof_life") != 1 {
		t.Fatalf("expected 1 active upload, got %d", dock.ActiveUploadsCount("prof_life"))
	}

	// Act 2: Transition to completed
	dock.UpdateItemStatus(item.ID, "completed", "")
	if dock.HasActiveUploads("prof_life") {
		t.Fatal("expected active uploads to be false after completion")
	}
	status := dock.GetStatus("prof_life")
	if status.PendingCount != 0 {
		t.Fatalf("expected 0 pending count after completion, got %d", status.PendingCount)
	}

	// Act 3: Clear completed
	dock.ClearCompleted("prof_life")
	statusCleared := dock.GetStatus("prof_life")
	if len(statusCleared.Items) != 0 {
		t.Fatalf("expected 0 items after clearing completed, got %d", len(statusCleared.Items))
	}

	// Act 4: Remove by ID
	item2, _ := dock.Enqueue(models.UploadQueueItem{
		ProfileID: "prof_life",
		Remote:    "local",
		Path:      "test",
		FileName:  "item2.txt",
	})
	if !dock.RemoveItem(item2.ID) {
		t.Fatal("expected RemoveItem to succeed")
	}
	if _, found := dock.GetItem(item2.ID); found {
		t.Fatal("expected item2 to be gone")
	}
}
