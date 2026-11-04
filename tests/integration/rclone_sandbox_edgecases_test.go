package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/engine/review"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

// TestRcloneSandbox_TrailingSpaceDirectoryReconciliation verifies that directories
// created on disk with trailing whitespace (PR #68) are properly reconciled during review
// execution without failing or throwing unhandled filesystem errors against real Rclone.
func TestRcloneSandbox_TrailingSpaceDirectoryReconciliation(t *testing.T) {
	rcClient, sandboxDir := startRcloneSandbox(t)
	ctx := context.Background()

	localDir := filepath.Join(sandboxDir, "local_trail")
	remoteDir := filepath.Join(sandboxDir, "remote_trail")
	backupDir := filepath.Join(sandboxDir, "backups")
	walPath := filepath.Join(sandboxDir, "review.wal")

	// Create local directory with trailing whitespace in its leaf name
	dirWithTrailingSpace := filepath.Join(localDir, "Alpha ")
	if err := os.MkdirAll(dirWithTrailingSpace, 0750); err != nil {
		t.Fatalf("failed to create directory with trailing space: %v", err)
	}
	sampleFile := filepath.Join(dirWithTrailingSpace, "doc.txt")
	if err := os.WriteFile(sampleFile, []byte("trailing-space-content"), 0600); err != nil {
		t.Fatalf("failed to write test file in trailing-space dir: %v", err)
	}

	executor := review.NewResolutionExecutor(localDir, remoteDir, backupDir, walPath, rcClient)

	// Candidate discovered with trimmed whitespace: "Alpha"
	session := &models.ReviewSession{
		ID:        "sess_trail_1",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Status:    models.ReviewStatusCreated,
		Deletions: []models.DeletionCandidate{
			{
				ID:        "cand_trail_1",
				Path:      "Alpha", // trimmed version
				Name:      "Alpha",
				IsDir:     true,
				Source:    models.SideRemote,
				Decision:  models.DecisionAcceptDelete,
				Resolved:  true,
				Condition: models.ConditionTooManyDeletes,
			},
		},
	}

	if err := executor.ExecuteSession(ctx, session); err != nil {
		t.Fatalf("ExecuteSession failed on trailing-space directory reconciliation: %v", err)
	}

	// Verify local directory was deleted
	if _, err := os.Stat(dirWithTrailingSpace); !os.IsNotExist(err) {
		t.Errorf("expected trailing-space directory to be deleted, stat err: %v", err)
	}

	// Verify session completed
	if session.Status != models.ReviewStatusReady {
		t.Errorf("expected session status READY, got %s", session.Status)
	}
}

// TestRcloneSandbox_Idempotent404Handling verifies that when remote files/directories have already
// been deleted or are absent during resolution (PR #68), the real Rclone RC error is safely caught
// and treated as idempotent success rather than aborting batch review execution.
func TestRcloneSandbox_Idempotent404Handling(t *testing.T) {
	rcClient, sandboxDir := startRcloneSandbox(t)
	ctx := context.Background()

	localDir := filepath.Join(sandboxDir, "local_404")
	remoteDir := filepath.Join(sandboxDir, "remote_404")
	backupDir := filepath.Join(sandboxDir, "backups_404")
	walPath := filepath.Join(sandboxDir, "review_404.wal")

	if err := os.MkdirAll(localDir, 0750); err != nil {
		t.Fatalf("failed to create localDir: %v", err)
	}
	if err := os.MkdirAll(remoteDir, 0750); err != nil {
		t.Fatalf("failed to create remoteDir: %v", err)
	}

	executor := review.NewResolutionExecutor(localDir, remoteDir, backupDir, walPath, rcClient)

	// Both candidates target items that DO NOT exist on the remote side
	session := &models.ReviewSession{
		ID:        "sess_404_1",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Status:    models.ReviewStatusCreated,
		Deletions: []models.DeletionCandidate{
			{
				ID:        "cand_absent_file",
				Path:      "nonexistent_file.txt",
				Name:      "nonexistent_file.txt",
				IsDir:     false,
				Source:    models.SideLocal, // deletion happened locally -> delete on remote
				Decision:  models.DecisionAcceptDelete,
				Resolved:  true,
				Condition: models.ConditionTooManyDeletes,
			},
			{
				ID:        "cand_absent_dir",
				Path:      "nonexistent_directory",
				Name:      "nonexistent_directory",
				IsDir:     true,
				Source:    models.SideLocal, // deletion happened locally -> purge dir on remote
				Decision:  models.DecisionAcceptDelete,
				Resolved:  true,
				Condition: models.ConditionTooManyDeletes,
			},
		},
	}

	// Should not return error even though upstream Rclone RC returns 404/not found
	if err := executor.ExecuteSession(ctx, session); err != nil {
		t.Fatalf("ExecuteSession aborted on 404 not-found item: %v", err)
	}

	if session.Status != models.ReviewStatusReady {
		t.Errorf("expected session status READY, got %s", session.Status)
	}
}

// TestRcloneSandbox_MixedDirectoryAndFileReviewExecution exercises real Rclone RC
// operations ensuring that directory candidates use Mkdir / PurgeDir while file candidates
// use CopyFile / DeleteFile, completely preventing HTTP 500 errors from Rclone.
func TestRcloneSandbox_MixedDirectoryAndFileReviewExecution(t *testing.T) {
	rcClient, sandboxDir := startRcloneSandbox(t)
	ctx := context.Background()

	localDir := filepath.Join(sandboxDir, "local_mixed")
	remoteDir := filepath.Join(sandboxDir, "remote_mixed")
	backupDir := filepath.Join(sandboxDir, "backups_mixed")
	walPath := filepath.Join(sandboxDir, "review_mixed.wal")

	// Seed remote side with a subfolder and a nested file
	remoteSubDir := filepath.Join(remoteDir, "archive")
	if err := os.MkdirAll(remoteSubDir, 0750); err != nil {
		t.Fatalf("failed to create remote subfolder: %v", err)
	}
	remoteFilePath := filepath.Join(remoteSubDir, "document.pdf")
	if err := os.WriteFile(remoteFilePath, []byte("pdf-binary-payload"), 0600); err != nil {
		t.Fatalf("failed to write remote nested file: %v", err)
	}

	// Local side is empty
	if err := os.MkdirAll(localDir, 0750); err != nil {
		t.Fatalf("failed to create localDir: %v", err)
	}

	executor := review.NewResolutionExecutor(localDir, remoteDir, backupDir, walPath, rcClient)

	// Session restores both the directory and the nested file from remote to local
	session := &models.ReviewSession{
		ID:        "sess_mixed_1",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Status:    models.ReviewStatusCreated,
		Deletions: []models.DeletionCandidate{
			{
				ID:        "cand_restore_dir",
				Path:      "archive",
				Name:      "archive",
				IsDir:     true,
				Source:    models.SideLocal, // local was deleted -> restore from remote
				Decision:  models.DecisionRestoreItem,
				Resolved:  true,
				Condition: models.ConditionTooManyDeletes,
			},
			{
				ID:        "cand_restore_file",
				Path:      "archive/document.pdf",
				Name:      "document.pdf",
				IsDir:     false,
				Source:    models.SideLocal, // local was deleted -> restore from remote
				Decision:  models.DecisionRestoreItem,
				Resolved:  true,
				Condition: models.ConditionTooManyDeletes,
			},
		},
	}

	if err := executor.ExecuteSession(ctx, session); err != nil {
		t.Fatalf("ExecuteSession failed on mixed directory/file restoration: %v", err)
	}

	// Verify local directory was created
	localSubDir := filepath.Join(localDir, "archive")
	if fi, err := os.Stat(localSubDir); err != nil || !fi.IsDir() {
		t.Fatalf("expected local directory 'archive' to exist, stat err: %v", err)
	}

	// Verify local file was copied from remote via real Rclone RC CopyFile
	localFile := filepath.Join(localSubDir, "document.pdf")
	content, err := os.ReadFile(localFile)
	if err != nil || string(content) != "pdf-binary-payload" {
		t.Fatalf("restored file mismatch: got %q, err: %v", string(content), err)
	}

	if session.Status != models.ReviewStatusReady {
		t.Errorf("expected session status READY, got %s", session.Status)
	}
}
