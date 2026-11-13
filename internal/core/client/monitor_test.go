package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestMonitorJobSuccess(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/job/status" {
			c := atomic.AddInt32(&calls, 1)
			if c < 2 {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"finished": false,
				})
			} else {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"finished": true,
					"success":  true,
					"output": map[string]interface{}{
						"transfers": 3,
					},
				})
			}
		}
	}))
	defer srv.Close()

	c := NewRcloneClient(srv.URL, "", "")
	res := c.MonitorJob(context.Background(), 123, 10*time.Millisecond, 20*time.Millisecond, 1.5, 3, 5*time.Second)
	if !res.Success {
		t.Fatalf("expected job success, got error: %s", res.ErrorMessage)
	}
	if res.FilesSynced != 3 {
		t.Fatalf("expected 3 files synced, got %d", res.FilesSynced)
	}
}

func TestMonitorJobFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/job/status" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"finished": true,
				"success":  false,
				"error":    "too many deletes",
			})
		}
	}))
	defer srv.Close()

	c := NewRcloneClient(srv.URL, "", "")
	res := c.MonitorJob(context.Background(), 123, 10*time.Millisecond, 20*time.Millisecond, 1.5, 3, 5*time.Second)
	if res.Success {
		t.Fatal("expected job failure")
	}
	if res.ErrorMessage != "too many deletes" {
		t.Fatalf("expected 'too many deletes' error message, got %s", res.ErrorMessage)
	}
}

func TestMonitorJobConsecutiveErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewRcloneClient(srv.URL, "", "")
	res := c.MonitorJob(context.Background(), 123, 5*time.Millisecond, 10*time.Millisecond, 1.5, 2, 2*time.Second)
	if res.Success {
		t.Fatal("expected failure on consecutive errors")
	}
	if res.ErrorType != "MONITOR_ERROR" {
		t.Fatalf("expected MONITOR_ERROR, got: %s", res.ErrorType)
	}
}

func TestMonitorJob_GroupStatsTransfers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/job/status":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"finished": true,
				"success":  true,
				"group":    "bisync-group-123",
				"output": map[string]interface{}{
					"workDir": "/tmp/work",
				},
			})
		case "/core/stats":
			var req map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req["group"] == "bisync-group-123" {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"transfers": 2,
				})
			} else {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"transfers": 0,
				})
			}
		}
	}))
	defer srv.Close()

	c := NewRcloneClient(srv.URL, "", "")
	res := c.MonitorJob(context.Background(), 123, 5*time.Millisecond, 10*time.Millisecond, 1.5, 3, 2*time.Second)
	if !res.Success {
		t.Fatalf("expected job success, got error: %s", res.ErrorMessage)
	}
	if res.FilesSynced != 2 {
		t.Fatalf("expected 2 files synced from /core/stats group, got %d", res.FilesSynced)
	}
}

func TestMonitorJob_OutputTextFallbackTransfers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/job/status":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"finished": true,
				"success":  true,
				"output": map[string]interface{}{
					"output": `{"level":"info","msg":"Copied (new)","object":"file1.txt"}
{"level":"info","msg":"Copied (replaced)","object":"file2.txt"}
Copied (new) file3.txt
Bisync successful`,
				},
			})
		case "/core/stats":
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := NewRcloneClient(srv.URL, "", "")
	res := c.MonitorJob(context.Background(), 123, 5*time.Millisecond, 10*time.Millisecond, 1.5, 3, 2*time.Second)
	if !res.Success {
		t.Fatalf("expected job success, got error: %s", res.ErrorMessage)
	}
	if res.FilesSynced != 3 {
		t.Fatalf("expected 3 files synced from output text fallback, got %d", res.FilesSynced)
	}
}
