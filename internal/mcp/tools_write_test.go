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

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slemmercs/rclone-bisync/internal/app"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

func TestWriteTools_ReadOnlyGuard(t *testing.T) {
	srv, err := NewServer(nil, ServerConfig{
		Version:  "v1.0.0-test",
		ReadOnly: true,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ctx := context.Background()

	// 1. bisync_trigger_sync blocked
	res, err := srv.handleTriggerSync(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{"profile_id": "test-prof"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "read-only") {
		t.Errorf("expected read-only rejection for trigger_sync, got: %v", res)
	}

	// 2. bisync_resolve_review blocked
	res, err = srv.handleResolveReview(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{"decision": "KEEP_LOCAL"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "read-only") {
		t.Errorf("expected read-only rejection for resolve_review, got: %v", res)
	}

	// 3. bisync_dedupe_by_hash blocked
	res, err = srv.handleDedupeByHash(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{"remote": "gdrive:"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "read-only") {
		t.Errorf("expected read-only rejection for dedupe_by_hash, got: %v", res)
	}
}

func TestWriteTools_TriggerSyncConfirmation(t *testing.T) {
	tempDir := t.TempDir()
	profilesFile := filepath.Join(tempDir, "profiles.json")

	store, err := profiles.NewStore(profilesFile, "")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	prof := &profiles.Profile{
		ID:        "prof-sync-1",
		Name:      "Documents",
		LocalPath: tempDir,
		Remote:    "local",
		Enabled:   true,
	}
	if err := store.Save(prof, "test"); err != nil {
		t.Fatalf("failed to save profile: %v", err)
	}

	executedSync := false
	executor := func(ctx context.Context, p *profiles.Profile, opts profiles.SyncOptions) error {
		executedSync = true
		return nil
	}
	scheduler := profiles.NewScheduler(store, executor)
	profSvc := profiles.NewService(store, nil, scheduler)

	ctr := &app.Container{
		Config:         &config.Config{LocalPath: tempDir},
		ProfileService: profSvc,
	}

	srv, err := NewServer(ctr, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ctx := context.Background()
	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	// 1. Dry run should NOT require confirmation
	res, err := srv.handleTriggerSync(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"profile_id": "prof-sync-1",
				"dry_run":    true,
				"resync":     true,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error on dry_run: %v", res)
	}

	// 2. Real resync (dry_run=false, resync=true) WITHOUT confirmation token
	res, err = srv.handleTriggerSync(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"profile_id": "prof-sync-1",
				"dry_run":    false,
				"resync":     true,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error response: %v", res)
	}

	text := res.Content[0].(mcp.TextContent).Text
	var ticketResp struct {
		Status string `json:"status"`
		Token  string `json:"token"`
	}
	if err := json.Unmarshal([]byte(text), &ticketResp); err != nil {
		t.Fatalf("failed to parse ticket response: %v", err)
	}
	if ticketResp.Status != "confirmation_required" || ticketResp.Token == "" {
		t.Fatalf("expected confirmation_required with token, got: %s", text)
	}

	// 3. Real resync with invalid token
	res, err = srv.handleTriggerSync(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"profile_id":         "prof-sync-1",
				"dry_run":            false,
				"resync":             true,
				"confirmation_token": "invalid-token-1234",
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error on invalid confirmation token")
	}

	// 4. Real resync with valid confirmation token
	res, err = srv.handleTriggerSync(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"profile_id":         "prof-sync-1",
				"dry_run":            false,
				"resync":             true,
				"confirmation_token": ticketResp.Token,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error with valid token: %v", res)
	}
	if !executedSync {
		t.Error("expected sync executor to be invoked")
	}
}

func TestWriteTools_DedupeConfirmation(t *testing.T) {
	srv, err := NewServer(nil, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ctx := context.Background()

	// 1. mode="list" does NOT require confirmation
	res, err := srv.handleDedupeByHash(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"remote": "gdrive",
				"mode":   "list",
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error on list mode: %v", res)
	}

	// 2. mode="oldest" without confirmation token
	res, err = srv.handleDedupeByHash(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"remote": "gdrive",
				"mode":   "oldest",
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error response: %v", res)
	}

	text := res.Content[0].(mcp.TextContent).Text
	var ticketResp struct {
		Status string `json:"status"`
		Token  string `json:"token"`
	}
	if err := json.Unmarshal([]byte(text), &ticketResp); err != nil {
		t.Fatalf("failed to parse ticket response: %v", err)
	}
	if ticketResp.Status != "confirmation_required" || ticketResp.Token == "" {
		t.Fatalf("expected confirmation_required with token, got: %s", text)
	}

	// 3. mode="oldest" with valid token
	res, err = srv.handleDedupeByHash(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"remote":             "gdrive",
				"mode":               "oldest",
				"confirmation_token": ticketResp.Token,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error with valid token: %v", res)
	}
}

func TestWriteTools_ResolveReviewConfirmation(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:          tempDir,
		ActiveReviewFile: filepath.Join(tempDir, "review.json"),
	}

	sess := &models.ReviewSession{
		ID:     "session-test-mcp",
		Status: models.ReviewStatusReviewing,
		Conflicts: []models.ConflictCandidate{
			{ID: "cand-1", Path: "file1.txt", Name: "file1.txt"},
		},
	}
	sessBytes, err := json.Marshal(sess)
	if err != nil {
		t.Fatalf("failed to marshal session: %v", err)
	}
	if err := os.WriteFile(cfg.ActiveReviewFile, sessBytes, 0600); err != nil {
		t.Fatalf("failed to write review file: %v", err)
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{})
	}))
	defer mockServer.Close()
	rcClient := client.NewRcloneClient(mockServer.URL, "", "")

	reviewSvc := services.NewReviewService(cfg, rcClient)
	ctr := &app.Container{
		Config:        cfg,
		Client:        rcClient,
		ReviewService: reviewSvc,
	}

	srv, err := NewServer(ctr, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ctx := context.Background()

	// 1. Missing decision parameter
	res, err := srv.handleResolveReview(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error on missing decision")
	}

	// 2. Stage decisions without execute (should NOT require confirmation)
	res, err = srv.handleResolveReview(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": "session-test-mcp",
				"decision":   "KEEP_LOCAL",
				"execute":    false,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error on staging: %v", res)
	}

	// 3. Execute without confirmation token
	res, err = srv.handleResolveReview(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": "session-test-mcp",
				"decision":   "KEEP_LOCAL",
				"execute":    true,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error response: %v", res)
	}

	text := res.Content[0].(mcp.TextContent).Text
	var ticketResp struct {
		Status string `json:"status"`
		Token  string `json:"token"`
	}
	if err := json.Unmarshal([]byte(text), &ticketResp); err != nil {
		t.Fatalf("failed to parse ticket response: %v", err)
	}
	if ticketResp.Status != "confirmation_required" || ticketResp.Token == "" {
		t.Fatalf("expected confirmation_required with token, got: %s", text)
	}

	// 4. Execute with invalid confirmation token
	res, err = srv.handleResolveReview(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id":         "session-test-mcp",
				"decision":           "KEEP_LOCAL",
				"execute":            true,
				"confirmation_token": "bad-token",
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error on invalid token")
	}

	// 5. Execute with valid confirmation token
	res, err = srv.handleResolveReview(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id":         "session-test-mcp",
				"decision":           "KEEP_LOCAL",
				"execute":            true,
				"confirmation_token": ticketResp.Token,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error with valid token: %v", res)
	}
}
