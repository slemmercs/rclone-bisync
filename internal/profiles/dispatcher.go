package profiles

import (
	"context"
	"log/slog"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/core/client"
)

// EngineOptionsClient defines the dynamic engine configuration interface for the scheduler.
type EngineOptionsClient interface {
	GetMainOptions(ctx context.Context) (*client.MainOptions, error)
	SetMainOptions(ctx context.Context, opts *client.MainOptions) error
}

// QueueItem represents a queued synchronization execution job.
type QueueItem struct {
	ProfileID   string
	Opts        SyncOptions
	Signature   EngineSignature
	TargetFlags ProcessGlobalFlags
	EnqueuedAt  time.Time
	StartedAt   *time.Time
	ErrCh       chan error
}

// QueueJobDTO provides safe serialized inspection of an active or pending queue job.
type QueueJobDTO struct {
	ProfileID   string          `json:"profile_id"`
	ProfileName string          `json:"profile_name"`
	Origin      string          `json:"origin"`
	DryRun      bool            `json:"dry_run"`
	Resync      bool            `json:"resync"`
	Force       bool            `json:"force"`
	Signature   EngineSignature `json:"signature"`
	EnqueuedAt  time.Time       `json:"enqueued_at"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	WaitSeconds float64         `json:"wait_seconds"`
}

// QueueSnapshotDTO provides a point-in-time snapshot of scheduler queue state.
type QueueSnapshotDTO struct {
	Paused          bool            `json:"paused"`
	ActiveSignature EngineSignature `json:"active_signature"`
	MaxConcurrency  int             `json:"max_concurrency"`
	RunningCount    int             `json:"running_count"`
	QueuedCount     int             `json:"queued_count"`
	ActiveJobs      []QueueJobDTO   `json:"active_jobs"`
	QueuedJobs      []QueueJobDTO   `json:"queued_jobs"`
}

// SchedulerOption configures the Scheduler.
type SchedulerOption func(*Scheduler)

// WithEngineClient configures an Rclone options client for dynamic process-global flag switching.
func WithEngineClient(c EngineOptionsClient) SchedulerOption {
	return func(s *Scheduler) {
		s.engineClient = c
	}
}

// WithFleetFlags configures a supplier function for global fleet custom flags.
func WithFleetFlags(fn func() string) SchedulerOption {
	return func(s *Scheduler) {
		s.fleetFlagsFn = fn
	}
}

// WithMaxConcurrency sets the maximum number of parallel profile syncs.
func WithMaxConcurrency(max int) SchedulerOption {
	return func(s *Scheduler) {
		if max > 0 {
			s.maxConcurrency = max
		}
	}
}

// WithMaxQueueWait sets the maximum duration a queued job can wait before triggering an SLA-mandated cohort switch.
func WithMaxQueueWait(d time.Duration) SchedulerOption {
	return func(s *Scheduler) {
		if d > 0 {
			s.maxQueueWait = d
		}
	}
}

// runDispatcher processes the centralized run queue, coordinating opportunistic batching,
// starvation-free SLA cohort switches, and baseline option restoration.
func (s *Scheduler) runDispatcher(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.mu.Lock()
			s.restoreBaselineLocked(context.Background())
			s.mu.Unlock()
			return
		case <-s.notifyCh:
			s.processDispatch(ctx)
		case <-ticker.C:
			s.processDispatch(ctx)
		}
	}
}

func (s *Scheduler) signalDispatcherLocked() {
	select {
	case s.notifyCh <- struct{}{}:
	default:
	}
}

func (s *Scheduler) getFleetFlagsLocked() string {
	if s.fleetFlagsFn != nil {
		return s.fleetFlagsFn()
	}
	return ""
}

func (s *Scheduler) switchSignatureLocked(ctx context.Context, sig EngineSignature, flags ProcessGlobalFlags) {
	s.activeSignature = sig
	if s.engineClient == nil {
		return
	}
	opts := flags.ToMainOptions()
	s.currentOptions = opts

	switchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.engineClient.SetMainOptions(switchCtx, opts); err != nil {
		slog.Warn("failed to switch Rclone main engine options; proceeding with current settings",
			"signature", sig, "error", err)
	}
}

func (s *Scheduler) restoreBaselineLocked(ctx context.Context) {
	if s.engineClient == nil || s.baselineOptions == nil {
		s.activeSignature = DefaultEngineSignature
		return
	}
	if s.activeSignature == DefaultEngineSignature && s.currentOptions == s.baselineOptions {
		return
	}

	restoreCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.engineClient.SetMainOptions(restoreCtx, s.baselineOptions); err != nil {
		slog.Warn("failed to restore baseline Rclone engine options", "error", err)
	}
	s.currentOptions = s.baselineOptions
	s.activeSignature = DefaultEngineSignature
}

func (s *Scheduler) removeFromQueueLocked(target *QueueItem) {
	for i, item := range s.queue {
		if item == target {
			s.queue = append(s.queue[:i], s.queue[i+1:]...)
			return
		}
	}
}

// processDispatch evaluates the FIFO queue and dispatches compatible jobs.
func (s *Scheduler) processDispatch(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped || s.paused {
		return
	}

	// 1. If queue is empty and active cohort has drained, restore baseline options
	if len(s.queue) == 0 {
		if len(s.activeJobs) == 0 && s.activeSignature != DefaultEngineSignature {
			s.restoreBaselineLocked(ctx)
		}
		return
	}

	// 2. Evaluate starvation on the oldest item in the queue
	oldest := s.queue[0]
	isOldestStarved := time.Since(oldest.EnqueuedAt) >= s.maxQueueWait

	// If oldest item requires a different signature:
	if oldest.Signature != s.activeSignature {
		if len(s.activeJobs) == 0 {
			// All active jobs have drained -> switch signature immediately
			s.switchSignatureLocked(ctx, oldest.Signature, oldest.TargetFlags)
		} else if isOldestStarved {
			// Oldest item is starved! Halt new admissions to let active jobs drain
			return
		}
	}

	// If active jobs are running with a different signature and oldest is starved, hold off
	if len(s.activeJobs) > 0 && oldest.Signature != s.activeSignature && isOldestStarved {
		return
	}

	// 3. Dispatch compatible jobs up to available concurrency limit
	availableSlots := s.maxConcurrency - len(s.activeJobs)
	if availableSlots <= 0 {
		return
	}

	// If no jobs are active, adopt the signature of the queue head
	if len(s.activeJobs) == 0 && s.activeSignature != oldest.Signature {
		s.switchSignatureLocked(ctx, oldest.Signature, oldest.TargetFlags)
	}

	var toDispatch []*QueueItem
	var remaining []*QueueItem

	for _, item := range s.queue {
		if len(toDispatch) < availableSlots && item.Signature == s.activeSignature {
			toDispatch = append(toDispatch, item)
		} else {
			remaining = append(remaining, item)
		}
	}

	s.queue = remaining

	// Launch dispatched jobs in isolated goroutines
	now := time.Now().UTC()
	for _, item := range toDispatch {
		item.StartedAt = &now
		s.activeJobs[item.ProfileID] = item
		if w, ok := s.workers[item.ProfileID]; ok {
			w.isSyncing.Store(true)
		}
		s.wg.Add(1)
		go s.runJob(ctx, item)
	}
}

// GetQueueSnapshot generates a point-in-time snapshot of the dispatcher and execution queue.
func (s *Scheduler) GetQueueSnapshot() QueueSnapshotDTO {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snap := QueueSnapshotDTO{
		Paused:          s.paused,
		ActiveSignature: s.activeSignature,
		MaxConcurrency:  s.maxConcurrency,
		RunningCount:    len(s.activeJobs),
		QueuedCount:     len(s.queue),
		ActiveJobs:      make([]QueueJobDTO, 0, len(s.activeJobs)),
		QueuedJobs:      make([]QueueJobDTO, 0, len(s.queue)),
	}

	for _, item := range s.activeJobs {
		snap.ActiveJobs = append(snap.ActiveJobs, s.toJobDTO(item))
	}
	for _, item := range s.queue {
		snap.QueuedJobs = append(snap.QueuedJobs, s.toJobDTO(item))
	}

	return snap
}

func (s *Scheduler) toJobDTO(item *QueueItem) QueueJobDTO {
	name := item.ProfileID
	if s.store != nil {
		if prof, err := s.store.Get(item.ProfileID); err == nil && prof.Name != "" {
			name = prof.Name
		}
	}
	return QueueJobDTO{
		ProfileID:   item.ProfileID,
		ProfileName: name,
		Origin:      item.Opts.Origin,
		DryRun:      item.Opts.DryRun,
		Resync:      item.Opts.Resync,
		Force:       item.Opts.Force,
		Signature:   item.Signature,
		EnqueuedAt:  item.EnqueuedAt,
		StartedAt:   item.StartedAt,
		WaitSeconds: time.Since(item.EnqueuedAt).Seconds(),
	}
}
