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
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
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

func TestAuditRecords_ProfileNameResolution(t *testing.T) {
	srv, auditSvc := setupTestServerWithAudit(t)
	defer auditSvc.Close()

	// Setup profile store & service
	tmpDir := t.TempDir()
	pStore, err := profiles.NewStore(filepath.Join(tmpDir, "profiles.json"), filepath.Join(tmpDir, "history.json"))
	if err != nil {
		t.Fatalf("failed to create profile store: %v", err)
	}
	_ = pStore.Save(&profiles.Profile{
		ID:        "prof_abc123",
		Name:      "OneDrive (Personal)",
		Remote:    "onedrive:",
		LocalPath: filepath.Join(tmpDir, "onedrive"),
	}, "test setup")

	pService := profiles.NewService(pStore, nil, nil)
	srv.ProfileService = pService

	// Record audit events
	_, _ = auditSvc.RecordEvent("prof_abc123", audit.ActionSyncExecuted, "SYSTEM", map[string]any{"files": 5})
	_, _ = auditSvc.RecordEvent("default", audit.ActionSyncExecuted, "SYSTEM", map[string]any{"files": 1})

	// 1. GET /api/audit/records without filter -> should enrich profile_name
	req := httptest.NewRequest(http.MethodGet, "/api/audit/records", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Records []struct {
			ProfileID   string `json:"profile_id"`
			ProfileName string `json:"profile_name"`
		} `json:"records"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.Total != 2 {
		t.Fatalf("expected total 2, got %d", resp.Total)
	}
	if resp.Records[0].ProfileName != "Default" && resp.Records[0].ProfileName != "OneDrive (Personal)" {
		t.Fatalf("expected friendly name, got %s", resp.Records[0].ProfileName)
	}

	// 2. Filter by human friendly profile name "OneDrive (Personal)"
	req = httptest.NewRequest(http.MethodGet, "/api/audit/records?profile_id=OneDrive+(Personal)", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var filteredResp struct {
		Records []struct {
			ProfileID   string `json:"profile_id"`
			ProfileName string `json:"profile_name"`
		} `json:"records"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &filteredResp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if filteredResp.Total != 1 {
		t.Fatalf("expected 1 record when filtered by friendly name, got %d", filteredResp.Total)
	}
	if filteredResp.Records[0].ProfileID != "prof_abc123" || filteredResp.Records[0].ProfileName != "OneDrive (Personal)" {
		t.Fatalf("unexpected record details: %+v", filteredResp.Records[0])
	}
}

func TestHandleRemoteDedupe_AuditLogging(t *testing.T) {
	srv, auditSvc := setupTestServerWithAudit(t)
	defer auditSvc.Close()

	payload, _ := json.Marshal(map[string]any{
		"remote":  "gdrive",
		"path":    "documents",
		"mode":    "oldest",
		"by_hash": false,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/remotes/dedupe", strings.NewReader(string(payload)))
	req.Header.Set("X-Dashboard-Request", "1")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got: %d", w.Code)
	}

	records, total, err := auditSvc.Query(audit.Filter{Action: audit.ActionDedupeExecuted})
	if err != nil {
		t.Fatalf("failed to query audit records: %v", err)
	}
	if total != 1 || len(records) != 1 {
		t.Fatalf("expected 1 DEDUPE_EXECUTED record, got %d", total)
	}
	if records[0].Action != audit.ActionDedupeExecuted {
		t.Errorf("expected action %s, got %s", audit.ActionDedupeExecuted, records[0].Action)
	}
	if !strings.Contains(string(records[0].Details), `"remote":"gdrive"`) {
		t.Errorf("expected remote gdrive in details, got %s", string(records[0].Details))
	}
	if !strings.Contains(string(records[0].Details), `"run_id":"dedupe-`) {
		t.Errorf("expected run_id in details, got %s", string(records[0].Details))
	}
	if strings.Contains(string(records[0].Details), `"output"`) {
		t.Errorf("expected output to NOT be stored in audit ledger details, got %s", string(records[0].Details))
	}

	var resp models.DedupeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !strings.HasPrefix(resp.RunID, "dedupe-") {
		t.Errorf("expected response RunID to start with dedupe-, got %s", resp.RunID)
	}
}

func TestRemoteDedupe_ListModeLogs(t *testing.T) {
	srv, auditSvc := setupTestServerWithAudit(t)
	defer auditSvc.Close()

	payload, _ := json.Marshal(map[string]any{
		"remote":  "gdrive",
		"path":    "documents",
		"mode":    "list",
		"by_hash": true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/remotes/dedupe", strings.NewReader(string(payload)))
	req.Header.Set("X-Dashboard-Request", "1")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got: %d", w.Code)
	}

	var resp models.DedupeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !strings.HasPrefix(resp.RunID, "dedupe-") {
		t.Errorf("expected response RunID to start with dedupe-, got %s", resp.RunID)
	}

	// Verify no ActionDedupeExecuted audit record for list preview mode
	records, total, err := auditSvc.Query(audit.Filter{Action: audit.ActionDedupeExecuted})
	if err != nil {
		t.Fatalf("failed to query audit records: %v", err)
	}
	if total != 0 || len(records) != 0 {
		t.Fatalf("expected 0 DEDUPE_EXECUTED records for list mode, got %d", total)
	}
}
