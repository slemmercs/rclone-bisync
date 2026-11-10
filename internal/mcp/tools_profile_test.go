package mcp

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slemmercs/rclone-bisync/internal/app"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
)

func TestProfileTools_ReadOnlyGuard(t *testing.T) {
	srv, err := NewServer(nil, ServerConfig{
		Version:  "v1.0.0-test",
		ReadOnly: true,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// 1. bisync_create_profile
	res, err := callMCP(srv.handleCreateProfile, map[string]any{"name": "p1", "remote": "local", "local_path": "/tmp"})
	if err != nil || !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "read-only") {
		t.Errorf("expected read-only rejection for create_profile, got: %v, err: %v", res, err)
	}

	// 2. bisync_update_profile
	res, err = callMCP(srv.handleUpdateProfile, map[string]any{"profile_id": "p1"})
	if err != nil || !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "read-only") {
		t.Errorf("expected read-only rejection for update_profile, got: %v, err: %v", res, err)
	}

	// 3. bisync_delete_profile
	res, err = callMCP(srv.handleDeleteProfile, map[string]any{"profile_id": "p1"})
	if err != nil || !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "read-only") {
		t.Errorf("expected read-only rejection for delete_profile, got: %v, err: %v", res, err)
	}

	// 4. bisync_rollback_profile
	res, err = callMCP(srv.handleRollbackProfile, map[string]any{"profile_id": "p1", "version": 1})
	if err != nil || !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "read-only") {
		t.Errorf("expected read-only rejection for rollback_profile, got: %v, err: %v", res, err)
	}
}

func TestProfileTools_CreateAndPathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	profStore, err := profiles.NewStore(filepath.Join(tmpDir, "profiles.json"), filepath.Join(tmpDir, "history.json"))
	if err != nil {
		t.Fatalf("failed to create profStore: %v", err)
	}
	profSvc := profiles.NewService(profStore, nil, nil)
	ctr := &app.Container{
		ProfileService: profSvc,
		Config:         &config.Config{LocalPath: tmpDir},
	}

	srv, err := NewServer(ctr, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// 1. Path traversal rejected
	res, err := callMCP(srv.handleCreateProfile, map[string]any{
		"name":       "Bad Traversal",
		"remote":     "local",
		"local_path": tmpDir + "/../etc",
	})
	if err != nil || !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "path traversal") {
		t.Fatalf("expected path traversal rejection, got: %v", res)
	}

	// 2. Valid creation
	syncDir := filepath.Join(tmpDir, "sync_data")
	res, err = callMCP(srv.handleCreateProfile, map[string]any{
		"name":          "Valid Profile",
		"remote":        "local",
		"local_path":    syncDir,
		"schedule":      "0 * * * *",
		"conflict_mode": "newer",
		"max_delete":    20,
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error creating profile: %v", res)
	}
	text := res.Content[0].(mcp.TextContent).Text
	var created profiles.Profile
	if err := json.Unmarshal([]byte(text), &created); err != nil {
		t.Fatalf("failed to unmarshal created profile: %v", err)
	}
	if created.Name != "Valid Profile" || created.Flags.ConflictResolve == nil || *created.Flags.ConflictResolve != "newer" || created.Flags.MaxDelete == nil || string(*created.Flags.MaxDelete) != "20" {
		t.Fatalf("unexpected profile values: %+v", created)
	}
}

func TestProfileTools_Update(t *testing.T) {
	tmpDir := t.TempDir()
	profStore, err := profiles.NewStore(filepath.Join(tmpDir, "profiles.json"), filepath.Join(tmpDir, "history.json"))
	if err != nil {
		t.Fatalf("failed to create profStore: %v", err)
	}
	profSvc := profiles.NewService(profStore, nil, nil)
	ctr := &app.Container{
		ProfileService: profSvc,
	}

	srv, err := NewServer(ctr, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Create initial profile
	p, err := profSvc.Create(context.Background(), &profiles.Profile{
		ID:        "prof-upd-1",
		Name:      "Initial Name",
		Remote:    "local",
		LocalPath: tmpDir,
		Enabled:   true,
	})
	if err != nil {
		t.Fatalf("failed to create initial profile: %v", err)
	}

	// Update name, schedule, enabled
	res, err := callMCP(srv.handleUpdateProfile, map[string]any{
		"profile_id": p.ID,
		"name":       "Updated MCP Name",
		"schedule":   "*/15 * * * *",
		"enabled":    false,
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error updating profile: %v", res)
	}
	text := res.Content[0].(mcp.TextContent).Text
	var updated profiles.Profile
	if err := json.Unmarshal([]byte(text), &updated); err != nil {
		t.Fatalf("failed to unmarshal updated profile: %v", err)
	}
	if updated.Name != "Updated MCP Name" || updated.Schedule != "*/15 * * * *" || updated.Enabled != false {
		t.Fatalf("unexpected updated values: %+v", updated)
	}
}

func TestProfileTools_DeleteConfirmation(t *testing.T) {
	tmpDir := t.TempDir()
	profStore, err := profiles.NewStore(filepath.Join(tmpDir, "profiles.json"), filepath.Join(tmpDir, "history.json"))
	if err != nil {
		t.Fatalf("failed to create profStore: %v", err)
	}
	profSvc := profiles.NewService(profStore, nil, nil)
	ctr := &app.Container{
		ProfileService: profSvc,
	}

	srv, err := NewServer(ctr, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	p, err := profSvc.Create(context.Background(), &profiles.Profile{
		ID:        "prof-del-1",
		Name:      "To Delete",
		Remote:    "local",
		LocalPath: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	// 1. Delete without token -> confirmation required
	res, err := callMCP(srv.handleDeleteProfile, map[string]any{
		"profile_id": p.ID,
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error requesting delete ticket: %v", res)
	}
	text := res.Content[0].(mcp.TextContent).Text
	var ticketPayload struct {
		Status string `json:"status"`
		Token  string `json:"token"`
	}
	if err := json.Unmarshal([]byte(text), &ticketPayload); err != nil {
		t.Fatalf("failed to parse ticket payload: %v", err)
	}
	if ticketPayload.Status != "confirmation_required" || ticketPayload.Token == "" {
		t.Fatalf("expected confirmation_required with token, got: %+v", ticketPayload)
	}

	// Profile still exists
	if _, err := profSvc.Get(p.ID); err != nil {
		t.Fatalf("profile should still exist before confirmation")
	}

	// 2. Delete with token -> success
	res, err = callMCP(srv.handleDeleteProfile, map[string]any{
		"profile_id":         p.ID,
		"confirmation_token": ticketPayload.Token,
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error confirming delete: %v", res)
	}

	// Profile now deleted
	if _, err := profSvc.Get(p.ID); err == nil {
		t.Fatalf("expected profile to be deleted")
	}
}

func TestProfileTools_RollbackConfirmation(t *testing.T) {
	tmpDir := t.TempDir()
	profStore, err := profiles.NewStore(filepath.Join(tmpDir, "profiles.json"), filepath.Join(tmpDir, "history.json"))
	if err != nil {
		t.Fatalf("failed to create profStore: %v", err)
	}
	profSvc := profiles.NewService(profStore, nil, nil)
	ctr := &app.Container{
		ProfileService: profSvc,
	}

	srv, err := NewServer(ctr, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Create profile (version 1)
	p, err := profSvc.Create(context.Background(), &profiles.Profile{
		ID:        "prof-rb-1",
		Name:      "Original Name",
		Remote:    "local",
		LocalPath: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	// Update profile (version 2)
	p.Name = "Modified Name"
	_, err = profSvc.Update(context.Background(), p, "Modified for test")
	if err != nil {
		t.Fatalf("failed to update profile: %v", err)
	}

	// 1. Call rollback without token -> returns confirmation ticket
	res, err := callMCP(srv.handleRollbackProfile, map[string]any{
		"profile_id": p.ID,
		"version":    1,
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error requesting rollback ticket: %v", res)
	}
	text := res.Content[0].(mcp.TextContent).Text
	var ticketPayload struct {
		Status string `json:"status"`
		Token  string `json:"token"`
	}
	if err := json.Unmarshal([]byte(text), &ticketPayload); err != nil {
		t.Fatalf("failed to parse ticket payload: %v", err)
	}
	if ticketPayload.Status != "confirmation_required" || ticketPayload.Token == "" {
		t.Fatalf("expected confirmation_required with token, got: %+v", ticketPayload)
	}

	// 2. Call rollback with token -> restored to version 1
	res, err = callMCP(srv.handleRollbackProfile, map[string]any{
		"profile_id":         p.ID,
		"version":            1,
		"confirmation_token": ticketPayload.Token,
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error confirming rollback: %v", res)
	}
	text = res.Content[0].(mcp.TextContent).Text
	var rolledBack profiles.Profile
	if err := json.Unmarshal([]byte(text), &rolledBack); err != nil {
		t.Fatalf("failed to unmarshal rolled back profile: %v", err)
	}
	if rolledBack.Name != "Original Name" {
		t.Fatalf("expected rolled back name 'Original Name', got: %s", rolledBack.Name)
	}
}

