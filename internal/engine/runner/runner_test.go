package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/core/lock"
	"github.com/slemmercs/rclone-bisync/internal/core/logger"
	"github.com/slemmercs/rclone-bisync/internal/core/state"
	"github.com/slemmercs/rclone-bisync/internal/models"
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

	// Verify structured log output in bisync.log (RUNNING followed by SUCCESS)
	logData, err := os.ReadFile(cfg.LogFile)
	if err != nil {
		t.Fatalf("failed to read bisync.log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(logData)), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 log entries (RUNNING and SUCCESS), got %d entries: %s", len(lines), string(logData))
	}

	var runningEntry map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &runningEntry); err != nil {
		t.Fatalf("failed to parse RUNNING log entry: %v, content=%s", err, lines[0])
	}
	if runningEntry["run_id"] != "test_run" || runningEntry["status"] != "RUNNING" || runningEntry["job_id"] != float64(10) {
		t.Fatalf("unexpected RUNNING log entry content: %+v", runningEntry)
	}

	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(lines[1]), &entry); err != nil {
		t.Fatalf("failed to parse SUCCESS bisync.log entry: %v, content=%s", err, lines[1])
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
	if ok, err := r.RunWithError(context.Background(), false, false, false, "cli", "", false); ok || err == nil || !strings.Contains(err.Error(), "sync dispatch failed") {
		t.Fatalf("expected dispatch error from RunWithError, got ok=%v, err=%v", ok, err)
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
	if ok, err := rCompilerErr.RunWithError(context.Background(), false, false, false, "cli", "", false); ok || err == nil || !strings.Contains(err.Error(), "configuration/canary check failed") {
		t.Fatalf("expected canary check error from RunWithError, got ok=%v, err=%v", ok, err)
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

	// Act 1b: 0-transfer routine background run (origin: "daemon") must NOT append to audit ledger
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
	success0 := r0.Run(context.Background(), false, false, false, "daemon", "run_audit_02_empty", false)
	if !success0 {
		t.Fatal("expected 0-transfer daemon run to succeed")
	}
	recordsAfter0, totalAfter0, err := audit.QueryRecords(db, audit.Filter{})
	if err != nil {
		t.Fatalf("failed to query records: %v", err)
	}
	if totalAfter0 != 1 || len(recordsAfter0) != 1 {
		t.Fatalf("expected audit records count to remain 1 after 0-transfer daemon run, got: %d", totalAfter0)
	}

	// Act 1c: 0-transfer resync run (origin: "daemon", resync: true) MUST append to audit ledger
	successResync := r0.Run(context.Background(), true, false, false, "daemon", "run_audit_03_resync", false)
	if !successResync {
		t.Fatal("expected resync run to succeed")
	}
	recordsAfterResync, totalAfterResync, err := audit.QueryRecords(db, audit.Filter{})
	if err != nil {
		t.Fatalf("failed to query records: %v", err)
	}
	if totalAfterResync != 2 || len(recordsAfterResync) != 2 {
		t.Fatalf("expected audit records count to be 2 after 0-transfer resync run, got: %d", totalAfterResync)
	}
	var resyncPayload map[string]any
	_ = json.Unmarshal(recordsAfterResync[0].Details, &resyncPayload)
	if resyncPayload["resync"] != true {
		t.Errorf("expected resync: true in audit payload, got: %+v", resyncPayload)
	}

	// Act 1d: 0-transfer user action (origin: "dashboard") MUST append to audit ledger
	successUser := r0.Run(context.Background(), false, false, false, "dashboard", "run_audit_04_user", false)
	if !successUser {
		t.Fatal("expected user dashboard run to succeed")
	}
	recordsAfterUser, totalAfterUser, err := audit.QueryRecords(db, audit.Filter{})
	if err != nil {
		t.Fatalf("failed to query records: %v", err)
	}
	if totalAfterUser != 3 || len(recordsAfterUser) != 3 {
		t.Fatalf("expected audit records count to be 3 after user dashboard run, got: %d", totalAfterUser)
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

	// Assert 3: Audit DB retains the 3 valid records; non-mutating aborts are NOT logged to audit.db
	_, total2, err := audit.QueryRecords(db, audit.Filter{})
	if err != nil {
		t.Fatalf("failed to query audit records: %v", err)
	}
	if total2 != 3 {
		t.Fatalf("expected audit records count to remain 3 after aborted run, got: %d", total2)
	}

	// Assert 4: Ledger SHA-256 chain integrity verified with 3 blocks
	valid, count, err := audit.VerifyLedger(db)
	if err != nil || !valid || count != 3 {
		t.Fatalf("expected valid ledger chain with 3 blocks, got valid=%v, count=%d, err=%v", valid, count, err)
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
func TestSyncRunner_ProfileLogFormatting(t *testing.T) {
	// 1. Empty config
	r1 := &SyncRunner{Config: &config.Config{}}
	if got := r1.formatLogMsg("Sync succeeded"); got != "Sync succeeded" {
		t.Fatalf("expected 'Sync succeeded', got '%s'", got)
	}

	// 2. Profile ID only
	r2 := &SyncRunner{Config: &config.Config{ProfileID: "onedrive"}}
	if got := r2.formatLogMsg("Sync succeeded"); got != "[onedrive] Sync succeeded" {
		t.Fatalf("expected '[onedrive] Sync succeeded', got '%s'", got)
	}

	// 3. Profile Name takes precedence
	r3 := &SyncRunner{Config: &config.Config{ProfileID: "gdrive", ProfileName: "Google Drive"}}
	if got := r3.formatLogMsg("Sync succeeded"); got != "[Google Drive] Sync succeeded" {
		t.Fatalf("expected '[Google Drive] Sync succeeded', got '%s'", got)
	}

	// 4. "default" profile is not prefixed
	r4 := &SyncRunner{Config: &config.Config{ProfileID: "default", ProfileName: "default"}}
	if got := r4.formatLogMsg("Sync succeeded"); got != "Sync succeeded" {
		t.Fatalf("expected 'Sync succeeded', got '%s'", got)
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

func TestSyncRunner_ActionLabeling(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "action_run.json")

	cfg := &config.Config{
		BaseDir:     tmpDir,
		LogFile:     logPath,
		ProfileID:   "onedrive",
		ProfileName: "OneDrive (Test)",
	}

	r := NewSyncRunner(cfg)

	// 1. Dry run
	r.handleSyncSuccess("run_dry", 1, "dashboard", time.Now(), 2.0, client.MonitorJobResult{Success: true, FilesSynced: 0}, false, false, true, true, state.NotificationState{})
	data, _ := os.ReadFile(logPath)
	if !strings.Contains(string(data), "[OneDrive (Test)] Dry run succeeded in 2.0s (0 file(s) transferred)") {
		t.Fatalf("expected dry run message, got: %s", string(data))
	}

	// 2. Resync
	_ = os.Remove(logPath)
	r.Logger, _ = logger.New(logPath, false)
	r.handleSyncSuccess("run_resync", 2, "dashboard", time.Now(), 5.0, client.MonitorJobResult{Success: true, FilesSynced: 3}, true, false, false, true, state.NotificationState{})
	data, _ = os.ReadFile(logPath)
	if !strings.Contains(string(data), "[OneDrive (Test)] Resync succeeded in 5.0s (3 file(s) transferred)") {
		t.Fatalf("expected resync message, got: %s", string(data))
	}

	// 3. Normal Sync
	_ = os.Remove(logPath)
	r.Logger, _ = logger.New(logPath, false)
	r.handleSyncSuccess("run_sync", 3, "dashboard", time.Now(), 1.5, client.MonitorJobResult{Success: true, FilesSynced: 1}, false, false, false, true, state.NotificationState{})
	data, _ = os.ReadFile(logPath)
	if !strings.Contains(string(data), "[OneDrive (Test)] Sync succeeded in 1.5s (1 file(s) transferred)") {
		t.Fatalf("expected sync message, got: %s", string(data))
	}
}

func TestRunWithError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sync/bisync":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"jobid": 42})
		case "/job/status":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"finished": true,
				"success":  true,
				"output":   map[string]interface{}{"transfers": 1},
			})
		}
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "bisync.log")

	cfg := &config.Config{
		RCAPIURL:                 srv.URL,
		LocalPath:                tmpDir,
		RemotePath:               "gdrive:",
		LogFile:                  logPath,
		LockFile:                 filepath.Join(tmpDir, "bisync.lock"),
		NotificationStateFile:    filepath.Join(tmpDir, "notify.json"),
		PendingDeletionsFile:     filepath.Join(tmpDir, "deletions.json"),
		RateLimitBackoffFile:     filepath.Join(tmpDir, "rate.json"),
		MetricsFile:              filepath.Join(tmpDir, "metrics.jsonl"),
		JobPollInitialSleepSec:   0.005,
		JobPollMaxSleepSec:       0.01,
		JobPollBackoffMultiplier: 1.0,
		JobTimeoutSeconds:        5,
		ProfileID:                "test_profile",
		ProfileName:              "Test Profile",
	}

	r := NewSyncRunner(cfg)

	// 1. Dry run execution
	ok, err := r.RunWithError(context.Background(), false, false, true, "dashboard", "run_dry_test", true)
	if !ok || err != nil {
		t.Fatalf("expected dry run to succeed, got ok=%v, err=%v", ok, err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read bisync.log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 log entries, got %d: %s", len(lines), string(data))
	}

	var dryRunEntry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &dryRunEntry); err != nil {
		t.Fatalf("failed to parse RUNNING dry-run entry: %v", err)
	}
	if dryRunEntry["status"] != "RUNNING" || dryRunEntry["run_id"] != "run_dry_test" {
		t.Fatalf("expected RUNNING status for dry run, got %+v", dryRunEntry)
	}
	if !strings.Contains(fmt.Sprint(dryRunEntry["msg"]), "Dry run started") {
		t.Fatalf("expected 'Dry run started' in msg, got %v", dryRunEntry["msg"])
	}

	// 2. Resync execution
	_ = os.Remove(logPath)
	r.Logger, _ = logger.New(logPath, false)
	ok, err = r.RunWithError(context.Background(), true, false, false, "dashboard", "run_resync_test", true)
	if !ok || err != nil {
		t.Fatalf("expected resync to succeed, got ok=%v, err=%v", ok, err)
	}

	data, err = os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read bisync.log: %v", err)
	}
	lines = strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 log entries, got %d: %s", len(lines), string(data))
	}

	var resyncEntry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &resyncEntry); err != nil {
		t.Fatalf("failed to parse RUNNING resync entry: %v", err)
	}
	if resyncEntry["status"] != "RUNNING" || resyncEntry["run_id"] != "run_resync_test" {
		t.Fatalf("expected RUNNING status for resync, got %+v", resyncEntry)
	}
	if !strings.Contains(fmt.Sprint(resyncEntry["msg"]), "Resync started") {
		t.Fatalf("expected 'Resync started' in msg, got %v", resyncEntry["msg"])
	}

	// 3. Normal sync execution
	_ = os.Remove(logPath)
	r.Logger, _ = logger.New(logPath, false)
	ok, err = r.RunWithError(context.Background(), false, false, false, "dashboard", "run_sync_test", true)
	if !ok || err != nil {
		t.Fatalf("expected normal sync to succeed, got ok=%v, err=%v", ok, err)
	}

	data, err = os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read bisync.log: %v", err)
	}
	lines = strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 log entries, got %d: %s", len(lines), string(data))
	}

	var syncEntry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &syncEntry); err != nil {
		t.Fatalf("failed to parse RUNNING sync entry: %v", err)
	}
	if syncEntry["status"] != "RUNNING" || syncEntry["run_id"] != "run_sync_test" {
		t.Fatalf("expected RUNNING status for normal sync, got %+v", syncEntry)
	}
	if !strings.Contains(fmt.Sprint(syncEntry["msg"]), "Sync started") {
		t.Fatalf("expected 'Sync started' in msg, got %v", syncEntry["msg"])
	}
}

func TestSyncRunner_TransferredFilesIngestion(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "bisync.log")

	cfg := &config.Config{
		BaseDir:     tmpDir,
		LogFile:     logPath,
		ProfileID:   "drive",
		ProfileName: "Google Drive",
	}
	r := NewSyncRunner(cfg)

	monitorRes := client.MonitorJobResult{
		Success:     true,
		FilesSynced: 2,
		TransferredFiles: []models.SyncFileOp{
			{Path: "subfolder/a.txt", Action: "COPIED", Msg: "subfolder/a.txt: Copied (new)"},
			{Path: "notes.pdf", Action: "REPLACED", Msg: "notes.pdf: Copied (replaced)"},
		},
	}

	ok := r.handleSyncSuccess("run_files_1", 101, "dashboard", time.Now(), 1.2, monitorRes, false, false, false, true, state.NotificationState{})
	if !ok {
		t.Fatalf("expected handleSyncSuccess to return true")
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 { // 1 parent sync + 2 sync_file
		t.Fatalf("expected 3 log entries, got %d: %s", len(lines), string(data))
	}

	var file1, file2 map[string]interface{}
	if err := json.Unmarshal([]byte(lines[1]), &file1); err != nil {
		t.Fatalf("failed to parse file1 JSON: %v", err)
	}
	if file1["type"] != "sync_file" || file1["object"] != "subfolder/a.txt" || file1["action"] != "COPIED" {
		t.Fatalf("unexpected file1 entry: %+v", file1)
	}

	if err := json.Unmarshal([]byte(lines[2]), &file2); err != nil {
		t.Fatalf("failed to parse file2 JSON: %v", err)
	}
	if file2["type"] != "sync_file" || file2["object"] != "notes.pdf" || file2["action"] != "REPLACED" {
		t.Fatalf("unexpected file2 entry: %+v", file2)
	}
}

func TestSyncRunner_TransferredFilesCappingAndOverflow(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "bisync.log")

	cfg := &config.Config{
		BaseDir:     tmpDir,
		LogFile:     logPath,
		ProfileID:   "backup",
		ProfileName: "Backup",
	}
	r := NewSyncRunner(cfg)

	// Generate 55 operations
	ops := make([]models.SyncFileOp, 55)
	for i := 0; i < 55; i++ {
		ops[i] = models.SyncFileOp{
			Path:   fmt.Sprintf("file_%d.txt", i),
			Action: "COPIED",
			Msg:    fmt.Sprintf("file_%d.txt: Copied (new)", i),
		}
	}

	monitorRes := client.MonitorJobResult{
		Success:          true,
		FilesSynced:      55,
		TransferredFiles: ops,
	}

	ok := r.handleSyncSuccess("run_overflow_1", 102, "dashboard", time.Now(), 3.5, monitorRes, false, false, false, true, state.NotificationState{})
	if !ok {
		t.Fatalf("expected handleSyncSuccess to return true")
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	// Expected: 1 parent sync + 50 sync_file + 1 sync_overflow = 52
	if len(lines) != 52 {
		t.Fatalf("expected 52 entries (1 parent + 50 capped files + 1 overflow), got %d", len(lines))
	}

	var overflowEntry map[string]interface{}
	if err := json.Unmarshal([]byte(lines[51]), &overflowEntry); err != nil {
		t.Fatalf("failed to parse overflow entry: %v", err)
	}
	if overflowEntry["type"] != "sync_overflow" {
		t.Fatalf("expected type sync_overflow, got %v", overflowEntry["type"])
	}
	if !strings.Contains(fmt.Sprint(overflowEntry["msg"]), "5 more transferred files omitted") {
		t.Fatalf("expected overflow count in msg, got %v", overflowEntry["msg"])
	}
}

func TestSyncRunner_TransferredFilesDaemonLogFallback(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "bisync.log")
	daemonLog := filepath.Join(tmpDir, "rclone-daemon.log")

	// Write daemon log with transfers
	rawDaemon := "2026/11/13 12:00:00 INFO  : fallback/item1.txt: Copied (new)\n2026/11/13 12:00:01 INFO  : fallback/item2.txt: Deleted\n"
	if err := os.WriteFile(daemonLog, []byte(rawDaemon), 0644); err != nil {
		t.Fatalf("failed to write daemon log: %v", err)
	}

	cfg := &config.Config{
		BaseDir:     tmpDir,
		LogFile:     logPath,
		ProfileID:   "fallback-prof",
		ProfileName: "Fallback",
	}
	r := NewSyncRunner(cfg)

	// TransferredFiles empty, but FilesSynced = 2
	monitorRes := client.MonitorJobResult{
		Success:          true,
		FilesSynced:      2,
		TransferredFiles: nil,
	}

	ok := r.handleSyncSuccess("run_fallback_1", 103, "dashboard", time.Now(), 0.8, monitorRes, false, false, false, true, state.NotificationState{})
	if !ok {
		t.Fatalf("expected handleSyncSuccess to return true")
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 { // 1 parent sync + 2 fallback sync_file
		t.Fatalf("expected 3 log entries from daemon log fallback, got %d: %s", len(lines), string(data))
	}

	var f1 map[string]interface{}
	if err := json.Unmarshal([]byte(lines[1]), &f1); err != nil {
		t.Fatalf("failed to parse fallback entry: %v", err)
	}
	if f1["object"] != "fallback/item1.txt" || f1["action"] != "COPIED" {
		t.Fatalf("unexpected fallback file1: %+v", f1)
	}
}
