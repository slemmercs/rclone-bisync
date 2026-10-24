package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

func setupTestServerWithAudit(t *testing.T) (*Server, *services.AuditService) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_audit.db")

	db, err := audit.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open audit db: %v", err)
	}

	auditSvc := services.NewAuditService(db, nil)
	cfg := &config.Config{
		BaseDir:     tmpDir,
		AuditDBPath: dbPath,
	}

	s := NewServer(cfg, nil, nil)
	s.AuditService = auditSvc
	return s, auditSvc
}

func TestAuditEndpoints(t *testing.T) {
	srv, auditSvc := setupTestServerWithAudit(t)
	defer auditSvc.Close()

	// 1. Initial GET /api/audit/records (empty)
	req := httptest.NewRequest(http.MethodGet, "/api/audit/records", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var emptyResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &emptyResp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if emptyResp["status"] != "success" || emptyResp["total"].(float64) != 0 {
		t.Fatalf("unexpected empty response: %+v", emptyResp)
	}

	// 2. Append records
	_, _ = auditSvc.RecordEvent("gdrive", audit.ActionSyncExecuted, "SYSTEM", map[string]any{"ok": true})
	_, _ = auditSvc.RecordEvent("gdrive", audit.ActionConflictResolved, "USER", map[string]any{"file": "test.txt"})

	// 3. GET /api/audit/records (with data)
	req = httptest.NewRequest(http.MethodGet, "/api/audit/records?limit=10&page=1", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["total"].(float64) != 2 {
		t.Fatalf("expected total 2, got %v", resp["total"])
	}

	// 4. GET /api/audit/verify
	req = httptest.NewRequest(http.MethodGet, "/api/audit/verify", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var verifyResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &verifyResp); err != nil {
		t.Fatalf("failed to parse verify response: %v", err)
	}
	if verifyResp["valid"] != true || verifyResp["total_records"].(float64) != 2 {
		t.Fatalf("expected valid=true and total=2, got: %+v", verifyResp)
	}
	if verifyResp["head_hash"] == "" || verifyResp["head_hash"] == audit.GenesisHash {
		t.Fatalf("expected non-genesis head hash, got %v", verifyResp["head_hash"])
	}

	// 5. GET /api/audit/export?format=jsonl
	req = httptest.NewRequest(http.MethodGet, "/api/audit/export?format=jsonl", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "ndjson") {
		t.Fatalf("expected ndjson content-type, got %s", w.Header().Get("Content-Type"))
	}
	lines := strings.Split(strings.TrimSpace(w.Body.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 exported lines, got %d", len(lines))
	}

	// 6. GET /api/audit/export?format=csv
	req = httptest.NewRequest(http.MethodGet, "/api/audit/export?format=csv", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "csv") {
		t.Fatalf("expected csv content-type, got %s", w.Header().Get("Content-Type"))
	}
	csvLines := strings.Split(strings.TrimSpace(w.Body.String()), "\n")
	if len(csvLines) != 3 { // Header + 2 rows
		t.Fatalf("expected 3 csv lines, got %d", len(csvLines))
	}
}

func TestAuditEndpoints_NilService(t *testing.T) {
	srv := NewServer(&config.Config{}, nil, nil)
	srv.AuditService = nil

	// GET /api/audit/records
	req := httptest.NewRequest(http.MethodGet, "/api/audit/records", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on nil service, got %d", w.Code)
	}

	// GET /api/audit/verify
	req = httptest.NewRequest(http.MethodGet, "/api/audit/verify", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on nil service, got %d", w.Code)
	}

	// GET /api/audit/export
	req = httptest.NewRequest(http.MethodGet, "/api/audit/export", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 on nil service export, got %d", w.Code)
	}
}
