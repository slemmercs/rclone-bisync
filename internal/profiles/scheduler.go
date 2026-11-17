package profiles

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/core/client"
)

var (
	ErrSchedulerStopped  = errors.New("scheduler is stopped")
	ErrProfileNotActive  = errors.New("profile is not active or enabled")
	ErrSyncInProgress    = errors.New("synchronization is already in progress for this profile")
	ErrSyncCancelled     = errors.New("sync cancelled by operator")
	ErrJobAlreadyRunning = errors.New("job is already running; cannot cancel queued entry")
	ErrJobNotQueued      = errors.New("job is not in queue")
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

// Scheduler coordinates concurrent, flock-isolated goroutines for synchronization profiles
// via a centralized FIFO fair execution queue and dynamic engine option switching.
type Scheduler struct {
	store           *Store
	executor        SyncExecutor
	mu              sync.RWMutex
	workers         map[string]*profileWorker
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	stopped         bool
	paused          bool
	engineClient    EngineOptionsClient
	fleetFlagsFn    func() string
	maxConcurrency  int
	maxQueueWait    time.Duration
	activeSignature EngineSignature
	baselineOptions *client.MainOptions
	currentOptions  *client.MainOptions
	activeJobs      map[string]*QueueItem
	queue           []*QueueItem
	notifyCh        chan struct{}
}

// NewScheduler creates a Scheduler instance with optional engine configuration parameters.
func NewScheduler(store *Store, executor SyncExecutor, opts ...SchedulerOption) *Scheduler {
	s := &Scheduler{
		store:           store,
		executor:        executor,
		workers:         make(map[string]*profileWorker),
		activeJobs:      make(map[string]*QueueItem),
		activeSignature: DefaultEngineSignature,
		maxConcurrency:  4,
		maxQueueWait:    60 * time.Second,
		notifyCh:        make(chan struct{}, 1),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Start begins scheduling loops and launches the unified queue dispatcher.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.stopped = false

	if s.engineClient != nil {
		if opts, err := s.engineClient.GetMainOptions(s.ctx); err == nil && opts != nil {
			s.baselineOptions = opts
			s.currentOptions = opts
		}
	}

	// Launch unified queue dispatcher
	s.wg.Add(1)
	dispCtx := s.ctx
	// #nosec G118 - dispatcher lifecycle is tied to scheduler cancel context
	go s.runDispatcher(dispCtx)

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

// Stop shuts down the scheduler, aborts pending jobs, and restores baseline engine options.
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

	// Abort pending queued jobs
	for _, item := range s.queue {
		if item.ErrCh != nil {
			item.ErrCh <- ErrSchedulerStopped
		}
	}
	s.queue = nil

	for id, w := range s.workers {
		w.cancel()
		delete(s.workers, id)
	}
	s.mu.Unlock()

	s.wg.Wait()

	s.mu.Lock()
	s.restoreBaselineLocked(context.Background())
	s.mu.Unlock()
}

// TriggerSync requests an immediate sync run for a profile with force flag.
func (s *Scheduler) TriggerSync(ctx context.Context, profileID string, force bool) error {
	return s.TriggerSyncWithOptions(ctx, profileID, SyncOptions{Force: force})
}

// TriggerSyncWithOptions requests an immediate sync run for a profile with custom options.
func (s *Scheduler) TriggerSyncWithOptions(ctx context.Context, profileID string, opts SyncOptions) error {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return ErrSchedulerStopped
	}

	_, exists := s.workers[profileID]
	if !exists {
		s.mu.Unlock()
		return ErrProfileNotActive
	}

	if _, active := s.activeJobs[profileID]; active {
		s.mu.Unlock()
		return ErrSyncInProgress
	}
	for _, item := range s.queue {
		if item.ProfileID == profileID {
			s.mu.Unlock()
			return ErrSyncInProgress
		}
	}

	prof, err := s.store.Get(profileID)
	if err != nil || !prof.Enabled || prof.Archived {
		s.mu.Unlock()
		return ErrProfileNotActive
	}

	fleetFlags := s.getFleetFlagsLocked()
	targetFlags, sig, err := ResolveEngineSignature(prof.Flags.CustomFlags, fleetFlags)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to resolve engine signature: %w", err)
	}

	item := &QueueItem{
		ProfileID:   profileID,
		Opts:        opts,
		Signature:   sig,
		TargetFlags: targetFlags,
		EnqueuedAt:  time.Now(),
		ErrCh:       make(chan error, 1),
	}

	s.queue = append(s.queue, item)
	s.signalDispatcherLocked()
	s.mu.Unlock()

	select {
	case err := <-item.ErrCh:
		return err
	case <-ctx.Done():
		s.mu.Lock()
		s.removeFromQueueLocked(item)
		s.mu.Unlock()
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
	_, active := s.activeJobs[profileID]
	return active
}

// IsProfileQueued reports whether a profile is queued awaiting execution.
func (s *Scheduler) IsProfileQueued(profileID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.stopped {
		return false
	}
	for _, item := range s.queue {
		if item.ProfileID == profileID {
			return true
		}
	}
	return false
}

// IsAnyRunning reports whether any profile worker is actively synchronizing.
func (s *Scheduler) IsAnyRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.stopped {
		return false
	}
	return len(s.activeJobs) > 0
}

// GetActiveSignature returns the currently active engine configuration signature.
func (s *Scheduler) GetActiveSignature() EngineSignature {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeSignature
}

// GetQueueLength returns the number of jobs waiting in the fair queue.
func (s *Scheduler) GetQueueLength() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.queue)
}

// GetQueuedProfileIDs returns a list of profile IDs currently in the queue.
func (s *Scheduler) GetQueuedProfileIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, len(s.queue))
	for i, item := range s.queue {
		ids[i] = item.ProfileID
	}
	return ids
}

// PauseDispatcher halts admission of new queued jobs while allowing active jobs to finish.
func (s *Scheduler) PauseDispatcher() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paused = true
}

// ResumeDispatcher enables admission of queued jobs and triggers immediate dispatch.
func (s *Scheduler) ResumeDispatcher() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paused = false
	s.signalDispatcherLocked()
}

// IsDispatcherPaused reports whether the dispatcher has halted admissions.
func (s *Scheduler) IsDispatcherPaused() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.paused
}

// CancelQueued removes a waiting job from the queue and cancels its execution.
func (s *Scheduler) CancelQueued(profileID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, running := s.activeJobs[profileID]; running {
		return ErrJobAlreadyRunning
	}

	for i, item := range s.queue {
		if item.ProfileID == profileID {
			s.queue = append(s.queue[:i], s.queue[i+1:]...)
			if item.ErrCh != nil {
				item.ErrCh <- ErrSyncCancelled
			}
			return nil
		}
	}

	return ErrJobNotQueued
}

// ClearQueue drains and cancels all pending queued jobs, restoring baseline engine options if idle.
func (s *Scheduler) ClearQueue() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := len(s.queue)
	if count == 0 {
		return 0
	}

	for _, item := range s.queue {
		if item.ErrCh != nil {
			item.ErrCh <- ErrSyncCancelled
		}
	}
	s.queue = nil

	if len(s.activeJobs) == 0 && s.activeSignature != DefaultEngineSignature {
		s.restoreBaselineLocked(context.Background())
	}

	return count
}

func (s *Scheduler) runJob(ctx context.Context, item *QueueItem) {
	defer s.wg.Done()

	err := s.executeJob(ctx, item)

	if item.ErrCh != nil {
		item.ErrCh <- err
	}

	s.mu.Lock()
	delete(s.activeJobs, item.ProfileID)
	if w, ok := s.workers[item.ProfileID]; ok {
		w.isSyncing.Store(false)
	}
	s.signalDispatcherLocked()
	s.mu.Unlock()
}

func (s *Scheduler) executeJob(ctx context.Context, item *QueueItem) error {
	prof, err := s.store.Get(item.ProfileID)
	if err != nil || !prof.Enabled || prof.Archived {
		return ErrProfileNotActive
	}

	state := prof.State
	state.Status = StatusRunning
	_ = s.store.UpdateState(item.ProfileID, state)

	startTime := time.Now().UTC()
	var execErr error
	if s.executor != nil {
		execErr = s.executor(ctx, prof, item.Opts)
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

	_ = s.store.UpdateState(item.ProfileID, state)
	return execErr
}
