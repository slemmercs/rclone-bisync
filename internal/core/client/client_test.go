package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/core/stats" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"bytes": 1024}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := NewRcloneClient(server.URL, "user", "pass")
	healthy, latency, err := c.CheckHealth(context.Background(), 1, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected health check error: %v", err)
	}
	if !healthy {
		t.Fatal("expected healthy to be true")
	}
	if latency <= 0 {
		t.Fatalf("expected positive latency, got %f", latency)
	}
}

func TestTriggerBisync(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sync/bisync" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"jobid": 42}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := NewRcloneClient(server.URL, "user", "pass")
	jobID, err := c.TriggerBisync(context.Background(), map[string]interface{}{"path1": "local", "path2": "remote:"}, "test-group")
	if err != nil {
		t.Fatalf("unexpected trigger error: %v", err)
	}
	if jobID != 42 {
		t.Fatalf("expected jobID 42, got %d", jobID)
	}
}

func TestListDirectoryWithCanary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/operations/list" {
			w.WriteHeader(http.StatusOK)
			res := map[string]interface{}{
				"list": []map[string]interface{}{
					{"Name": "file1.txt", "IsDir": false, "Size": 100},
					{"Name": "RCLONE_TEST", "IsDir": false, "Size": 0},
				},
			}
			_ = json.NewEncoder(w).Encode(res)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := NewRcloneClient(server.URL, "user", "pass")
	listing := c.ListDirectory(context.Background(), "gdrive:", "photos")
	if listing.Error != "" {
		t.Fatalf("unexpected listing error: %s", listing.Error)
	}
	if !listing.HasCanary {
		t.Fatal("expected listing to detect RCLONE_TEST canary")
	}
	if len(listing.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(listing.Items))
	}
}

func TestRemoteConfigSanitization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/config/get" {
			w.WriteHeader(http.StatusOK)
			res := map[string]interface{}{
				"type":          "drive",
				"scope":         "drive",
				"client_secret": "LEAKED_SECRET",
				"token":         "LEAKED_TOKEN",
			}
			_ = json.NewEncoder(w).Encode(res)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := NewRcloneClient(server.URL, "user", "pass")
	cfg := c.GetRemoteConfig(context.Background(), "gdrive:")
	if cfg["type"] != "drive" {
		t.Fatalf("expected type drive, got %v", cfg["type"])
	}
	if _, exists := cfg["client_secret"]; exists {
		t.Fatal("expected client_secret to be stripped by GetRemoteConfig")
	}
	if _, exists := cfg["token"]; exists {
		t.Fatal("expected token to be stripped by GetRemoteConfig")
	}
}

func TestTouchFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/operations/touch" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := NewRcloneClient(server.URL, "user", "pass")
	ok, err := c.TouchFile(context.Background(), "gdrive", "test.txt")
	if err != nil || !ok {
		t.Fatalf("expected TouchFile to succeed: %v", err)
	}

	// Error path
	cBad := NewRcloneClient("http://127.0.0.1:1", "user", "pass")
	if _, err := cBad.TouchFile(context.Background(), "gdrive", "test.txt"); err == nil {
		t.Fatal("expected error on connection failure")
	}
}

func TestCopyFileAndDir(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/operations/copyfile":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case "/sync/copy":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := NewRcloneClient(server.URL, "user", "pass")
	if ok, err := c.CopyFile(context.Background(), "src:", "a.txt", "dst:", "b.txt"); err != nil || !ok {
		t.Fatalf("expected CopyFile to succeed: %v", err)
	}
	if ok, err := c.CopyDir(context.Background(), "src:", "dst:"); err != nil || !ok {
		t.Fatalf("expected CopyDir to succeed: %v", err)
	}
}

func TestDeleteFileAndPurgeDir(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/operations/deletefile":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case "/operations/purge":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := NewRcloneClient(server.URL, "user", "pass")
	if ok, err := c.DeleteFile(context.Background(), "remote", "a.txt"); err != nil || !ok {
		t.Fatalf("expected DeleteFile to succeed: %v", err)
	}
	if ok, err := c.PurgeDir(context.Background(), "remote", "folder"); err != nil || !ok {
		t.Fatalf("expected PurgeDir to succeed: %v", err)
	}
}

func TestListRemotesAndAbout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/config/listremotes":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"remotes": ["gdrive", "onedrive"]}`))
		case "/operations/about":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"total": 1000000, "used": 500000, "free": 500000}`))
		case "/core/stats":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"bytes": 1024, "speed": 100}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := NewRcloneClient(server.URL, "user", "pass")

	remotes, err := c.ListRemotes(context.Background())
	if err != nil || len(remotes) != 2 {
		t.Fatalf("expected 2 remotes: %v, %v", remotes, err)
	}

	about := c.GetRemoteAbout(context.Background(), "gdrive")
	if about == nil || about["total"] != float64(1000000) {
		t.Fatalf("unexpected about response: %v", about)
	}

	stats, err := c.GetCoreStats(context.Background())
	if err != nil || len(stats) == 0 {
		t.Fatalf("unexpected stats: %v, %v", stats, err)
	}
}

func TestJobManagement(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/job/stop":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case "/job/list":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"jobids": [10, 20]}`))
		case "/job/status":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"finished": true, "success": true}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := NewRcloneClient(server.URL, "user", "pass")
	if ok := c.StopJob(context.Background(), 10); !ok {
		t.Fatalf("StopJob failed")
	}

	jobs := c.ListJobs(context.Background())
	if len(jobs) != 2 {
		t.Fatalf("ListJobs failed, got: %v", jobs)
	}

	status, err := c.GetJobStatus(context.Background(), 10)
	if err != nil || status["finished"] != true || status["success"] != true {
		t.Fatalf("GetJobStatus failed: %v, %v", status, err)
	}
}

func TestCheckHealthFailureAndRetries(t *testing.T) {
	// 1. Non-200 status code
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := NewRcloneClient(server.URL, "user", "pass")
	healthy, _, err := c.CheckHealth(context.Background(), 1, 1*time.Millisecond)
	if healthy || err == nil {
		t.Fatalf("expected unhealthy for 500 status, got healthy=%v, err=%v", healthy, err)
	}

	// 2. Unreachable host
	cBad := NewRcloneClient("http://127.0.0.1:1", "user", "pass")
	h2, _, err2 := cBad.CheckHealth(context.Background(), 1, 1*time.Millisecond)
	if h2 || err2 == nil {
		t.Fatalf("expected error for unreachable host")
	}
}

func TestClientErrorBranches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server failure"))
	}))
	defer server.Close()

	c := NewRcloneClient(server.URL, "user", "pass")
	ctx := context.Background()

	// 1. ListDirectory error
	listing := c.ListDirectory(ctx, "gdrive", "folder")
	if listing.Error == "" {
		t.Fatal("expected listing error on 500 status")
	}

	// 2. ListRemotes error
	if _, err := c.ListRemotes(ctx); err == nil {
		t.Fatal("expected ListRemotes error on 500 status")
	}

	// 3. GetRemoteAbout error
	if about := c.GetRemoteAbout(ctx, "gdrive"); about != nil {
		t.Fatalf("expected nil about on 500 status: %v", about)
	}

	// 4. GetCoreStats error
	if _, err := c.GetCoreStats(ctx); err == nil {
		t.Fatal("expected GetCoreStats error on 500 status")
	}

	// 5. CopyFile, CopyDir, DeleteFile, PurgeDir error paths
	if _, err := c.CopyFile(ctx, "a", "1", "b", "2"); err == nil {
		t.Fatal("expected error on CopyFile 500")
	}
	if _, err := c.CopyDir(ctx, "a", "b"); err == nil {
		t.Fatal("expected error on CopyDir 500")
	}
	if _, err := c.DeleteFile(ctx, "a", "1"); err == nil {
		t.Fatal("expected error on DeleteFile 500")
	}
	if _, err := c.PurgeDir(ctx, "a", "dir"); err == nil {
		t.Fatal("expected error on PurgeDir 500")
	}
	// 6. TriggerBisync error
	if _, err := c.TriggerBisync(ctx, map[string]interface{}{}, "test-group"); err == nil {
		t.Fatal("expected error on TriggerBisync 500")
	}

	// 7. GetRemoteConfig error
	if cfg := c.GetRemoteConfig(ctx, "invalid"); cfg != nil {
		t.Fatal("expected nil GetRemoteConfig on 500")
	}

}

func TestOAuthClient(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/config/get":
			var req map[string]string
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req["name"] == "drive_shared" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"type": "drive",
					"scope": "drive",
					"client_id": "",
					"client_secret": "",
					"token": "{\"access_token\":\"ya29.sample\",\"token_type\":\"Bearer\",\"refresh_token\":\"1//sample\",\"expiry\":\"2026-10-22T20:00:00Z\"}"
				}`))
			} else if req["name"] == "drive_custom" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"type": "drive",
					"scope": "drive",
					"client_id": "custom-id.apps.googleusercontent.com",
					"client_secret": "custom-secret",
					"token": "{\"access_token\":\"ya29.sample2\",\"token_type\":\"Bearer\",\"refresh_token\":\"1//sample2\",\"expiry\":\"2026-10-22T21:00:00Z\"}"
				}`))
			} else if req["name"] == "bad_json_token" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"type": "drive",
					"token": "not-a-valid-json"
				}`))
			} else if req["name"] == "map_token" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"type": "onedrive",
					"token": {"access_token":"token123","refresh_token":"ref123","expiry":"2026-10-23T00:00:00Z"}
				}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error": "remote not found"}`))
			}
		case "/config/update":
			var req map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req["name"] == "drive_custom" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{}`))
			} else {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error": "cannot update remote"}`))
			}
		case "/config/oauthstatus":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "running", "authUrl": "http://127.0.0.1:53682/auth?state=123"}`))
		case "/config/oauthstop":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := NewRcloneClient(server.URL, "user", "pass")
	ctx := context.Background()

	// Act & Assert 1: GetRemoteAuthConfig for shared client_id remote
	infoShared, err := c.GetRemoteAuthConfig(ctx, "drive_shared:")
	if err != nil {
		t.Fatalf("unexpected error for drive_shared: %v", err)
	}
	if infoShared["type"] != "drive" {
		t.Fatalf("expected type drive, got %v", infoShared["type"])
	}
	if infoShared["client_id"] != "" {
		t.Fatalf("expected empty client_id, got %v", infoShared["client_id"])
	}
	if infoShared["has_client_secret"] != false {
		t.Fatalf("expected has_client_secret=false, got %v", infoShared["has_client_secret"])
	}
	if infoShared["has_token"] != true {
		t.Fatalf("expected has_token=true, got %v", infoShared["has_token"])
	}
	if infoShared["token_expiry"] != "2026-10-22T20:00:00Z" {
		t.Fatalf("expected expiry 2026-10-22T20:00:00Z, got %v", infoShared["token_expiry"])
	}
	// Assert no secret keys leaked
	if _, leaked := infoShared["client_secret"]; leaked {
		t.Fatal("security violation: client_secret leaked in GetRemoteAuthConfig")
	}
	if _, leaked := infoShared["token"]; leaked {
		t.Fatal("security violation: raw token leaked in GetRemoteAuthConfig")
	}

	// Act & Assert 2: GetRemoteAuthConfig for custom credentials remote
	infoCustom, err := c.GetRemoteAuthConfig(ctx, "drive_custom")
	if err != nil {
		t.Fatalf("unexpected error for drive_custom: %v", err)
	}
	if infoCustom["client_id"] != "custom-id.apps.googleusercontent.com" {
		t.Fatalf("unexpected client_id: %v", infoCustom["client_id"])
	}
	if infoCustom["has_client_secret"] != true {
		t.Fatalf("expected has_client_secret=true, got %v", infoCustom["has_client_secret"])
	}

	// Act & Assert 3: bad json token and map token
	infoBad, err := c.GetRemoteAuthConfig(ctx, "bad_json_token")
	if err != nil || infoBad["has_token"] != true {
		t.Fatalf("expected has_token=true for bad_json_token fallback, got: %v, %v", infoBad, err)
	}
	infoMap, err := c.GetRemoteAuthConfig(ctx, "map_token")
	if err != nil || infoMap["has_token"] != true || infoMap["token_expiry"] != "2026-10-23T00:00:00Z" {
		t.Fatalf("expected has_token=true and expiry for map_token, got: %v, %v", infoMap, err)
	}

	// Act & Assert 4: not found remote
	if _, err := c.GetRemoteAuthConfig(ctx, "nonexistent"); err == nil {
		t.Fatal("expected error for nonexistent remote")
	}

	// Act & Assert 5: UpdateRemoteConfig success and error
	if err := c.UpdateRemoteConfig(ctx, "drive_custom", map[string]string{"client_id": "new-id"}); err != nil {
		t.Fatalf("expected UpdateRemoteConfig to succeed: %v", err)
	}
	if err := c.UpdateRemoteConfig(ctx, "unknown_remote", map[string]string{"client_id": "bad"}); err == nil {
		t.Fatal("expected error on UpdateRemoteConfig for unknown_remote")
	}

	// Act & Assert 6: OAuthStatus
	status, authURL, err := c.OAuthStatus(ctx)
	if err != nil {
		t.Fatalf("expected OAuthStatus to succeed: %v", err)
	}
	if status != "running" || authURL != "http://127.0.0.1:53682/auth?state=123" {
		t.Fatalf("unexpected OAuthStatus result: status=%s, authURL=%s", status, authURL)
	}

	// Act & Assert 7: OAuthStop
	if err := c.OAuthStop(ctx); err != nil {
		t.Fatalf("expected OAuthStop to succeed: %v", err)
	}
}
