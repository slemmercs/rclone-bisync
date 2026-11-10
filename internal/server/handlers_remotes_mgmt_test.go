package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

func TestRemoteMgmt_Handlers(t *testing.T) {
	tmpDir := t.TempDir()
	profStore, err := profiles.NewStore(filepath.Join(tmpDir, "profiles.json"), filepath.Join(tmpDir, "history.json"))
	if err != nil {
		t.Fatalf("failed to create profStore: %v", err)
	}
	profSvc := profiles.NewService(profStore, nil, nil)

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/config/providers":
			resp := models.ProvidersResponse{
				Providers: []models.Provider{
					{Name: "drive", Description: "Google Drive"},
					{Name: "s3", Description: "Amazon S3"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		case "/config/listremotes":
			_ = json.NewEncoder(w).Encode(map[string][]string{"remotes": {"active_rem:"}})
		case "/config/create":
			w.WriteHeader(http.StatusOK)
		case "/config/delete":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	rcClient := client.NewRcloneClient(mockServer.URL, "", "")
	telSvc := services.NewTelemetryService(&config.Config{}, rcClient)
	wizSvc := services.NewRemoteWizardService(&config.Config{}, rcClient, nil, profSvc, telSvc)

	srv := &Server{
		RemoteWizardService: wizSvc,
		ProfileService:      profSvc,
		Mux:                 http.NewServeMux(),
	}
	srv.routes()

	// 1. GET /api/remotes/providers
	req := httptest.NewRequest(http.MethodGet, "/api/remotes/providers", nil)
	rec := httptest.NewRecorder()
	srv.Mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for providers, got %d", rec.Code)
	}
	var provResp map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&provResp)
	if provResp["success"] != true {
		t.Fatalf("expected success: true, got %+v", provResp)
	}

	// 2. POST /api/remotes/create (Validation failure: empty name)
	badBody, _ := json.Marshal(models.CreateRemoteBridgeRequest{Name: ""})
	req = httptest.NewRequest(http.MethodPost, "/api/remotes/create", bytes.NewReader(badBody))
	rec = httptest.NewRecorder()
	srv.Mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
	}

	// 3. POST /api/remotes/create (Success with Profile Bridge)
	syncDir := filepath.Join(tmpDir, "local_sync")
	validBody, _ := json.Marshal(models.CreateRemoteBridgeRequest{
		Name:          "prod_cloud",
		Type:          "drive",
		Parameters:    map[string]interface{}{"scope": "drive"},
		CreateProfile: true,
		LocalPath:     syncDir,
		RemoteSubpath: "team_drive",
		ProfileName:   "Production Cloud",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/remotes/create", bytes.NewReader(validBody))
	rec = httptest.NewRecorder()
	srv.Mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d (%s)", rec.Code, rec.Body.String())
	}
	var createRes models.CreateRemoteResult
	_ = json.NewDecoder(rec.Body).Decode(&createRes)
	if !createRes.Success || createRes.ProfileID == "" {
		t.Fatalf("expected successful remote and profile creation, got %+v", createRes)
	}

	// 4. DELETE /api/remotes/prod_cloud without force -> 409 Conflict because profile depends on it
	req = httptest.NewRequest(http.MethodDelete, "/api/remotes/prod_cloud", nil)
	rec = httptest.NewRecorder()
	srv.Mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d (%s)", rec.Code, rec.Body.String())
	}

	// 5. DELETE /api/remotes/prod_cloud?force=true -> 200 OK
	req = httptest.NewRequest(http.MethodDelete, "/api/remotes/prod_cloud?force=true", nil)
	rec = httptest.NewRecorder()
	srv.Mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with force=true, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestRemoteMgmt_ServiceUnavailable(t *testing.T) {
	srv := &Server{
		Mux: http.NewServeMux(),
	}
	srv.routes()

	req := httptest.NewRequest(http.MethodGet, "/api/remotes/providers", nil)
	rec := httptest.NewRecorder()
	srv.Mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 Service Unavailable, got %d", rec.Code)
	}
}
