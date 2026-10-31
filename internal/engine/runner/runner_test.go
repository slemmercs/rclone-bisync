package runner

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/audit"
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
				"output":   map[string]interface{}{"transfers": 2},
			})
		}
	}))
	defer srv.Close()

	tmpDir, _ := os.MkdirTemp("", "runner_test")
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		RCAPIURL:                 srv.URL,
		LocalPath:                tmpDir,
		RemotePath:               "gdrive:",
		LogFile:                  tmpDir + "/bisync.log",
		LockFile:                 tmpDir + "/bisync.lock",
		NotificationStateFile:    tmpDir + "/notify.json",
		PendingDeletionsFile:     tmpDir + "/deletions.json",
		RateLimitBackoffFile:     tmpDir + "/rate.json",
		MetricsFile:              tmpDir + "/metrics.jsonl",
		JobPollInitialSleepSec:   0.01,
		JobPollMaxSleepSec:       0.05,
		JobPollBackoffMultiplier: 1.5,
		JobTimeoutSeconds:        60,
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
				"output":   map[string]interface{}{"transfers": 1},
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

func TestSyncRunner_AuditExecuted_And_Aborted(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "runner_audit_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	auditDBPath := tmpDir + "/audit.db"
	db, err := audit.OpenDB(auditDBPath)
	if err != nil {
		t.Fatalf("failed to open audit db: %v", err)
	}
	defer db.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sync/bisync":
			_ = json.NewEncoder(w).Encode(map[string]any{"jobid": 42})
		case "/job/status":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"finished": true,
				"success":  true,
				"output":   map[string]any{"transfers": 5},
			})
		}
	}))
	defer srv.Close()

	logFile := tmpDir + "/bisync.log"
	cfg := &config.Config{
		RCAPIURL:                 srv.URL,
		LocalPath:                tmpDir,
		RemotePath:               "gdrive:",
		AuditDBPath:              auditDBPath,
		LogFile:                  logFile,
		LockFile:                 tmpDir + "/bisync.lock",
		NotificationStateFile:    tmpDir + "/notify.json",
		PendingDeletionsFile:     tmpDir + "/deletions.json",
		JobPollInitialSleepSec:   0.005,
		JobPollMaxSleepSec:       0.01,
		JobPollBackoffMultiplier: 1.0,
		JobTimeoutSeconds:        5,
	}

	r := NewSyncRunner(cfg)
	if r.Audit == nil {
		t.Fatal("expected runner.Audit to be initialized from AuditDBPath")
	}

	// Act 1: Successful sync with files transferred and origin: "dashboard"
	success := r.Run(context.Background(), false, false, false, "dashboard", "run_audit_01", false)
	if !success {
		t.Fatal("expected sync run to succeed")
	}

	// Assert 1: Audit DB has SYNC_EXECUTED
	records, total, err := audit.QueryRecords(db, audit.Filter{})
	if err != nil {
		t.Fatalf("failed to query audit records: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 audit record, got: %d", total)
	}
	if records[0].Action != audit.ActionSyncExecuted {
		t.Errorf("expected Action %s, got: %s", audit.ActionSyncExecuted, records[0].Action)
	}
	if records[0].Actor != "USER" {
		t.Errorf("expected Actor USER for origin dashboard, got: %s", records[0].Actor)
	}

	// Assert 2: Dual-logging written to bisync.log with "type":"audit"
	logBytes, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read bisync.log: %v", err)
	}
	logContent := string(logBytes)
	if !strings.Contains(logContent, `"type":"audit"`) {
		t.Errorf("expected bisync.log to contain dual-logged audit entry: %s", logContent)
	}
	if !strings.Contains(logContent, "SYNC_EXECUTED") {
		t.Errorf("expected bisync.log to contain SYNC_EXECUTED: %s", logContent)
	}

	// Act 1b: Option 2 verification: 0-transfer run (even with origin: "dashboard") must NOT append to audit ledger
	srv0 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sync/bisync":
			_ = json.NewEncoder(w).Encode(map[string]any{"jobid": 43})
		case "/job/status":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"finished": true,
				"success":  true,
				"output":   map[string]any{"transfers": 0},
			})
		}
	}))
	defer srv0.Close()
	cfg0 := *cfg
	cfg0.RCAPIURL = srv0.URL
	r0 := NewSyncRunner(&cfg0)
	success0 := r0.Run(context.Background(), false, false, false, "dashboard", "run_audit_02_empty", false)
	if !success0 {
		t.Fatal("expected 0-transfer run to succeed")
	}
	recordsAfter0, totalAfter0, err := audit.QueryRecords(db, audit.Filter{})
	if err != nil {
		t.Fatalf("failed to query records: %v", err)
	}
	if totalAfter0 != 1 || len(recordsAfter0) != 1 {
		t.Fatalf("expected audit records count to remain 1 after 0-transfer run (Option 2), got: %d", totalAfter0)
	}

	// Act 2: Circuit break / abort scenario (Lock contention)
	lm := lock.NewLockManager(cfg.LockFile, 300, false)
	acq, err := lm.Acquire()
	if err != nil || !acq {
		t.Fatalf("failed to acquire outer lock: %v", err)
	}

	aborted := r.Run(context.Background(), false, false, false, "cli", "run_abort_01", true)
	if aborted {
		t.Fatal("expected sync run to abort due to lock contention")
	}
	_ = lm.Release()

	// Assert 3: Audit DB retains only the 1 mutating record; non-mutating aborts are NOT logged to audit.db
	records2, total2, err := audit.QueryRecords(db, audit.Filter{})
	if err != nil {
		t.Fatalf("failed to query audit records: %v", err)
	}
	if total2 != 1 {
		t.Fatalf("expected audit records count to remain 1 after aborted non-mutating run, got: %d", total2)
	}
	if records2[0].Action != audit.ActionSyncExecuted {
		t.Errorf("expected Action %s, got: %s", audit.ActionSyncExecuted, records2[0].Action)
	}

	// Assert 4: Ledger SHA-256 chain integrity verified with 1 block
	valid, count, err := audit.VerifyLedger(db)
	if err != nil || !valid || count != 1 {
		t.Fatalf("expected valid ledger chain with 1 block, got valid=%v, count=%d, err=%v", valid, count, err)
	}
}

func TestSyncRunner_NotificationTitle(t *testing.T) {
	// Scenario 1: Default/empty profile configuration
	r1 := &SyncRunner{Config: &config.Config{}}
	if got := r1.notificationTitle("Sync Successful"); got != "Sync Successful" {
		t.Fatalf("expected 'Sync Successful', got '%s'", got)
	}

	// Scenario 2: ProfileID only
	r2 := &SyncRunner{Config: &config.Config{ProfileID: "documents"}}
	if got := r2.notificationTitle("Sync Successful"); got != "[documents] Sync Successful" {
		t.Fatalf("expected '[documents] Sync Successful', got '%s'", got)
	}

	// Scenario 3: ProfileName takes precedence over ProfileID
	r3 := &SyncRunner{Config: &config.Config{ProfileID: "prof-123", ProfileName: "Work Docs"}}
	if got := r3.notificationTitle("Sync Aborted"); got != "[Work Docs] Sync Aborted" {
		t.Fatalf("expected '[Work Docs] Sync Aborted', got '%s'", got)
	}

	// Scenario 4: "default" profile name/ID should not be prefixed
	r4 := &SyncRunner{Config: &config.Config{ProfileID: "default", ProfileName: "default"}}
	if got := r4.notificationTitle("Sync Failure"); got != "Sync Failure" {
		t.Fatalf("expected 'Sync Failure', got '%s'", got)
	}
}
func TestSyncRunner_DebugModePropagation(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "debug_run.json")

	cfg := &config.Config{
		BaseDir:   tmpDir,
		LogFile:   logPath,
		DebugMode: true,
		LockFile:  filepath.Join(tmpDir, "sync.lock"),
	}

	r := NewSyncRunner(cfg)
	if r.Logger == nil {
		t.Fatal("expected logger to not be nil")
	}
	if r.Logger.Level() != slog.LevelDebug {
		t.Fatalf("expected logger level to be LevelDebug, got %v", r.Logger.Level())
	}
}
