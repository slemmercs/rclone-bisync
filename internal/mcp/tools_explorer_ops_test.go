package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slemmercs/rclone-bisync/internal/app"
	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

func setupExplorerMCPTest(t *testing.T) (*Server, string, *sql.DB, *httptest.Server) {
	t.Helper()
	tmpDir := t.TempDir()
	auditDBPath := filepath.Join(tmpDir, "audit.db")
	db, err := audit.OpenDB(auditDBPath)
	if err != nil {
		t.Fatalf("failed to open audit db: %v", err)
	}
	auditSvc := services.NewAuditService(db, nil)
	cfg := &config.Config{LocalPath: tmpDir}

	mockRC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/operations/copyfile":
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			srcFs, _ := body["srcFs"].(string)
			srcRemote, _ := body["srcRemote"].(string)
			dstFs, _ := body["dstFs"].(string)
			dstRemote, _ := body["dstRemote"].(string)
			data, _ := os.ReadFile(filepath.Join(srcFs, srcRemote))
			_ = os.WriteFile(filepath.Join(dstFs, dstRemote), data, 0644)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))

	rcClient := client.NewRcloneClient(mockRC.URL, "", "")
	expSvc := services.NewExplorerService(cfg, rcClient, nil)

	ctr := &app.Container{
		Config:          cfg,
		AuditService:    auditSvc,
		ExplorerService: expSvc,
		Client:          rcClient,
	}

	srv, err := NewServer(ctr, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		mockRC.Close()
		t.Fatalf("failed to create server: %v", err)
	}

	return srv, tmpDir, db, mockRC
}

func TestExplorerOpsTools_ReadOnlyGuard(t *testing.T) {
	srv, err := NewServer(nil, ServerConfig{
		Version:  "v1.0.0-test",
		ReadOnly: true,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	mutatingTools := []struct {
		name    string
		handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		args    map[string]any
	}{
		{"mkdir", srv.handleMakeDir, map[string]any{"remote": "local", "path": "dir1"}},
		{"copy", srv.handleCopyItem, map[string]any{"src_remote": "local", "src_path": "a", "dst_remote": "local", "dst_path": "b"}},
		{"move", srv.handleMoveItem, map[string]any{"src_remote": "local", "src_path": "a", "dst_remote": "local", "dst_path": "b"}},
		{"delete", srv.handleDeleteItem, map[string]any{"remote": "local", "path": "a"}},
		{"purge", srv.handlePurgeDir, map[string]any{"remote": "local", "path": "dir1"}},
	}

	for _, tc := range mutatingTools {
		res, err := callMCP(tc.handler, tc.args)
		if err != nil || !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "read-only") {
			t.Errorf("expected read-only rejection for %s, got: %v, err: %v", tc.name, res, err)
		}
	}
}

func TestExplorerOpsTools_PreviewFile(t *testing.T) {
	srv, tmpDir, db, mockRC := setupExplorerMCPTest(t)
	defer db.Close()
	defer mockRC.Close()

	testFile := filepath.Join(tmpDir, "readme.txt")
	if err := os.WriteFile(testFile, []byte("Hello MCP Explorer!"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// 1. Valid preview
	res, err := callMCP(srv.handlePreviewFile, map[string]any{
		"remote": "local",
		"path":   "readme.txt",
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error previewing file: %v", res)
	}
	text := res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "Hello MCP Explorer!") {
		t.Fatalf("expected file contents in preview, got: %s", text)
	}

	// 2. Traversal rejection
	res, err = callMCP(srv.handlePreviewFile, map[string]any{
		"remote": "local",
		"path":   "../secret.txt",
	})
	if err != nil || !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "traversal") {
		t.Fatalf("expected traversal rejection, got: %v", res)
	}
}

func TestExplorerOpsTools_Mkdir(t *testing.T) {
	srv, tmpDir, db, mockRC := setupExplorerMCPTest(t)
	defer db.Close()
	defer mockRC.Close()

	res, err := callMCP(srv.handleMakeDir, map[string]any{
		"remote": "local",
		"path":   "nested/new_dir",
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error making dir: %v", res)
	}

	dirInfo, err := os.Stat(filepath.Join(tmpDir, "nested", "new_dir"))
	if err != nil || !dirInfo.IsDir() {
		t.Fatalf("expected directory to be created on disk, err: %v", err)
	}

	// Verify audit event
	events, _, _ := srv.container.AuditService.Query(audit.Filter{Action: audit.ActionWorkspaceDirCreated})
	if len(events) == 0 || events[0].Action != audit.ActionWorkspaceDirCreated {
		t.Fatalf("expected WORKSPACE_DIR_CREATED audit event, got: %+v", events)
	}
}

func TestExplorerOpsTools_CopyAndMove(t *testing.T) {
	srv, tmpDir, db, mockRC := setupExplorerMCPTest(t)
	defer db.Close()
	defer mockRC.Close()

	srcFile := filepath.Join(tmpDir, "origin.txt")
	if err := os.WriteFile(srcFile, []byte("Content to duplicate"), 0644); err != nil {
		t.Fatalf("failed to write origin file: %v", err)
	}

	// 1. Copy origin.txt -> copied.txt
	res, err := callMCP(srv.handleCopyItem, map[string]any{
		"src_remote": "local",
		"src_path":   "origin.txt",
		"dst_remote": "local",
		"dst_path":   "copied.txt",
		"is_dir":     false,
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error copying item: %v", res)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "copied.txt")); err != nil {
		t.Fatalf("expected copied.txt to exist: %v", err)
	}

	// 2. Move copied.txt -> moved.txt
	res, err = callMCP(srv.handleMoveItem, map[string]any{
		"src_remote": "local",
		"src_path":   "copied.txt",
		"dst_remote": "local",
		"dst_path":   "moved.txt",
		"is_dir":     false,
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error moving item: %v", res)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "copied.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected copied.txt to be gone after move")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "moved.txt")); err != nil {
		t.Fatalf("expected moved.txt to exist: %v", err)
	}
}

func TestExplorerOpsTools_CanaryProtection(t *testing.T) {
	srv, tmpDir, db, mockRC := setupExplorerMCPTest(t)
	defer db.Close()
	defer mockRC.Close()

	canaryFile := filepath.Join(tmpDir, "RCLONE_TEST")
	if err := os.WriteFile(canaryFile, []byte("canary"), 0644); err != nil {
		t.Fatalf("failed to write canary: %v", err)
	}

	// 1. Delete canary rejected
	res, err := callMCP(srv.handleDeleteItem, map[string]any{
		"remote": "local",
		"path":   "RCLONE_TEST",
	})
	if err != nil || !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "canary") {
		t.Fatalf("expected canary rejection on delete, got: %v", res)
	}

	// 2. Move canary rejected
	res, err = callMCP(srv.handleMoveItem, map[string]any{
		"src_remote": "local",
		"src_path":   "RCLONE_TEST",
		"dst_remote": "local",
		"dst_path":   "RENAMED_TEST",
	})
	if err != nil || !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "canary") {
		t.Fatalf("expected canary rejection on move, got: %v", res)
	}

	// 3. Purge directory with canary rejected
	res, err = callMCP(srv.handlePurgeDir, map[string]any{
		"remote": "local",
		"path":   "sub/RCLONE_TEST",
	})
	if err != nil || !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "canary") {
		t.Fatalf("expected canary rejection on purge, got: %v", res)
	}
}

func TestExplorerOpsTools_DeleteConfirmation(t *testing.T) {
	srv, tmpDir, db, mockRC := setupExplorerMCPTest(t)
	defer db.Close()
	defer mockRC.Close()

	victim := filepath.Join(tmpDir, "to_delete.txt")
	if err := os.WriteFile(victim, []byte("delete me"), 0644); err != nil {
		t.Fatalf("failed to write victim file: %v", err)
	}

	// 1. Delete without confirmation token -> returns confirmation ticket
	res, err := callMCP(srv.handleDeleteItem, map[string]any{
		"remote": "local",
		"path":   "to_delete.txt",
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

	// Verify file still exists on disk
	if _, err := os.Stat(victim); err != nil {
		t.Fatalf("file should still exist before confirmation")
	}

	// 2. Delete with valid confirmation token -> success
	res, err = callMCP(srv.handleDeleteItem, map[string]any{
		"remote":             "local",
		"path":               "to_delete.txt",
		"confirmation_token": ticketPayload.Token,
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error confirming delete: %v", res)
	}

	// Verify file deleted
	if _, err := os.Stat(victim); !os.IsNotExist(err) {
		t.Fatalf("expected file to be deleted from disk")
	}
}

func TestExplorerOpsTools_PurgeConfirmation(t *testing.T) {
	srv, tmpDir, db, mockRC := setupExplorerMCPTest(t)
	defer db.Close()
	defer mockRC.Close()

	purgeDir := filepath.Join(tmpDir, "folder_to_purge")
	if err := os.MkdirAll(filepath.Join(purgeDir, "nested"), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(purgeDir, "nested", "data.txt"), []byte("data"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// 1. Purge without confirmation token -> confirmation required
	res, err := callMCP(srv.handlePurgeDir, map[string]any{
		"remote": "local",
		"path":   "folder_to_purge",
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error requesting purge ticket: %v", res)
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

	// Verify directory still exists on disk
	if _, err := os.Stat(purgeDir); err != nil {
		t.Fatalf("directory should still exist before confirmation")
	}

	// 2. Purge with valid confirmation token -> success
	res, err = callMCP(srv.handlePurgeDir, map[string]any{
		"remote":             "local",
		"path":               "folder_to_purge",
		"confirmation_token": ticketPayload.Token,
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error confirming purge: %v", res)
	}

	// Verify directory purged from disk
	if _, err := os.Stat(purgeDir); !os.IsNotExist(err) {
		t.Fatalf("expected directory to be deleted from disk")
	}
}
