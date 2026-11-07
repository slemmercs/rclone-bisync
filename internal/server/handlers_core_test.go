package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

func setupTestServerWithCore(rcServerURL string) *Server {
	cfg := &config.Config{}
	var cl *client.RcloneClient
	if rcServerURL != "" {
		cl = client.NewRcloneClient(rcServerURL, "", "")
	}
	coreSvc := services.NewCoreService(cfg, cl)

	s := &Server{
		Config:      cfg,
		CoreService: coreSvc,
		Mux:         http.NewServeMux(),
	}
	s.routes()
	return s
}

func TestHandleGetBandwidthLimit_Success(t *testing.T) {
	// Arrange
	mockRC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"bytesPerSecond": 10485760, "rate": "10M"}`))
	}))
	defer mockRC.Close()

	srv := setupTestServerWithCore(mockRC.URL)

	// Act
	req := httptest.NewRequest(http.MethodGet, "/api/core/bwlimit", nil)
	w := httptest.NewRecorder()
	srv.Mux.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}
	var res models.BandwidthLimitResponse
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.RateFormatted != "10M" || res.BytesPerSecond != 10485760 {
		t.Errorf("unexpected limit response: %+v", res)
	}
}

func TestHandleSetBandwidthLimit_Success(t *testing.T) {
	// Arrange
	mockRC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"bytesPerSecond": 5242880, "rate": "5M"}`))
	}))
	defer mockRC.Close()

	srv := setupTestServerWithCore(mockRC.URL)

	// Act
	body := bytes.NewBufferString(`{"rate": "5M"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/core/bwlimit", body)
	w := httptest.NewRecorder()
	srv.Mux.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}
	var res models.BandwidthLimitResponse
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.RateFormatted != "5M" {
		t.Errorf("expected 5M, got %s", res.RateFormatted)
	}
}

func TestHandleSetBandwidthLimit_ValidationErrors(t *testing.T) {
	srv := setupTestServerWithCore("http://localhost:5572")

	// Invalid body
	req := httptest.NewRequest(http.MethodPost, "/api/core/bwlimit", bytes.NewBufferString(`invalid-json`))
	w := httptest.NewRecorder()
	srv.Mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", w.Code)
	}

	// Missing rate
	req = httptest.NewRequest(http.MethodPost, "/api/core/bwlimit", bytes.NewBufferString(`{"rate": "   "}`))
	w = httptest.NewRecorder()
	srv.Mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", w.Code)
	}
}

func TestHandleGetCoreStats_Success(t *testing.T) {
	// Arrange
	mockRC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"bytes": 1048576,
			"speed": 52428.8,
			"transfers": 1,
			"checks": 0,
			"errors": 0,
			"fatalErrors": false,
			"transferring": []
		}`))
	}))
	defer mockRC.Close()

	srv := setupTestServerWithCore(mockRC.URL)

	// Act
	req := httptest.NewRequest(http.MethodGet, "/api/core/stats", nil)
	w := httptest.NewRecorder()
	srv.Mux.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}
	var stats models.CoreStatsResponse
	if err := json.NewDecoder(w.Body).Decode(&stats); err != nil {
		t.Fatalf("failed to decode stats: %v", err)
	}
	if stats.Bytes != 1048576 || stats.Speed != 52428.8 {
		t.Errorf("unexpected stats: %+v", stats)
	}
}

func TestCoreEndpoints_NilService(t *testing.T) {
	// Arrange: Server without CoreService
	srv := &Server{
		Config: &config.Config{},
		Mux:    http.NewServeMux(),
	}
	srv.routes()

	// Act & Assert GET bwlimit
	req := httptest.NewRequest(http.MethodGet, "/api/core/bwlimit", nil)
	w := httptest.NewRecorder()
	srv.Mux.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 Service Unavailable, got %d", w.Code)
	}

	// Act & Assert POST bwlimit
	req = httptest.NewRequest(http.MethodPost, "/api/core/bwlimit", bytes.NewBufferString(`{"rate":"10M"}`))
	w = httptest.NewRecorder()
	srv.Mux.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 Service Unavailable, got %d", w.Code)
	}

	// Act & Assert GET stats
	req = httptest.NewRequest(http.MethodGet, "/api/core/stats", nil)
	w = httptest.NewRecorder()
	srv.Mux.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 Service Unavailable, got %d", w.Code)
	}
}
