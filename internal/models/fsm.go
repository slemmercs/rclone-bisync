package models

import (
	"time"
)

// SyncFSMState defines the explicit lifecycle states of the storage engine for serialization.
type SyncFSMState string

// FSMStatusPayload surfaces current state and recent transitions to the HTTP API.
type FSMStatusPayload struct {
	ProfileID      string            `json:"profile_id"`
	CurrentState   string            `json:"current_state"`
	LastTransition *StateChangeEvent `json:"last_transition,omitempty"`
	IsActive       bool              `json:"is_active"`
	CanTriggerSync bool              `json:"can_trigger_sync"`
	PendingReview  bool              `json:"pending_review"`
}

// StateChangeEvent mirrors engine.StateChangeEvent for API serialization.
type StateChangeEvent struct {
	ProfileID string    `json:"profile_id"`
	RunID     string    `json:"run_id,omitempty"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Reason    string    `json:"reason,omitempty"`
	ErrorType string    `json:"error_type,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
