package mcp

import (
	"errors"
	"testing"
	"time"
)

func TestTicketManager_CreateAndValidate(t *testing.T) {
	tm := NewTicketManager(1 * time.Minute)

	params := map[string]string{"resync": "true", "force": "false"}
	ticket := tm.CreateTicket("trigger_sync", "profile-photos", params)

	if ticket.ID == "" {
		t.Fatal("expected non-empty ticket ID")
	}
	if ticket.Action != "trigger_sync" {
		t.Errorf("expected action 'trigger_sync', got %s", ticket.Action)
	}
	if ticket.Target != "profile-photos" {
		t.Errorf("expected target 'profile-photos', got %s", ticket.Target)
	}
	if ticket.Params["resync"] != "true" {
		t.Errorf("expected param resync=true, got %s", ticket.Params["resync"])
	}

	// Validate ticket without consuming
	valid, err := tm.ValidateTicket(ticket.ID, "trigger_sync", "profile-photos")
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if valid.ID != ticket.ID {
		t.Errorf("expected ticket ID %s, got %s", ticket.ID, valid.ID)
	}

	// Action mismatch
	_, err = tm.ValidateTicket(ticket.ID, "resolve_review", "profile-photos")
	if !errors.Is(err, ErrActionMismatch) {
		t.Errorf("expected ErrActionMismatch, got %v", err)
	}

	// Target mismatch
	_, err = tm.ValidateTicket(ticket.ID, "trigger_sync", "profile-documents")
	if !errors.Is(err, ErrTargetMismatch) {
		t.Errorf("expected ErrTargetMismatch, got %v", err)
	}

	// Parameter fingerprint match
	_, err = tm.ValidateTicket(ticket.ID, "trigger_sync", "profile-photos", map[string]string{"resync": "true"})
	if err != nil {
		t.Errorf("expected parameter validation match, got: %v", err)
	}

	// Parameter fingerprint mismatch
	_, err = tm.ValidateTicket(ticket.ID, "trigger_sync", "profile-photos", map[string]string{"resync": "false"})
	if !errors.Is(err, ErrParamsMismatch) {
		t.Errorf("expected ErrParamsMismatch, got %v", err)
	}

	// Non-existent ticket
	_, err = tm.ValidateTicket("non-existent-id", "trigger_sync", "profile-photos")
	if !errors.Is(err, ErrTicketNotFound) {
		t.Errorf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestTicketManager_Consume(t *testing.T) {
	tm := NewTicketManager(1 * time.Minute)

	ticket := tm.CreateTicket("resolve_review", "sess-123", nil)

	// Parameter mismatch on consume
	tParams := tm.CreateTicket("trigger_sync", "prof-1", map[string]string{"mode": "oldest"})
	_, err := tm.ConsumeTicket(tParams.ID, "trigger_sync", "prof-1", map[string]string{"mode": "newest"})
	if !errors.Is(err, ErrParamsMismatch) {
		t.Errorf("expected ErrParamsMismatch on consume, got %v", err)
	}

	// Consume ticket
	consumed, err := tm.ConsumeTicket(ticket.ID, "resolve_review", "sess-123")
	if err != nil {
		t.Fatalf("failed to consume ticket: %v", err)
	}
	if consumed.ID != ticket.ID {
		t.Errorf("consumed ticket ID mismatch")
	}

	// Attempting to consume again should fail with ErrTicketNotFound
	_, err = tm.ConsumeTicket(ticket.ID, "resolve_review", "sess-123")
	if !errors.Is(err, ErrTicketNotFound) {
		t.Errorf("expected ErrTicketNotFound on second consume, got %v", err)
	}
}

func TestTicketManager_ExpirationAndPrune(t *testing.T) {
	// 5ms TTL
	tm := NewTicketManager(5 * time.Millisecond)

	ticket := tm.CreateTicket("dedupe_by_hash", "profile-1", nil)

	time.Sleep(10 * time.Millisecond)

	// Validate should return ErrTicketExpired
	_, err := tm.ValidateTicket(ticket.ID, "dedupe_by_hash", "profile-1")
	if !errors.Is(err, ErrTicketExpired) {
		t.Errorf("expected ErrTicketExpired, got %v", err)
	}

	// Consume should return ErrTicketExpired and delete
	_, err = tm.ConsumeTicket(ticket.ID, "dedupe_by_hash", "profile-1")
	if !errors.Is(err, ErrTicketExpired) {
		t.Errorf("expected ErrTicketExpired on consume, got %v", err)
	}

	// Prune test
	t2 := tm.CreateTicket("action", "target", nil)
	_ = t2
	time.Sleep(10 * time.Millisecond)
	pruned := tm.Prune()
	if pruned != 1 {
		t.Errorf("expected 1 pruned ticket, got %d", pruned)
	}
}

func TestTicketManager_DefaultTTL(t *testing.T) {
	tm := NewTicketManager(0)
	if tm.ttl != 5*time.Minute {
		t.Errorf("expected default TTL of 5m, got %v", tm.ttl)
	}
}
