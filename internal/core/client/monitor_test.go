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
