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
