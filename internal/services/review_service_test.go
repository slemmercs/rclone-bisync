package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/engine/review"
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

func TestReviewService_AutoSkipAndFailedRetry(t *testing.T) {
	cfg, _ := createTestConfig(t)
	isAuthorized := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isAuthorized {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"401 Unauthorized"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	rcClient := client.NewRcloneClient(server.URL, "user", "pass")
	svc := NewReviewService(cfg, rcClient)

	// Create dummy session with 2 candidates
	sess, err := svc.DiscoverSession(context.Background(), "TOO_MANY_DELETES", "Safety abort")
	if err != nil || sess == nil {
		t.Fatalf("DiscoverSession failed: %v", err)
	}

	// Manually inject 2 deletion candidates
	svc.mu.Lock()
	sess.Deletions = []models.DeletionCandidate{
		{
			ID:       "del-auth",
			Path:     "auth_test.pdf",
			Source:   models.SideLocal,
			Decision: "",
			Resolved: false,
		},
		{
			ID:       "del-pending",
			Path:     "pending_test.pdf",
			Source:   models.SideLocal,
			Decision: "",
			Resolved: false,
		},
	}
	svc.mu.Unlock()

	// Stage first candidate using alias "restore-local"
	updated, err := svc.StageCandidateDecision(sess.ID, "del-auth", "restore-local")
	if err != nil {
		t.Fatalf("StageCandidateDecision with restore-local alias failed: %v", err)
	}
	if updated.Deletions[0].Decision != models.DecisionRestoreItem {
		t.Fatalf("expected restore-local to normalize to restore-item, got %s", updated.Deletions[0].Decision)
	}

	// Apply review session with pending candidate remaining:
	// It will auto-mark del-pending as skip, then attempt to restore del-auth.
	// Since isAuthorized == false, it will fail with 401 Unauthorized!
	_, err = svc.ApplyReviewSession(context.Background(), sess.ID)
	if err == nil {
		t.Fatal("expected error due to 401 Unauthorized, got nil")
	}

	// Session should now be FAILED
	active, _ := svc.GetActiveSession(context.Background())
	if active.Status != models.ReviewStatusFailed {
		t.Fatalf("expected FAILED status after 401, got %s", active.Status)
	}

	// Re-stage decision to 'skip' on FAILED session
	reStaged, err := svc.StageCandidateDecision(sess.ID, "del-auth", models.DecisionSkip)
	if err != nil {
		t.Fatalf("failed to stage decision on FAILED session: %v", err)
	}
	if reStaged.Status != models.ReviewStatusReviewing {
		t.Fatalf("expected status REVIEWING after re-staging, got %s", reStaged.Status)
	}

	// Re-apply: since all candidates are now skipped, apply should succeed without remote calls
	applied, err := svc.ApplyReviewSession(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("expected second apply to succeed with skipped candidates, got: %v", err)
	}
	if applied.Status != models.ReviewStatusReady {
		t.Fatalf("expected READY status, got %s", applied.Status)
	}
}

func TestReviewService_ApplyBatchRecommendations(t *testing.T) {
	mockRC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer mockRC.Close()

	cfg, _ := createTestConfig(t)
	rcClient := client.NewRcloneClient(mockRC.URL, "", "")
	svc := NewReviewService(cfg, rcClient)

	now := time.Now().UTC()
	sess := &models.ReviewSession{
		ID:        "batch-test-session",
		CreatedAt: now,
		Status:    models.ReviewStatusReviewing,
		Conflicts: []models.ConflictCandidate{
			{
				ID:            "conf-temp",
				Path:          "~$TempDoc.docx",
				Name:          "~$TempDoc.docx",
				LocalSize:     100,
				LocalModTime:  now.Format(time.RFC3339),
				RemoteSize:    100,
				RemoteModTime: now.Add(-time.Hour).Format(time.RFC3339),
			},
			{
				ID:            "conf-remote-newer",
				Path:          "notes.txt",
				Name:          "notes.txt",
				LocalSize:     100,
				LocalModTime:  now.Add(-2 * time.Hour).Format(time.RFC3339),
				RemoteSize:    120,
				RemoteModTime: now.Format(time.RFC3339),
			},
			{
				ID:            "conf-truncation",
				Path:          "corrupted.pdf",
				Name:          "corrupted.pdf",
				LocalSize:     0,
				LocalModTime:  now.Format(time.RFC3339),
				RemoteSize:    50000,
				RemoteModTime: now.Add(-10 * time.Minute).Format(time.RFC3339),
			},
		},
	}

	svc.mu.Lock()
	svc.session = sess
	svc.mu.Unlock()

	// Apply batch with minConfidence 0.70 and maxRisk LOW
	staged, err := svc.ApplyBatchRecommendations(context.Background(), sess.ID, 0.70, "LOW")
	if err != nil {
		t.Fatalf("unexpected error applying batch recommendations: %v", err)
	}

	if staged != 2 {
		t.Fatalf("expected 2 candidates staged, got %d", staged)
	}

	active, err := svc.GetActiveSession(context.Background())
	if err != nil || active == nil {
		t.Fatalf("expected active session, err: %v", err)
	}

	if active.Conflicts[0].Decision != models.DecisionKeepLocal {
		t.Errorf("expected conf-temp to be keep-local, got %s", active.Conflicts[0].Decision)
	}
	if active.Conflicts[1].Decision != models.DecisionKeepRemote {
		t.Errorf("expected conf-remote-newer to be keep-remote, got %s", active.Conflicts[1].Decision)
	}
	if active.Conflicts[2].Decision != "" || active.Conflicts[2].Resolved {
		t.Errorf("expected conf-truncation (CRITICAL risk) to remain unresolved, got %s", active.Conflicts[2].Decision)
	}
}

func TestReviewService_DeletionRecommendations(t *testing.T) {
	cfg, _ := createTestConfig(t)
	svc := NewReviewService(cfg, nil)

	sess := &models.ReviewSession{
		ID:        "del-rec-test",
		CreatedAt: time.Now().UTC(),
		Status:    models.ReviewStatusReviewing,
		Deletions: []models.DeletionCandidate{
			{
				ID:     "del-canary",
				Path:   "RCLONE_TEST",
				Name:   "RCLONE_TEST",
				Source: models.SideRemote,
			},
			{
				ID:     "del-templock",
				Path:   "sub/~$doc.docx",
				Name:   "~$doc.docx",
				Source: models.SideLocal,
			},
			{
				ID:     "del-userlocal",
				Path:   "data/file.txt",
				Name:   "file.txt",
				Source: models.SideLocal,
			},
		},
	}

	svc.mu.Lock()
	svc.session = sess
	svc.mu.Unlock()

	active, err := svc.GetActiveSession(context.Background())
	if err != nil || active == nil {
		t.Fatalf("expected active session: %v", err)
	}

	// 1. Canary has CRITICAL risk and restore-item suggestion
	canaryRec := active.Deletions[0].Recommendation
	if canaryRec == nil || canaryRec.Risk != "CRITICAL" || canaryRec.SuggestedAction != string(models.DecisionRestoreItem) {
		t.Fatalf("expected CRITICAL restore-item for canary, got: %+v", canaryRec)
	}

	// 2. Temp lock has LOW risk and accept-delete suggestion
	tempRec := active.Deletions[1].Recommendation
	if tempRec == nil || tempRec.Risk != "LOW" || tempRec.SuggestedAction != string(models.DecisionAcceptDelete) {
		t.Fatalf("expected LOW accept-delete for temp lock, got: %+v", tempRec)
	}

	// 3. User local deletion has LOW risk and accept-delete suggestion
	userRec := active.Deletions[2].Recommendation
	if userRec == nil || userRec.Risk != "LOW" || userRec.SuggestedAction != string(models.DecisionAcceptDelete) {
		t.Fatalf("expected LOW accept-delete for user local deletion, got: %+v", userRec)
	}

	// Batch auto-stage with LOW risk should stage the 2 safe deletions, leaving canary unresolved
	staged, err := svc.ApplyBatchRecommendations(context.Background(), sess.ID, 0.70, "LOW")
	if err != nil {
		t.Fatalf("ApplyBatchRecommendations failed: %v", err)
	}
	if staged != 2 {
		t.Fatalf("expected 2 deletions staged, got %d", staged)
	}
	if active.Deletions[0].Decision != "" {
		t.Fatalf("expected canary to remain unresolved, got: %s", active.Deletions[0].Decision)
	}
	if active.Deletions[1].Decision != models.DecisionAcceptDelete {
		t.Fatalf("expected temp lock to be accept-delete, got: %s", active.Deletions[1].Decision)
	}
	if active.Deletions[2].Decision != models.DecisionAcceptDelete {
		t.Fatalf("expected user local to be accept-delete, got: %s", active.Deletions[2].Decision)
	}
}

func TestReviewService_DiscoverFromDaemonLog(t *testing.T) {
	cfg, _ := createTestConfig(t)
	svc := NewReviewService(cfg, nil)

	// Daemon log with files containing spaces
	daemonLog := filepath.Join(cfg.BaseDir, "rclone-daemon.log")
	logContent := `{"time":"2026-10-24T22:16:00Z","level":"info","msg":"- Path2             File was deleted                            - Csaba/etc/Daimler Corporate Color Palette.xlsx"}` + "\n" +
		`{"time":"2026-10-24T22:16:00Z","level":"error","msg":"too many deletes (>0%, 4 of 7061) on Path2 \"/home/csaba/gdrive/\". Run with --force if desired."}` + "\n"
	_ = os.WriteFile(daemonLog, []byte(logContent), 0644)

	// bisync.log has only summary error
	_ = os.WriteFile(cfg.LogFile, []byte(`{"time":"2026-10-24T22:16:00+02:00","level":"error","msg":"Sync failed [TOO_MANY_DELETES]: too many deletes"}`+"\n"), 0644)

	// Pending deletions marker exists
	_ = os.WriteFile(cfg.PendingDeletionsFile, []byte(`{"aborted_at":123456,"has_deletions":true,"error":"too many deletes"}`), 0644)

	sess, err := svc.GetActiveSession(context.Background())
	if err != nil || sess == nil {
		t.Fatalf("expected active session auto-discovered from daemon log: %v, err: %v", sess, err)
	}

	if len(sess.Deletions) != 1 {
		t.Fatalf("expected 1 deletion candidate, got %d", len(sess.Deletions))
	}
	expectedPath := "Csaba/etc/Daimler Corporate Color Palette.xlsx"
	if sess.Deletions[0].Path != expectedPath {
		t.Fatalf("expected candidate path %q, got %q", expectedPath, sess.Deletions[0].Path)
	}
}

func TestReviewService_AutoDismissEmptySession(t *testing.T) {
	cfg, _ := createTestConfig(t)
	svc := NewReviewService(cfg, nil)

	// Empty logs
	_ = os.WriteFile(cfg.LogFile, []byte(""), 0644)

	// Pending deletions marker exists but no logs have files
	_ = os.WriteFile(cfg.PendingDeletionsFile, []byte(`{"aborted_at":123456,"has_deletions":true,"error":"too many deletes"}`), 0644)

	sess, err := svc.GetActiveSession(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess != nil {
		t.Fatalf("expected empty session to be cleaned up and return nil, got: %+v", sess)
	}

	// Pending deletions file should be removed
	if _, err := os.Stat(cfg.PendingDeletionsFile); !os.IsNotExist(err) {
		t.Fatal("expected pending deletions file to be removed when empty")
	}
	if svc.HasBlockingReview() {
		t.Fatal("expected no blocking review for empty session")
	}
}

func TestReviewService_ParseCandidateTime(t *testing.T) {
	// 1. Empty string
	if parsed := parseCandidateTime(""); !parsed.IsZero() {
		t.Fatalf("expected zero time for empty string, got %v", parsed)
	}

	// 2. RFC3339Nano
	nanoStr := "2026-10-25T13:45:00.123456789Z"
	if parsed := parseCandidateTime(nanoStr); parsed.IsZero() || parsed.Year() != 2026 || parsed.Nanosecond() != 123456789 {
		t.Fatalf("unexpected parse for RFC3339Nano: %v", parsed)
	}

	// 3. RFC3339
	rfcStr := "2026-10-25T13:45:00Z"
	if parsed := parseCandidateTime(rfcStr); parsed.IsZero() || parsed.Year() != 2026 || parsed.Minute() != 45 {
		t.Fatalf("unexpected parse for RFC3339: %v", parsed)
	}

	// 4. "2006-01-02 15:04:05"
	stdStr := "2026-10-25 13:45:00"
	if parsed := parseCandidateTime(stdStr); parsed.IsZero() || parsed.Year() != 2026 || parsed.Hour() != 13 {
		t.Fatalf("unexpected parse for standard format: %v", parsed)
	}

	// 5. Invalid string
	if parsed := parseCandidateTime("not-a-timestamp"); !parsed.IsZero() {
		t.Fatalf("expected zero time for invalid string, got %v", parsed)
	}
}

func TestReviewService_RollbackAndAudit(t *testing.T) {
	cfg, _ := createTestConfig(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	rcClient := client.NewRcloneClient(server.URL, "", "")
	svc := NewReviewService(cfg, rcClient)

	auditSvc, _, _ := setupTestAuditService(t)
	svc.Audit = auditSvc

	ctx := context.Background()

	// 1. Rollback on non-existent session
	if _, err := svc.RollbackReviewSession(ctx, "nonexistent-session"); err == nil {
		t.Fatal("expected error on nonexistent session rollback")
	}

	// 2. Set up local file and pending deletion to create a discovered session
	localFile := filepath.Join(cfg.LocalPath, "item_to_delete.txt")
	_ = os.WriteFile(localFile, []byte("content"), 0644)
	_ = os.WriteFile(cfg.PendingDeletionsFile, []byte(`{"aborted_at":123456,"has_deletions":true,"files":["item_to_delete.txt"],"error":"Too many deletes"}`), 0644)
	_ = os.WriteFile(cfg.LogFile, []byte("ERROR : File was deleted - item_to_delete.txt: Safety abort: too many deletes"), 0644)

	sess, err := svc.GetActiveSession(ctx)
	if err != nil || sess == nil {
		t.Fatalf("failed to discover session: %v", err)
	}

	// Stage decision and apply
	candID := sess.Deletions[0].ID
	_, err = svc.StageCandidateDecision(sess.ID, candID, models.DecisionAcceptDelete)
	if err != nil {
		t.Fatalf("failed to stage decision: %v", err)
	}

	applied, err := svc.ApplyReviewSession(ctx, sess.ID)
	if err != nil {
		t.Fatalf("failed to apply session: %v", err)
	}

	// Re-assign session to simulate rollback on that session
	svc.session = applied
	rolledBack, err := svc.RollbackReviewSession(ctx, applied.ID)
	if err != nil {
		t.Fatalf("failed to rollback session: %v", err)
	}
	if rolledBack == nil {
		t.Fatal("expected rolled back session, got nil")
	}

	// Verify audit record exists for rollback
	records, _, err := auditSvc.Query(audit.Filter{
		Action: audit.ActionRollbackExecuted,
	})
	if err != nil || len(records) == 0 {
		t.Fatalf("expected audit event for rollback, got %d records, err: %v", len(records), err)
	}
}

func TestReviewService_PathHelpersAndNilConfig(t *testing.T) {
	// 1. Nil config
	svcNil := &ReviewService{}
	if path := svcNil.sessionFilePath(); path != "" {
		t.Fatalf("expected empty sessionFilePath, got %q", path)
	}
	if path := svcNil.walFilePath(); path != "" {
		t.Fatalf("expected empty walFilePath, got %q", path)
	}
	if path := svcNil.backupsDirPath(); path != "" {
		t.Fatalf("expected empty backupsDirPath, got %q", path)
	}
	if path := svcNil.reviewDirPath(); path != "" {
		t.Fatalf("expected empty reviewDirPath, got %q", path)
	}
	if _, err := svcNil.DiscoverSession(context.Background(), "ERR", "msg"); err == nil {
		t.Fatal("expected error on nil config DiscoverSession")
	}
	if pruned, err := svcNil.PruneBackups(); pruned != 0 || err != nil {
		t.Fatalf("expected (0, nil) from PruneBackups on nil config, got (%d, %v)", pruned, err)
	}

	// 2. BaseDir fallback paths
	tmpDir := t.TempDir()
	cfgBase := &config.Config{
		BaseDir: tmpDir,
	}
	svcBase := NewReviewService(cfgBase, nil)
	if expected := filepath.Join(tmpDir, "active_review_session.json"); svcBase.sessionFilePath() != expected {
		t.Fatalf("expected %q, got %q", expected, svcBase.sessionFilePath())
	}
	if expected := filepath.Join(tmpDir, "review.wal"); svcBase.walFilePath() != expected {
		t.Fatalf("expected %q, got %q", expected, svcBase.walFilePath())
	}
	if expected := filepath.Join(tmpDir, "review_backups"); svcBase.backupsDirPath() != expected {
		t.Fatalf("expected %q, got %q", expected, svcBase.backupsDirPath())
	}
	if expected := filepath.Join(tmpDir, "review_sessions"); svcBase.reviewDirPath() != expected {
		t.Fatalf("expected %q, got %q", expected, svcBase.reviewDirPath())
	}
}

func TestReviewService_BatchAndStagingEdgeCases(t *testing.T) {
	cfg, _ := createTestConfig(t)
	svc := NewReviewService(cfg, nil)
	ctx := context.Background()

	// 1. ApplyBatchRecommendations with no active session
	if _, err := svc.ApplyBatchRecommendations(ctx, "", 0.8, "LOW"); err == nil {
		t.Fatal("expected error when no active session")
	}

	// 2. Setup active session
	localFile := filepath.Join(cfg.LocalPath, "test_file.txt")
	_ = os.WriteFile(localFile, []byte("content"), 0644)
	_ = os.WriteFile(cfg.PendingDeletionsFile, []byte(`{"aborted_at":123456,"has_deletions":true,"files":["test_file.txt"],"error":"Too many deletes"}`), 0644)
	_ = os.WriteFile(cfg.LogFile, []byte("ERROR : File was deleted - test_file.txt: Safety abort: too many deletes"), 0644)

	sess, err := svc.GetActiveSession(ctx)
	if err != nil || sess == nil {
		t.Fatalf("failed to get active session: %v", err)
	}

	// 3. Batch apply with mismatched session ID
	if _, err := svc.ApplyBatchRecommendations(ctx, "wrong-id", 0.8, "LOW"); err == nil {
		t.Fatal("expected error with mismatched session ID")
	}

	// 4. Stage decision with unknown candidate ID
	if _, err := svc.StageCandidateDecision(sess.ID, "unknown-candidate", models.DecisionAcceptDelete); err == nil {
		t.Fatal("expected error for unknown candidate ID")
	}

	// 5. Dismiss with invalid session ID
	if err := svc.DismissReviewSession("wrong-session-id"); err == nil {
		t.Fatal("expected error dismissing wrong session ID")
	}

	// 6. Dismiss on active session with AuditService
	auditSvc, _, _ := setupTestAuditService(t)
	svc.Audit = auditSvc
	if err := svc.DismissReviewSession(sess.ID); err != nil {
		t.Fatalf("failed to dismiss session: %v", err)
	}
	records, _, err := auditSvc.Query(audit.Filter{
		Action: audit.ActionReviewDismissed,
	})
	if err != nil || len(records) == 0 {
		t.Fatalf("expected audit event for dismiss, got %d records, err: %v", len(records), err)
	}
}

func TestReviewService_NewWithWALReconciliation(t *testing.T) {
	cfg, tmpDir := createTestConfig(t)
	sess := review.NewSession("CONFLICT", "err", nil, nil)
	sessFile := filepath.Join(tmpDir, "active_review_session.json")
	walFile := filepath.Join(tmpDir, "review.wal")
	_ = review.SaveActiveSession(sessFile, sess)
	_ = review.AppendWAL(walFile, models.WALEntry{
		SessionID: sess.ID,
		Step:      "MUTATION_COMPLETED",
		ItemPath:  "test.txt",
		Type:      models.DecisionAcceptDelete,
	})

	svc := NewReviewService(cfg, nil)
	if svc.session == nil || svc.session.ID != sess.ID {
		t.Fatalf("expected reconciled session loaded, got %v", svc.session)
	}
}
