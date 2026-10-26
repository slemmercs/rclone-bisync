package profiles

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseSchedule(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"manual", 0},
		{"", 0},
		{"daily", 24 * time.Hour},
		{"hourly", 1 * time.Hour},
		{"15m", 15 * time.Minute},
		{"2h", 2 * time.Hour},
		{"invalid", 0},
	}

	for _, tt := range tests {
		got := ParseSchedule(tt.input)
		if got != tt.expected {
			t.Errorf("ParseSchedule(%q) = %v; expected %v", tt.input, got, tt.expected)
		}
	}
}

func TestScheduler_ExecutionAndFlock(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(filepath.Join(tmpDir, "p.json"), filepath.Join(tmpDir, "h.json"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	p := &Profile{
		ID:        "sync-work",
		Name:      "Work Sync",
		Remote:    "gdrive",
		LocalPath: "/tmp",
		Enabled:   true,
		Schedule:  "manual",
		Safety: SafetySettings{
			FailureThreshold: 2,
		},
	}
	if err := store.Save(p, "Init"); err != nil {
		t.Fatalf("failed to save profile: %v", err)
	}

	var execCount int64
	var shouldFail bool

	executor := func(ctx context.Context, prof *Profile, force bool) error {
		atomic.AddInt64(&execCount, 1)
		if shouldFail {
			return errors.New("simulated sync error")
		}
		return nil
	}

	sched := NewScheduler(store, executor)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	// 1. Trigger manual sync successfully
	if err := sched.TriggerSync(context.Background(), "sync-work", false); err != nil {
		t.Fatalf("expected sync success, got: %v", err)
	}
	if atomic.LoadInt64(&execCount) != 1 {
		t.Fatalf("expected execCount 1, got %d", atomic.LoadInt64(&execCount))
	}

	updatedP, _ := store.Get("sync-work")
	if updatedP.State.Status != StatusHealthy || updatedP.State.ConsecutiveFailures != 0 {
		t.Fatalf("expected healthy status, got %+v", updatedP.State)
	}

	// 2. Trigger failure 1
	shouldFail = true
	err = sched.TriggerSync(context.Background(), "sync-work", false)
	if err == nil {
		t.Fatalf("expected error from failed sync")
	}
	updatedP, _ = store.Get("sync-work")
	if updatedP.State.Status != StatusSyncFailed || updatedP.State.ConsecutiveFailures != 1 {
		t.Fatalf("expected StatusSyncFailed and 1 failure, got %+v", updatedP.State)
	}

	// 3. Trigger failure 2 -> reaches threshold 2 -> StatusPaused
	err = sched.TriggerSync(context.Background(), "sync-work", false)
	if err == nil {
		t.Fatalf("expected error from failed sync")
	}
	updatedP, _ = store.Get("sync-work")
	if updatedP.State.Status != StatusPaused || updatedP.State.ConsecutiveFailures != 2 {
		t.Fatalf("expected StatusPaused after 2 failures, got %+v", updatedP.State)
	}

	// 4. Test missing profile trigger
	if err := sched.TriggerSync(context.Background(), "unknown", false); err != ErrProfileNotActive {
		t.Fatalf("expected ErrProfileNotActive, got: %v", err)
	}

	// 5. Test Stop and Trigger on stopped
	sched.Stop()
	if err := sched.TriggerSync(context.Background(), "sync-work", false); err != ErrSchedulerStopped {
		t.Fatalf("expected ErrSchedulerStopped, got: %v", err)
	}
}

func TestScheduler_RestartAndStopProfile(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(filepath.Join(tmpDir, "p.json"), filepath.Join(tmpDir, "h.json"))
	p := &Profile{
		ID:        "prof-1",
		Name:      "P1",
		Remote:    "remote1",
		LocalPath: "/tmp",
		Enabled:   true,
		Schedule:  "manual",
	}
	_ = store.Save(p, "Init")

	sched := NewScheduler(store, nil)
	_ = sched.Start(context.Background())
	defer sched.Stop()

	// Stop profile worker
	sched.StopProfile("prof-1")
	if err := sched.TriggerSync(context.Background(), "prof-1", false); err != ErrProfileNotActive {
		t.Fatalf("expected ErrProfileNotActive after StopProfile, got: %v", err)
	}

	// Restart profile worker
	sched.RestartProfile(p)
	// Now worker is active again (though executor is nil, so it will succeed with nil error)
	if err := sched.TriggerSync(context.Background(), "prof-1", false); err != nil {
		t.Fatalf("expected nil error after RestartProfile, got: %v", err)
	}
}
