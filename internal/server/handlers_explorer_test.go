package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

func setupTestServerWithProfilesAndExplorer(t *testing.T) (*Server, string, string) {
	t.Helper()
	tmpDir := t.TempDir()

	defaultLocal := filepath.Join(tmpDir, "default_local")
	profileLocal := filepath.Join(tmpDir, "profile_local")
	_ = os.MkdirAll(defaultLocal, 0755)
	_ = os.MkdirAll(profileLocal, 0755)

	// Create test files in both directories
	_ = os.WriteFile(filepath.Join(defaultLocal, "default_file.txt"), []byte("default"), 0644)
	_ = os.WriteFile(filepath.Join(profileLocal, "profile_file.txt"), []byte("profile"), 0644)

	cfg := &config.Config{
		BaseDir:   tmpDir,
		LocalPath: defaultLocal,
	}

	pStore, err := profiles.NewStore(filepath.Join(tmpDir, "profiles.json"), filepath.Join(tmpDir, "history.json"))
	if err != nil {
		t.Fatalf("failed to create profile store: %v", err)
	}

	_ = pStore.Save(&profiles.Profile{
		ID:        "prof_onedrive",
		Name:      "OneDrive Work",
		Remote:    "onedrive:",
		LocalPath: profileLocal,
	}, "init profile")

	pService := profiles.NewService(pStore, nil, nil)
	srv := NewServer(cfg, nil, nil)
	srv.ProfileService = pService

	canarySvc := services.NewCanaryService(cfg, nil)
	srv.CanaryService = canarySvc

	explorerSvc := services.NewExplorerService(cfg, nil, canarySvc)
	srv.ExplorerService = explorerSvc

	return srv, defaultLocal, profileLocal
}

func TestHandleBrowse_ProfileLocalRouting(t *testing.T) {
	srv, _, _ := setupTestServerWithProfilesAndExplorer(t)

	// 1. Browse with default profile (or empty profile_id) -> reads defaultLocal
	body, _ := json.Marshal(models.BrowseRequest{
		Remote:    "local",
		Path:      "",
		ProfileID: "default",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/browse", bytes.NewBuffer(body))
	req.Header.Set("X-Dashboard-Request", "1")
	req.Header.Set("Origin", "http://localhost:8080")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var res models.BrowseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	hasDefaultFile := false
	for _, it := range res.Items {
		if name, ok := it["Name"].(string); ok && name == "default_file.txt" {
			hasDefaultFile = true
		}
	}
	if !hasDefaultFile {
		t.Fatalf("expected default_file.txt in default browse, got: %+v", res.Items)
	}

	// 2. Browse with profile_id = "prof_onedrive" -> routes to profileLocal
	body, _ = json.Marshal(models.BrowseRequest{
		Remote:    "local",
		Path:      "",
		ProfileID: "prof_onedrive",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/browse", bytes.NewBuffer(body))
	req.Header.Set("X-Dashboard-Request", "1")
	req.Header.Set("Origin", "http://localhost:8080")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var profRes models.BrowseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &profRes); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	hasProfileFile := false
	for _, it := range profRes.Items {
		if name, ok := it["Name"].(string); ok && name == "profile_file.txt" {
			hasProfileFile = true
		}
	}
	if !hasProfileFile {
		t.Fatalf("expected profile_file.txt in prof_onedrive browse, got: %+v", profRes.Items)
	}
}

func TestHandleProvisionCanary_ProfileRouting(t *testing.T) {
	srv, _, profileLocal := setupTestServerWithProfilesAndExplorer(t)

	body, _ := json.Marshal(models.BrowseRequest{
		Remote:    "local",
		Path:      "",
		ProfileID: "prof_onedrive",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/browse/canary", bytes.NewBuffer(body))
	req.Header.Set("X-Dashboard-Request", "1")
	req.Header.Set("Origin", "http://localhost:8080")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	canaryPath := filepath.Join(profileLocal, "RCLONE_TEST")
	if _, err := os.Stat(canaryPath); os.IsNotExist(err) {
		t.Fatalf("expected RCLONE_TEST to exist in profile local folder: %s", canaryPath)
	}
}

func TestHandleDeleteItem_ProfileRouting(t *testing.T) {
	srv, _, profileLocal := setupTestServerWithProfilesAndExplorer(t)

	targetFile := filepath.Join(profileLocal, "profile_file.txt")
	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		t.Fatalf("target file does not exist before delete")
	}

	body, _ := json.Marshal(models.DeleteRequest{
		Remote:    "local",
		Path:      "profile_file.txt",
		IsDir:     false,
		ProfileID: "prof_onedrive",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/browse/delete", bytes.NewBuffer(body))
	req.Header.Set("X-Dashboard-Request", "1")
	req.Header.Set("Origin", "http://localhost:8080")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if _, err := os.Stat(targetFile); !os.IsNotExist(err) {
		t.Fatalf("expected profile_file.txt to be deleted")
	}
}

func TestHandleCopyItem_ProfileRouting(t *testing.T) {
	srv, _, profileLocal := setupTestServerWithProfilesAndExplorer(t)

	var lastSrcFs, lastDstFs string
	mockRC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/operations/copyfile" {
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if src, ok := body["srcFs"].(string); ok {
				lastSrcFs = src
			}
			if dst, ok := body["dstFs"].(string); ok {
				lastDstFs = dst
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockRC.Close()

	rcClient := client.NewRcloneClient(mockRC.URL, "user", "pass")
	srv.ExplorerService.Client = rcClient

	body, _ := json.Marshal(models.CopyRequest{
		SrcRemote: "local",
		SrcPath:   "profile_file.txt",
		DstRemote: "local",
		DstPath:   "profile_file_copy.txt",
		IsDir:     false,
		ProfileID: "prof_onedrive",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/browse/copy", bytes.NewBuffer(body))
	req.Header.Set("X-Dashboard-Request", "1")
	req.Header.Set("Origin", "http://localhost:8080")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var res models.CopyResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if res.Status != "success" {
		t.Fatalf("expected status success, got: %+v", res)
	}

	// Verify the mocked copy request received the profile's local directory path
	if lastSrcFs != profileLocal || lastDstFs != profileLocal {
		t.Fatalf("expected srcFs and dstFs to be %s, got srcFs=%s, dstFs=%s", profileLocal, lastSrcFs, lastDstFs)
	}
}

func TestHandleCopyItem_RemoteSubdirectoryDestination(t *testing.T) {
	srv, _, profileLocal := setupTestServerWithProfilesAndExplorer(t)

	var lastSrcFs, lastSrcRemote, lastDstFs, lastDstRemote string
	mockRC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/operations/copyfile" {
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if src, ok := body["srcFs"].(string); ok {
				lastSrcFs = src
			}
			if srcRem, ok := body["srcRemote"].(string); ok {
				lastSrcRemote = srcRem
			}
			if dst, ok := body["dstFs"].(string); ok {
				lastDstFs = dst
			}
			if dstRem, ok := body["dstRemote"].(string); ok {
				lastDstRemote = dstRem
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockRC.Close()

	rcClient := client.NewRcloneClient(mockRC.URL, "user", "pass")
	srv.ExplorerService.Client = rcClient

	tests := []struct {
		name          string
		dstRemote     string
		dstPath       string
		wantDstFs     string
		wantDstRemote string
	}{
		{
			name:          "remote root destination",
			dstRemote:     "onedrive",
			dstPath:       "",
			wantDstFs:     "onedrive:",
			wantDstRemote: "profile_file.txt",
		},
		{
			name:          "remote subdirectory destination without trailing colon",
			dstRemote:     "onedrive",
			dstPath:       "Documents",
			wantDstFs:     "onedrive:Documents",
			wantDstRemote: "profile_file.txt",
		},
		{
			name:          "remote nested subdirectory destination",
			dstRemote:     "onedrive",
			dstPath:       "Documents/Projects",
			wantDstFs:     "onedrive:Documents/Projects",
			wantDstRemote: "profile_file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lastSrcFs, lastSrcRemote, lastDstFs, lastDstRemote = "", "", "", ""
			body, _ := json.Marshal(models.CopyRequest{
				SrcRemote: "local",
				SrcPath:   "profile_file.txt",
				DstRemote: tt.dstRemote,
				DstPath:   tt.dstPath,
				IsDir:     false,
				ProfileID: "prof_onedrive",
			})
			req := httptest.NewRequest(http.MethodPost, "/api/browse/copy", bytes.NewBuffer(body))
			req.Header.Set("X-Dashboard-Request", "1")
			req.Header.Set("Origin", "http://localhost:8080")
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", w.Code)
			}

			if lastSrcFs != profileLocal {
				t.Errorf("srcFs = %q, want %q", lastSrcFs, profileLocal)
			}
			if lastSrcRemote != "profile_file.txt" {
				t.Errorf("srcRemote = %q, want 'profile_file.txt'", lastSrcRemote)
			}
			if lastDstFs != tt.wantDstFs {
				t.Errorf("dstFs = %q, want %q", lastDstFs, tt.wantDstFs)
			}
			if lastDstRemote != tt.wantDstRemote {
				t.Errorf("dstRemote = %q, want %q", lastDstRemote, tt.wantDstRemote)
			}
		})
	}
}
