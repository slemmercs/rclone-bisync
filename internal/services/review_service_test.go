package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

func createTestConfig(t *testing.T) (*config.Config, string) {
	t.Helper()
	tmpDir := t.TempDir()

	localDir := filepath.Join(tmpDir, "local")
	backupsDir := filepath.Join(tmpDir, "backups")
	archiveDir := filepath.Join(tmpDir, "archive")
	_ = os.MkdirAll(localDir, 0755)
	_ = os.MkdirAll(backupsDir, 0755)
	_ = os.MkdirAll(archiveDir, 0755)

	cfg := &config.Config{
		LocalPath:            localDir,
		RemotePath:           "gdrive:data",
		BaseDir:              tmpDir,
		LockFile:             filepath.Join(tmpDir, "bisync.lock"),
		PendingDeletionsFile: filepath.Join(tmpDir, "pending.json"),
		LogFile:              filepath.Join(tmpDir, "bisync.log"),
		ActiveReviewFile:     filepath.Join(tmpDir, "active_review_session.json"),
		ReviewDir:            tmpDir,
		ReviewBackupsDir:     backupsDir,
		ReviewBackupTTLHours: 24.0,
	}
	return cfg, tmpDir
}

func TestReviewService_DiscoveryAndLifecycle(t *testing.T) {
	cfg, _ := createTestConfig(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	rcClient := client.NewRcloneClient(server.URL, "", "")
	svc := NewReviewService(cfg, rcClient)

	// Initially no session
	if svc.HasBlockingReview() {
		t.Fatal("expected no blocking review initially")
	}

	sess, err := svc.GetActiveSession(context.Background())
	if err != nil || sess != nil {
		t.Fatalf("expected nil active session, got %v, err: %v", sess, err)
	}

	// Create a dummy pending deletions file and a local file
	localFile := filepath.Join(cfg.LocalPath, "deleted_item.txt")
	_ = os.WriteFile(localFile, []byte("file to be deleted"), 0644)
	_ = os.WriteFile(cfg.PendingDeletionsFile, []byte(`{"aborted_at":123456,"has_deletions":true,"files":["deleted_item.txt"],"error":"Too many deletes"}`), 0644)
	_ = os.WriteFile(cfg.LogFile, []byte("ERROR : File was deleted - deleted_item.txt: Safety abort: too many deletes"), 0644)

	// Now GetActiveSession should auto-discover
	sess, err = svc.GetActiveSession(context.Background())
	if err != nil || sess == nil {
		t.Fatalf("expected auto-discovered session, got %v, err: %v", sess, err)
	}
	if !svc.HasBlockingReview() {
		t.Fatal("expected HasBlockingReview to be true")
	}

	// Session file should exist on disk
	if _, err := os.Stat(cfg.ActiveReviewFile); err != nil {
		t.Fatalf("expected session file to be written: %v", err)
	}

	// Candidate count should be at least 1
	if len(sess.Deletions) == 0 {
		t.Fatalf("expected deletion candidates discovered, got %d", len(sess.Deletions))
	}

	// Test decision staging
	candID := sess.Deletions[0].ID
	updated, err := svc.StageCandidateDecision(sess.ID, candID, models.DecisionAcceptDelete)
	if err != nil || updated == nil {
		t.Fatalf("failed staging decision: %v", err)
	}
	if !updated.Deletions[0].Resolved {
		t.Fatal("expected candidate to be resolved")
	}

	// Staging on wrong session ID should error
	if _, err := svc.StageCandidateDecision("nonexistent", candID, models.DecisionAcceptDelete); err == nil {
		t.Fatal("expected error for nonexistent session ID")
	}

	// Apply review session
	applied, err := svc.ApplyReviewSession(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("failed applying session: %v", err)
	}
	if applied.Status != models.ReviewStatusReady {
		t.Fatalf("expected applied session READY, got %s", applied.Status)
	}

	// Active session should now be cleared
	if svc.HasBlockingReview() {
		t.Fatal("expected no blocking review after successful apply")
	}

	// Pending deletions file should be removed
	if _, err := os.Stat(cfg.PendingDeletionsFile); !os.IsNotExist(err) {
		t.Fatal("expected pending deletions file to be removed")
	}
}

func TestReviewService_DismissAndPrune(t *testing.T) {
	cfg, _ := createTestConfig(t)
	svc := NewReviewService(cfg, nil)

	// Force discovery pass
	sess, err := svc.DiscoverSession(context.Background(), "CONFLICT", "Conflict detected")
	if err != nil || sess == nil {
		t.Fatalf("DiscoverSession failed: %v", err)
	}

	if !svc.HasBlockingReview() {
		t.Fatal("expected blocking review")
	}

	// Dismiss session
	if err := svc.DismissReviewSession(sess.ID); err != nil {
		t.Fatalf("DismissReviewSession failed: %v", err)
	}

	if svc.HasBlockingReview() {
		t.Fatal("expected no blocking review after dismiss")
	}

	// Dismiss with invalid session ID
	if err := svc.DismissReviewSession("bad-id"); err == nil {
		t.Fatal("expected error on invalid dismiss ID")
	}

	// Prune backups
	pruned, err := svc.PruneBackups()
	if err != nil || pruned != 0 {
		t.Fatalf("unexpected prune result: %d, err: %v", pruned, err)
	}
}
