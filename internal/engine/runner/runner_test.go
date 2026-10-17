package runner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
)

func TestSyncRunnerExecution(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sync/bisync":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"jobid": 10})
		case "/job/status":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"finished": true,
				"success":  true,
				"output": map[string]interface{}{"transfers": 2},
			})
		}
	}))
	defer srv.Close()

	tmpDir, _ := os.MkdirTemp("", "runner_test")
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		RCAPIURL:          srv.URL,
		LocalPath:         tmpDir,
		RemotePath:        "gdrive:",
		LockFile:          tmpDir + "/bisync.lock",
		NotificationStateFile: tmpDir + "/notify.json",
		PendingDeletionsFile:  tmpDir + "/deletions.json",
		RateLimitBackoffFile:  tmpDir + "/rate.json",
		MetricsFile:           tmpDir + "/metrics.jsonl",
		JobPollInitialSleepSec: 0.01,
		JobPollMaxSleepSec:     0.05,
		JobPollBackoffMultiplier: 1.5,
		JobTimeoutSeconds:      60,
	}

	r := NewSyncRunner(cfg)
	success := r.Run(context.Background(), false, false, false, "test", "test_run", true)
	if !success {
		t.Fatalf("expected sync run to succeed")
	}
}
