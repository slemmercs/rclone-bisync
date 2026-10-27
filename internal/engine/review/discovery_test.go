package review

import (
	"os"
	"path/filepath"
	"strings"
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
func TestDiscoverCandidates_FilenamesWithSpaces(t *testing.T) {
	logLines := []string{
		`{"time":"2026-10-24T22:16:00Z","level":"info","msg":"- Path2             File was deleted                            - Csaba/etc/Daimler Corporate Color Palette.xlsx"}`,
		`{"time":"2026-10-24T22:16:00Z","level":"info","msg":"- Path2             File was deleted                            - Csaba/etc/Lime Aioli.docx"}`,
		`{"time":"2026-10-24T22:16:00Z","level":"error","msg":"too many deletes (>0%, 4 of 7061) on Path2 \"/home/csaba/gdrive/\". Run with --force if desired."}`,
	}

	deletions, _, err := DiscoverCandidates(logLines, "", "")
	if err != nil {
		t.Fatalf("DiscoverCandidates failed: %v", err)
	}

	if len(deletions) != 2 {
		t.Fatalf("expected 2 deletions, got %d", len(deletions))
	}
	expected0 := "Csaba/etc/Daimler Corporate Color Palette.xlsx"
	expected1 := "Csaba/etc/Lime Aioli.docx"
	if deletions[0].Path != expected0 && deletions[1].Path != expected0 {
		t.Fatalf("expected deletion with spaces %q, got: %v", expected0, deletions)
	}
	if deletions[0].Path != expected1 && deletions[1].Path != expected1 {
		t.Fatalf("expected deletion with spaces %q, got: %v", expected1, deletions)
	}
}
func TestDiscoverCandidates_NestedDaemonJSON(t *testing.T) {
	daemonLogLine := `{"time":"2026-10-24T22:50:23.841+02:00","level":"notice","msg":"{\"time\":\"2026-10-24T22:50:23.509+02:00\",\"level\":\"info\",\"msg\":\"- Path1             File was deleted                            - Csaba/etc/Ausgleichberechnung.xlsx\",\"source\":\"bisync/deltas.go:188\"}\n{\"time\":\"2026-10-24T22:50:23.530+02:00\",\"level\":\"info\",\"msg\":\"- Path2             File was deleted                            - Csaba/etc/Ossobuco mit Gremolata  Le Creuset DE.pdf\",\"source\":\"bisync/deltas.go:188\"}\n{\"time\":\"2026-10-24T22:50:23.531+02:00\",\"level\":\"error\",\"msg\":\"too many deletes (>0%, 1 of 7058) on Path1 \\\"gdrive{tCwNP}:/\\\". Run with --force if desired.\",\"source\":\"bisync/deltas.go:557\"}"}`

	deletions, _, err := DiscoverCandidates([]string{daemonLogLine}, "", "")
	if err != nil {
		t.Fatalf("DiscoverCandidates failed: %v", err)
	}

	if len(deletions) != 2 {
		t.Fatalf("expected 2 deletions parsed from nested JSON, got %d: %v", len(deletions), deletions)
	}

	var foundPath1, foundPath2 bool
	for _, d := range deletions {
		if strings.Contains(d.Path, "source") || strings.Contains(d.Path, `"`) || strings.Contains(d.Path, `}`) {
			t.Fatalf("corrupted deletion path detected: %q", d.Path)
		}
		if d.Path == "Csaba/etc/Ausgleichberechnung.xlsx" {
			foundPath1 = true
			if d.Source != models.SideRemote {
				t.Errorf("expected SideRemote for Path1 deletion, got: %s", d.Source)
			}
		}
		if d.Path == "Csaba/etc/Ossobuco mit Gremolata  Le Creuset DE.pdf" {
			foundPath2 = true
			if d.Source != models.SideLocal {
				t.Errorf("expected SideLocal for Path2 deletion, got: %s", d.Source)
			}
		}
	}
	if !foundPath1 {
		t.Errorf("failed to discover Ausgleichberechnung.xlsx")
	}
	if !foundPath2 {
		t.Errorf("failed to discover Ossobuco mit Gremolata  Le Creuset DE.pdf")
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
