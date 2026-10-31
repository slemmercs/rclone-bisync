package profiles

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrSchedulerStopped = errors.New("scheduler is stopped")
	ErrProfileNotActive = errors.New("profile is not active or enabled")
	ErrSyncInProgress   = errors.New("synchronization is already in progress for this profile")
)

// SyncOptions specifies parameters for a synchronization run.
type SyncOptions struct {
	Force  bool   `json:"force"`
	Resync bool   `json:"resync"`
	DryRun bool   `json:"dry_run"`
	Origin string `json:"origin,omitempty"`
}

// SyncExecutor executes a single sync run for a profile.
type SyncExecutor func(ctx context.Context, p *Profile, opts SyncOptions) error

type triggerReq struct {
	opts  SyncOptions
	errCh chan error
}

type profileWorker struct {
	profileID string
	triggerCh chan triggerReq
	cancel    context.CancelFunc
	running   sync.Mutex
	isSyncing atomic.Bool
}

// Scheduler coordinates concurrent, flock-isolated goroutines for synchronization profiles.
type Scheduler struct {
	store    *Store
	executor SyncExecutor
	mu       sync.RWMutex
	workers  map[string]*profileWorker
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	stopped  bool
}

// NewScheduler creates a Scheduler instance.
func NewScheduler(store *Store, executor SyncExecutor) *Scheduler {
	return &Scheduler{
		store:    store,
		executor: executor,
		workers:  make(map[string]*profileWorker),
	}
}

// Start begins scheduling loops for all enabled profiles.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.stopped = false

	if s.store == nil {
		return nil
	}

	for _, p := range s.store.List(false) {
		if p.Enabled && !p.Archived {
			s.startWorkerLocked(p)
		}
	}
	return nil
}

// Stop shuts down all profile workers gracefully.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	if s.cancel != nil {
		s.cancel()
	}
	for id, w := range s.workers {
		w.cancel()
		delete(s.workers, id)
	}
	s.mu.Unlock()

	s.wg.Wait()
}

// TriggerSync requests an immediate sync run for a profile with force flag.
func (s *Scheduler) TriggerSync(ctx context.Context, profileID string, force bool) error {
	return s.TriggerSyncWithOptions(ctx, profileID, SyncOptions{Force: force})
}

// TriggerSyncWithOptions requests an immediate sync run for a profile with custom options.
func (s *Scheduler) TriggerSyncWithOptions(ctx context.Context, profileID string, opts SyncOptions) error {
	s.mu.RLock()
	if s.stopped {
		s.mu.RUnlock()
		return ErrSchedulerStopped
	}
	w, exists := s.workers[profileID]
	s.mu.RUnlock()

	if !exists {
		return ErrProfileNotActive
	}

	req := triggerReq{
		opts:  opts,
		errCh: make(chan error, 1),
	}

	select {
	case w.triggerCh <- req:
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrSyncInProgress
	}

	select {
	case err := <-req.errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// IsProfileRunning reports whether a profile is actively executing a synchronization cycle.
func (s *Scheduler) IsProfileRunning(profileID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.stopped {
		return false
	}
	w, exists := s.workers[profileID]
	if !exists {
		return false
	}
	return w.isSyncing.Load()
}

// IsAnyRunning reports whether any profile worker is actively synchronizing.
func (s *Scheduler) IsAnyRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.stopped {
		return false
	}
	for _, w := range s.workers {
		if w.isSyncing.Load() {
			return true
		}
	}
	return false
}

// RestartProfile updates or starts a profile worker reflecting new configuration.
func (s *Scheduler) RestartProfile(p *Profile) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return
	}

	s.stopWorkerLocked(p.ID)
	if p.Enabled && !p.Archived {
		s.startWorkerLocked(p)
	}
}

// StopProfile halts background execution for a profile.
func (s *Scheduler) StopProfile(profileID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopWorkerLocked(profileID)
}

func (s *Scheduler) stopWorkerLocked(profileID string) {
	if w, exists := s.workers[profileID]; exists {
		w.cancel()
		delete(s.workers, profileID)
	}
}

func (s *Scheduler) startWorkerLocked(p *Profile) {
	if s.ctx == nil {
		return
	}
	wCtx, wCancel := context.WithCancel(s.ctx)
	w := &profileWorker{
		profileID: p.ID,
		triggerCh: make(chan triggerReq, 1),
		cancel:    wCancel,
	}
	s.workers[p.ID] = w

	s.wg.Add(1)
	go s.runWorker(wCtx, w, p)
}

func (s *Scheduler) runWorker(ctx context.Context, w *profileWorker, initialProf *Profile) {
	defer s.wg.Done()

	profileID := w.profileID
	interval := ParseSchedule(initialProf.Schedule)
	var ticker *time.Ticker
	var tickerCh <-chan time.Time

	if interval > 0 {
		ticker = time.NewTicker(interval)
		tickerCh = ticker.C
		defer ticker.Stop()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-tickerCh:
			s.executeProfileSync(ctx, w, profileID, SyncOptions{Origin: "scheduler"}, nil)
		case req := <-w.triggerCh:
			s.executeProfileSync(ctx, w, profileID, req.opts, req.errCh)
		}
	}
}

func (s *Scheduler) executeProfileSync(
	ctx context.Context,
	w *profileWorker,
	profileID string,
	opts SyncOptions,
	errCh chan error,
) {
	if !w.running.TryLock() {
		if errCh != nil {
			errCh <- ErrSyncInProgress
		}
		return
	}
	w.isSyncing.Store(true)
	defer func() {
		w.isSyncing.Store(false)
		w.running.Unlock()
	}()

	prof, err := s.store.Get(profileID)
	if err != nil || !prof.Enabled || prof.Archived {
		if errCh != nil {
			errCh <- ErrProfileNotActive
		}
		return
	}

	state := prof.State
	state.Status = StatusRunning
	_ = s.store.UpdateState(profileID, state)

	startTime := time.Now().UTC()
	var execErr error
	if s.executor != nil {
		execErr = s.executor(ctx, prof, opts)
	}
	duration := time.Since(startTime).Seconds()

	now := time.Now().UTC()
	state.LastSync = &now
	state.LastDurationSec = duration

	interval := ParseSchedule(prof.Schedule)
	if interval > 0 {
		next := now.Add(interval)
		state.NextSync = &next
	}

	if execErr != nil {
		state.ConsecutiveFailures++
		state.LastError = execErr.Error()
		if prof.Safety.FailureThreshold > 0 && state.ConsecutiveFailures >= prof.Safety.FailureThreshold {
			state.Status = StatusPaused
		} else {
			state.Status = StatusSyncFailed
		}
	} else {
		state.ConsecutiveFailures = 0
		state.LastError = ""
		state.Status = StatusHealthy
	}

	_ = s.store.UpdateState(profileID, state)
	w.isSyncing.Store(false)

	if errCh != nil {
		errCh <- execErr
	}
}

// ParseSchedule converts schedule expressions into time.Duration.
func ParseSchedule(schedule string) time.Duration {
	s := strings.ToLower(strings.TrimSpace(schedule))
	switch s {
	case "", "manual":
		return 0
	case "daily":
		return 24 * time.Hour
	case "hourly":
		return 1 * time.Hour
	default:
		d, err := time.ParseDuration(s)
		if err != nil || d <= 0 {
			slog.Warn("invalid schedule, defaulting to manual", "schedule", schedule)
			return 0
		}
		return d
	}
}
