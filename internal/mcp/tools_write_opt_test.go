package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slemmercs/rclone-bisync/internal/app"
	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

func TestWriteTools_ResolveReview_DismissAndValidation(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:          tempDir,
		ActiveReviewFile: filepath.Join(tempDir, "review.json"),
	}

	sess := &models.ReviewSession{
		ID:     "session-dismiss-test",
		Status: models.ReviewStatusReviewing,
	}
	sessBytes, _ := json.Marshal(sess)
	_ = os.WriteFile(cfg.ActiveReviewFile, sessBytes, 0600)

	reviewSvc := services.NewReviewService(cfg, nil)
	ctr := &app.Container{
		Config:        cfg,
		ReviewService: reviewSvc,
	}

	srv, err := NewServer(ctr, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	ctx := context.Background()

	// 1. Invalid decision string
	res, err := srv.handleResolveReview(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{"decision": "invalid-action-xyz"},
		},
	})
	if err != nil || !res.IsError {
		t.Fatalf("expected error on invalid decision: %v, res: %v", err, res)
	}

	// 2. Dismiss decision
	res, err = srv.handleResolveReview(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": "session-dismiss-test",
				"decision":   "dismiss",
			},
		},
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error on dismiss: %v, res: %v", err, res)
	}
}

func TestWriteTools_DedupeAuditAndTruncation(t *testing.T) {
	tempDir := t.TempDir()
	auditPath := filepath.Join(tempDir, "audit.db")
	db, err := audit.OpenDB(auditPath)
	if err != nil {
		t.Fatalf("failed to open audit db: %v", err)
	}
	defer db.Close()
	auditSvc := services.NewAuditService(db, nil)

	// Mock RC dedupe returning long text with duplicates deleted
	longOutput := strings.Repeat("Found duplicate file.txt\nDeleted duplicate copy file.txt\n", 50)
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"result": longOutput})
	}))
	defer mockServer.Close()
	rcClient := client.NewRcloneClient(mockServer.URL, "", "")

	ctr := &app.Container{
		Client:       rcClient,
		AuditService: auditSvc,
	}

	srv, err := NewServer(ctr, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	ctx := context.Background()

	// 1. Get confirmation token
	res, _ := srv.handleDedupeByHash(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{"remote": "drive", "mode": "oldest", "by_hash": true},
		},
	})
	var ticketResp struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal([]byte(res.Content[0].(mcp.TextContent).Text), &ticketResp)

	// 2. Execute dedupe with token
	resExec, err := srv.handleDedupeByHash(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"remote":             "drive",
				"mode":               "oldest",
				"by_hash":            true,
				"confirmation_token": ticketResp.Token,
			},
		},
	})
	if err != nil || resExec.IsError {
		t.Fatalf("unexpected dedupe execution error: %v", err)
	}

	var data map[string]any
	_ = json.Unmarshal([]byte(resExec.Content[0].(mcp.TextContent).Text), &data)
	if data["deleted"].(float64) != 50 {
		t.Errorf("expected 50 deleted items, got %v", data["deleted"])
	}
	if data["truncated"].(bool) != true {
		t.Error("expected output to be truncated")
	}

	// 3. Verify audit ledger record was written without unbounded raw_output
	records, total, err := auditSvc.Query(audit.Filter{Action: audit.ActionDedupeExecuted})
	if err != nil || total != 1 {
		t.Fatalf("expected 1 audit record, got total=%d err=%v", total, err)
	}
	detailsStr := string(records[0].Details)
	if strings.Contains(detailsStr, "raw_output") {
		t.Error("audit record should not contain raw_output per ADR-004")
	}
	if !strings.Contains(detailsStr, `"deleted":50`) {
		t.Errorf("audit record should contain deleted count, got: %s", detailsStr)
	}
}
