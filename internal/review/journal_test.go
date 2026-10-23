package review

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

func TestJournal_SaveAndLoadActiveSession(t *testing.T) {
	tmpDir := t.TempDir()
	activeFile := filepath.Join(tmpDir, "active.json")

	// Initially missing
	loaded, err := LoadActiveSession(activeFile)
	if err != nil || loaded != nil {
		t.Fatalf("expected nil when missing, got: %v, err: %v", loaded, err)
	}

	// Create and save
	sess := NewSession("TOO_MANY_DELETES", "Need review", []models.DeletionCandidate{
		{
			ID:       "del-1",
			Path:     "test.txt",
			Resolved: false,
		},
	}, nil)

	if err := SaveActiveSession(activeFile, sess); err != nil {
		t.Fatalf("SaveActiveSession failed: %v", err)
	}

	loaded, err = LoadActiveSession(activeFile)
	if err != nil {
		t.Fatalf("LoadActiveSession failed: %v", err)
	}
	if loaded == nil || loaded.ID != sess.ID {
		t.Fatalf("loaded session does not match original: %+v", loaded)
	}
	if len(loaded.Deletions) != 1 {
		t.Fatalf("expected 1 deletion candidate, got %d", len(loaded.Deletions))
	}
}

func TestJournal_ArchiveSession(t *testing.T) {
	tmpDir := t.TempDir()
	reviewDir := filepath.Join(tmpDir, "archive")
	activeFile := filepath.Join(tmpDir, "active.json")

	sess := NewSession("ERROR", "Archived session", nil, nil)
	sess.Status = models.ReviewStatusComplete
	if err := SaveActiveSession(activeFile, sess); err != nil {
		t.Fatalf("SaveActiveSession failed: %v", err)
	}

	if err := ArchiveSession(reviewDir, activeFile, sess); err != nil {
		t.Fatalf("ArchiveSession failed: %v", err)
	}

	// Active file should no longer exist
	if _, err := os.Stat(activeFile); !os.IsNotExist(err) {
		t.Fatalf("expected activeFile to be removed after archive")
	}

	// Archived file should exist
	archivedPath := filepath.Join(reviewDir, sess.ID+".json")
	if _, err := os.Stat(archivedPath); err != nil {
		t.Fatalf("expected archive file %s to exist: %v", archivedPath, err)
	}
}

func TestJournal_WALAppendAndRead(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "session.wal")

	// Empty read on missing
	entries, err := ReadWAL(walPath)
	if err != nil || len(entries) != 0 {
		t.Fatalf("expected empty entries on missing WAL, got %v, err: %v", entries, err)
	}

	e1 := models.WALEntry{
		SessionID: "sess-1",
		ActionID:  "act-1",
		Step:      "MUTATION_STARTED",
		ItemPath:  "doc.txt",
		Type:      models.DecisionAcceptDelete,
	}
	e2 := models.WALEntry{
		SessionID: "sess-1",
		ActionID:  "act-1",
		Step:      "MUTATION_COMPLETED",
		ItemPath:  "doc.txt",
		Type:      models.DecisionAcceptDelete,
	}

	if err := AppendWAL(walPath, e1); err != nil {
		t.Fatalf("AppendWAL failed: %v", err)
	}
	if err := AppendWAL(walPath, e2); err != nil {
		t.Fatalf("AppendWAL failed: %v", err)
	}

	entries, err = ReadWAL(walPath)
	if err != nil {
		t.Fatalf("ReadWAL failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Step != "MUTATION_STARTED" || entries[1].Step != "MUTATION_COMPLETED" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}

func TestJournal_ReconcileSessionWithWAL(t *testing.T) {
	now := time.Now().UTC()
	sess := &models.ReviewSession{
		ID:     "sess-crash",
		Status: models.ReviewStatusApplying,
		Actions: []models.ResolutionAction{
			{
				ID:       "act-1",
				Status:   "pending",
				ItemPath: "done.txt",
			},
			{
				ID:       "act-2",
				Status:   "pending",
				ItemPath: "interrupted.txt",
			},
		},
	}

	walEntries := []models.WALEntry{
		{
			ActionID:  "act-1",
			Step:      "MUTATION_COMPLETED",
			Timestamp: now.Format(time.RFC3339),
		},
		{
			ActionID:  "act-2",
			Step:      "MUTATION_STARTED",
			Timestamp: now.Format(time.RFC3339),
		},
	}

	reconciled := ReconcileSessionWithWAL(sess, walEntries)
	if reconciled.Status != models.ReviewStatusFailed {
		t.Fatalf("expected FAILED status due to interrupted action, got %s", reconciled.Status)
	}
	if reconciled.Actions[0].Status != "completed" {
		t.Fatalf("expected act-1 completed, got %s", reconciled.Actions[0].Status)
	}
	if reconciled.Actions[1].Status != "interrupted" {
		t.Fatalf("expected act-2 interrupted, got %s", reconciled.Actions[1].Status)
	}

	// Test full completion scenario
	sessAllDone := &models.ReviewSession{
		ID:     "sess-done",
		Status: models.ReviewStatusApplying,
		Actions: []models.ResolutionAction{
			{ID: "act-1", Status: "pending"},
		},
	}
	walDone := []models.WALEntry{
		{ActionID: "act-1", Step: "MUTATION_COMPLETED"},
	}
	reconciledDone := ReconcileSessionWithWAL(sessAllDone, walDone)
	if reconciledDone.Status != models.ReviewStatusReady {
		t.Fatalf("expected READY status when all actions completed, got %s", reconciledDone.Status)
	}
	if reconciledDone.Actions[0].Status != "completed" {
		t.Fatalf("expected act-1 completed, got %s", reconciledDone.Actions[0].Status)
	}
}


func TestJournal_EdgeCases(t *testing.T) {
	// Empty path handling
	if err := SaveActiveSession("", &models.ReviewSession{}); err == nil {
		t.Fatal("expected error with empty activeFilePath")
	}
	if err := SaveActiveSession("/tmp/foo", nil); err == nil {
		t.Fatal("expected error with nil session")
	}

	loaded, err := LoadActiveSession("")
	if err != nil || loaded != nil {
		t.Fatalf("expected nil for empty LoadActiveSession path")
	}

	if err := ArchiveSession("", "", nil); err == nil {
		t.Fatal("expected error with nil session in ArchiveSession")
	}
	if err := ArchiveSession("", "", &models.ReviewSession{}); err == nil {
		t.Fatal("expected error with empty reviewDir in ArchiveSession")
	}

	if err := AppendWAL("", models.WALEntry{}); err == nil {
		t.Fatal("expected error with empty walPath")
	}

	entries, err := ReadWAL("")
	if err != nil || len(entries) != 0 {
		t.Fatal("expected empty entries with empty walPath")
	}

	if res := ReconcileSessionWithWAL(nil, nil); res != nil {
		t.Fatal("expected nil for nil session in ReconcileSessionWithWAL")
	}

	sess := &models.ReviewSession{}
	if res := ReconcileSessionWithWAL(sess, nil); res != sess {
		t.Fatal("expected same session for empty WAL")
	}

	// Corrupt JSON in active file
	tmpDir := t.TempDir()
	badFile := filepath.Join(tmpDir, "bad.json")
	_ = os.WriteFile(badFile, []byte("{corrupt"), 0644)
	if _, err := LoadActiveSession(badFile); err == nil {
		t.Fatal("expected error loading corrupt JSON")
	}

	// Corrupt line in WAL
	badWAL := filepath.Join(tmpDir, "bad.wal")
	_ = os.WriteFile(badWAL, []byte("not-json\n"), 0644)
	if _, err := ReadWAL(badWAL); err == nil {
		t.Fatal("expected error reading corrupt WAL")
	}

	// Reconcile with action error step
	actionSess := &models.ReviewSession{
		Status: models.ReviewStatusApplying,
		Actions: []models.ResolutionAction{
			{ID: "act-err", Status: "pending"},
		},
	}
	errWAL := []models.WALEntry{
		{ActionID: "act-err", Step: "MUTATION_FAILED", Error: "permission denied"},
	}
	reconciled := ReconcileSessionWithWAL(actionSess, errWAL)
	if reconciled.Status != models.ReviewStatusFailed || reconciled.Actions[0].Status != "failed" {
		t.Fatalf("expected failed action, got status %s", reconciled.Actions[0].Status)
	}
}
