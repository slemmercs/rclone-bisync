package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slemmercs/rclone-bisync/internal/app"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

func TestReadTools_NilContainer(t *testing.T) {
	srv, err := NewServer(nil, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ctx := context.Background()

	// 1. bisync_list_profiles with nil container
	res, err := srv.handleListProfiles(ctx, mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Content) == 0 {
		t.Fatal("expected non-empty content in result")
	}

	// 2. bisync_get_status with nil container
	res, err = srv.handleGetStatus(ctx, mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Content) == 0 {
		t.Fatal("expected non-empty content in result")
	}

	// 3. bisync_list_remotes with nil container
	res, err = srv.handleListRemotes(ctx, mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Content) == 0 {
		t.Fatal("expected non-empty content in result")
	}

	// 4. bisync_get_sync_history with nil container
	res, err = srv.handleGetSyncHistory(ctx, mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Content) == 0 {
		t.Fatal("expected non-empty content in result")
	}

	// 5. bisync_inspect_conflict_session with nil container
	res, err = srv.handleInspectConflictSession(ctx, mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Content) == 0 {
		t.Fatal("expected non-empty content in result")
	}

	// 6. bisync_explore_path with nil container
	res, err = srv.handleExplorePath(ctx, mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Error("expected error for nil container in explore_path")
	}
}

func TestReadTools_WithServices(t *testing.T) {
	tempDir := t.TempDir()
	profilesFile := filepath.Join(tempDir, "profiles.json")

	store, err := profiles.NewStore(profilesFile, "")
	if err != nil {
		t.Fatalf("failed to create profile store: %v", err)
	}

	prof := &profiles.Profile{
		ID:        "test-profile-1",
		Name:      "Work Docs",
		LocalPath: tempDir,
		Remote:    "local",
	}
	if err := store.Save(prof, "test"); err != nil {
		t.Fatalf("failed to save profile: %v", err)
	}

	profSvc := profiles.NewService(store, nil, nil)
	cfg := &config.Config{
		LocalPath: tempDir,
	}
	telSvc := services.NewTelemetryService(cfg, nil)
	expSvc := services.NewExplorerService(cfg, nil, nil)

	ctr := &app.Container{
		Config:           cfg,
		ProfileService:   profSvc,
		TelemetryService: telSvc,
		ExplorerService:  expSvc,
	}

	srv, err := NewServer(ctr, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ctx := context.Background()

	// 1. bisync_list_profiles
	res, err := srv.handleListProfiles(ctx, mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatal("unexpected tool error in list_profiles")
	}

	// 2. bisync_get_status
	res, err = srv.handleGetStatus(ctx, mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatal("unexpected tool error in get_status")
	}

	// 3. bisync_explore_path with valid local path
	testFile := filepath.Join(tempDir, "sample.txt")
	_ = os.WriteFile(testFile, []byte("hello"), 0644)

	exploreReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"remote":     "local",
				"path":       "",
				"profile_id": "test-profile-1",
			},
		},
	}
	res, err = srv.handleExplorePath(ctx, exploreReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error in explore_path: %v", res)
	}

	// 4. bisync_explore_path with path traversal
	traversalReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"remote": "local",
				"path":   "../../etc",
			},
		},
	}
	res, err = srv.handleExplorePath(ctx, traversalReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Error("expected error for path traversal")
	}
}

func TestResources_Read(t *testing.T) {
	tempDir := t.TempDir()
	profilesFile := filepath.Join(tempDir, "profiles.json")

	store, err := profiles.NewStore(profilesFile, "")
	if err != nil {
		t.Fatalf("failed to create profile store: %v", err)
	}

	prof := &profiles.Profile{
		ID:        "prof-res-1",
		Name:      "Cloud Backup",
		LocalPath: tempDir,
		Remote:    "drive:",
	}
	if err := store.Save(prof, "test"); err != nil {
		t.Fatalf("failed to save profile: %v", err)
	}

	profSvc := profiles.NewService(store, nil, nil)
	cfg := &config.Config{LocalPath: tempDir}
	telSvc := services.NewTelemetryService(cfg, nil)

	ctr := &app.Container{
		Config:           cfg,
		ProfileService:   profSvc,
		TelemetryService: telSvc,
	}

	srv, err := NewServer(ctr, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ctx := context.Background()

	// 1. bisync://profiles
	contents, err := srv.handleResourceProfiles(ctx, mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{URI: "bisync://profiles"},
	})
	if err != nil {
		t.Fatalf("failed to read bisync://profiles: %v", err)
	}
	if len(contents) == 0 {
		t.Fatal("expected contents for bisync://profiles")
	}

	// 2. bisync://status
	contents, err = srv.handleResourceStatus(ctx, mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{URI: "bisync://status"},
	})
	if err != nil {
		t.Fatalf("failed to read bisync://status: %v", err)
	}
	if len(contents) == 0 {
		t.Fatal("expected contents for bisync://status")
	}

	// 3. bisync://profile/{id}
	contents, err = srv.handleResourceProfileByID(ctx, mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{URI: "bisync://profile/prof-res-1"},
	})
	if err != nil {
		t.Fatalf("failed to read bisync://profile/prof-res-1: %v", err)
	}
	if len(contents) == 0 {
		t.Fatal("expected contents for profile by ID")
	}

	// Non-existent profile
	_, err = srv.handleResourceProfileByID(ctx, mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{URI: "bisync://profile/does-not-exist"},
	})
	if err == nil {
		t.Error("expected error for non-existent profile")
	}

	// 4. bisync://history/{id}
	contents, err = srv.handleResourceHistoryByID(ctx, mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{URI: "bisync://history/prof-res-1"},
	})
	if err != nil {
		t.Fatalf("failed to read bisync://history/prof-res-1: %v", err)
	}
	if len(contents) == 0 {
		t.Fatal("expected contents for history by ID")
	}
}
