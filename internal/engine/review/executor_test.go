package review

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

func TestExecutor_AcceptDelete(t *testing.T) {
	tmpDir := t.TempDir()
	localRoot := filepath.Join(tmpDir, "local")
	backupDir := filepath.Join(tmpDir, "backups")
	walPath := filepath.Join(tmpDir, "review.wal")
	_ = os.MkdirAll(localRoot, 0755)

	localFile := filepath.Join(localRoot, "doc.txt")
	_ = os.WriteFile(localFile, []byte("local document"), 0644)

	var remoteDeleted string
	mockClient := &mockRemoteClient{
		deleteFileFunc: func(ctx context.Context, fs, remote string) (bool, error) {
			remoteDeleted = remote
			return true, nil
		},
	}

	exec := NewResolutionExecutor(localRoot, "gdrive", backupDir, walPath, mockClient)

	// Case 1: Source Remote -> delete local file
	sess := NewSession("TOO_MANY_DELETES", "Review", []models.DeletionCandidate{
		{
			ID:       "del-1",
			Path:     "doc.txt",
			Source:   models.SideRemote,
			Decision: models.DecisionAcceptDelete,
			Resolved: true,
		},
	}, nil)

	if err := exec.ExecuteSession(context.Background(), sess); err != nil {
		t.Fatalf("ExecuteSession failed: %v", err)
	}

	if sess.Status != models.ReviewStatusReady {
		t.Fatalf("expected session status READY, got %s", sess.Status)
	}

	// Local file should be removed
	if _, err := os.Stat(localFile); !os.IsNotExist(err) {
		t.Fatal("expected local file to be deleted")
	}

	// Backup should exist
	if len(sess.Actions) != 1 || sess.Actions[0].BackupPath == "" {
		t.Fatalf("expected action with backup path, got %+v", sess.Actions)
	}

	// WAL should contain entries
	walEntries, err := ReadWAL(walPath)
	if err != nil || len(walEntries) == 0 {
		t.Fatalf("expected WAL entries, got %v, err: %v", walEntries, err)
	}

	// Case 2: Source Local -> delete remote file
	sess2 := NewSession("TOO_MANY_DELETES", "Review", []models.DeletionCandidate{
		{
			ID:       "del-2",
			Path:     "cloud.txt",
			Source:   models.SideLocal,
			Decision: models.DecisionAcceptDelete,
			Resolved: true,
		},
	}, nil)

	if err := exec.ExecuteSession(context.Background(), sess2); err != nil {
		t.Fatalf("ExecuteSession failed: %v", err)
	}
	if remoteDeleted != "cloud.txt" {
		t.Fatalf("expected remote file cloud.txt to be deleted, got %s", remoteDeleted)
	}
}
func TestExecutor_AcceptDelete_AlreadyAbsentRemote(t *testing.T) {
	tmpDir := t.TempDir()
	localRoot := filepath.Join(tmpDir, "local")
	backupDir := filepath.Join(tmpDir, "backups")
	walPath := filepath.Join(tmpDir, "review.wal")
	_ = os.MkdirAll(localRoot, 0755)

	mockClient := &mockRemoteClient{
		deleteFileFunc: func(ctx context.Context, fs, remote string) (bool, error) {
			// Simulate 404 object not found from Rclone RC
			return false, fmt.Errorf("deletefile failed (HTTP 404): {\"error\": \"object not found\"}")
		},
	}

	exec := NewResolutionExecutor(localRoot, "gdrive", backupDir, walPath, mockClient)

	sess := NewSession("TOO_MANY_DELETES", "Review needed", []models.DeletionCandidate{
		{
			ID:       "del-absent",
			Path:     "already_gone.tar.gz",
			Source:   models.SideLocal,
			Decision: models.DecisionAcceptDelete,
			Resolved: true,
		},
	}, nil)

	if err := exec.ExecuteSession(context.Background(), sess); err != nil {
		t.Fatalf("expected idempotent success when remote file is already absent, got: %v", err)
	}

	if sess.Status != models.ReviewStatusReady {
		t.Fatalf("expected status READY after successful execution, got %s", sess.Status)
	}
}

func TestExecutor_RestoreItem(t *testing.T) {
	tmpDir := t.TempDir()
	localRoot := filepath.Join(tmpDir, "local")
	backupDir := filepath.Join(tmpDir, "backups")
	walPath := filepath.Join(tmpDir, "review.wal")
	_ = os.MkdirAll(localRoot, 0755)

	localFile := filepath.Join(localRoot, "restored.txt")
	_ = os.WriteFile(localFile, []byte("restore me"), 0644)

	var copiedDst string
	mockClient := &mockRemoteClient{
		copyFileFunc: func(ctx context.Context, srcFs, srcRemote, dstFs, dstRemote string) (bool, error) {
			copiedDst = dstRemote
			return true, nil
		},
	}

	exec := NewResolutionExecutor(localRoot, "gdrive", backupDir, walPath, mockClient)

	// SideRemote: local exists, restore to remote
	sess := NewSession("TOO_MANY_DELETES", "Review", []models.DeletionCandidate{
		{
			ID:       "del-1",
			Path:     "restored.txt",
			Source:   models.SideRemote,
			Decision: models.DecisionRestoreItem,
			Resolved: true,
		},
	}, nil)

	if err := exec.ExecuteSession(context.Background(), sess); err != nil {
		t.Fatalf("ExecuteSession failed: %v", err)
	}

	if copiedDst != "restored.txt" {
		t.Fatalf("expected restore to copy to restored.txt, got %s", copiedDst)
	}
}

func TestExecutor_Conflicts(t *testing.T) {
	tmpDir := t.TempDir()
	localRoot := filepath.Join(tmpDir, "local")
	backupDir := filepath.Join(tmpDir, "backups")
	walPath := filepath.Join(tmpDir, "review.wal")
	_ = os.MkdirAll(localRoot, 0755)

	localFile := filepath.Join(localRoot, "conflict.txt")
	_ = os.WriteFile(localFile, []byte("conflict local"), 0644)

	var copyCalls []string
	mockClient := &mockRemoteClient{
		copyFileFunc: func(ctx context.Context, srcFs, srcRemote, dstFs, dstRemote string) (bool, error) {
			copyCalls = append(copyCalls, fmt.Sprintf("%s -> %s", srcRemote, dstRemote))
			return true, nil
		},
	}

	exec := NewResolutionExecutor(localRoot, "gdrive", backupDir, walPath, mockClient)

	// Keep Local
	sessLocal := NewSession("CONFLICT", "Review", nil, []models.ConflictCandidate{
		{
			ID:       "conf-1",
			Path:     "conflict.txt",
			Decision: models.DecisionKeepLocal,
			Resolved: true,
		},
	})
	if err := exec.ExecuteSession(context.Background(), sessLocal); err != nil {
		t.Fatalf("ExecuteSession keep-local failed: %v", err)
	}
	if len(copyCalls) == 0 {
		t.Fatal("expected copy calls for keep-local")
	}

	// Keep Remote
	sessRemote := NewSession("CONFLICT", "Review", nil, []models.ConflictCandidate{
		{
			ID:       "conf-2",
			Path:     "conflict.txt",
			Decision: models.DecisionKeepRemote,
			Resolved: true,
		},
	})
	if err := exec.ExecuteSession(context.Background(), sessRemote); err != nil {
		t.Fatalf("ExecuteSession keep-remote failed: %v", err)
	}

	// Keep Both
	sessBoth := NewSession("CONFLICT", "Review", nil, []models.ConflictCandidate{
		{
			ID:       "conf-3",
			Path:     "conflict.txt",
			Decision: models.DecisionKeepBoth,
			Resolved: true,
		},
	})
	if err := exec.ExecuteSession(context.Background(), sessBoth); err != nil {
		t.Fatalf("ExecuteSession keep-both failed: %v", err)
	}
}

func TestExecutor_RollbackSession(t *testing.T) {
	tmpDir := t.TempDir()
	localRoot := filepath.Join(tmpDir, "local")
	backupDir := filepath.Join(tmpDir, "backups")
	walPath := filepath.Join(tmpDir, "review.wal")
	_ = os.MkdirAll(localRoot, 0755)

	localFile := filepath.Join(localRoot, "revert.txt")
	_ = os.WriteFile(localFile, []byte("original content before delete"), 0644)

	mockClient := &mockRemoteClient{}
	exec := NewResolutionExecutor(localRoot, "gdrive", backupDir, walPath, mockClient)

	sess := NewSession("TOO_MANY_DELETES", "Review", []models.DeletionCandidate{
		{
			ID:       "del-rev",
			Path:     "revert.txt",
			Source:   models.SideRemote,
			Decision: models.DecisionAcceptDelete,
			Resolved: true,
		},
	}, nil)

	// Execute deletion (file will be backed up then deleted)
	if err := exec.ExecuteSession(context.Background(), sess); err != nil {
		t.Fatalf("ExecuteSession failed: %v", err)
	}
	if _, err := os.Stat(localFile); !os.IsNotExist(err) {
		t.Fatal("expected file to be deleted")
	}

	// Now Rollback!
	if err := RollbackSession(context.Background(), sess, exec); err != nil {
		t.Fatalf("RollbackSession failed: %v", err)
	}

	// Session should be back in REVIEWING status
	if sess.Status != models.ReviewStatusReviewing {
		t.Fatalf("expected status REVIEWING after rollback, got %s", sess.Status)
	}

	// Action should be rolled_back
	if sess.Actions[0].Status != "rolled_back" {
		t.Fatalf("expected action status rolled_back, got %s", sess.Actions[0].Status)
	}

	// Local file should be restored!
	restoredData, err := os.ReadFile(localFile)
	if err != nil {
		t.Fatalf("failed reading restored file: %v", err)
	}
	if string(restoredData) != "original content before delete" {
		t.Fatalf("restored content mismatch: %s", string(restoredData))
	}
}

func TestExecutor_ErrorHandling(t *testing.T) {
	exec := NewResolutionExecutor("/tmp", "gdrive", "/tmp/backups", "/tmp/wal", nil)

	// Nil session
	if err := exec.ExecuteSession(context.Background(), nil); err == nil {
		t.Fatal("expected error with nil session")
	}

	// Nil session on rollback
	if err := RollbackSession(context.Background(), nil, exec); err == nil {
		t.Fatal("expected error with nil session in rollback")
	}
	if err := RollbackSession(context.Background(), &models.ReviewSession{}, nil); err == nil {
		t.Fatal("expected error with nil executor in rollback")
	}

	// Path traversal candidate in session
	sess := NewSession("ERROR", "Traverse", []models.DeletionCandidate{
		{
			ID:       "del-bad",
			Path:     "../escape.txt",
			Source:   models.SideRemote,
			Decision: models.DecisionAcceptDelete,
			Resolved: true,
		},
	}, nil)

	if err := exec.ExecuteSession(context.Background(), sess); err == nil {
		t.Fatal("expected path traversal rejection during execution")
	}
	if sess.Status != models.ReviewStatusFailed {
		t.Fatalf("expected FAILED session status after rejection, got %s", sess.Status)
	}
}

func TestExecutor_RollbackRemoteAndRestore(t *testing.T) {
	tmpDir := t.TempDir()
	localRoot := filepath.Join(tmpDir, "local")
	backupDir := filepath.Join(tmpDir, "backups")
	_ = os.MkdirAll(localRoot, 0755)

	var restoredRemoteDst string
	var deletedRemote string
	mockClient := &mockRemoteClient{
		copyFileFunc: func(ctx context.Context, srcFs, srcRemote, dstFs, dstRemote string) (bool, error) {
			restoredRemoteDst = dstRemote
			return true, nil
		},
		deleteFileFunc: func(ctx context.Context, fs, remote string) (bool, error) {
			deletedRemote = remote
			return true, nil
		},
	}

	exec := NewResolutionExecutor(localRoot, "gdrive", backupDir, "", mockClient)

	// Rollback session with remote backup
	sessRemote := &models.ReviewSession{
		ID:     "sess-remote-rev",
		Status: models.ReviewStatusApplying,
		Actions: []models.ResolutionAction{
			{
				ID:         "act-1",
				ItemPath:   "doc.pdf",
				Type:       models.DecisionAcceptDelete,
				Status:     "completed",
				BackupPath: ".review_backups/sess-remote-rev/doc.pdf",
			},
			{
				ID:       "act-2",
				ItemPath: "new_file.txt",
				Type:     models.DecisionRestoreItem,
				Status:   "completed",
			},
		},
	}

	if err := RollbackSession(context.Background(), sessRemote, exec); err != nil {
		t.Fatalf("RollbackSession failed: %v", err)
	}

	if restoredRemoteDst != "doc.pdf" {
		t.Fatalf("expected remote file doc.pdf to be restored, got %s", restoredRemoteDst)
	}
	if deletedRemote != "new_file.txt" {
		t.Fatalf("expected restored file new_file.txt to be removed on rollback, got %s", deletedRemote)
	}
}

func TestExecutor_ClientErrors(t *testing.T) {
	tmpDir := t.TempDir()
	localRoot := filepath.Join(tmpDir, "local")
	_ = os.MkdirAll(localRoot, 0755)

	errClient := &mockRemoteClient{
		copyFileFunc: func(ctx context.Context, srcFs, srcRemote, dstFs, dstRemote string) (bool, error) {
			return false, fmt.Errorf("remote connection failed")
		},
		deleteFileFunc: func(ctx context.Context, fs, remote string) (bool, error) {
			return false, fmt.Errorf("remote delete failed")
		},
	}

	exec := NewResolutionExecutor(localRoot, "gdrive", tmpDir, "", errClient)

	// Failed delete
	sess := NewSession("ERROR", "Msg", []models.DeletionCandidate{
		{
			ID:       "del-err",
			Path:     "remote.txt",
			Source:   models.SideLocal,
			Decision: models.DecisionAcceptDelete,
			Resolved: true,
		},
	}, nil)

	if err := exec.ExecuteSession(context.Background(), sess); err == nil {
		t.Fatal("expected error on failed remote delete")
	}

	// Failed restore
	sessRestore := NewSession("ERROR", "Msg", []models.DeletionCandidate{
		{
			ID:       "rest-err",
			Path:     "remote.txt",
			Source:   models.SideLocal,
			Decision: models.DecisionRestoreItem,
			Resolved: true,
		},
	}, nil)

	if err := exec.ExecuteSession(context.Background(), sessRestore); err == nil {
		t.Fatal("expected error on failed restore copy")
	}
}

func TestExecutor_HTTP401UnauthorizedAndRetry(t *testing.T) {
	tmpDir := t.TempDir()
	localRoot := filepath.Join(tmpDir, "local")
	walPath := filepath.Join(tmpDir, "review.wal")
	_ = os.MkdirAll(localRoot, 0755)

	unauthClient := &mockRemoteClient{
		copyFileFunc: func(ctx context.Context, srcFs, srcRemote, dstFs, dstRemote string) (bool, error) {
			return false, fmt.Errorf("copyfile failed (HTTP 401): 401 Unauthorized")
		},
	}

	exec := NewResolutionExecutor(localRoot, "gdrive", tmpDir, walPath, unauthClient)

	sess := NewSession("TOO_MANY_DELETES", "Auth fail test", []models.DeletionCandidate{
		{
			ID:       "del-401",
			Path:     "document.pdf",
			Source:   models.SideLocal,
			Decision: models.DecisionRestoreItem,
			Resolved: true,
		},
	}, nil)

	// Execute should fail with 401
	err := exec.ExecuteSession(context.Background(), sess)
	if err == nil {
		t.Fatal("expected 401 error, got nil")
	}
	if sess.Status != models.ReviewStatusFailed {
		t.Fatalf("expected session status FAILED, got %s", sess.Status)
	}

	// Staging on FAILED session should transition it back to REVIEWING
	if err := StageDecision(sess, "del-401", models.DecisionSkip); err != nil {
		t.Fatalf("failed to stage decision on FAILED session: %v", err)
	}
	if sess.Status != models.ReviewStatusReviewing {
		t.Fatalf("expected session status REVIEWING after re-staging, got %s", sess.Status)
	}

	// Retrying execution with skip should now succeed
	if err := exec.ExecuteSession(context.Background(), sess); err != nil {
		t.Fatalf("expected retry with skip to succeed, got: %v", err)
	}
	if sess.Status != models.ReviewStatusReady {
		t.Fatalf("expected session status READY, got %s", sess.Status)
	}
}

func TestExecutor_SkipCandidates(t *testing.T) {
	tmpDir := t.TempDir()
	localRoot := filepath.Join(tmpDir, "local")
	_ = os.MkdirAll(localRoot, 0755)

	called := false
	mockClient := &mockRemoteClient{
		copyFileFunc: func(ctx context.Context, srcFs, srcRemote, dstFs, dstRemote string) (bool, error) {
			called = true
			return true, nil
		},
		deleteFileFunc: func(ctx context.Context, fs, remote string) (bool, error) {
			called = true
			return true, nil
		},
	}

	exec := NewResolutionExecutor(localRoot, "gdrive", tmpDir, "", mockClient)

	sess := NewSession("MIXED", "Skip test", []models.DeletionCandidate{
		{
			ID:       "del-skip",
			Path:     "file.txt",
			Source:   models.SideRemote,
			Decision: models.DecisionSkip,
			Resolved: true,
		},
	}, []models.ConflictCandidate{
		{
			ID:       "conf-skip",
			Path:     "conf.txt",
			Decision: models.DecisionSkip,
			Resolved: true,
		},
	})

	if err := exec.ExecuteSession(context.Background(), sess); err != nil {
		t.Fatalf("ExecuteSession failed with skipped items: %v", err)
	}

	if sess.Status != models.ReviewStatusReady {
		t.Fatalf("expected READY status, got %s", sess.Status)
	}
	if called {
		t.Fatal("expected no client mutations for skipped items")
	}
}

func TestExecutor_TrailingSpaceDirectoryAndResumeCompleted(t *testing.T) {
	tmpDir := t.TempDir()
	localRoot := filepath.Join(tmpDir, "local")
	backupDir := filepath.Join(tmpDir, "backups")
	walPath := filepath.Join(tmpDir, "review.wal")

	trailingSpaceDir := filepath.Join(localRoot, "MOHU Szemétszállítás ")
	_ = os.MkdirAll(trailingSpaceDir, 0755)
	relPath := "MOHU Szemétszállítás /MOHU_2024-02-13_305375343.pdf"
	_ = os.WriteFile(filepath.Join(localRoot, relPath), []byte("pdf data"), 0644)

	var copiedItems []string
	var capturedSrcFs string
	mockClient := &mockRemoteClient{
		copyFileFunc: func(ctx context.Context, srcFs, srcRemote, dstFs, dstRemote string) (bool, error) {
			capturedSrcFs = srcFs
			copiedItems = append(copiedItems, srcRemote)
			return true, nil
		},
	}

	exec := NewResolutionExecutor(localRoot, "gdrive", backupDir, walPath, mockClient)
	sess := NewSession("TOO_MANY_DELETES", "Review", []models.DeletionCandidate{
		{
			ID:       "del-already-done",
			Path:     "already_done.pdf",
			Source:   models.SideRemote,
			Decision: models.DecisionRestoreItem,
			Resolved: true,
		},
		{
			ID:       "del-trailing-space",
			Path:     relPath,
			Source:   models.SideRemote,
			Decision: models.DecisionRestoreItem,
			Resolved: true,
		},
	}, nil)
	sess.Actions = []models.ResolutionAction{
		{
			ID:       "act-del-1",
			Type:     models.DecisionRestoreItem,
			ItemPath: "already_done.pdf",
			Status:   "completed",
		},
	}

	if err := exec.ExecuteSession(context.Background(), sess); err != nil {
		t.Fatalf("ExecuteSession failed: %v", err)
	}

	if len(copiedItems) != 1 || copiedItems[0] != relPath {
		t.Fatalf("expected only %q to be copied on resume, got %v", relPath, copiedItems)
	}
	if capturedSrcFs != filepath.Clean(localRoot) {
		t.Fatalf("expected srcFs to be localRoot %q, got %q", filepath.Clean(localRoot), capturedSrcFs)
	}
	if len(sess.Actions) != 2 {
		t.Fatalf("expected 2 total completed actions, got %d", len(sess.Actions))
	}
}

func TestExecutor_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	localRoot := filepath.Join(tmpDir, "local")
	backupDir := filepath.Join(tmpDir, "backups")
	walPath := filepath.Join(tmpDir, "review.wal")

	// 1. Directory restore to remote (source: remote -> local dir uploaded via Mkdir)
	localDir := filepath.Join(localRoot, "Alfi", "SubDir")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("failed to create local dir: %v", err)
	}

	var remoteMkdirs []string
	var remotePurges []string
	var copyFileCalled bool
	var deleteFileCalled bool

	mockClient := &mockRemoteClient{
		copyFileFunc: func(ctx context.Context, srcFs, srcRemote, dstFs, dstRemote string) (bool, error) {
			copyFileCalled = true
			return true, nil
		},
		deleteFileFunc: func(ctx context.Context, fs, remote string) (bool, error) {
			deleteFileCalled = true
			return true, nil
		},
		mkdirFunc: func(ctx context.Context, fs, remote string) (bool, error) {
			remoteMkdirs = append(remoteMkdirs, remote)
			return true, nil
		},
		purgeDirFunc: func(ctx context.Context, fs, remote string) (bool, error) {
			remotePurges = append(remotePurges, remote)
			return true, nil
		},
	}

	exec := NewResolutionExecutor(localRoot, "gdrive", backupDir, walPath, mockClient)

	sess := NewSession("TOO_MANY_DELETES", "Review", []models.DeletionCandidate{
		{
			ID:       "del-dir-remote-restore",
			Path:     "Alfi",
			IsDir:    true,
			Source:   models.SideRemote,
			Decision: models.DecisionRestoreItem,
			Resolved: true,
		},
		{
			ID:       "del-dir-local-restore",
			Path:     "RemoteOnlyDir",
			IsDir:    true,
			Source:   models.SideLocal,
			Decision: models.DecisionRestoreItem,
			Resolved: true,
		},
		{
			ID:       "del-dir-remote-delete",
			Path:     "ToDeleteLocalDir",
			IsDir:    true,
			Source:   models.SideRemote,
			Decision: models.DecisionAcceptDelete,
			Resolved: true,
		},
		{
			ID:       "del-dir-local-delete",
			Path:     "ToDeleteRemoteDir",
			IsDir:    true,
			Source:   models.SideLocal,
			Decision: models.DecisionAcceptDelete,
			Resolved: true,
		},
	}, nil)

	// Create ToDeleteLocalDir with a nested file
	toDeleteDir := filepath.Join(localRoot, "ToDeleteLocalDir")
	_ = os.MkdirAll(toDeleteDir, 0755)
	_ = os.WriteFile(filepath.Join(toDeleteDir, "nested.txt"), []byte("nested"), 0644)

	if err := exec.ExecuteSession(context.Background(), sess); err != nil {
		t.Fatalf("ExecuteSession failed on directory candidates: %v", err)
	}

	// Assertions:
	// 1. CopyFile and DeleteFile should NOT be called for directory operations
	if copyFileCalled {
		t.Errorf("expected CopyFile NOT to be called on directory candidates")
	}
	if deleteFileCalled {
		t.Errorf("expected DeleteFile NOT to be called on directory candidates")
	}

	// 2. Mkdir called for remote restore
	if len(remoteMkdirs) != 1 || remoteMkdirs[0] != "Alfi" {
		t.Errorf("expected Mkdir to be called for 'Alfi', got %v", remoteMkdirs)
	}

	// 3. Local dir recreated for local restore
	remoteOnlyLocal := filepath.Join(localRoot, "RemoteOnlyDir")
	if fi, err := os.Stat(remoteOnlyLocal); err != nil || !fi.IsDir() {
		t.Errorf("expected local directory 'RemoteOnlyDir' to be created, stat error: %v", err)
	}

	// 4. Local dir removed on accept delete
	if _, err := os.Stat(toDeleteDir); !os.IsNotExist(err) {
		t.Errorf("expected 'ToDeleteLocalDir' to be removed from local disk")
	}

	// 5. PurgeDir called for remote deletion
	if len(remotePurges) != 1 || remotePurges[0] != "ToDeleteRemoteDir" {
		t.Errorf("expected PurgeDir to be called for 'ToDeleteRemoteDir', got %v", remotePurges)
	}
}

func TestExecutor_QueueTopologicalOrderAndErrorIsolation(t *testing.T) {
	tmpDir := t.TempDir()
	localRoot := filepath.Join(tmpDir, "local")
	_ = os.MkdirAll(localRoot, 0755)

	var ops []string
	client := &mockRemoteClient{
		mkdirFunc: func(ctx context.Context, fs, dir string) (bool, error) {
			ops = append(ops, "mkdir:"+dir)
			return true, nil
		},
		copyFileFunc: func(ctx context.Context, srcFs, srcRemote, dstFs, dstRemote string) (bool, error) {
			ops = append(ops, "copy:"+srcRemote)
			return true, nil
		},
	}

	exec := NewResolutionExecutor(localRoot, "gdrive", tmpDir, "", client)
	exec.QueueConfig = &QueueConfig{
		MaxRetries:     1,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     2 * time.Millisecond,
	}

	// Staged in inverted order: nested file before parent directories
	sess := NewSession("TOO_MANY_DELETES", "Topological test", []models.DeletionCandidate{
		{
			ID:       "file",
			Path:     "root/sub/nested.txt",
			IsDir:    false,
			Source:   models.SideRemote,
			Decision: models.DecisionRestoreItem,
			Resolved: true,
		},
		{
			ID:       "root",
			Path:     "root",
			IsDir:    true,
			Source:   models.SideRemote,
			Decision: models.DecisionRestoreItem,
			Resolved: true,
		},
		{
			ID:       "sub",
			Path:     "root/sub",
			IsDir:    true,
			Source:   models.SideRemote,
			Decision: models.DecisionRestoreItem,
			Resolved: true,
		},
	}, nil)

	// Create local nested file so copy to remote succeeds
	_ = os.MkdirAll(filepath.Join(localRoot, "root/sub"), 0755)
	_ = os.WriteFile(filepath.Join(localRoot, "root/sub/nested.txt"), []byte("nested content"), 0644)

	if err := exec.ExecuteSession(context.Background(), sess); err != nil {
		t.Fatalf("ExecuteSession failed: %v", err)
	}

	// Verify topological ordering: root mkdir -> sub mkdir -> nested file copy
	expectedOps := []string{"mkdir:root", "mkdir:root/sub", "copy:root/sub/nested.txt"}
	if len(ops) != len(expectedOps) {
		t.Fatalf("expected ops %v, got %v", expectedOps, ops)
	}
	for i, exp := range expectedOps {
		if ops[i] != exp {
			t.Errorf("at position %d: expected %s, got %s", i, exp, ops[i])
		}
	}
}
