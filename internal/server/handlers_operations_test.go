package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

func TestHandleOperations_MoveItem_Success(t *testing.T) {
	// Arrange
	srv, _, profileLocal := setupTestServerWithProfilesAndExplorer(t)
	srcFile := filepath.Join(profileLocal, "doc.txt")
	_ = os.WriteFile(srcFile, []byte("content"), 0644)

	reqBody, _ := json.Marshal(models.MoveRequest{
		ProfileID: "prof_onedrive",
		SrcRemote: "local",
		SrcPath:   "doc.txt",
		DstRemote: "local",
		DstPath:   "moved_doc.txt",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/operations/move", bytes.NewReader(reqBody))
	req.Header.Set("X-Dashboard-Request", "1")
	w := httptest.NewRecorder()

	// Act
	srv.Mux.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp models.MoveResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "success" {
		t.Fatalf("expected success, got %s: %s", resp.Status, resp.Error)
	}
	if _, err := os.Stat(filepath.Join(profileLocal, "moved_doc.txt")); os.IsNotExist(err) {
		t.Fatal("expected moved file to exist")
	}
}

func TestHandleOperations_MoveItem_TraversalAndCanary(t *testing.T) {
	// Arrange
	srv, _, profileLocal := setupTestServerWithProfilesAndExplorer(t)

	// 1. Traversal attempt
	reqBody, _ := json.Marshal(models.MoveRequest{
		ProfileID: "prof_onedrive",
		SrcRemote: "local",
		SrcPath:   "../escaped.txt",
		DstRemote: "local",
		DstPath:   "stolen.txt",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/operations/move", bytes.NewReader(reqBody))
	req.Header.Set("X-Dashboard-Request", "1")
	w := httptest.NewRecorder()
	srv.Mux.ServeHTTP(w, req)

	var resp models.MoveResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "error" {
		t.Fatalf("expected traversal error, got %s", resp.Status)
	}

	// 2. Canary protection
	_ = os.WriteFile(filepath.Join(profileLocal, "RCLONE_TEST"), []byte("canary"), 0644)
	canaryBody, _ := json.Marshal(models.MoveRequest{
		ProfileID: "prof_onedrive",
		SrcRemote: "local",
		SrcPath:   "RCLONE_TEST",
		DstRemote: "local",
		DstPath:   "canary_backup",
	})

	req2 := httptest.NewRequest(http.MethodPost, "/api/operations/move", bytes.NewReader(canaryBody))
	req2.Header.Set("X-Dashboard-Request", "1")
	w2 := httptest.NewRecorder()
	srv.Mux.ServeHTTP(w2, req2)

	var resp2 models.MoveResponse
	_ = json.NewDecoder(w2.Body).Decode(&resp2)
	if resp2.Status != "error" {
		t.Fatalf("expected canary error, got %s", resp2.Status)
	}
}

func TestHandleOperations_PurgeDirectory_Success(t *testing.T) {
	// Arrange
	srv, _, profileLocal := setupTestServerWithProfilesAndExplorer(t)
	purgeDir := filepath.Join(profileLocal, "purge_folder")
	_ = os.MkdirAll(purgeDir, 0755)
	_ = os.WriteFile(filepath.Join(purgeDir, "subfile.txt"), []byte("data"), 0644)

	reqBody, _ := json.Marshal(models.PurgeRequest{
		ProfileID: "prof_onedrive",
		Remote:    "local",
		Path:      "purge_folder",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/operations/purge", bytes.NewReader(reqBody))
	req.Header.Set("X-Dashboard-Request", "1")
	w := httptest.NewRecorder()

	// Act
	srv.Mux.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp models.PurgeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "success" {
		t.Fatalf("expected success, got %s: %s", resp.Status, resp.Error)
	}
	if _, err := os.Stat(purgeDir); !os.IsNotExist(err) {
		t.Fatal("expected directory to be deleted")
	}
}

func TestHandleOperations_PurgeDirectory_TraversalAndCanary(t *testing.T) {
	// Arrange
	srv, _, profileLocal := setupTestServerWithProfilesAndExplorer(t)

	// 1. Traversal attempt
	reqBody, _ := json.Marshal(models.PurgeRequest{
		ProfileID: "prof_onedrive",
		Remote:    "local",
		Path:      "../escaped",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/operations/purge", bytes.NewReader(reqBody))
	req.Header.Set("X-Dashboard-Request", "1")
	w := httptest.NewRecorder()
	srv.Mux.ServeHTTP(w, req)

	var resp models.PurgeResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "error" {
		t.Fatalf("expected traversal error, got %s", resp.Status)
	}

	// 2. Directory with canary
	canaryDir := filepath.Join(profileLocal, "protected_dir")
	_ = os.MkdirAll(canaryDir, 0755)
	_ = os.WriteFile(filepath.Join(canaryDir, "RCLONE_TEST"), []byte("canary"), 0644)

	canaryBody, _ := json.Marshal(models.PurgeRequest{
		ProfileID: "prof_onedrive",
		Remote:    "local",
		Path:      "protected_dir",
	})
	req2 := httptest.NewRequest(http.MethodPost, "/api/operations/purge", bytes.NewReader(canaryBody))
	req2.Header.Set("X-Dashboard-Request", "1")
	w2 := httptest.NewRecorder()
	srv.Mux.ServeHTTP(w2, req2)

	var resp2 models.PurgeResponse
	_ = json.NewDecoder(w2.Body).Decode(&resp2)
	if resp2.Status != "error" {
		t.Fatalf("expected canary error, got %s", resp2.Status)
	}
}

func TestHandleOperations_PublicLink(t *testing.T) {
	// Arrange
	srv, _, _ := setupTestServerWithProfilesAndExplorer(t)

	// Mock RC server for public link
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"url": "https://onedrive.live.com/sharedlink"}`))
	}))
	defer ts.Close()

	rcClient := client.NewRcloneClient(ts.URL, "", "")
	srv.ExplorerService = services.NewExplorerService(srv.Config, rcClient, srv.CanaryService)

	// 1. GET local -> unsupported
	reqLocal := httptest.NewRequest(http.MethodGet, "/api/operations/publiclink?remote=local&path=file.pdf", nil)
	reqLocal.Header.Set("X-Dashboard-Request", "1")
	wLocal := httptest.NewRecorder()
	srv.Mux.ServeHTTP(wLocal, reqLocal)

	var respLocal models.PublicLinkResponse
	_ = json.NewDecoder(wLocal.Body).Decode(&respLocal)
	if respLocal.Supported {
		t.Fatal("expected local public link to be unsupported")
	}

	// 2. GET remote -> success
	reqRemote := httptest.NewRequest(http.MethodGet, "/api/operations/publiclink?remote=onedrive:&path=file.pdf", nil)
	reqRemote.Header.Set("X-Dashboard-Request", "1")
	wRemote := httptest.NewRecorder()
	srv.Mux.ServeHTTP(wRemote, reqRemote)

	var respRemote models.PublicLinkResponse
	_ = json.NewDecoder(wRemote.Body).Decode(&respRemote)
	if !respRemote.Supported || respRemote.URL != "https://onedrive.live.com/sharedlink" {
		t.Fatalf("expected remote public link success, got %+v", respRemote)
	}

	// 3. Traversal -> rejected
	reqTrav := httptest.NewRequest(http.MethodGet, "/api/operations/publiclink?remote=onedrive:&path=../etc/passwd", nil)
	reqTrav.Header.Set("X-Dashboard-Request", "1")
	wTrav := httptest.NewRecorder()
	srv.Mux.ServeHTTP(wTrav, reqTrav)

	var respTrav models.PublicLinkResponse
	_ = json.NewDecoder(wTrav.Body).Decode(&respTrav)
	if respTrav.Status != "error" {
		t.Fatal("expected traversal error")
	}
}

