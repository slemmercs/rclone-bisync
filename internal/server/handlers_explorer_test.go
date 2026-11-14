package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/audit"
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
		ID:              "prof_onedrive",
		Name:            "OneDrive Work",
		Remote:          "onedrive:",
		LocalPath:       profileLocal,
		ExcludePatterns: []string{"Personal Vault/**", "*.tmp"},
	}, "init profile")

	pService := profiles.NewService(pStore, nil, nil)
	srv := NewServer(cfg, nil, nil)
	srv.ProfileService = pService
	srv.UploadDock = services.NewUploadDock(pService)

	canarySvc := services.NewCanaryService(cfg, nil)
	srv.CanaryService = canarySvc

	explorerSvc := services.NewExplorerService(cfg, nil, canarySvc)
	srv.ExplorerService = explorerSvc

	auditDBPath := filepath.Join(tmpDir, "test_audit.db")
	if auditDB, err := audit.OpenDB(auditDBPath); err == nil {
		auditSvc := services.NewAuditService(auditDB, nil)
		auditSvc.DBPath = auditDBPath
		srv.AuditService = auditSvc
		t.Cleanup(func() { _ = auditSvc.Close() })
	}

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
func TestHandleCopyItem_ExcludePatternsForwarding(t *testing.T) {
	srv, _, profileLocal := setupTestServerWithProfilesAndExplorer(t)

	var receivedExcludeRules []interface{}
	mockRC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sync/copy" {
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if filter, ok := body["_filter"].(map[string]interface{}); ok {
				if rules, ok := filter["ExcludeRule"].([]interface{}); ok {
					receivedExcludeRules = rules
				}
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockRC.Close()

	rcClient := client.NewRcloneClient(mockRC.URL, "user", "pass")
	srv.ExplorerService.Client = rcClient

	subDir := filepath.Join(profileLocal, "myfolder")
	_ = os.MkdirAll(subDir, 0755)

	body, _ := json.Marshal(models.CopyRequest{
		SrcRemote: "onedrive:",
		SrcPath:   "",
		DstRemote: "local",
		DstPath:   "myfolder",
		IsDir:     true,
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

	if len(receivedExcludeRules) != 2 || receivedExcludeRules[0] != "Personal Vault/**" || receivedExcludeRules[1] != "*.tmp" {
		t.Fatalf("expected exclude rules to be forwarded to sync/copy, got: %v", receivedExcludeRules)
	}
}

func TestExplorer_PathTraversalForbidden(t *testing.T) {
	srv, _, _ := setupTestServerWithProfilesAndExplorer(t)

	// 1. Browse with traversal
	browseBody, _ := json.Marshal(models.BrowseRequest{
		Remote: "local",
		Path:   "../../../etc",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/browse", bytes.NewBuffer(browseBody))
	req.Header.Set("X-Dashboard-Request", "1")
	req.Header.Set("Origin", "http://localhost:8080")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with error body, got %d", w.Code)
	}
	var browseRes client.DirectoryListing
	_ = json.Unmarshal(w.Body.Bytes(), &browseRes)
	if browseRes.Error != "Path traversal forbidden" {
		t.Fatalf("expected path traversal error, got: %s", browseRes.Error)
	}

	// 2. Canary with traversal
	canaryBody, _ := json.Marshal(models.BrowseRequest{
		Remote: "local",
		Path:   "../../../root",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/browse/canary", bytes.NewBuffer(canaryBody))
	req.Header.Set("X-Dashboard-Request", "1")
	req.Header.Set("Origin", "http://localhost:8080")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with error body, got %d", w.Code)
	}
	var canaryRes models.CanaryResponse
	_ = json.Unmarshal(w.Body.Bytes(), &canaryRes)
	if canaryRes.Status != "error" || canaryRes.Error != "Path traversal forbidden" {
		t.Fatalf("expected path traversal error, got: %+v", canaryRes)
	}

	// 3. Copy with traversal
	copyBody, _ := json.Marshal(models.CopyRequest{
		SrcRemote: "local",
		SrcPath:   "../../../etc/shadow",
		DstRemote: "local",
		DstPath:   "stolen.txt",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/browse/copy", bytes.NewBuffer(copyBody))
	req.Header.Set("X-Dashboard-Request", "1")
	req.Header.Set("Origin", "http://localhost:8080")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with error body, got %d", w.Code)
	}
	var copyRes models.CopyResponse
	_ = json.Unmarshal(w.Body.Bytes(), &copyRes)
	if copyRes.Status != "error" || copyRes.Error != "Path traversal forbidden" {
		t.Fatalf("expected path traversal error, got: %+v", copyRes)
	}

	// 4. Delete with traversal
	delBody, _ := json.Marshal(models.DeleteRequest{
		Remote: "local",
		Path:   "../../../etc/passwd",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/browse/delete", bytes.NewBuffer(delBody))
	req.Header.Set("X-Dashboard-Request", "1")
	req.Header.Set("Origin", "http://localhost:8080")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with error body, got %d", w.Code)
	}
	var delRes models.DeleteResponse
	_ = json.Unmarshal(w.Body.Bytes(), &delRes)
	if delRes.Status != "error" || delRes.Error != "Path traversal forbidden" {
		t.Fatalf("expected path traversal error, got: %+v", delRes)
	}
}

func TestHandleExplorerUpload_SuccessAndSafety(t *testing.T) {
	srv, defaultLocal, profileLocal := setupTestServerWithProfilesAndExplorer(t)

	// 1. Normal upload to default local
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	_ = w.WriteField("remote", "local")
	_ = w.WriteField("path", "")
	part, err := w.CreateFormFile("file", "uploaded_doc.txt")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = part.Write([]byte("hello world upload content"))
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/explorer/upload", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("X-Dashboard-Request", "1")
	req.Header.Set("Origin", "http://localhost:8080")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var res models.UploadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.Status != "success" || res.Path != "uploaded_doc.txt" {
		t.Fatalf("expected success, got: %+v", res)
	}
	content, err := os.ReadFile(filepath.Join(defaultLocal, "uploaded_doc.txt"))
	if err != nil || string(content) != "hello world upload content" {
		t.Fatalf("uploaded file content mismatch: %v (data: %s)", err, string(content))
	}

	// 2. Traversal attempt in path
	b.Reset()
	w = multipart.NewWriter(&b)
	_ = w.WriteField("remote", "local")
	_ = w.WriteField("path", "../../../etc")
	part, _ = w.CreateFormFile("file", "hacked.txt")
	_, _ = part.Write([]byte("evil"))
	_ = w.Close()

	req = httptest.NewRequest(http.MethodPost, "/api/explorer/upload", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("X-Dashboard-Request", "1")
	req.Header.Set("Origin", "http://localhost:8080")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	var errRes models.UploadResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &errRes)
	if errRes.Status != "error" || errRes.Error != "Path traversal forbidden" {
		t.Fatalf("expected path traversal forbidden error, got: %+v", errRes)
	}

	// 3. Canary file upload attempt (RCLONE_TEST)
	b.Reset()
	w = multipart.NewWriter(&b)
	_ = w.WriteField("remote", "local")
	_ = w.WriteField("path", "")
	part, _ = w.CreateFormFile("file", "RCLONE_TEST")
	_, _ = part.Write([]byte("fake canary"))
	_ = w.Close()

	req = httptest.NewRequest(http.MethodPost, "/api/explorer/upload", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("X-Dashboard-Request", "1")
	req.Header.Set("Origin", "http://localhost:8080")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	var canaryRes models.UploadResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &canaryRes)
	if canaryRes.Status != "error" || canaryRes.Error != "Canary test file (RCLONE_TEST) cannot be uploaded or overwritten" {
		t.Fatalf("expected canary protection error, got: %+v", canaryRes)
	}

	// 4. Upload scoped to profile_id = "prof_onedrive"
	b.Reset()
	w = multipart.NewWriter(&b)
	_ = w.WriteField("remote", "local")
	_ = w.WriteField("path", "")
	_ = w.WriteField("profile_id", "prof_onedrive")
	part, _ = w.CreateFormFile("file", "profile_doc.txt")
	_, _ = part.Write([]byte("onedrive profile data"))
	_ = w.Close()

	req = httptest.NewRequest(http.MethodPost, "/api/explorer/upload", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("X-Dashboard-Request", "1")
	req.Header.Set("Origin", "http://localhost:8080")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	var profRes models.UploadResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &profRes)
	if profRes.Status != "success" || profRes.Path != "profile_doc.txt" {
		t.Fatalf("expected profile upload success, got: %+v", profRes)
	}
	content, err = os.ReadFile(filepath.Join(profileLocal, "profile_doc.txt"))
	if err != nil || string(content) != "onedrive profile data" {
		t.Fatalf("profile uploaded file content mismatch: %v (data: %s)", err, string(content))
	}
}

func TestExplorerAudit_Operations(t *testing.T) {
	srv, _, profileLocal := setupTestServerWithProfilesAndExplorer(t)

	mockRC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockRC.Close()
	srv.ExplorerService.Client = client.NewRcloneClient(mockRC.URL, "user", "pass")

	// 1. Upload item with profile_id
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	_ = w.WriteField("remote", "local")
	_ = w.WriteField("path", "")
	_ = w.WriteField("profile_id", "prof_onedrive")
	part, _ := w.CreateFormFile("file", "audit_upload.txt")
	_, _ = part.Write([]byte("audit upload content"))
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/explorer/upload", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("X-Dashboard-Request", "1")
	req.Header.Set("Origin", "http://localhost:8080")
	req.Header.Set("X-Forwarded-User", "auditor@domain.test")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// 2. Copy item
	copyBody, _ := json.Marshal(models.CopyRequest{
		SrcRemote: "local",
		SrcPath:   "audit_upload.txt",
		DstRemote: "local",
		DstPath:   "audit_upload_copy.txt",
		ProfileID: "prof_onedrive",
	})
	reqCopy := httptest.NewRequest(http.MethodPost, "/api/browse/copy", bytes.NewBuffer(copyBody))
	reqCopy.Header.Set("Content-Type", "application/json")
	reqCopy.Header.Set("X-Dashboard-Request", "1")
	reqCopy.Header.Set("Origin", "http://localhost:8080")
	reqCopy.Header.Set("X-Forwarded-User", "auditor@domain.test")
	recCopy := httptest.NewRecorder()
	srv.ServeHTTP(recCopy, reqCopy)

	if recCopy.Code != http.StatusOK {
		t.Fatalf("expected 200 on copy, got %d", recCopy.Code)
	}

	// 3. Delete item (create file on disk so local deletion succeeds)
	_ = os.WriteFile(filepath.Join(profileLocal, "audit_upload_copy.txt"), []byte("copy"), 0644)
	delBody, _ := json.Marshal(models.DeleteRequest{
		Remote:    "local",
		Path:      "audit_upload_copy.txt",
		ProfileID: "prof_onedrive",
	})
	reqDel := httptest.NewRequest(http.MethodPost, "/api/browse/delete", bytes.NewBuffer(delBody))
	reqDel.Header.Set("Content-Type", "application/json")
	reqDel.Header.Set("X-Dashboard-Request", "1")
	reqDel.Header.Set("Origin", "http://localhost:8080")
	reqDel.Header.Set("X-Forwarded-User", "auditor@domain.test")
	recDel := httptest.NewRecorder()
	srv.ServeHTTP(recDel, reqDel)

	if recDel.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete, got %d", recDel.Code)
	}

	// Query audit records and verify all three operations were recorded
	recs, _, err := srv.AuditService.Query(audit.Filter{ProfileID: "prof_onedrive"})
	if err != nil {
		t.Fatalf("failed to query audit records: %v", err)
	}

	foundUpload := false
	foundCopy := false
	foundDelete := false

	for _, r := range recs {
		if r.Actor != "auditor@domain.test" {
			t.Errorf("expected actor auditor@domain.test, got %s", r.Actor)
		}
		if r.ProfileID != "prof_onedrive" {
			t.Errorf("expected profile prof_onedrive, got %s", r.ProfileID)
		}
		switch r.Action {
		case audit.ActionWorkspaceFileUploaded:
			foundUpload = true
			if !strings.Contains(string(r.Details), "audit_upload.txt") || !strings.Contains(string(r.Details), `"item_path"`) {
				t.Errorf("expected details to contain audit_upload.txt and item_path, got %s", string(r.Details))
			}
		case audit.ActionWorkspaceItemCopied:
			foundCopy = true
			if !strings.Contains(string(r.Details), "audit_upload.txt") {
				t.Errorf("expected details to contain audit_upload.txt, got %s", string(r.Details))
			}
		case audit.ActionWorkspaceItemDeleted:
			foundDelete = true
			if !strings.Contains(string(r.Details), "audit_upload_copy.txt") {
				t.Errorf("expected details to contain audit_upload_copy.txt, got %s", string(r.Details))
			}
		}
	}

	if !foundUpload {
		t.Errorf("expected ActionWorkspaceFileUploaded audit record")
	}
	if !foundCopy {
		t.Errorf("expected ActionWorkspaceItemCopied audit record")
	}
	if !foundDelete {
		t.Errorf("expected ActionWorkspaceItemDeleted audit record")
	}

	// Clean up created file in profileLocal
	_ = os.Remove(filepath.Join(profileLocal, "audit_upload.txt"))
}

func TestResolveExcludePatterns(t *testing.T) {
	srv, _, _ := setupTestServerWithProfilesAndExplorer(t)

	// 1. Profile with exclude patterns
	patterns := srv.resolveExcludePatterns("prof_onedrive")
	if len(patterns) != 2 || patterns[0] != "Personal Vault/**" || patterns[1] != "*.tmp" {
		t.Errorf("unexpected patterns for prof_onedrive: %v", patterns)
	}

	// 2. Non-existent profile, but config has --exclude flags
	srv.Config.RcloneCommonArgs = []string{"--fast-list", "--exclude", "*.log", "--exclude", "node_modules/**"}
	patternsFromConfig := srv.resolveExcludePatterns("non_existent")
	if len(patternsFromConfig) != 2 || patternsFromConfig[0] != "*.log" || patternsFromConfig[1] != "node_modules/**" {
		t.Errorf("unexpected patterns from config: %v", patternsFromConfig)
	}

	// 3. No profile, empty common args
	srv.Config.RcloneCommonArgs = nil
	patternsNil := srv.resolveExcludePatterns("")
	if len(patternsNil) != 0 {
		t.Errorf("expected empty patterns, got: %v", patternsNil)
	}

	// 4. resolveLocalPath fallback when profile missing
	path := srv.resolveLocalPath("non_existent")
	if path != srv.Config.LocalPath {
		t.Errorf("expected local path %q, got %q", srv.Config.LocalPath, path)
	}
}

func TestHandleBrowseCompare(t *testing.T) {
	srv, defaultLocal, _ := setupTestServerWithProfilesAndExplorer(t)

	// Create test files
	f1 := filepath.Join(defaultLocal, "compare1.txt")
	f2 := filepath.Join(defaultLocal, "compare2.txt")
	f3 := filepath.Join(defaultLocal, "compare3.txt")

	_ = os.WriteFile(f1, []byte("same data"), 0644)
	_ = os.WriteFile(f2, []byte("same data"), 0644)
	_ = os.WriteFile(f3, []byte("other data"), 0644)

	// 1. Success matching comparison
	reqBody, _ := json.Marshal(map[string]interface{}{
		"items": []models.CompareItemRequest{
			{Remote: "local", Path: "compare1.txt"},
			{Remote: "local", Path: "compare2.txt"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/browse/compare", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Dashboard-Request", "1")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var res models.FileCompareResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !res.Matches || res.Algorithm != "sha256" || len(res.Clusters) != 1 {
		t.Fatalf("expected matching result with 1 cluster, got: %+v", res)
	}

	// 2. Traversal attempt
	reqTravBody, _ := json.Marshal(map[string]interface{}{
		"items": []models.CompareItemRequest{
			{Remote: "local", Path: "../outside.txt"},
			{Remote: "local", Path: "compare1.txt"},
		},
	})
	reqTrav := httptest.NewRequest(http.MethodPost, "/api/browse/compare", bytes.NewReader(reqTravBody))
	reqTrav.Header.Set("Content-Type", "application/json")
	reqTrav.Header.Set("X-Dashboard-Request", "1")
	recTrav := httptest.NewRecorder()
	srv.ServeHTTP(recTrav, reqTrav)

	var resTrav models.FileCompareResponse
	_ = json.Unmarshal(recTrav.Body.Bytes(), &resTrav)
	if resTrav.Matches {
		t.Errorf("expected traversal comparison to not match")
	}

	// 3. Less than 2 items
	reqFewBody, _ := json.Marshal(map[string]interface{}{
		"items": []models.CompareItemRequest{
			{Remote: "local", Path: "compare1.txt"},
		},
	})
	reqFew := httptest.NewRequest(http.MethodPost, "/api/browse/compare", bytes.NewReader(reqFewBody))
	reqFew.Header.Set("Content-Type", "application/json")
	reqFew.Header.Set("X-Dashboard-Request", "1")
	recFew := httptest.NewRecorder()
	srv.ServeHTTP(recFew, reqFew)

	var resFew models.FileCompareResponse
	_ = json.Unmarshal(recFew.Body.Bytes(), &resFew)
	if resFew.Status != "error" {
		t.Errorf("expected error for <2 items, got: %+v", resFew)
	}
}

