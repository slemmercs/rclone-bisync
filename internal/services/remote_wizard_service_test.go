package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
)

func TestRemoteWizard_CategorizeProvider(t *testing.T) {
	tests := []struct {
		pType    string
		expected models.ProviderCategory
	}{
		{"drive", models.CategoryCloud},
		{"onedrive", models.CategoryCloud},
		{"dropbox", models.CategoryCloud},
		{"s3", models.CategoryObject},
		{"b2", models.CategoryObject},
		{"azureblob", models.CategoryObject},
		{"sftp", models.CategoryProtocol},
		{"webdav", models.CategoryProtocol},
		{"alias", models.CategoryVirtual},
		{"crypt", models.CategoryVirtual},
		{"unknown_backend", models.CategoryOther},
	}

	for _, tc := range tests {
		got := CategorizeProvider(tc.pType)
		if got != tc.expected {
			t.Errorf("CategorizeProvider(%q) = %q; want %q", tc.pType, got, tc.expected)
		}
	}
}

func TestRemoteWizard_GetProvidersCatalog(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/config/providers" {
			callCount++
			resp := models.ProvidersResponse{
				Providers: []models.Provider{
					{Name: "drive", Description: "Google Drive"},
					{Name: "s3", Description: "Amazon S3"},
					{Name: "sftp", Description: "SSH/SFTP"},
					{Name: "hidden_type", Description: "Secret", Hide: true},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := client.NewRcloneClient(server.URL, "", "")
	svc := NewRemoteWizardService(&config.Config{}, c, nil, nil, nil)

	cat, err := svc.GetProvidersCatalog(context.Background(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cat.Providers) != 3 {
		t.Fatalf("expected 3 non-hidden providers, got %d", len(cat.Providers))
	}
	for _, p := range cat.Providers {
		if p.Name == "drive" && !p.SupportsOAuth {
			t.Errorf("expected drive to have SupportsOAuth=true, got false")
		}
		if p.Name == "s3" && p.SupportsOAuth {
			t.Errorf("expected s3 to have SupportsOAuth=false, got true")
		}
		if p.Name == "sftp" && p.SupportsOAuth {
			t.Errorf("expected sftp to have SupportsOAuth=false, got true")
		}
	}
	if callCount != 1 {
		t.Fatalf("expected 1 call, got %d", callCount)
	}

	catCached, err := svc.GetProvidersCatalog(context.Background(), false)
	if err != nil {
		t.Fatalf("unexpected cached error: %v", err)
	}
	if len(catCached.Providers) != 3 {
		t.Fatalf("expected 3 cached providers, got %d", len(catCached.Providers))
	}
	if callCount != 1 {
		t.Fatalf("expected call count to remain 1, got %d", callCount)
	}

	_, err = svc.GetProvidersCatalog(context.Background(), true)
	if err != nil {
		t.Fatalf("unexpected error on force refresh: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected call count 2 after force refresh, got %d", callCount)
	}
}

func TestRemoteWizard_CreateRemote(t *testing.T) {
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
			_ = json.NewEncoder(w).Encode(map[string][]string{"remotes": {"existing_rem:"}})
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
	telSvc := NewTelemetryService(&config.Config{}, c)
	svc := NewRemoteWizardService(&config.Config{}, c, nil, profSvc, telSvc)

	// Empty name validation
	_, err = svc.CreateRemote(context.Background(), models.CreateRemoteBridgeRequest{Name: ""}, "test")
	if err == nil {
		t.Fatalf("expected error for empty name")
	}

	// Invalid characters
	_, err = svc.CreateRemote(context.Background(), models.CreateRemoteBridgeRequest{Name: "invalid name/slash"}, "test")
	if err == nil {
		t.Fatalf("expected error for invalid characters")
	}

	// Existing remote collision
	_, err = svc.CreateRemote(context.Background(), models.CreateRemoteBridgeRequest{Name: "existing_rem", Type: "s3"}, "test")
	if err == nil {
		t.Fatalf("expected error for existing remote collision")
	}

	// Successful creation with Profile Bridge
	res, err := svc.CreateRemote(context.Background(), models.CreateRemoteBridgeRequest{
		Name:          "new_backup",
		Type:          "s3",
		Parameters:    map[string]interface{}{"provider": "AWS"},
		CreateProfile: true,
		LocalPath:     filepath.Join(tmpDir, "sync_folder"),
		RemoteSubpath: "backups/prod",
		ProfileName:   "Backup Profile",
		Schedule:      "0 * * * *",
	}, "test-actor")

	if err != nil {
		t.Fatalf("unexpected error creating remote: %v", err)
	}
	if !res.Success || res.RemoteName != "new_backup" {
		t.Fatalf("unexpected response: %+v", res)
	}
	if res.NeedsAuth || res.Status != "created" {
		t.Fatalf("expected created status and needsAuth=false, got: %+v", res)
	}
	if res.ProfileID == "" || res.ProfileName != "Backup Profile" {
		t.Fatalf("expected profile ID and name, got: %+v", res)
	}

	prof, err := profStore.Get(res.ProfileID)
	if err != nil || prof.FullRemoteTarget() != "new_backup:backups/prod" {
		t.Fatalf("expected created profile with target new_backup:backups/prod, got %+v (err: %v)", prof, err)
	}

	// Test OAuth Pending Flow
	oauthRunning = true
	resOAuth, err := svc.CreateRemote(context.Background(), models.CreateRemoteBridgeRequest{
		Name: "gdrive_interactive",
		Type: "drive",
	}, "test-actor")
	if err != nil {
		t.Fatalf("unexpected error creating oauth remote: %v", err)
	}
	if !resOAuth.NeedsAuth || resOAuth.Status != "oauth_pending" || resOAuth.AuthURL != "http://127.0.0.1:53682/auth?state=xyz" {
		t.Fatalf("expected oauth_pending status with authURL, got: %+v", resOAuth)
	}
}

func TestRemoteWizard_DeleteRemote(t *testing.T) {
	tmpDir := t.TempDir()
	profStore, err := profiles.NewStore(filepath.Join(tmpDir, "profiles.json"), filepath.Join(tmpDir, "history.json"))
	if err != nil {
		t.Fatalf("failed to create profStore: %v", err)
	}
	profSvc := profiles.NewService(profStore, nil, nil)

	_, err = profSvc.Create(context.Background(), &profiles.Profile{
		ID:         "prof-1",
		Name:       "Production Sync",
		Remote:     "target_remote",
		RemotePath: "bucket/data",
		LocalPath:  tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

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
	svc := NewRemoteWizardService(&config.Config{}, c, nil, profSvc, nil)

	// Blocked without force
	err = svc.DeleteRemote(context.Background(), "target_remote", false, "test")
	if err == nil {
		t.Fatalf("expected error deleting remote referenced by profile")
	}

	// Succeeds with force=true
	err = svc.DeleteRemote(context.Background(), "target_remote", true, "test")
	if err != nil {
		t.Fatalf("expected deletion to succeed with force: %v", err)
	}
	if !deleted {
		t.Fatalf("expected delete request to be sent to mock server")
	}
}
