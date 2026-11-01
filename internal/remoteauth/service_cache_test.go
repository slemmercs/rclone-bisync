package remoteauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
)

func TestRemoteAuthService_GetAuthStatusAndCaching(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		switch r.URL.Path {
		case "/config/listremotes":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"remotes": ["cached_remote"]}`))
		case "/config/get":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"type":"s3","endpoint":"s3.amazonaws.com"}`))
		case "/operations/about":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"total": 1000}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rClient := client.NewRcloneClient(server.URL, "user", "pass")
	svc := NewRemoteAuthService(&config.Config{}, rClient, nil)
	ctx := context.Background()

	// 1. Initial call populates cache
	status1, err := svc.GetAuthStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected error from GetAuthStatus: %v", err)
	}
	if len(status1.Remotes) != 1 || status1.Remotes[0].Name != "cached_remote" {
		t.Fatalf("unexpected remotes in status: %+v", status1)
	}
	countAfterFirst := atomic.LoadInt32(&requestCount)
	if countAfterFirst == 0 {
		t.Fatal("expected server to receive requests")
	}

	// 2. Second call without forceRefresh hits cache
	status2, err := svc.GetAuthStatusCached(ctx, false)
	if err != nil {
		t.Fatalf("unexpected error from GetAuthStatusCached: %v", err)
	}
	if status2.Summary.TotalRemotes != status1.Summary.TotalRemotes {
		t.Fatalf("expected identical cached response, got %+v", status2)
	}
	if atomic.LoadInt32(&requestCount) != countAfterFirst {
		t.Fatal("expected request count not to increase on cache hit")
	}

	// 3. InvalidateAuthCache clears cached time
	svc.InvalidateAuthCache()
	status3, err := svc.GetAuthStatusCached(ctx, false)
	if err != nil {
		t.Fatalf("unexpected error after cache invalidation: %v", err)
	}
	if status3.Summary.TotalRemotes != 1 {
		t.Fatalf("expected 1 remote after re-fetch, got %d", status3.Summary.TotalRemotes)
	}
	if atomic.LoadInt32(&requestCount) <= countAfterFirst {
		t.Fatal("expected request count to increase after cache invalidation")
	}

	// 4. Force refresh bypasses cache even if valid
	countBeforeForce := atomic.LoadInt32(&requestCount)
	status4, err := svc.GetAuthStatusCached(ctx, true)
	if err != nil {
		t.Fatalf("unexpected error on force refresh: %v", err)
	}
	if status4.Summary.TotalRemotes != 1 {
		t.Fatalf("expected 1 remote on force refresh, got %d", status4.Summary.TotalRemotes)
	}
	if atomic.LoadInt32(&requestCount) <= countBeforeForce {
		t.Fatal("expected request count to increase on force refresh")
	}
}

func TestRemoteAuthService_NewServiceDefaultClient(t *testing.T) {
	cfg := &config.Config{
		RCUser:   "myuser",
		RCPass:   "mypass",
		RCAPIURL: "http://127.0.0.1:5572",
	}

	svc := NewRemoteAuthService(cfg, nil, nil)
	if svc.Client == nil {
		t.Fatal("expected client to be initialized from config")
	}
	if svc.Runner == nil {
		t.Fatal("expected default command runner to be set")
	}
}

func TestRemoteAuthService_DefaultCommandRunner(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Successful execution
	out, err := DefaultCommandRunner(ctx, "echo", nil, "hello-rclone")
	if err != nil {
		t.Fatalf("expected echo to succeed, got: %v", err)
	}
	if out != "hello-rclone" {
		t.Fatalf("expected 'hello-rclone', got %q", out)
	}

	// Failed execution (nonexistent binary)
	_, err = DefaultCommandRunner(ctx, "nonexistent-rclone-bin-xyz-1234", nil)
	if err == nil {
		t.Fatal("expected error for nonexistent binary")
	}
}

func TestRemoteAuthService_CollectSessionEnvironment(t *testing.T) {
	env := collectSessionEnvironment()
	if len(env) == 0 {
		t.Fatal("expected collectSessionEnvironment to return environment variables")
	}
}

func TestRemoteAuthService_ValidateRemote_NilClient(t *testing.T) {
	svc := NewRemoteAuthService(nil, nil, nil)
	svc.Client = nil

	resp, err := svc.ValidateRemote(context.Background(), "gdrive")
	if err == nil {
		t.Fatal("expected error when client is nil")
	}
	if resp.Valid {
		t.Fatal("expected valid to be false when client is nil")
	}
}
