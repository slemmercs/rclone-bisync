package profiles

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/core/client"
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

	executor := func(ctx context.Context, prof *Profile, opts SyncOptions) error {
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

func TestScheduler_RunningStateQueries(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(filepath.Join(tmpDir, "p.json"), filepath.Join(tmpDir, "h.json"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	p1 := &Profile{
		ID:        "prof-running-1",
		Name:      "Running 1",
		Remote:    "gdrive",
		LocalPath: "/tmp",
		Enabled:   true,
		Schedule:  "manual",
	}
	p2 := &Profile{
		ID:        "prof-idle-2",
		Name:      "Idle 2",
		Remote:    "onedrive",
		LocalPath: "/tmp",
		Enabled:   true,
		Schedule:  "manual",
	}
	_ = store.Save(p1, "Init 1")
	_ = store.Save(p2, "Init 2")

	started := make(chan struct{})
	release := make(chan struct{})

	executor := func(ctx context.Context, p *Profile, opts SyncOptions) error {
		if p.ID == "prof-running-1" {
			close(started)
			<-release
		}
		return nil
	}

	sched := NewScheduler(store, executor)
	svc := NewService(store, nil, sched)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := sched.Start(ctx); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	// Initially idle
	if sched.IsAnyRunning() {
		t.Fatalf("expected IsAnyRunning = false initially")
	}
	if svc.IsProfileSyncing("prof-running-1") {
		t.Fatalf("expected prof-running-1 idle initially")
	}

	doneCh := make(chan error, 1)
	go func() {
		doneCh <- sched.TriggerSync(context.Background(), "prof-running-1", false)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for sync to begin")
	}

	// Verify running state queries
	if !sched.IsProfileRunning("prof-running-1") {
		t.Errorf("expected IsProfileRunning(prof-running-1) = true")
	}
	if sched.IsProfileRunning("prof-idle-2") {
		t.Errorf("expected IsProfileRunning(prof-idle-2) = false")
	}
	if !sched.IsAnyRunning() {
		t.Errorf("expected IsAnyRunning = true")
	}
	if !svc.IsAnySyncing() {
		t.Errorf("expected svc.IsAnySyncing = true")
	}
	if !svc.IsProfileSyncing("prof-running-1") {
		t.Errorf("expected svc.IsProfileSyncing(prof-running-1) = true")
	}
	if svc.IsProfileSyncing("prof-idle-2") {
		t.Errorf("expected svc.IsProfileSyncing(prof-idle-2) = false")
	}

	// Release and finish
	close(release)
	if err := <-doneCh; err != nil {
		t.Fatalf("expected nil error from sync, got: %v", err)
	}

	if sched.IsProfileRunning("prof-running-1") {
		t.Errorf("expected IsProfileRunning(prof-running-1) = false after completion")
	}
	if sched.IsAnyRunning() {
		t.Errorf("expected IsAnyRunning = false after completion")
	}
	if svc.IsAnySyncing() {
		t.Errorf("expected svc.IsAnySyncing = false after completion")
	}

	// Stopped scheduler
	sched.Stop()
	if sched.IsAnyRunning() {
		t.Errorf("expected IsAnyRunning = false when stopped")
	}
	if sched.IsProfileRunning("prof-running-1") {
		t.Errorf("expected IsProfileRunning = false when stopped")
	}
}
type mockEngineClient struct {
	mu           sync.Mutex
	mainOptions  *client.MainOptions
	getCalls     int
	setCalls     []client.MainOptions
	failSetCalls bool
}

func (m *mockEngineClient) GetMainOptions(ctx context.Context) (*client.MainOptions, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getCalls++
	if m.mainOptions == nil {
		return &client.MainOptions{
			Checkers: client.IntPtr(8),
			TPSLimit: client.Float64Ptr(0),
		}, nil
	}
	return m.mainOptions, nil
}

func (m *mockEngineClient) SetMainOptions(ctx context.Context, opts *client.MainOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failSetCalls {
		return errors.New("simulated set options failure")
	}
	if opts != nil {
		m.setCalls = append(m.setCalls, *opts)
		m.mainOptions = opts
	}
	return nil
}

func TestScheduler_FairQueueAndClustering(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(filepath.Join(tmpDir, "p.json"), filepath.Join(tmpDir, "h.json"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// P1 and P2 share signature SIG_CHK=8_TPS=4
	p1 := &Profile{
		ID:        "prof-c1",
		Name:      "C1",
		Remote:    "remote1",
		LocalPath: "/tmp",
		Enabled:   true,
		Schedule:  "manual",
		Flags:     ProfileFlags{CustomFlags: "--checkers 8 --tpslimit 4"},
	}
	p2 := &Profile{
		ID:        "prof-c2",
		Name:      "C2",
		Remote:    "remote2",
		LocalPath: "/tmp",
		Enabled:   true,
		Schedule:  "manual",
		Flags:     ProfileFlags{CustomFlags: "--tpslimit 4.0 --checkers 8"},
	}
	// P3 has differing signature SIG_CHK=16_TPS=10
	p3 := &Profile{
		ID:        "prof-c3",
		Name:      "C3",
		Remote:    "remote3",
		LocalPath: "/tmp",
		Enabled:   true,
		Schedule:  "manual",
		Flags:     ProfileFlags{CustomFlags: "--checkers 16 --tpslimit 10"},
	}
	_ = store.Save(p1, "Init 1")
	_ = store.Save(p2, "Init 2")
	_ = store.Save(p3, "Init 3")

	mockEngine := &mockEngineClient{
		mainOptions: &client.MainOptions{
			Checkers: client.IntPtr(4),
			TPSLimit: client.Float64Ptr(0),
		},
	}

	var activeConcurrent int64
	var maxConcurrentObserved int64
	p1p2Barrier := make(chan struct{})

	executor := func(ctx context.Context, p *Profile, opts SyncOptions) error {
		cur := atomic.AddInt64(&activeConcurrent, 1)
		defer atomic.AddInt64(&activeConcurrent, -1)

		for {
			oldMax := atomic.LoadInt64(&maxConcurrentObserved)
			if cur <= oldMax || atomic.CompareAndSwapInt64(&maxConcurrentObserved, oldMax, cur) {
				break
			}
		}

		if p.ID == "prof-c1" || p.ID == "prof-c2" {
			<-p1p2Barrier
		}
		return nil
	}

	sched := NewScheduler(store, executor,
		WithEngineClient(mockEngine),
		WithMaxConcurrency(2),
		WithMaxQueueWait(5*time.Second),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := sched.Start(ctx); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}
	defer sched.Stop()

	// Enqueue P1 and P2 (compatible)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = sched.TriggerSync(context.Background(), "prof-c1", false)
	}()
	go func() {
		defer wg.Done()
		_ = sched.TriggerSync(context.Background(), "prof-c2", false)
	}()

	// Wait for both to be running concurrently
	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt64(&activeConcurrent) != 2 {
		t.Fatalf("expected P1 and P2 to execute concurrently in parallel, got %d", atomic.LoadInt64(&activeConcurrent))
	}

	// Trigger P3 (incompatible signature)
	p3Done := make(chan error, 1)
	go func() {
		p3Done <- sched.TriggerSync(context.Background(), "prof-c3", false)
	}()

	// Verify P3 is queued and waiting while P1 & P2 run
	time.Sleep(50 * time.Millisecond)
	if !sched.IsProfileQueued("prof-c3") {
		t.Errorf("expected P3 to be queued waiting for cohort to drain")
	}

	// Release P1 & P2
	close(p1p2Barrier)
	wg.Wait()

	// Wait for P3 to finish
	select {
	case err := <-p3Done:
		if err != nil {
			t.Fatalf("P3 failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for P3")
	}

	if atomic.LoadInt64(&maxConcurrentObserved) != 2 {
		t.Errorf("expected maximum concurrency of 2, observed %d", atomic.LoadInt64(&maxConcurrentObserved))
	}

	// Verify SetMainOptions was called for P1/P2 (SIG_CHK=8_TPS=4) and then P3 (SIG_CHK=16_TPS=10)
	mockEngine.mu.Lock()
	defer mockEngine.mu.Unlock()
	if len(mockEngine.setCalls) < 2 {
		t.Fatalf("expected at least 2 SetMainOptions calls, got %d", len(mockEngine.setCalls))
	}
}

func TestScheduler_StarvationAgingSLA(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(filepath.Join(tmpDir, "p.json"), filepath.Join(tmpDir, "h.json"))

	pA1 := &Profile{
		ID:        "prof-a1",
		Name:      "A1",
		Remote:    "r1",
		LocalPath: "/tmp",
		Enabled:   true,
		Schedule:  "manual",
		Flags:     ProfileFlags{CustomFlags: "--tpslimit 4"},
	}
	pB := &Profile{
		ID:        "prof-b",
		Name:      "B",
		Remote:    "r2",
		LocalPath: "/tmp",
		Enabled:   true,
		Schedule:  "manual",
		Flags:     ProfileFlags{CustomFlags: "--tpslimit 12"},
	}
	pA2 := &Profile{
		ID:        "prof-a2",
		Name:      "A2",
		Remote:    "r3",
		LocalPath: "/tmp",
		Enabled:   true,
		Schedule:  "manual",
		Flags:     ProfileFlags{CustomFlags: "--tpslimit 4"},
	}
	_ = store.Save(pA1, "Init A1")
	_ = store.Save(pB, "Init B")
	_ = store.Save(pA2, "Init A2")

	var a1BarrierOnce sync.Once
	a1Barrier := make(chan struct{})
	closeA1 := func() { a1BarrierOnce.Do(func() { close(a1Barrier) }) }
	defer closeA1()

	var execOrder []string
	var orderMu sync.Mutex

	executor := func(ctx context.Context, p *Profile, opts SyncOptions) error {
		orderMu.Lock()
		execOrder = append(execOrder, p.ID)
		orderMu.Unlock()

		if p.ID == "prof-a1" {
			<-a1Barrier
		}
		return nil
	}

	sched := NewScheduler(store, executor,
		WithMaxConcurrency(2),
		WithMaxQueueWait(30*time.Millisecond), // Short SLA for fast testing
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = sched.Start(ctx)
	defer sched.Stop()

	// 1. Trigger A1 (starts executing)
	go func() {
		_ = sched.TriggerSync(context.Background(), "prof-a1", false)
	}()
	time.Sleep(30 * time.Millisecond)

	// 2. Queue B (different signature)
	go func() {
		_ = sched.TriggerSync(context.Background(), "prof-b", false)
	}()

	// 3. Wait for B's queue wait to exceed 30ms SLA so B becomes starved
	time.Sleep(50 * time.Millisecond)

	// 4. Now queue A2 (same signature as A1)
	a2Done := make(chan error, 1)
	go func() {
		a2Done <- sched.TriggerSync(context.Background(), "prof-a2", false)
	}()
	time.Sleep(30 * time.Millisecond)

	// Even though slot is available, A2 should NOT be dispatched because B is starved!
	orderMu.Lock()
	if len(execOrder) > 1 {
		t.Fatalf("expected only A1 running due to B's SLA starvation barrier, got: %v", execOrder)
	}
	orderMu.Unlock()

	// 5. Release A1
	closeA1()

	// Wait for A2 to finish
	select {
	case <-a2Done:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for A2")
	}

	orderMu.Lock()
	defer orderMu.Unlock()
	// B must execute before A2
	if len(execOrder) != 3 || execOrder[0] != "prof-a1" || execOrder[1] != "prof-b" || execOrder[2] != "prof-a2" {
		t.Fatalf("expected execution order [prof-a1, prof-b, prof-a2], got: %v", execOrder)
	}
}

func TestScheduler_BaselineRestore(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(filepath.Join(tmpDir, "p.json"), filepath.Join(tmpDir, "h.json"))

	p := &Profile{
		ID:        "prof-custom",
		Name:      "Custom",
		Remote:    "r1",
		LocalPath: "/tmp",
		Enabled:   true,
		Schedule:  "manual",
		Flags:     ProfileFlags{CustomFlags: "--checkers 12 --tpslimit 2"},
	}
	_ = store.Save(p, "Init")

	mockEngine := &mockEngineClient{
		mainOptions: &client.MainOptions{
			Checkers: client.IntPtr(8),
			TPSLimit: client.Float64Ptr(0),
		},
	}

	sched := NewScheduler(store, nil,
		WithEngineClient(mockEngine),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = sched.Start(ctx)

	// Trigger custom run
	if err := sched.TriggerSync(context.Background(), "prof-custom", false); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	// Wait for queue to drain and baseline restore
	time.Sleep(100 * time.Millisecond)

	mockEngine.mu.Lock()
	defer mockEngine.mu.Unlock()
	if len(mockEngine.setCalls) < 2 {
		t.Fatalf("expected at least 2 SetMainOptions calls (switch + restore), got %d", len(mockEngine.setCalls))
	}
	lastCall := mockEngine.setCalls[len(mockEngine.setCalls)-1]
	if *lastCall.Checkers != 8 {
		t.Errorf("expected restored checkers to be baseline (8), got %d", *lastCall.Checkers)
	}

	sched.Stop()
}

func TestScheduler_OptionSwitchErrorFallback(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(filepath.Join(tmpDir, "p.json"), filepath.Join(tmpDir, "h.json"))

	p := &Profile{
		ID:        "prof-fallback",
		Name:      "Fallback",
		Remote:    "r1",
		LocalPath: "/tmp",
		Enabled:   true,
		Schedule:  "manual",
		Flags:     ProfileFlags{CustomFlags: "--tpslimit 5"},
	}
	_ = store.Save(p, "Init")

	mockEngine := &mockEngineClient{
		failSetCalls: true,
	}

	var executed bool
	executor := func(ctx context.Context, p *Profile, opts SyncOptions) error {
		executed = true
		return nil
	}

	sched := NewScheduler(store, executor,
		WithEngineClient(mockEngine),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = sched.Start(ctx)
	defer sched.Stop()

	if err := sched.TriggerSync(context.Background(), "prof-fallback", false); err != nil {
		t.Fatalf("expected profile execution to succeed despite SetMainOptions error, got: %v", err)
	}
	if !executed {
		t.Fatal("expected executor to have run")
	}
}

func TestScheduler_DuplicateEnqueueRejected(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(filepath.Join(tmpDir, "p.json"), filepath.Join(tmpDir, "h.json"))

	p := &Profile{
		ID:        "prof-dup",
		Name:      "Dup",
		Remote:    "r1",
		LocalPath: "/tmp",
		Enabled:   true,
		Schedule:  "manual",
	}
	_ = store.Save(p, "Init")

	barrier := make(chan struct{})
	executor := func(ctx context.Context, p *Profile, opts SyncOptions) error {
		<-barrier
		return nil
	}

	sched := NewScheduler(store, executor)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = sched.Start(ctx)
	defer sched.Stop()

	go func() {
		_ = sched.TriggerSync(context.Background(), "prof-dup", false)
	}()

	time.Sleep(50 * time.Millisecond)

	err := sched.TriggerSync(context.Background(), "prof-dup", false)
	if err != ErrSyncInProgress {
		t.Fatalf("expected ErrSyncInProgress for concurrent trigger, got: %v", err)
	}

	close(barrier)
}

func TestScheduler_StopDrainsPendingQueue(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(filepath.Join(tmpDir, "p.json"), filepath.Join(tmpDir, "h.json"))

	p1 := &Profile{
		ID:        "prof-d1",
		Name:      "D1",
		Remote:    "r1",
		LocalPath: "/tmp",
		Enabled:   true,
		Schedule:  "manual",
	}
	p2 := &Profile{
		ID:        "prof-d2",
		Name:      "D2",
		Remote:    "r2",
		LocalPath: "/tmp",
		Enabled:   true,
		Schedule:  "manual",
	}
	_ = store.Save(p1, "Init 1")
	_ = store.Save(p2, "Init 2")

	barrier := make(chan struct{})
	executor := func(ctx context.Context, p *Profile, opts SyncOptions) error {
		<-barrier
		return nil
	}

	sched := NewScheduler(store, executor, WithMaxConcurrency(1))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = sched.Start(ctx)

	go func() {
		_ = sched.TriggerSync(context.Background(), "prof-d1", false)
	}()
	time.Sleep(40 * time.Millisecond)

	p2ErrCh := make(chan error, 1)
	go func() {
		p2ErrCh <- sched.TriggerSync(context.Background(), "prof-d2", false)
	}()
	time.Sleep(40 * time.Millisecond)

	close(barrier)
	sched.Stop()

	select {
	case err := <-p2ErrCh:
		if err != ErrSchedulerStopped {
			t.Fatalf("expected ErrSchedulerStopped, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for queued job error")
	}
}

