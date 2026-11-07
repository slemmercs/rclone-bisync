package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

func setupEncryptionTestServer(t *testing.T) (*Server, string) {
	tmpDir := t.TempDir()
	confPath := filepath.Join(tmpDir, "rclone.conf")

	cfg := &config.Config{RcloneConfigFile: confPath}
	mockRunner := func(ctx context.Context, name string, env []string, args ...string) (string, error) {
		return "success", nil
	}
	encSvc := services.NewConfigEncryptionService(cfg, nil, mockRunner)

	s := &Server{
		Config:                  cfg,
		ConfigEncryptionService: encSvc,
		Mux:                     http.NewServeMux(),
	}
	s.Mux.HandleFunc("GET /api/config/encryption", s.handleGetConfigEncryption)
	s.Mux.HandleFunc("POST /api/config/encryption/set", s.handleSetConfigEncryption)
	s.Mux.HandleFunc("POST /api/config/encryption/remove", s.handleRemoveConfigEncryption)

	return s, confPath
}

func TestHandleGetConfigEncryption(t *testing.T) {
	s, _ := setupEncryptionTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/config/encryption", nil)
	w := httptest.NewRecorder()
	s.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var res models.EncryptionStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if res.IsEncrypted {
		t.Errorf("expected not encrypted, got true")
	}
}

func TestHandleSetConfigEncryption_Success(t *testing.T) {
	s, _ := setupEncryptionTestServer(t)

	payload := models.SetEncryptionRequest{Password: "secret123"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/config/encryption/set", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var res models.EncryptionActionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if res.Status != "success" {
		t.Errorf("expected status success, got %s", res.Status)
	}
}

func TestHandleSetConfigEncryption_EmptyPassword(t *testing.T) {
	s, _ := setupEncryptionTestServer(t)

	payload := models.SetEncryptionRequest{Password: ""}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/config/encryption/set", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for empty password, got %d", w.Code)
	}
}

func TestHandleRemoveConfigEncryption(t *testing.T) {
	s, _ := setupEncryptionTestServer(t)

	payload := models.RemoveEncryptionRequest{Password: "secret123"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/config/encryption/remove", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}
