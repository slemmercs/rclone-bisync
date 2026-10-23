package review

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/core/state"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

func TestDiscoverCandidates_StructuredPendingFile(t *testing.T) {
	tmpDir := t.TempDir()
	pendingFile := filepath.Join(tmpDir, "pending.json")
	localRoot := filepath.Join(tmpDir, "local")
	_ = os.MkdirAll(localRoot, 0755)

	// Create local file to check metadata population
	localFile := filepath.Join(localRoot, "report.pdf")
	_ = os.WriteFile(localFile, []byte("report-data-12345"), 0644)

	pendingData := state.PendingDeletionsData{
		HasDeletions: true,
		TotalCount:   2,
		Source:       "Google Drive (remote)",
		TargetAction: "Will be deleted from local disk",
		Files:        []string{"report.pdf", "archive/data.zip"},
	}
	if err := state.RecordPendingDeletions(pendingFile, pendingData); err != nil {
		t.Fatalf("failed to record pending deletions: %v", err)
	}

	deletions, conflicts, err := DiscoverCandidates(nil, pendingFile, localRoot)
	if err != nil {
		t.Fatalf("DiscoverCandidates failed: %v", err)
	}

	if len(deletions) != 2 {
		t.Fatalf("expected 2 deletion candidates, got %d", len(deletions))
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected 0 conflict candidates, got %d", len(conflicts))
	}

	// Verify report.pdf has local metadata populated
	var reportCand *models.DeletionCandidate
	for i := range deletions {
		if deletions[i].Path == "report.pdf" {
			reportCand = &deletions[i]
			break
		}
	}
	if reportCand == nil {
		t.Fatal("expected report.pdf candidate")
	}
	if reportCand.Size != 17 {
		t.Fatalf("expected size 17, got %d", reportCand.Size)
	}
	if reportCand.Source != models.SideRemote {
		t.Fatalf("expected remote source, got %s", reportCand.Source)
	}
}

func TestDiscoverCandidates_LogLines(t *testing.T) {
	tmpDir := t.TempDir()
	localRoot := filepath.Join(tmpDir, "local")
	_ = os.MkdirAll(localRoot, 0755)

	logLines := []string{
		`{"time":"2026-10-23T10:00:00Z","level":"error","msg":"Fatal bisync error: 50% Path2 deletions exceed safety threshold: too many deletes"}`,
		`{"time":"2026-10-23T10:00:00Z","level":"info","msg":"- File was deleted: doc1.docx"}`,
		`{"time":"2026-10-23T10:00:00Z","level":"info","msg":"- File was deleted: sub/doc2.xlsx"}`,
		`{"time":"2026-10-23T10:00:00Z","level":"error","msg":"Conflict: config.yaml differs between Path1 and Path2"}`,
	}

	deletions, conflicts, err := DiscoverCandidates(logLines, "", localRoot)
	if err != nil {
		t.Fatalf("DiscoverCandidates failed: %v", err)
	}

	if len(deletions) != 2 {
		t.Fatalf("expected 2 deletions from logs, got %d", len(deletions))
	}
	// Path2 indicates local deletions targeting remote
	if deletions[0].Source != models.SideLocal {
		t.Fatalf("expected local source from Path2 log, got %s", deletions[0].Source)
	}

	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict from logs, got %d", len(conflicts))
	}
	if conflicts[0].Path != "config.yaml" {
		t.Fatalf("expected config.yaml, got %s", conflicts[0].Path)
	}
}

func TestDiscoverCandidates_FilesystemConflicts(t *testing.T) {
	tmpDir := t.TempDir()
	localRoot := filepath.Join(tmpDir, "local")
	_ = os.MkdirAll(filepath.Join(localRoot, "sub"), 0755)

	conflictFile := filepath.Join(localRoot, "sub", "project.conflict-1.txt")
	_ = os.WriteFile(conflictFile, []byte("conflict content"), 0644)

	deletions, conflicts, err := DiscoverCandidates(nil, "", localRoot)
	if err != nil {
		t.Fatalf("DiscoverCandidates failed: %v", err)
	}

	if len(deletions) != 0 {
		t.Fatalf("expected 0 deletions, got %d", len(deletions))
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict found on filesystem, got %d", len(conflicts))
	}

	expectedPath := filepath.Join("sub", "project.txt")
	if conflicts[0].Path != expectedPath {
		t.Fatalf("expected normalized conflict path %s, got %s", expectedPath, conflicts[0].Path)
	}
}

func TestDiscoverCandidates_LogVariants(t *testing.T) {
	logLines := []string{
		"", // empty line
		`{"msg":"random non-json info message"}`,
		`conflict on "document.docx" between path1 and path2`,
		`nested line with {"msg":"File was deleted: file1.txt\nFile was deleted: file2.txt"}`,
	}

	deletions, conflicts, err := DiscoverCandidates(logLines, "", "")
	if err != nil {
		t.Fatalf("DiscoverCandidates failed: %v", err)
	}

	if len(conflicts) != 1 || conflicts[0].Path != "document.docx" {
		t.Fatalf("expected conflict on document.docx, got %+v", conflicts)
	}
	if len(deletions) != 2 {
		t.Fatalf("expected 2 deletions from nested log, got %d", len(deletions))
	}
}
