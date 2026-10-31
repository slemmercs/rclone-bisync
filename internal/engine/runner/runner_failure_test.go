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
	"github.com/slemmercs/rclone-bisync/internal/core/logger"
	"github.com/slemmercs/rclone-bisync/internal/core/state"
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
