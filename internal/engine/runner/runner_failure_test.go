package runner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/core/lock"
	"github.com/slemmercs/rclone-bisync/internal/core/logger"
	"github.com/slemmercs/rclone-bisync/internal/core/state"
	"github.com/slemmercs/rclone-bisync/internal/engine"
	"github.com/slemmercs/rclone-bisync/internal/engine/review"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

func TestSyncRunner_NewSyncRunnerNilConfig(t *testing.T) {
	r := NewSyncRunner(nil)
	if r == nil {
		t.Fatal("expected non-nil runner")
	}
	if r.Config == nil {
		t.Fatal("expected non-nil config inside runner")
	}
	if label := r.profileLabel(); label != "" {
		t.Errorf("expected empty profile label, got %s", label)
	}
}

func TestSyncRunner_DebugModePropagationDuringRun(t *testing.T) {
	var capturedArgs []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sync/bisync":
			var req map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&req)
			if params, ok := req["_config"].(map[string]interface{}); ok {
				if args, ok := params["args"].([]interface{}); ok {
					for _, a := range args {
						capturedArgs = append(capturedArgs, a.(string))
					}
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"jobid": 20})
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
	logPath := filepath.Join(tmpDir, "debug_run.log")

	cfg := &config.Config{
		RCAPIURL:                 srv.URL,
		LocalPath:                tmpDir,
		RemotePath:               "gdrive:",
		LogFile:                  logPath,
		LockFile:                 filepath.Join(tmpDir, "lock.lock"),
		DebugMode:                true,
		RcloneCommonArgs:         []string{"--fast-list"},
		JobPollInitialSleepSec:   0.005,
		JobPollMaxSleepSec:       0.01,
		JobPollBackoffMultiplier: 1.0,
		JobTimeoutSeconds:        5,
	}

	r := NewSyncRunner(cfg)
	success := r.Run(context.Background(), false, false, false, "cli", "debug_run_1", true)
	if !success {
		t.Fatal("expected debug run to succeed")
	}

	capturedArgs = nil
	cfg.RcloneCommonArgs = []string{"-v"}
	r2 := NewSyncRunner(cfg)
	success2 := r2.Run(context.Background(), false, false, false, "cli", "debug_run_2", true)
	if !success2 {
		t.Fatal("expected debug run 2 to succeed")
	}
}

func TestSyncRunner_HandleSyncFailure_ReviewSessionAndExistingMerge(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "bisync.log")
	daemonLog := filepath.Join(tmpDir, "rclone-daemon.log")
	activeReviewFile := filepath.Join(tmpDir, "active_review.json")
	pendingDeletionsFile := filepath.Join(tmpDir, "deletions.json")
	notifyFile := filepath.Join(tmpDir, "notify.json")
	metricsFile := filepath.Join(tmpDir, "metrics.jsonl")

	delLogContent := `2026/10/31 12:00:00 ERROR : deleted_file.txt: Deletion of 1 file(s) would exceed delete limit of 0
2026/10/31 12:00:01 ERROR : conflict_file.txt: Path exists on both Path1 and Path2 with different content
`
	_ = os.WriteFile(daemonLog, []byte(delLogContent), 0600)
	_ = os.WriteFile(logFile, []byte(delLogContent), 0600)

	existingSess := review.NewSession(
		"TOO_MANY_DELETES",
		"Safety abort: too many deletes",
		[]models.DeletionCandidate{
			{Path: "deleted_file.txt", Resolved: true, Decision: "approve"},
		},
		[]models.ConflictCandidate{
			{Path: "conflict_file.txt", Resolved: true, Decision: "keep_newer"},
		},
	)
	_ = review.Transition(existingSess, models.ReviewStatusReviewing)
	_ = review.SaveActiveSession(activeReviewFile, existingSess)

	cfg := &config.Config{
		BaseDir:               tmpDir,
		LogFile:               logFile,
		ActiveReviewFile:      activeReviewFile,
		PendingDeletionsFile:  pendingDeletionsFile,
		NotificationStateFile: notifyFile,
		MetricsFile:           metricsFile,
		EnableMetrics:         true,
		ProfileID:             "work",
		ProfileName:           "Work Profile",
	}

	r := NewSyncRunner(cfg)
	l, _ := logger.New(logFile, false)
	r.Logger = l

	monitorRes := client.MonitorJobResult{
		Success:      false,
		ErrorMessage: "Safety abort: too many deletes",
		ErrorType:    "TOO_MANY_DELETES",
		FilesSynced:  0,
	}

	notifState := state.NotificationState{
		LastErrorType:      "",
		LastErrorTimestamp: 0,
	}

	failed := r.handleSyncFailure(
		"run_fail_1",
		100,
		"dashboard",
		time.Now().Add(-5*time.Second),
		5.0,
		monitorRes,
		false,
		false,
		false,
		false,
		notifState,
	)

	if failed {
		t.Fatal("expected handleSyncFailure to return false")
	}

	loaded, err := review.LoadActiveSession(activeReviewFile)
	if err != nil || loaded == nil {
		t.Fatalf("expected active review file to be loaded: %v", err)
	}
	if len(loaded.Deletions) == 0 || !loaded.Deletions[0].Resolved || loaded.Deletions[0].Decision != "approve" {
		t.Errorf("expected merged deletion candidate to keep approve decision, got %+v", loaded.Deletions)
	}

	if _, err := os.Stat(metricsFile); os.IsNotExist(err) {
		t.Errorf("expected metrics file to be populated")
	}
	if _, err := os.Stat(pendingDeletionsFile); os.IsNotExist(err) {
		t.Errorf("expected pending deletions file to be populated")
	}
}

func TestSyncRunner_HandleSyncFailure_RateLimitAndCriticalNotification(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "bisync.log")
	rateLimitFile := filepath.Join(tmpDir, "rate_limit.json")
	notifyFile := filepath.Join(tmpDir, "notify.json")

	cfg := &config.Config{
		BaseDir:               tmpDir,
		LogFile:               logFile,
		RateLimitBackoffFile:  rateLimitFile,
		NotificationStateFile: notifyFile,
		ProfileID:             "backup",
	}

	r := NewSyncRunner(cfg)
	l, _ := logger.New(logFile, false)
	r.Logger = l

	monitorRate := client.MonitorJobResult{
		Success:      false,
		ErrorMessage: "Rate limit reached",
		ErrorType:    "RATE_LIMIT",
	}

	_ = r.handleSyncFailure(
		"run_rate",
		101,
		"cli",
		time.Now(),
		1.0,
		monitorRate,
		false,
		false,
		false,
		true,
		state.NotificationState{},
	)

	if _, err := os.Stat(rateLimitFile); os.IsNotExist(err) {
		t.Errorf("expected rate limit file to be created")
	}

	monitorCrit := client.MonitorJobResult{
		Success:      false,
		ErrorMessage: "Check access canary missing",
		ErrorType:    "CHECK_ACCESS_FAILED",
	}

	_ = r.handleSyncFailure(
		"run_crit",
		102,
		"cli",
		time.Now(),
		1.0,
		monitorCrit,
		false,
		false,
		false,
		false,
		state.NotificationState{},
	)

	if _, err := os.Stat(notifyFile); os.IsNotExist(err) {
		t.Errorf("expected notify state file to be created")
	}
}


func TestSyncRunner_FSM_SuccessContract(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sync/bisync":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"jobid": 501})
		case "/job/status":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"finished": true,
				"success":  true,
				"output":   map[string]interface{}{"transfers": 3},
			})
		}
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	cfg := &config.Config{
		RCAPIURL:                 srv.URL,
		LocalPath:                tmpDir,
		RemotePath:               "remote:bucket",
		LockFile:                 filepath.Join(tmpDir, "sync.lock"),
		LogFile:                  filepath.Join(tmpDir, "sync.log"),
		ProfileID:                "fsm-success-prof",
		JobPollInitialSleepSec:   0.005,
		JobPollMaxSleepSec:       0.01,
		JobPollBackoffMultiplier: 1.0,
		JobTimeoutSeconds:        5,
	}

	r := NewSyncRunner(cfg)
	if r.FSM == nil || r.FSM.Current() != engine.StateIdle {
		t.Fatalf("expected initial FSM state IDLE, got %v", r.FSM)
	}

	var transitions []engine.SyncFSMState
	r.FSM.OnTransition(func(evt engine.StateChangeEvent) {
		transitions = append(transitions, evt.To)
	})

	success, err := r.RunWithError(context.Background(), false, false, false, "test", "run-fsm-succ", true)
	if !success || err != nil {
		t.Fatalf("expected sync success, got err: %v", err)
	}

	if r.FSM.Current() != engine.StateCompleted {
		t.Fatalf("expected final state COMPLETED, got %s", r.FSM.Current())
	}

	// Verify transition sequence: QUEUED -> RUNNING -> COMPLETED
	expected := []engine.SyncFSMState{engine.StateQueued, engine.StateRunning, engine.StateCompleted}
	if len(transitions) != len(expected) {
		t.Fatalf("expected %d transitions, got %d (%v)", len(expected), len(transitions), transitions)
	}
	for i, exp := range expected {
		if transitions[i] != exp {
			t.Errorf("transition %d: expected %s, got %s", i, exp, transitions[i])
		}
	}
}

func TestSyncRunner_FSM_LockContentionContract(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "contention.lock")

	lm := lock.NewLockManager(lockPath, 3600, false)
	acquired, err := lm.Acquire()
	if err != nil || !acquired {
		t.Fatalf("failed to hold initial lock: %v", err)
	}
	defer lm.Release()

	cfg := &config.Config{
		LocalPath: tmpDir,
		LockFile:  lockPath,
		ProfileID: "lock-fail-prof",
	}

	r := NewSyncRunner(cfg)
	success, err := r.RunWithError(context.Background(), false, false, false, "test", "run-fsm-lock", true)
	if success || err == nil {
		t.Fatal("expected run to fail due to lock contention")
	}

	if r.FSM.Current() != engine.StateAborted {
		t.Fatalf("expected FSM state ABORTED, got %s", r.FSM.Current())
	}
	last := r.FSM.LastEvent()
	if last.ErrorType != "LOCK_CONTENTION" {
		t.Fatalf("expected error type LOCK_CONTENTION, got %s", last.ErrorType)
	}
}

func TestSyncRunner_FSM_RateLimitBackoffContract(t *testing.T) {
	tmpDir := t.TempDir()
	rateFile := filepath.Join(tmpDir, "rate_limit.flag")
	_ = os.WriteFile(rateFile, []byte(`{"active":true}`), 0644)

	cfg := &config.Config{
		LocalPath:            tmpDir,
		LockFile:             filepath.Join(tmpDir, "test.lock"),
		RateLimitBackoffFile: rateFile,
		ProfileID:            "rate-limit-prof",
	}

	r := NewSyncRunner(cfg)
	success, err := r.RunWithError(context.Background(), false, false, false, "test", "run-fsm-rate", true)
	if success || err == nil {
		t.Fatal("expected run to fail due to rate limit")
	}

	if r.FSM.Current() != engine.StateAborted {
		t.Fatalf("expected FSM state ABORTED, got %s", r.FSM.Current())
	}
	last := r.FSM.LastEvent()
	if last.ErrorType != "RATE_LIMIT_BACKOFF" {
		t.Fatalf("expected error type RATE_LIMIT_BACKOFF, got %s", last.ErrorType)
	}
}

func TestSyncRunner_FSM_DispatchFailureContract(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "connection refused by daemon", http.StatusBadGateway)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	cfg := &config.Config{
		RCAPIURL:   srv.URL,
		LocalPath:  tmpDir,
		RemotePath: "remote:bucket",
		LockFile:   filepath.Join(tmpDir, "dispatch.lock"),
		ProfileID:  "dispatch-fail-prof",
	}

	r := NewSyncRunner(cfg)
	success, err := r.RunWithError(context.Background(), false, false, false, "test", "run-fsm-dispatch", true)
	if success || err == nil {
		t.Fatal("expected run to fail due to dispatch error")
	}

	if r.FSM.Current() != engine.StateAborted {
		t.Fatalf("expected FSM state ABORTED, got %s", r.FSM.Current())
	}
}

func TestSyncRunner_FSM_ConflictReviewRequiredContract(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sync/bisync":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"jobid": 601})
		case "/job/status":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"finished": true,
				"success":  false,
				"error":    "Safety abort: too many deletes: 5 files to delete exceeds limit 0 (use --force or adjust max-delete)",
			})
		}
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	cfg := &config.Config{
		RCAPIURL:                 srv.URL,
		LocalPath:                tmpDir,
		RemotePath:               "remote:bucket",
		LockFile:                 filepath.Join(tmpDir, "conflict.lock"),
		LogFile:                  filepath.Join(tmpDir, "conflict.log"),
		PendingDeletionsFile:     filepath.Join(tmpDir, "deletions.json"),
		ActiveReviewFile:         filepath.Join(tmpDir, "review.json"),
		ProfileID:                "fsm-conflict-prof",
		JobPollInitialSleepSec:   0.005,
		JobPollMaxSleepSec:       0.01,
		JobPollBackoffMultiplier: 1.0,
		JobTimeoutSeconds:        5,
	}

	r := NewSyncRunner(cfg)
	success, err := r.RunWithError(context.Background(), false, false, false, "test", "run-fsm-conflict", true)
	if success {
		t.Fatal("expected sync to fail with review required")
	}
	if err == nil {
		t.Fatal("expected error returned for review required")
	}

	if r.FSM.Current() != engine.StateReviewRequired {
		t.Fatalf("expected FSM state REVIEW_REQUIRED, got %s", r.FSM.Current())
	}
	last := r.FSM.LastEvent()
	if last.ErrorType != "TOO_MANY_DELETES" {
		t.Fatalf("expected error type TOO_MANY_DELETES, got %s", last.ErrorType)
	}
}

func TestSyncRunner_FSM_JobFatalAbortContract(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sync/bisync":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"jobid": 701})
		case "/job/status":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"finished": true,
				"success":  false,
				"error":    "fatal: out of memory during file listing",
			})
		}
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	cfg := &config.Config{
		RCAPIURL:                 srv.URL,
		LocalPath:                tmpDir,
		RemotePath:               "remote:bucket",
		LockFile:                 filepath.Join(tmpDir, "fatal.lock"),
		ProfileID:                "fsm-fatal-prof",
		JobPollInitialSleepSec:   0.005,
		JobPollMaxSleepSec:       0.01,
		JobPollBackoffMultiplier: 1.0,
		JobTimeoutSeconds:        5,
	}

	r := NewSyncRunner(cfg)
	success, err := r.RunWithError(context.Background(), false, false, false, "test", "run-fsm-fatal", true)
	if success || err == nil {
		t.Fatal("expected sync to fail with fatal error")
	}

	if r.FSM.Current() != engine.StateAborted {
		t.Fatalf("expected FSM state ABORTED, got %s", r.FSM.Current())
	}
}

func TestSyncRunner_FSM_MultiRunResetContract(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sync/bisync":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"jobid": 801})
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
	cfg := &config.Config{
		RCAPIURL:                 srv.URL,
		LocalPath:                tmpDir,
		RemotePath:               "remote:bucket",
		LockFile:                 filepath.Join(tmpDir, "multirun.lock"),
		ProfileID:                "fsm-multirun-prof",
		JobPollInitialSleepSec:   0.005,
		JobPollMaxSleepSec:       0.01,
		JobPollBackoffMultiplier: 1.0,
		JobTimeoutSeconds:        5,
	}

	r := NewSyncRunner(cfg)

	// Run 1
	s1, err1 := r.RunWithError(context.Background(), false, false, false, "test", "run-1", true)
	if !s1 || err1 != nil {
		t.Fatalf("run 1 failed: %v", err1)
	}
	if r.FSM.Current() != engine.StateCompleted {
		t.Fatalf("expected COMPLETED after run 1, got %s", r.FSM.Current())
	}

	// Run 2 immediately on the same runner instance
	s2, err2 := r.RunWithError(context.Background(), false, false, false, "test", "run-2", true)
	if !s2 || err2 != nil {
		t.Fatalf("run 2 failed: %v", err2)
	}
	if r.FSM.Current() != engine.StateCompleted {
		t.Fatalf("expected COMPLETED after run 2, got %s", r.FSM.Current())
	}
}

