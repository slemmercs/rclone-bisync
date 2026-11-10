package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestRemoteTools_ReadOnlyGuard(t *testing.T) {
	srv, err := NewServer(nil, ServerConfig{
		Version:  "v1.0.0-test",
		ReadOnly: true,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// 1. bisync_create_remote blocked in read-only
	res, err := callMCP(srv.handleCreateRemote, map[string]any{
		"name": "my_s3",
		"type": "s3",
	})
	if err != nil || !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "read-only") {
		t.Errorf("expected read-only rejection for create_remote, got: %v, err: %v", res, err)
	}

	// 2. bisync_delete_remote blocked in read-only
	res, err = callMCP(srv.handleDeleteRemote, map[string]any{
		"name": "my_s3",
	})
	if err != nil || !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "read-only") {
		t.Errorf("expected read-only rejection for delete_remote, got: %v, err: %v", res, err)
	}
}

func TestRemoteTools_ListProviders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/config/providers" {
			_ = json.NewEncoder(w).Encode(models.ProvidersResponse{
				Providers: []models.Provider{
					{Name: "drive", Description: "Google Drive"},
					{Name: "s3", Description: "Amazon S3 Compliant"},
					{Name: "sftp", Description: "SSH/SFTP"},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := client.NewRcloneClient(server.URL, "", "")
	wizSvc := services.NewRemoteWizardService(&config.Config{}, c, nil, nil, nil)
	ctr := &app.Container{
		RemoteWizardService: wizSvc,
	}

	srv, err := NewServer(ctr, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// 1. Query all
	res, err := callMCP(srv.handleListProviders, map[string]any{})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error listing providers: %v", res)
	}
	text := res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "drive") || !strings.Contains(text, "s3") {
		t.Fatalf("expected drive and s3 in output, got: %s", text)
	}

	// 2. Query filtered by category
	res, err = callMCP(srv.handleListProviders, map[string]any{"category": "cloud"})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error filtering providers: %v", res)
	}
	text = res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "drive") || strings.Contains(text, "sftp") {
		t.Fatalf("expected only cloud providers, got: %s", text)
	}
}

func TestRemoteTools_CreateRemote(t *testing.T) {
	tmpDir := t.TempDir()
	profStore, err := profiles.NewStore(filepath.Join(tmpDir, "profiles.json"), filepath.Join(tmpDir, "history.json"))
	if err != nil {
		t.Fatalf("failed to create profStore: %v", err)
	}
	profSvc := profiles.NewService(profStore, nil, nil)

	oauthRunning := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/config/listremotes":
			_ = json.NewEncoder(w).Encode(map[string][]string{"remotes": {}})
		case "/config/create":
			w.WriteHeader(http.StatusOK)
		case "/config/oauthstatus":
			if oauthRunning {
				_ = json.NewEncoder(w).Encode(map[string]any{"status": "running", "authUrl": "http://127.0.0.1:53682/auth?state=xyz"})
			} else {
				_ = json.NewEncoder(w).Encode(map[string]any{"status": "idle"})
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := client.NewRcloneClient(server.URL, "", "")
	telSvc := services.NewTelemetryService(&config.Config{}, c)
	wizSvc := services.NewRemoteWizardService(&config.Config{}, c, nil, profSvc, telSvc)
	ctr := &app.Container{
		RemoteWizardService: wizSvc,
		ProfileService:      profSvc,
	}

	srv, err := NewServer(ctr, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// 1. Create S3 remote with profile bridge
	res, err := callMCP(srv.handleCreateRemote, map[string]any{
		"name":           "prod_s3",
		"type":           "s3",
		"parameters":     map[string]any{"provider": "AWS"},
		"create_profile": true,
		"local_path":     filepath.Join(tmpDir, "s3_local"),
		"profile_name":   "Prod S3 Profile",
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error creating remote: %v", res)
	}
	text := res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "prod_s3") || !strings.Contains(text, "Prod S3 Profile") {
		t.Fatalf("unexpected create remote result: %s", text)
	}

	// 2. Create interactive OAuth remote
	oauthRunning = true
	res, err = callMCP(srv.handleCreateRemote, map[string]any{
		"name": "gdrive_personal",
		"type": "drive",
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error creating oauth remote: %v", res)
	}
	text = res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "oauth_pending") || !strings.Contains(text, "127.0.0.1:53682") {
		t.Fatalf("expected oauth_pending status with authURL, got: %s", text)
	}
}

func TestRemoteTools_DeleteRemoteConfirmation(t *testing.T) {
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/config/delete" {
			deleted = true
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := client.NewRcloneClient(server.URL, "", "")
	wizSvc := services.NewRemoteWizardService(&config.Config{}, c, nil, nil, nil)
	ctr := &app.Container{
		RemoteWizardService: wizSvc,
	}

	srv, err := NewServer(ctr, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// 1. Call without confirmation token -> returns confirmation ticket
	res, err := callMCP(srv.handleDeleteRemote, map[string]any{
		"name": "target_rem",
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
	if deleted {
		t.Fatalf("remote should not be deleted before confirmation")
	}

	// 2. Call with valid token -> deletes remote
	res, err = callMCP(srv.handleDeleteRemote, map[string]any{
		"name":               "target_rem",
		"confirmation_token": ticketPayload.Token,
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error confirming delete: %v", res)
	}
	if !deleted {
		t.Fatalf("expected remote to be deleted after valid confirmation")
	}
}
