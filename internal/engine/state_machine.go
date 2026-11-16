package engine

import (
	"fmt"
	"sync"
	"time"
)

// SyncFSMState defines the explicit lifecycle states of the storage engine.
type SyncFSMState string

const (
	StateIdle           SyncFSMState = "IDLE"
	StateQueued         SyncFSMState = "QUEUED"
	StateRunning        SyncFSMState = "RUNNING"
	StateReviewRequired SyncFSMState = "REVIEW_REQUIRED"
	StateResolving      SyncFSMState = "RESOLVING"
	StateCompleted      SyncFSMState = "COMPLETED"
	StateAborted        SyncFSMState = "ABORTED"
)

// StateChangeEvent represents an immutable record of an FSM transition.
type StateChangeEvent struct {
	ProfileID string       `json:"profile_id"`
	RunID     string       `json:"run_id,omitempty"`
	From      SyncFSMState `json:"from"`
	To        SyncFSMState `json:"to"`
	Reason    string       `json:"reason,omitempty"`
	ErrorType string       `json:"error_type,omitempty"`
	Timestamp time.Time    `json:"timestamp"`
}

// TransitionHook is invoked whenever a state transition successfully occurs.
type TransitionHook func(event StateChangeEvent)

// CanTransition validates whether transitioning from state 'from' to state 'to' is allowed.
func CanTransition(from, to SyncFSMState) bool {
	if from == to {
		return true
	}
	switch from {
	case StateIdle:
		return to == StateQueued || to == StateRunning
	case StateQueued:
		return to == StateRunning || to == StateAborted
	case StateRunning:
		return to == StateCompleted || to == StateReviewRequired || to == StateAborted
	case StateReviewRequired:
		return to == StateResolving || to == StateAborted
	case StateResolving:
		return to == StateRunning || to == StateCompleted || to == StateAborted
	case StateCompleted, StateAborted:
		return to == StateIdle
	default:
		return false
	}
}

// SyncFSM provides a thread-safe finite state machine for workload lifecycle coordination.
type SyncFSM struct {
	mu        sync.RWMutex
	profileID string
	current   SyncFSMState
	lastEvent StateChangeEvent
	listeners []TransitionHook
}

// NewSyncFSM initializes a new thread-safe state machine for the specified profile.
func NewSyncFSM(profileID string, initial SyncFSMState) *SyncFSM {
	if initial == "" {
		initial = StateIdle
	}
	now := time.Now().UTC()
	return &SyncFSM{
		profileID: profileID,
		current:   initial,
		lastEvent: StateChangeEvent{
			ProfileID: profileID,
			From:      initial,
			To:        initial,
			Reason:    "initialization",
			Timestamp: now,
		},
		listeners: make([]TransitionHook, 0),
	}
}

// Current returns the current state under a read lock.
func (f *SyncFSM) Current() SyncFSMState {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.current
}

// LastEvent returns the most recent state change event under a read lock.
func (f *SyncFSM) LastEvent() StateChangeEvent {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.lastEvent
}

// IsActive returns true if the engine is actively executing or resolving.
func (f *SyncFSM) IsActive() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.current == StateQueued || f.current == StateRunning || f.current == StateResolving
}

// CanTrigger returns true if a new synchronization run can be dispatched.
func (f *SyncFSM) CanTrigger() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.current == StateIdle || f.current == StateCompleted || f.current == StateAborted
}

// Reset transitions the FSM back to StateIdle if it is in StateCompleted or StateAborted.
func (f *SyncFSM) Reset() error {
	f.mu.Lock()
	cur := f.current
	if cur == StateIdle {
		f.mu.Unlock()
		return nil
	}
	f.mu.Unlock()
	_, err := f.Transition(StateIdle, "", "reset to idle", "")
	return err
}


// OnTransition registers a callback hook to be triggered on successful state transitions.
func (f *SyncFSM) OnTransition(hook TransitionHook) {
	if hook == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listeners = append(f.listeners, hook)
}

// Transition attempts to transition the FSM to target state 'to'.
func (f *SyncFSM) Transition(to SyncFSMState, runID string, reason string, errType string) (StateChangeEvent, error) {
	f.mu.Lock()
	from := f.current
	if !CanTransition(from, to) {
		f.mu.Unlock()
		return StateChangeEvent{}, fmt.Errorf("illegal transition from %s to %s for profile %s: %s", from, to, f.profileID, reason)
	}

	event := StateChangeEvent{
		ProfileID: f.profileID,
		RunID:     runID,
		From:      from,
		To:        to,
		Reason:    reason,
		ErrorType: errType,
		Timestamp: time.Now().UTC(),
	}

	f.current = to
	f.lastEvent = event

	// Copy listener slice to execute outside lock
	listenersCopy := make([]TransitionHook, len(f.listeners))
	copy(listenersCopy, f.listeners)
	f.mu.Unlock()

	for _, hook := range listenersCopy {
		hook(event)
	}

	return event, nil
}
