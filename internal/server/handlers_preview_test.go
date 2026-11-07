package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/app"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

func setupPreviewTestServer(t *testing.T) (*Server, string) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "sample.txt")
	if err := os.WriteFile(testFile, []byte("Preview test content"), 0600); err != nil {
		t.Fatal(err)
	}

	subDir := filepath.Join(tmpDir, "docs")
	if err := os.Mkdir(subDir, 0750); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{LocalPath: tmpDir}
	container, err := app.NewContainer(cfg)
	if err != nil {
		t.Fatal(err)
	}
	container.ExplorerService = services.NewExplorerService(cfg, nil, nil)

	s := &Server{
		Container:       container,
		Config:          cfg,
		ExplorerService: container.ExplorerService,
		Mux:             http.NewServeMux(),
	}
	s.Mux.HandleFunc("GET /api/explorer/preview", s.handleExplorerPreview)
	s.Mux.HandleFunc("GET /api/explorer/download", s.handleExplorerDownload)

	return s, tmpDir
}

func TestHandlePreview_Success(t *testing.T) {
	s, _ := setupPreviewTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/explorer/preview?remote=local&path=sample.txt", nil)
	w := httptest.NewRecorder()
	s.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var res models.FilePreviewResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if res.Name != "sample.txt" || res.Content != "Preview test content" {
		t.Errorf("unexpected preview response: %+v", res)
	}
}

func TestHandlePreview_MissingPath(t *testing.T) {
	s, _ := setupPreviewTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/explorer/preview?remote=local", nil)
	w := httptest.NewRecorder()
	s.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for missing path, got %d", w.Code)
	}
}

func TestHandlePreview_TraversalForbidden(t *testing.T) {
	s, _ := setupPreviewTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/explorer/preview?remote=local&path=../../etc/passwd", nil)
	w := httptest.NewRecorder()
	s.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403 for path traversal, got %d", w.Code)
	}
}

func TestHandlePreview_Directory(t *testing.T) {
	s, _ := setupPreviewTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/explorer/preview?remote=local&path=docs", nil)
	w := httptest.NewRecorder()
	s.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 when previewing directory, got %d", w.Code)
	}
}

func TestHandleDownload_Success(t *testing.T) {
	s, _ := setupPreviewTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/explorer/download?remote=local&path=sample.txt", nil)
	w := httptest.NewRecorder()
	s.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	body, _ := io.ReadAll(w.Body)
	if string(body) != "Preview test content" {
		t.Errorf("unexpected downloaded body: %s", string(body))
	}
	disp := w.Header().Get("Content-Disposition")
	if !strings.Contains(disp, `filename="sample.txt"`) {
		t.Errorf("unexpected Content-Disposition: %s", disp)
	}
}

func TestHandleDownload_Inline(t *testing.T) {
	s, tmpDir := setupPreviewTestServer(t)

	pdfFile := filepath.Join(tmpDir, "sample.pdf")
	if err := os.WriteFile(pdfFile, []byte("%PDF-1.4 test"), 0600); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/explorer/download?remote=local&path=sample.pdf&inline=true", nil)
	w := httptest.NewRecorder()
	s.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	disp := w.Header().Get("Content-Disposition")
	if !strings.Contains(disp, `inline; filename="sample.pdf"`) {
		t.Errorf("expected inline Content-Disposition, got: %s", disp)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/pdf" {
		t.Errorf("expected application/pdf Content-Type, got: %s", contentType)
	}
}

func TestHandleDownload_MissingPathAndDirectory(t *testing.T) {
	s, _ := setupPreviewTestServer(t)

	// Missing path
	req := httptest.NewRequest(http.MethodGet, "/api/explorer/download?remote=local", nil)
	w := httptest.NewRecorder()
	s.Mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	// Directory
	reqDir := httptest.NewRequest(http.MethodGet, "/api/explorer/download?remote=local&path=docs", nil)
	wDir := httptest.NewRecorder()
	s.Mux.ServeHTTP(wDir, reqDir)
	if wDir.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 when downloading directory, got %d", wDir.Code)
	}
}
