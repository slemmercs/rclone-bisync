package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/app"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

func TestHandleRemoteDedupeBatch_Success(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "file_a.txt")
	f2 := filepath.Join(tmpDir, "file_b.txt")
	_ = os.WriteFile(f1, []byte("data a"), 0600)
	_ = os.WriteFile(f2, []byte("data b"), 0600)

	cfg := &config.Config{LocalPath: tmpDir}
	container, err := app.NewContainer(cfg)
	if err != nil {
		t.Fatal(err)
	}
	container.ExplorerService = services.NewExplorerService(cfg, nil, nil)
	container.DedupeService = services.NewDedupeService(cfg, container.ExplorerService, nil, nil)

	s := &Server{
		Container:       container,
		Config:          cfg,
		ExplorerService: container.ExplorerService,
		DedupeService:   container.DedupeService,
		Mux:             http.NewServeMux(),
	}
	s.Mux.HandleFunc("POST /api/remotes/dedupe/batch", s.handleRemoteDedupeBatch)

	payload := models.DedupeBatchApplyRequest{
		Remote: "local",
		Path:   "",
		Actions: []models.DedupeStageAction{
			{Path: "file_a.txt", Action: "keep"},
			{Path: "file_b.txt", Action: "delete"},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/remotes/dedupe/batch", bytes.NewReader(body))
	w := httptest.NewRecorder()

	s.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var res models.DedupeBatchApplyResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.Status != "success" || res.SuccessCount != 2 {
		t.Errorf("unexpected batch result: %+v", res)
	}
	if _, err := os.Stat(f2); !os.IsNotExist(err) {
		t.Errorf("expected file_b.txt to be deleted")
	}
}

func TestHandleRemoteDedupeBatch_InvalidPayload(t *testing.T) {
	cfg := &config.Config{}
	s := &Server{
		Config: cfg,
		Mux:    http.NewServeMux(),
	}
	s.Mux.HandleFunc("POST /api/remotes/dedupe/batch", s.handleRemoteDedupeBatch)

	req := httptest.NewRequest(http.MethodPost, "/api/remotes/dedupe/batch", bytes.NewReader([]byte("{invalid-json")))
	w := httptest.NewRecorder()
	s.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid json, got %d", w.Code)
	}

	reqMissingRemote := httptest.NewRequest(http.MethodPost, "/api/remotes/dedupe/batch", bytes.NewReader([]byte(`{"actions":[]}`)))
	wMissing := httptest.NewRecorder()
	s.Mux.ServeHTTP(wMissing, reqMissingRemote)

	if wMissing.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for missing remote, got %d", wMissing.Code)
	}
}
