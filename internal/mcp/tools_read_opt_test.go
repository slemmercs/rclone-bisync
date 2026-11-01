package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slemmercs/rclone-bisync/internal/app"
	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

func TestReadTools_TokenOptimizations(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")
	logContent := `{"level":"info","msg":"sync started"}
{"level":"error","msg":"sync failed on remote"}
{"level":"info","msg":"resuming sync"}`
	_ = os.WriteFile(logFile, []byte(logContent), 0644)

	cfg := &config.Config{LocalPath: tempDir, LogFile: logFile}
	auditPath := filepath.Join(tempDir, "audit.db")
	db, err := audit.OpenDB(auditPath)
	if err != nil {
		t.Fatalf("failed to open audit db: %v", err)
	}
	defer db.Close()
	auditSvc := services.NewAuditService(db, nil)
	_, _ = auditSvc.RecordEvent("prof-opt", audit.ActionSyncExecuted, "mcp", map[string]any{"files": 5})

	ctr := &app.Container{
		Config:           cfg,
		TelemetryService: services.NewTelemetryService(cfg, nil),
		ExplorerService:  services.NewExplorerService(cfg, nil, nil),
		AuditService:     auditSvc,
		ReviewService:    services.NewReviewService(cfg, nil),
	}

	srv, err := NewServer(ctr, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	ctx := context.Background()

	// 1. Status Slim vs Expanded
	resSlim, err := srv.handleGetStatus(ctx, mcp.CallToolRequest{})
	if err != nil || resSlim.IsError {
		t.Fatalf("unexpected slim status error: %v", err)
	}
	var slimData map[string]any
	_ = json.Unmarshal([]byte(resSlim.Content[0].(mcp.TextContent).Text), &slimData)
	if _, ok := slimData["bisync_logs"]; ok {
		t.Error("expected slim status to omit bisync_logs by default")
	}

	resFull, err := srv.handleGetStatus(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{"include_logs": true}},
	})
	if err != nil || resFull.IsError {
		t.Fatalf("unexpected full status error: %v", err)
	}
	var fullData map[string]any
	_ = json.Unmarshal([]byte(resFull.Content[0].(mcp.TextContent).Text), &fullData)
	if _, ok := fullData["bisync_logs"]; !ok {
		t.Error("expected full status to include bisync_logs")
	}

	// 2. bisync_get_logs filtering
	resLogs, err := srv.handleGetLogs(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{"limit": 10, "level": "ERROR"}},
	})
	if err != nil || resLogs.IsError {
		t.Fatalf("unexpected get_logs error: %v", err)
	}

	// 3. bisync_get_sync_history hash stripping
	resHistorySlim, _ := srv.handleGetSyncHistory(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{"include_hashes": false}},
	})
	var histSlimData map[string]any
	_ = json.Unmarshal([]byte(resHistorySlim.Content[0].(mcp.TextContent).Text), &histSlimData)
	firstRec := histSlimData["records"].([]any)[0].(map[string]any)
	if _, hasHash := firstRec["hash"]; hasHash {
		t.Error("expected slim audit records to omit hash")
	}

	// 4. bisync_inspect_conflict_session unresolved_only
	activeReviewFile := filepath.Join(tempDir, "active_review.json")
	cfg.ActiveReviewFile = activeReviewFile
	session := &models.ReviewSession{
		ID:     "sess-test",
		Status: models.ReviewStatusReviewing,
		Conflicts: []models.ConflictCandidate{
			{ID: "c-1", Path: "resolved.txt", Resolved: true},
			{ID: "c-2", Path: "unresolved.txt", Resolved: false},
		},
	}
	sessBytes, _ := json.Marshal(session)
	_ = os.WriteFile(activeReviewFile, sessBytes, 0600)

	resInspect, err := srv.handleInspectConflictSession(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{"unresolved_only": true}},
	})
	if err != nil || resInspect.IsError {
		t.Fatalf("unexpected inspect error: %v", err)
	}
	var inspectData map[string]any
	_ = json.Unmarshal([]byte(resInspect.Content[0].(mcp.TextContent).Text), &inspectData)
	confs := inspectData["session"].(map[string]any)["conflicts"].([]any)
	if len(confs) != 1 {
		t.Fatalf("expected 1 unresolved conflict, got %d", len(confs))
	}

	// 5. bisync_explore_path pagination and filter
	_ = os.WriteFile(filepath.Join(tempDir, "alpha.txt"), []byte("a"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "beta.txt"), []byte("b"), 0644)
	resPaged, _ := srv.handleExplorePath(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{"remote": "local", "limit": 1, "offset": 0}},
	})
	var pagedData map[string]any
	_ = json.Unmarshal([]byte(resPaged.Content[0].(mcp.TextContent).Text), &pagedData)
	if pagedData["returned_items"].(float64) != 1 {
		t.Errorf("expected 1 returned item, got %v", pagedData["returned_items"])
	}

	resFiltered, _ := srv.handleExplorePath(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{"remote": "local", "filter": "beta"}},
	})
	var filteredData map[string]any
	_ = json.Unmarshal([]byte(resFiltered.Content[0].(mcp.TextContent).Text), &filteredData)
	if filteredData["total_items"].(float64) != 1 {
		t.Errorf("expected 1 item matching 'beta', got %v", filteredData["total_items"])
	}
}
