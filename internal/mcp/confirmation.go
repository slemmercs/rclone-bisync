package mcp

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	// ErrTicketNotFound is returned when the ticket ID does not exist in the store.
	ErrTicketNotFound = errors.New("confirmation ticket not found")
	// ErrTicketExpired is returned when the ticket's TTL has elapsed.
	ErrTicketExpired = errors.New("confirmation ticket has expired")
	// ErrActionMismatch is returned when the ticket was generated for a different action.
	ErrActionMismatch = errors.New("confirmation ticket action mismatch")
	// ErrTargetMismatch is returned when the ticket was generated for a different target.
	ErrTargetMismatch = errors.New("confirmation ticket target mismatch")
)

// Ticket represents a cryptographically verified confirmation token for destructive operations.
type Ticket struct {
	ID        string            `json:"id"`
	Action    string            `json:"action"`
	Target    string            `json:"target"`
	Params    map[string]string `json:"params,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	ExpiresAt time.Time         `json:"expires_at"`
}

// TicketManager manages confirmation tickets with automatic expiration and concurrency safety.
type TicketManager struct {
	mu      sync.RWMutex
	ttl     time.Duration
	tickets map[string]Ticket
}

// NewTicketManager initializes a TicketManager with the specified ticket time-to-live.
func NewTicketManager(ttl time.Duration) *TicketManager {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &TicketManager{
		ttl:     ttl,
		tickets: make(map[string]Ticket),
	}
}

// CreateTicket issues a new confirmation ticket for an action and target.
func (tm *TicketManager) CreateTicket(action, target string, params map[string]string) Ticket {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.pruneExpiredLocked(time.Now())

	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		// Fallback to timestamp-based unique seed if crypto/rand fails
		copy(tokenBytes, fmt.Sprintf("%016x", time.Now().UnixNano()))
	}
	id := hex.EncodeToString(tokenBytes)

	now := time.Now()
	ticket := Ticket{
		ID:        id,
		Action:    action,
		Target:    target,
		Params:    params,
		CreatedAt: now,
		ExpiresAt: now.Add(tm.ttl),
	}

	tm.tickets[id] = ticket
	return ticket
}

// ValidateTicket checks whether a ticket is valid without consuming it.
func (tm *TicketManager) ValidateTicket(ticketID, action, target string) (*Ticket, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	ticket, exists := tm.tickets[ticketID]
	if !exists {
		return nil, ErrTicketNotFound
	}

	if time.Now().After(ticket.ExpiresAt) {
		return nil, ErrTicketExpired
	}

	if ticket.Action != action {
		return nil, ErrActionMismatch
	}

	if ticket.Target != target {
		return nil, ErrTargetMismatch
	}

	res := ticket
	return &res, nil
}

// ConsumeTicket validates and atomically removes a ticket upon authorized execution.
func (tm *TicketManager) ConsumeTicket(ticketID, action, target string) (*Ticket, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	ticket, exists := tm.tickets[ticketID]
	if !exists {
		return nil, ErrTicketNotFound
	}

	if time.Now().After(ticket.ExpiresAt) {
		delete(tm.tickets, ticketID)
		return nil, ErrTicketExpired
	}

	if ticket.Action != action {
		return nil, ErrActionMismatch
	}

	if ticket.Target != target {
		return nil, ErrTargetMismatch
	}

	delete(tm.tickets, ticketID)
	res := ticket
	return &res, nil
}

// Prune removes expired tickets from memory.
func (tm *TicketManager) Prune() int {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.pruneExpiredLocked(time.Now())
}

func (tm *TicketManager) pruneExpiredLocked(now time.Time) int {
	pruned := 0
	for id, t := range tm.tickets {
		if now.After(t.ExpiresAt) {
			delete(tm.tickets, id)
			pruned++
		}
	}
	return pruned
}
