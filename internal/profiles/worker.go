package profiles

import (
	"context"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"
)

type profileWorker struct {
	profileID string
	cancel    context.CancelFunc
	isSyncing atomic.Bool
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

// StopProfile halts background execution and clears pending jobs for a profile.
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
	for i := len(s.queue) - 1; i >= 0; i-- {
		if s.queue[i].ProfileID == profileID {
			if s.queue[i].ErrCh != nil {
				s.queue[i].ErrCh <- ErrProfileNotActive
			}
			s.queue = append(s.queue[:i], s.queue[i+1:]...)
		}
	}
}

func (s *Scheduler) startWorkerLocked(p *Profile) {
	if s.ctx == nil {
		return
	}
	wCtx, wCancel := context.WithCancel(s.ctx)
	w := &profileWorker{
		profileID: p.ID,
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
			s.enqueueScheduledSync(profileID)
		}
	}
}

func (s *Scheduler) enqueueScheduledSync(profileID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return
	}
	if _, active := s.activeJobs[profileID]; active {
		return
	}
	for _, item := range s.queue {
		if item.ProfileID == profileID {
			return
		}
	}

	prof, err := s.store.Get(profileID)
	if err != nil || !prof.Enabled || prof.Archived {
		return
	}

	fleetFlags := s.getFleetFlagsLocked()
	targetFlags, sig, err := ResolveEngineSignature(prof.Flags.CustomFlags, fleetFlags)
	if err != nil {
		slog.Warn("failed to resolve engine signature for scheduled sync", "profile_id", profileID, "error", err)
		return
	}

	item := &QueueItem{
		ProfileID:   profileID,
		Opts:        SyncOptions{Origin: "scheduler"},
		Signature:   sig,
		TargetFlags: targetFlags,
		EnqueuedAt:  time.Now(),
		ErrCh:       nil,
	}

	s.queue = append(s.queue, item)
	s.signalDispatcherLocked()
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
