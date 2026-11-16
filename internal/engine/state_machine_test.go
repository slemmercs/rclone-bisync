package engine

import (
	"sync"
	"testing"
	"time"
)

func TestFSM_ValidTransitions(t *testing.T) {
	tests := []struct {
		name        string
		transitions []SyncFSMState
		expected    SyncFSMState
	}{
		{
			name:        "happy path sync run",
			transitions: []SyncFSMState{StateQueued, StateRunning, StateCompleted, StateIdle},
			expected:    StateIdle,
		},
		{
			name:        "sync with conflict review and resolution",
			transitions: []SyncFSMState{StateQueued, StateRunning, StateReviewRequired, StateResolving, StateCompleted, StateIdle},
			expected:    StateIdle,
		},
		{
			name:        "sync aborted from queue (lock contention)",
			transitions: []SyncFSMState{StateQueued, StateAborted, StateIdle},
			expected:    StateIdle,
		},
		{
			name:        "sync aborted from running (canary trip or crash)",
			transitions: []SyncFSMState{StateRunning, StateAborted, StateIdle},
			expected:    StateIdle,
		},
		{
			name:        "review rejected or aborted",
			transitions: []SyncFSMState{StateQueued, StateRunning, StateReviewRequired, StateAborted, StateIdle},
			expected:    StateIdle,
		},
		{
			name:        "resolution requiring resync run",
			transitions: []SyncFSMState{StateQueued, StateRunning, StateReviewRequired, StateResolving, StateRunning, StateCompleted, StateIdle},
			expected:    StateIdle,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fsm := NewSyncFSM("test-prof", StateIdle)
			for _, target := range tc.transitions {
				evt, err := fsm.Transition(target, "run-123", "step", "")
				if err != nil {
					t.Fatalf("unexpected transition error to %s: %v", target, err)
				}
				if evt.To != target {
					t.Fatalf("expected event target %s, got %s", target, evt.To)
				}
			}
			if fsm.Current() != tc.expected {
				t.Fatalf("expected final state %s, got %s", tc.expected, fsm.Current())
			}
		})
	}
}

func TestFSM_InvalidTransitions(t *testing.T) {
	invalidCases := []struct {
		from SyncFSMState
		to   SyncFSMState
	}{
		{from: StateIdle, to: StateCompleted},
		{from: StateIdle, to: StateReviewRequired},
		{from: StateIdle, to: StateResolving},
		{from: StateQueued, to: StateCompleted},
		{from: StateQueued, to: StateReviewRequired},
		{from: StateReviewRequired, to: StateCompleted},
		{from: StateCompleted, to: StateRunning},
		{from: StateAborted, to: StateRunning},
	}

	for _, c := range invalidCases {
		fsm := NewSyncFSM("invalid-test", c.from)
		_, err := fsm.Transition(c.to, "run-x", "testing invalid", "TEST_ERROR")
		if err == nil {
			t.Errorf("expected error transitioning from %s to %s, got nil", c.from, c.to)
		}
		if fsm.Current() != c.from {
			t.Errorf("state mutated after invalid transition: expected %s, got %s", c.from, fsm.Current())
		}
	}
}

func TestFSM_ListenersAndMetadata(t *testing.T) {
	fsm := NewSyncFSM("profile-abc", StateIdle)
	var captured []StateChangeEvent
	var mu sync.Mutex

	fsm.OnTransition(func(evt StateChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		captured = append(captured, evt)
	})

	_, err := fsm.Transition(StateQueued, "run-99", "queued for exec", "")
	if err != nil {
		t.Fatal(err)
	}

	_, err = fsm.Transition(StateRunning, "run-99", "lock acquired", "")
	if err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	count := len(captured)
	mu.Unlock()

	if count != 2 {
		t.Fatalf("expected 2 captured events, got %d", count)
	}

	last := fsm.LastEvent()
	if last.To != StateRunning || last.RunID != "run-99" || last.Reason != "lock acquired" {
		t.Fatalf("unexpected last event: %+v", last)
	}
}

func TestFSM_ActiveAndTriggerablePredicates(t *testing.T) {
	fsm := NewSyncFSM("profile-pred", StateIdle)

	if fsm.IsActive() {
		t.Error("expected idle state to be not active")
	}
	if !fsm.CanTrigger() {
		t.Error("expected idle state to be triggerable")
	}

	_, _ = fsm.Transition(StateQueued, "r1", "queue", "")
	if !fsm.IsActive() {
		t.Error("expected queued state to be active")
	}
	if fsm.CanTrigger() {
		t.Error("expected queued state to not be triggerable")
	}

	_, _ = fsm.Transition(StateRunning, "r1", "run", "")
	if !fsm.IsActive() {
		t.Error("expected running state to be active")
	}

	_, _ = fsm.Transition(StateCompleted, "r1", "done", "")
	if fsm.IsActive() {
		t.Error("expected completed state to be not active")
	}
	if !fsm.CanTrigger() {
		t.Error("expected completed state to be triggerable")
	}
}

func TestFSM_ConcurrentSafety(t *testing.T) {
	fsm := NewSyncFSM("concurrent-prof", StateIdle)
	var wg sync.WaitGroup

	// Run concurrent readers and transitions
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = fsm.Current()
				_ = fsm.IsActive()
				_ = fsm.CanTrigger()
				_ = fsm.LastEvent()
			}
		}()
	}

	// Controlled sequential transitions with concurrent listeners
	fsm.OnTransition(func(_ StateChangeEvent) {
		time.Sleep(10 * time.Microsecond)
	})

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(iter int) {
			defer wg.Done()
			fsmLocal := NewSyncFSM("thread-prof", StateIdle)
			_, _ = fsmLocal.Transition(StateQueued, "run", "q", "")
			_, _ = fsmLocal.Transition(StateRunning, "run", "r", "")
			_, _ = fsmLocal.Transition(StateCompleted, "run", "c", "")
			_, _ = fsmLocal.Transition(StateIdle, "run", "i", "")
		}(i)
	}

	wg.Wait()
}
