package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
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
	s.Mux.HandleFunc("POST /api/config/encryption/save-passphrase", s.handleSaveDaemonPassphrase)

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

	// Test empty password rejection
	payloadEmpty := models.RemoveEncryptionRequest{Password: ""}
	bodyEmpty, _ := json.Marshal(payloadEmpty)
	reqEmpty := httptest.NewRequest(http.MethodPost, "/api/config/encryption/remove", bytes.NewReader(bodyEmpty))
	wEmpty := httptest.NewRecorder()
	s.Mux.ServeHTTP(wEmpty, reqEmpty)

	if wEmpty.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for empty password, got %d", wEmpty.Code)
	}

	// Test valid password success
	payload := models.RemoveEncryptionRequest{Password: "secret123"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/config/encryption/remove", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestHandleSaveDaemonPassphrase(t *testing.T) {
	s, _ := setupEncryptionTestServer(t)

	// Case 1: Empty password
	payloadEmpty := models.SaveDaemonPassphraseRequest{Password: "   "}
	bodyEmpty, _ := json.Marshal(payloadEmpty)
	reqEmpty := httptest.NewRequest(http.MethodPost, "/api/config/encryption/save-passphrase", bytes.NewReader(bodyEmpty))
	wEmpty := httptest.NewRecorder()
	s.Mux.ServeHTTP(wEmpty, reqEmpty)

	if wEmpty.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for empty password, got %d", wEmpty.Code)
	}

	// Case 2: Invalid JSON
	reqBadJSON := httptest.NewRequest(http.MethodPost, "/api/config/encryption/save-passphrase", bytes.NewReader([]byte("{invalid-json")))
	wBadJSON := httptest.NewRecorder()
	s.Mux.ServeHTTP(wBadJSON, reqBadJSON)

	if wBadJSON.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid JSON, got %d", wBadJSON.Code)
	}

	// Case 3: Valid password
	payloadValid := models.SaveDaemonPassphraseRequest{Password: "validPass123"}
	bodyValid, _ := json.Marshal(payloadValid)
	reqValid := httptest.NewRequest(http.MethodPost, "/api/config/encryption/save-passphrase", bytes.NewReader(bodyValid))
	wValid := httptest.NewRecorder()
	s.Mux.ServeHTTP(wValid, reqValid)

	if wValid.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", wValid.Code, wValid.Body.String())
	}

	var res models.EncryptionActionResponse
	if err := json.Unmarshal(wValid.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.Status != "success" {
		t.Errorf("expected status 'success', got %s", res.Status)
	}

	// Case 4: Failing runner (invalid passphrase)
	failRunner := func(ctx context.Context, name string, env []string, args ...string) (string, error) {
		return "", fmt.Errorf("decryption failed")
	}
	s.ConfigEncryptionService.Runner = failRunner

	reqFail := httptest.NewRequest(http.MethodPost, "/api/config/encryption/save-passphrase", bytes.NewReader(bodyValid))
	wFail := httptest.NewRecorder()
	s.Mux.ServeHTTP(wFail, reqFail)

	if wFail.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 when decryption fails, got %d", wFail.Code)
	}
}

func TestHandleConfigEncryption_NilService(t *testing.T) {
	s := &Server{
		Mux: http.NewServeMux(),
	}
	s.Mux.HandleFunc("GET /api/config/encryption", s.handleGetConfigEncryption)
	s.Mux.HandleFunc("POST /api/config/encryption/set", s.handleSetConfigEncryption)
	s.Mux.HandleFunc("POST /api/config/encryption/remove", s.handleRemoveConfigEncryption)
	s.Mux.HandleFunc("POST /api/config/encryption/save-passphrase", s.handleSaveDaemonPassphrase)

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/config/encryption", ""},
		{http.MethodPost, "/api/config/encryption/set", `{"password":"abc"}`},
		{http.MethodPost, "/api/config/encryption/remove", `{"password":"abc"}`},
		{http.MethodPost, "/api/config/encryption/save-passphrase", `{"password":"abc"}`},
	}

	for _, ep := range endpoints {
		var req *http.Request
		if ep.body != "" {
			req = httptest.NewRequest(ep.method, ep.path, strings.NewReader(ep.body))
		} else {
			req = httptest.NewRequest(ep.method, ep.path, nil)
		}
		w := httptest.NewRecorder()
		s.Mux.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 for nil service at %s %s, got %d", ep.method, ep.path, w.Code)
		}
	}
}
