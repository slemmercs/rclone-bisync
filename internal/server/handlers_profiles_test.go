package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
)

func setupTestServerWithProfiles(t *testing.T) (*Server, *profiles.Service) {
	t.Helper()
	tmpDir := t.TempDir()
	profPath := filepath.Join(tmpDir, "profiles.json")
	histPath := filepath.Join(tmpDir, "profiles_history.json")

	store, err := profiles.NewStore(profPath, histPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	executed := make(map[string]bool)
	executor := func(ctx context.Context, p *profiles.Profile, opts profiles.SyncOptions) error {
		executed[p.ID] = true
		return nil
	}

	sched := profiles.NewScheduler(store, executor)
	_ = sched.Start(context.Background())
	t.Cleanup(func() {
		sched.Stop()
	})
	svc := profiles.NewService(store, nil, sched)

	cfg := &config.Config{
		BaseDir:             tmpDir,
		ProfilesFile:        profPath,
		ProfilesHistoryFile: histPath,
	}

	s := NewServer(cfg, nil, nil)
	s.ProfileService = svc
	return s, svc
}

func doJSONReq(s *Server, method, target string, body any, withHeader bool) *httptest.ResponseRecorder {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, target, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	if withHeader {
		req.Header.Set("X-Dashboard-Request", "1")
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	return w
}

func TestProfiles_CRUD_Endpoints(t *testing.T) {
	s, _ := setupTestServerWithProfiles(t)

	// 1. Initial GET /api/profiles (empty)
	w := doJSONReq(s, http.MethodGet, "/api/profiles", nil, false)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var listResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &listResp)
	if listResp["status"] != "success" || listResp["total"].(float64) != 0 {
		t.Fatalf("expected 0 profiles, got %+v", listResp)
	}

	// 2. CSRF rejection on POST /api/profiles without header
	workPath := filepath.Join(s.Config.BaseDir, "work")
	newProf := profiles.Profile{
		ID:         "prof-1",
		Name:       "Work Google Drive",
		Remote:     "work-drive",
		RemotePath: "documents",
		LocalPath:  workPath,
		Schedule:   "manual",
		Enabled:    true,
	}
	w = doJSONReq(s, http.MethodPost, "/api/profiles", newProf, false)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 CSRF rejection without header, got %d", w.Code)
	}

	// 3. Valid POST /api/profiles
	w = doJSONReq(s, http.MethodPost, "/api/profiles", newProf, true)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	// 4. Duplicate POST /api/profiles -> 409 Conflict
	w = doJSONReq(s, http.MethodPost, "/api/profiles", newProf, true)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for duplicate ID, got %d", w.Code)
	}

	// 5. GET /api/profiles/{id}
	w = doJSONReq(s, http.MethodGet, "/api/profiles/prof-1", nil, false)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var getResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &getResp)
	if getResp["status"] != "success" {
		t.Fatalf("expected success, got %+v", getResp)
	}

	// 6. GET /api/profiles/{id} not found
	w = doJSONReq(s, http.MethodGet, "/api/profiles/nonexistent", nil, false)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	// 7. PUT /api/profiles/{id}
	updatePayload := map[string]any{
		"name":       "Updated Work Drive",
		"remote":     "work-drive",
		"local_path": workPath,
		"schedule":   "manual",
		"reason":     "Updating display name",
	}
	w = doJSONReq(s, http.MethodPut, "/api/profiles/prof-1", updatePayload, true)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 8. PUT /api/profiles/{id} not found
	w = doJSONReq(s, http.MethodPut, "/api/profiles/missing", updatePayload, true)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing profile update, got %d", w.Code)
	}
}

func TestProfiles_CloneRollbackSyncEndpoints(t *testing.T) {
	s, svc := setupTestServerWithProfiles(t)

	// Create base profile
	drivePath := filepath.Join(s.Config.BaseDir, "drive")
	p := &profiles.Profile{
		ID:         "orig-1",
		Name:       "Original Drive",
		Enabled:    true,
		Remote:     "gdrive",
		LocalPath:  drivePath,
		Schedule:   "manual",
		ReviewMode: profiles.ReviewDeletions,
	}
	_, err := svc.Create(context.Background(), p)
	if err != nil {
		t.Fatalf("failed to create base profile: %v", err)
	}

	// 1. Clone profile: POST /api/profiles/{id}/clone
	cloneBody := map[string]string{
		"id":   "clone-1",
		"name": "Cloned Drive",
	}
	w := doJSONReq(s, http.MethodPost, "/api/profiles/orig-1/clone", cloneBody, true)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on clone, got %d: %s", w.Code, w.Body.String())
	}

	// 2. Clone duplicate target ID -> 409 Conflict
	w = doJSONReq(s, http.MethodPost, "/api/profiles/orig-1/clone", cloneBody, true)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict on duplicate clone, got %d", w.Code)
	}

	// 3. Update clone to create version history
	cloneUpdated := *p
	cloneUpdated.ID = "clone-1"
	cloneUpdated.Name = "Cloned Drive V2"
	_, _ = svc.Update(context.Background(), &cloneUpdated, "bumped version")

	// Check history endpoint: GET /api/profiles/{id}/history
	w = doJSONReq(s, http.MethodGet, "/api/profiles/clone-1/history", nil, false)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on history, got %d", w.Code)
	}
	var histResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &histResp)
	histItems := histResp["history"].([]any)
	if len(histItems) < 2 {
		t.Fatalf("expected at least 2 versions, got %d", len(histItems))
	}

	// 4. Rollback clone: POST /api/profiles/{id}/rollback
	rollbackBody := map[string]int{
		"target_version": 1,
	}
	w = doJSONReq(s, http.MethodPost, "/api/profiles/clone-1/rollback", rollbackBody, true)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on rollback, got %d: %s", w.Code, w.Body.String())
	}

	// 5. Trigger sync: POST /api/profiles/{id}/sync
	syncBody := map[string]bool{"force": false}
	w = doJSONReq(s, http.MethodPost, "/api/profiles/clone-1/sync", syncBody, true)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on sync trigger, got %d: %s", w.Code, w.Body.String())
	}

	// 6. Delete profile: DELETE /api/profiles/{id}
	w = doJSONReq(s, http.MethodDelete, "/api/profiles/clone-1", nil, true)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete, got %d: %s", w.Code, w.Body.String())
	}

	// 7. Verify deletion: GET /api/profiles/{id} -> 404
	w = doJSONReq(s, http.MethodGet, "/api/profiles/clone-1", nil, false)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", w.Code)
	}

	// 8. Test /api/action profile routing
	actionBody := map[string]string{
		"action":     "start_sync",
		"profile_id": "orig-1",
	}
	w = doJSONReq(s, http.MethodPost, "/api/action", actionBody, true)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on /api/action with profile_id, got %d: %s", w.Code, w.Body.String())
	}
	var actResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &actResp)
	if actResp["status"] != "success" {
		t.Fatalf("expected success on /api/action, got %+v", actResp)
	}

	// 9. Test POST /api/profiles with auto-generated ID
	autoProf := map[string]any{
		"name":       "Auto Profile",
		"remote":     "gdrive",
		"local_path": filepath.Join(s.Config.BaseDir, "auto_prof"),
		"enabled":    true,
	}
	w = doJSONReq(s, http.MethodPost, "/api/profiles", autoProf, true)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 on auto-id create, got %d: %s", w.Code, w.Body.String())
	}
	var autoResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &autoResp)
	createdProf := autoResp["profile"].(map[string]any)
	genID := createdProf["id"].(string)
	if !strings.HasPrefix(genID, "prof_") {
		t.Fatalf("expected generated ID with prof_ prefix, got %s", genID)
	}
}

func TestProfiles_NilServiceHandling(t *testing.T) {
	s := NewServer(&config.Config{}, nil, nil)
	s.ProfileService = nil

	w := doJSONReq(s, http.MethodGet, "/api/profiles", nil, false)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with empty list when ProfileService is nil, got %d", w.Code)
	}

	w = doJSONReq(s, http.MethodGet, "/api/profiles/default", nil, false)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when ProfileService is nil, got %d", w.Code)
	}

	w = doJSONReq(s, http.MethodPost, "/api/profiles", map[string]any{}, true)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 on POST when ProfileService is nil, got %d", w.Code)
	}
}

func TestProfiles_ExecutorExcludeRulesForwarding(t *testing.T) {
	tmpDir := t.TempDir()
	profPath := filepath.Join(tmpDir, "profiles.json")
	histPath := filepath.Join(tmpDir, "profiles_history.json")

	var receivedExcludeRules []any
	rcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/config/listremotes" {
			_ = json.NewEncoder(w).Encode(map[string]any{"remotes": []string{"mockremote"}})
			return
		}
		if r.URL.Path == "/sync/bisync" {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if filterMap, ok := body["_filter"].(map[string]any); ok {
				if rules, ok := filterMap["ExcludeRule"].([]any); ok {
					receivedExcludeRules = rules
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"jobid": 999})
			return
		}
		if r.URL.Path == "/job/status" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"finished": true,
				"success":  true,
				"error":    "",
			})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer rcSrv.Close()

	localDir := filepath.Join(tmpDir, "local_test")
	_ = os.MkdirAll(localDir, 0755)

	cfg := &config.Config{
		BaseDir:             tmpDir,
		ProfilesFile:        profPath,
		ProfilesHistoryFile: histPath,
		RCAPIURL:            rcSrv.URL,
		LockFile:            filepath.Join(tmpDir, "main.lock"),
		JobTimeoutSeconds:   60,
	}

	srv := NewServer(cfg, nil, nil)
	if srv.ProfileService == nil {
		t.Fatal("expected non-nil ProfileService")
	}

	testProf := &profiles.Profile{
		ID:              "prof-ex",
		Name:            "Exclude Profile",
		Remote:          "mockremote",
		LocalPath:       localDir,
		Schedule:        "manual",
		Enabled:         true,
		ExcludePatterns: []string{".git/**", "node_modules/**", "*.tmp"},
	}

	if _, err := srv.ProfileService.Create(context.Background(), testProf); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	err := srv.ProfileService.TriggerSync(context.Background(), "prof-ex", false)
	if err != nil {
		t.Fatalf("failed to trigger sync: %v", err)
	}

	if len(receivedExcludeRules) != 3 {
		t.Fatalf("expected 3 exclude rules forwarded to daemon, got %d: %v", len(receivedExcludeRules), receivedExcludeRules)
	}

	expected := []string{".git/**", "node_modules/**", "*.tmp"}
	for i, rule := range expected {
		if receivedExcludeRules[i] != rule {
			t.Errorf("rule[%d]: expected %s, got %v", i, rule, receivedExcludeRules[i])
		}
	}
}

func TestProfiles_PathTraversalForbidden(t *testing.T) {
	s, _ := setupTestServerWithProfiles(t)

	// 1. POST /api/profiles with relative path traversal
	badCreate := map[string]any{
		"name":       "Bad Profile",
		"remote":     "gdrive",
		"local_path": "../../../etc/passwd",
		"enabled":    true,
	}
	w := doJSONReq(s, http.MethodPost, "/api/profiles", badCreate, true)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on traversal path in create, got %d: %s", w.Code, w.Body.String())
	}

	// 2. PUT /api/profiles/:id with relative path traversal
	badUpdate := map[string]any{
		"local_path": "/var/tmp/../../etc",
		"reason":     "test",
	}
	w = doJSONReq(s, http.MethodPut, "/api/profiles/prof_test", badUpdate, true)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on traversal path in update, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProfiles_OperationalFlagsAndValidation(t *testing.T) {
	s, _ := setupTestServerWithProfiles(t)
	tmpDir := t.TempDir()

	// 1. Rejection of prohibited custom flags on create
	badCreate := map[string]any{
		"name":       "Prohibited Flags Profile",
		"remote":     "gdrive",
		"local_path": tmpDir + "/valid_path",
		"enabled":    true,
		"flags": map[string]any{
			"custom_flags": "--config /etc/rclone.conf",
		},
	}
	w := doJSONReq(s, http.MethodPost, "/api/profiles", badCreate, true)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on prohibited custom flags in create, got %d: %s", w.Code, w.Body.String())
	}

	// 2. Successful create with valid operational flags
	maxDel := 25
	cmpMode := "checksum"
	canary := false
	goodCreate := map[string]any{
		"id":         "prof_flags",
		"name":       "Operational Flags Profile",
		"remote":     "gdrive",
		"local_path": tmpDir + "/flags_path",
		"enabled":    true,
		"flags": map[string]any{
			"max_delete":        maxDel,
			"compare_mode":      cmpMode,
			"canary_validation": canary,
			"custom_flags":      "--bwlimit 10M",
		},
	}
	w = doJSONReq(s, http.MethodPost, "/api/profiles", goodCreate, true)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 on valid flags profile create, got %d: %s", w.Code, w.Body.String())
	}

	// 3. GET /api/profiles/:id returns profile with resolved_flags and global_defaults
	w = doJSONReq(s, http.MethodGet, "/api/profiles/prof_flags", nil, true)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on get profile, got %d: %s", w.Code, w.Body.String())
	}
	var getResp struct {
		Status         string                 `json:"status"`
		Profile        profiles.Profile       `json:"profile"`
		ResolvedFlags  profiles.ResolvedFlags `json:"resolved_flags"`
		GlobalDefaults config.GlobalDefaults  `json:"global_defaults"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("failed to decode get profile response: %v", err)
	}
	if getResp.ResolvedFlags.MaxDelete != 25 {
		t.Errorf("expected resolved MaxDelete 25, got %d", getResp.ResolvedFlags.MaxDelete)
	}
	if getResp.ResolvedFlags.CompareMode != "checksum" {
		t.Errorf("expected resolved CompareMode 'checksum', got %s", getResp.ResolvedFlags.CompareMode)
	}
	if getResp.ResolvedFlags.CanaryValidation != false {
		t.Errorf("expected resolved CanaryValidation false")
	}
	if getResp.ResolvedFlags.ConflictResolve != "newer" {
		t.Errorf("expected resolved ConflictResolve inherited 'newer', got %s", getResp.ResolvedFlags.ConflictResolve)
	}
	if getResp.GlobalDefaults.MaxDelete != 0 {
		t.Errorf("expected GlobalDefaults MaxDelete 0, got %d", getResp.GlobalDefaults.MaxDelete)
	}

	// 4. Rejection of prohibited custom flags on update
	badUpdate := map[string]any{
		"flags": map[string]any{
			"custom_flags": "--rc-pass hacked",
		},
		"reason": "testing prohibited flag on update",
	}
	w = doJSONReq(s, http.MethodPut, "/api/profiles/prof_flags", badUpdate, true)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on prohibited custom flags in update, got %d: %s", w.Code, w.Body.String())
	}
}
