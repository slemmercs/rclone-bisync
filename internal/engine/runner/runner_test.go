package runner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/lock"
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
		LogFile:           tmpDir + "/bisync.log",
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
	success := r.Run(context.Background(), false, false, false, "test", "test_run", false)
	if !success {
		t.Fatalf("expected sync run to succeed")
	}

	// Verify structured single-line log output in bisync.log
	logData, err := os.ReadFile(cfg.LogFile)
	if err != nil {
		t.Fatalf("failed to read bisync.log: %v", err)
	}

	var entry map[string]interface{}
	if err := json.Unmarshal(logData, &entry); err != nil {
		t.Fatalf("failed to parse bisync.log entry: %v, content=%s", err, string(logData))
	}

	if entry["run_id"] != "test_run" || entry["status"] != "SUCCESS" || entry["job_id"] != float64(10) {
		t.Fatalf("unexpected log entry content: %+v", entry)
	}
	if entry["start_time"] == nil || entry["end_time"] == nil {
		t.Fatalf("expected start_time and end_time in log entry: %+v", entry)
	}
}

func TestSyncRunnerLockContention(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "runner_lock_test")
	defer os.RemoveAll(tmpDir)

	lockFile := tmpDir + "/bisync.lock"
	cfg := &config.Config{
		LocalPath: tmpDir,
		LockFile:  lockFile,
	}

	lm := lock.NewLockManager(lockFile, 3600, false)
	acquired, err := lm.Acquire()
	if err != nil || !acquired {
		t.Fatalf("failed to hold initial lock: %v", err)
	}
	defer lm.Release()

	r := NewSyncRunner(cfg)
	success := r.Run(context.Background(), false, false, false, "test", "run-contention", true)
	if success {
		t.Fatal("expected Run to fail due to lock contention")
	}
}

func TestSyncRunnerRateLimitBackoffActive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sync/bisync":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"jobid": 11})
		case "/job/status":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"finished": true,
				"success":  true,
				"output": map[string]interface{}{"transfers": 1},
			})
		}
	}))
	defer srv.Close()

	tmpDir, _ := os.MkdirTemp("", "runner_rate_test")
	defer os.RemoveAll(tmpDir)

	rateFile := tmpDir + "/rate_limit.flag"
	_ = os.WriteFile(rateFile, []byte(`{"active":true}`), 0644)

	cfg := &config.Config{
		RCAPIURL:               srv.URL,
		LocalPath:              tmpDir,
		RemotePath:             "gdrive:",
		LockFile:               tmpDir + "/bisync.lock",
		RateLimitBackoffFile:   rateFile,
		JobPollInitialSleepSec: 0.01,
		JobPollMaxSleepSec:     0.05,
		JobTimeoutSeconds:      60,
	}

	r := NewSyncRunner(cfg)

	// 1. Without force -> skipped
	if success := r.Run(context.Background(), false, false, false, "test", "", true); success {
		t.Fatal("expected sync to be skipped while rate limit backoff is active")
	}

	// 2. With force -> overrides backoff
	if success := r.Run(context.Background(), false, true, false, "test", "", true); !success {
		t.Fatal("expected force=true to override rate limit backoff")
	}
}

func TestSyncRunnerJobFailures(t *testing.T) {
	var currentError string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sync/bisync":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"jobid": 12})
		case "/job/status":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"finished": true,
				"success":  false,
				"error":    currentError,
			})
		}
	}))
	defer srv.Close()

	tmpDir, _ := os.MkdirTemp("", "runner_fail_test")
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		RCAPIURL:               srv.URL,
		LocalPath:              tmpDir,
		RemotePath:             "gdrive:",
		LockFile:               tmpDir + "/bisync.lock",
		NotificationStateFile:  tmpDir + "/notify.json",
		PendingDeletionsFile:   tmpDir + "/deletions.json",
		RateLimitBackoffFile:   tmpDir + "/rate_limit.flag",
		MetricsFile:            tmpDir + "/metrics.jsonl",
		EnableMetrics:          true,
		JobPollInitialSleepSec: 0.01,
		JobPollMaxSleepSec:     0.05,
		JobTimeoutSeconds:      60,
	}

	r := NewSyncRunner(cfg)

	// 1. TOO_MANY_DELETES
	currentError = "Safety abort: too many deletes"
	if success := r.Run(context.Background(), false, false, false, "test", "run-del", true); success {
		t.Fatal("expected failure on too many deletes")
	}
	if _, err := os.Stat(cfg.PendingDeletionsFile); os.IsNotExist(err) {
		t.Fatal("expected PendingDeletionsFile to be created")
	}

	// 2. RATE_LIMIT
	currentError = "rateLimitExceeded: User rate limit exceeded"
	if success := r.Run(context.Background(), false, false, false, "test", "run-rate", true); success {
		t.Fatal("expected failure on rate limit")
	}
	if _, err := os.Stat(cfg.RateLimitBackoffFile); os.IsNotExist(err) {
		t.Fatal("expected RateLimitBackoffFile to be created")
	}
}

func TestSyncRunnerCompilerErrorAndTriggerError(t *testing.T) {
	// 1. Auth enabled NewSyncRunner
	cfgAuth := &config.Config{
		RCAuthEnabled: true,
		RCUser:        "testuser",
		RCPass:        "testpass",
		Features:      map[string]bool{"smart_notifications": true},
	}
	rAuth := NewSyncRunner(cfgAuth)
	if rAuth.Client == nil {
		t.Fatal("expected non-nil client")
	}

	// 2. TriggerBisync dispatch failure with notification
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("daemon error: internal error"))
	}))
	defer srv.Close()

	tmpDir, _ := os.MkdirTemp("", "runner_dispatch_err")
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		RCAPIURL:              srv.URL,
		LocalPath:             tmpDir,
		RemotePath:            "gdrive:",
		LockFile:              tmpDir + "/bisync.lock",
		NotificationStateFile: tmpDir + "/notify.json",
	}
	r := NewSyncRunner(cfg)
	if success := r.Run(context.Background(), false, false, false, "cli", "", false); success {
		t.Fatal("expected failure on dispatch error")
	}

	// 3. Compiler canary missing error branch
	cfgCompilerErr := &config.Config{
		LocalPath:        tmpDir,
		RemotePath:       "gdrive:",
		RcloneCommonArgs: []string{"--check-access"},
		LockFile:         tmpDir + "/bisync_comp.lock",
	}
	rCompilerErr := NewSyncRunner(cfgCompilerErr)
	if success := rCompilerErr.Run(context.Background(), false, false, false, "cli", "", false); success {
		t.Fatal("expected failure on missing canary check-access error")
	}

}

