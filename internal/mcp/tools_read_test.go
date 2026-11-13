package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slemmercs/rclone-bisync/internal/app"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
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

	// 5. bisync_inspect_conflict_session
	reviewSvc := services.NewReviewService(cfg, nil)
	ctr.ReviewService = reviewSvc
	res, err = srv.handleInspectConflictSession(ctx, mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Errorf("unexpected error in inspect_conflict_session: %v", res)
	}

	// Traversal session ID rejection
	res, err = srv.handleInspectConflictSession(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{"session_id": "../../etc/passwd"}},
	})
	if err != nil || !res.IsError {
		t.Fatalf("expected error on path traversal session_id in inspect_conflict_session, got: %v", res)
	}

	// 6. bisync_get_sync_history without audit service
	res, err = srv.handleGetSyncHistory(ctx, mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Errorf("unexpected error in get_sync_history: %v", res)
	}
	res, err = srv.handleGetSyncHistory(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{"limit": 500},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Errorf("unexpected error in get_sync_history with custom limit: %v", res)
	}

	// 7. bisync_list_remotes with mock RC server
	mockRC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"remotes": []string{"gdrive", "onedrive"},
		})
	}))
	defer mockRC.Close()
	ctr.Client = client.NewRcloneClient(mockRC.URL, "", "")
	ctr.TelemetryService = services.NewTelemetryService(cfg, ctr.Client)
	res, err = srv.handleListRemotes(ctx, mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Errorf("unexpected error in list_remotes: %v", res)
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

	// 5. bisync://conflicts/{session_id}
	activeReviewFile := filepath.Join(tempDir, "active_review.json")
	cfg.ActiveReviewFile = activeReviewFile
	reviewSess := &models.ReviewSession{ID: "sess-res-1", Status: models.ReviewStatusReviewing}
	sessData, _ := json.Marshal(reviewSess)
	_ = os.WriteFile(activeReviewFile, sessData, 0600)
	ctr.ReviewService = services.NewReviewService(cfg, nil)

	contents, err = srv.handleResourceConflictSession(ctx, mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{URI: "bisync://conflicts/sess-res-1"},
	})
	if err != nil {
		t.Fatalf("failed to read bisync://conflicts/sess-res-1: %v", err)
	}
	if len(contents) == 0 {
		t.Fatal("expected contents for conflict session resource")
	}

	// 6. Resource error edge cases
	// Empty history ID
	if _, err := srv.handleResourceHistoryByID(ctx, mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{URI: "bisync://history/"},
	}); err == nil {
		t.Error("expected error for empty history ID")
	}

	// Fallback to ProfileService when AuditService is nil
	ctr.AuditService = nil
	if contents, err := srv.handleResourceHistoryByID(ctx, mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{URI: "bisync://history/prof-res-1"},
	}); err != nil || len(contents) == 0 {
		t.Errorf("expected ProfileService history fallback, got: %v", err)
	}

	// Invalid conflict session ID (path traversal)
	if _, err := srv.handleResourceConflictSession(ctx, mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{URI: "bisync://conflicts/../escape"},
	}); err == nil {
		t.Error("expected error for traversal in conflict session ID")
	}

	// Nil ReviewService
	ctr.ReviewService = nil
	if _, err := srv.handleResourceConflictSession(ctx, mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{URI: "bisync://conflicts/sess-res-1"},
	}); err == nil {
		t.Error("expected error for nil ReviewService")
	}

}

func TestValidateFlags(t *testing.T) {
	tempDir := t.TempDir()
	profilesFile := filepath.Join(tempDir, "profiles.json")
	store, err := profiles.NewStore(profilesFile, "")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	maxDel := profiles.MaxDeleteValue("25%")
	cmp := "modtime"
	prof := &profiles.Profile{
		ID:        "prof-val",
		Name:      "Validation Profile",
		LocalPath: tempDir,
		Remote:    "gdrive:backups",
		Flags: profiles.ProfileFlags{
			MaxDelete:   &maxDel,
			CompareMode: &cmp,
			CustomFlags: "--track-renames",
		},
	}
	_ = store.Save(prof, "test")

	cfg := &config.Config{LocalPath: tempDir}
	ctr := &app.Container{
		Config:         cfg,
		ProfileService: profiles.NewService(store, nil, nil),
	}
	srv, err := NewServer(ctr, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	ctx := context.Background()

	// 1. Missing arguments
	resEmpty, _ := srv.handleValidateFlags(ctx, mcp.CallToolRequest{})
	if !resEmpty.IsError {
		t.Error("expected error for empty arguments in validate_flags")
	}

	// 2. Space-separated string flags
	resStr, err := srv.handleValidateFlags(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{"flags": "--max-delete 10% --track-renames --dry-run"}},
	})
	if err != nil || resStr.IsError {
		t.Fatalf("unexpected error for string flags: %v", err)
	}
	var dataStr map[string]any
	_ = json.Unmarshal([]byte(resStr.Content[0].(mcp.TextContent).Text), &dataStr)
	if dataStr["valid"] != true {
		t.Errorf("expected valid=true, got %v", dataStr["valid"])
	}
	paramsStr := dataStr["params"].(map[string]any)
	if paramsStr["maxDelete"].(float64) != 10 {
		t.Errorf("expected maxDelete 10, got %v", paramsStr["maxDelete"])
	}
	if paramsStr["dryRun"] != true {
		t.Errorf("expected dryRun true, got %v", paramsStr["dryRun"])
	}
	cfgStr := dataStr["config"].(map[string]any)
	if cfgStr["TrackRenames"] != true {
		t.Errorf("expected TrackRenames true, got %v", cfgStr["TrackRenames"])
	}

	// 3. Array of flag strings
	resArr, err := srv.handleValidateFlags(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{"flags": []any{"--buffer-size", "64M", "--retries", "5"}}},
	})
	if err != nil || resArr.IsError {
		t.Fatalf("unexpected error for array flags: %v", err)
	}
	var dataArr map[string]any
	_ = json.Unmarshal([]byte(resArr.Content[0].(mcp.TextContent).Text), &dataArr)
	cfgArr := dataArr["config"].(map[string]any)
	if cfgArr["BufferSize"] != "64M" || cfgArr["Retries"].(float64) != 5 {
		t.Errorf("unexpected config in array flags: %v", cfgArr)
	}

	// 4. Invalid flags (bad integer/float)
	resInv, err := srv.handleValidateFlags(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{"flags": "--max-delete badpct --retries notanumber"}},
	})
	if err != nil || resInv.IsError {
		t.Fatalf("unexpected tool failure for invalid flags: %v", err)
	}
	var dataInv map[string]any
	_ = json.Unmarshal([]byte(resInv.Content[0].(mcp.TextContent).Text), &dataInv)
	if dataInv["valid"] != false {
		t.Errorf("expected valid=false for invalid flags, got %v", dataInv["valid"])
	}
	errs := dataInv["errors"].([]any)
	if len(errs) != 2 {
		t.Errorf("expected 2 errors, got %d", len(errs))
	}

	// 5. Valid profile_id
	resProf, err := srv.handleValidateFlags(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{"profile_id": "prof-val"}},
	})
	if err != nil || resProf.IsError {
		t.Fatalf("unexpected error for profile_id: %v", err)
	}
	var dataProf map[string]any
	_ = json.Unmarshal([]byte(resProf.Content[0].(mcp.TextContent).Text), &dataProf)
	if dataProf["valid"] != true {
		t.Errorf("expected profile flags to be valid, got errors: %v", dataProf["errors"])
	}
	profParams := dataProf["params"].(map[string]any)
	if profParams["maxDelete"].(float64) != 25 {
		t.Errorf("expected maxDelete 25 from profile, got %v", profParams["maxDelete"])
	}

	// 6. Non-existent profile_id
	resBadProf, _ := srv.handleValidateFlags(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{"profile_id": "prof-does-not-exist"}},
	})
	if !resBadProf.IsError {
		t.Error("expected error for non-existent profile_id")
	}
}
