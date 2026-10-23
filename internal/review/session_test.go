package review

import (
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

func TestSessionFSM_Transitions(t *testing.T) {
	sess := NewSession("TOO_MANY_DELETES", "Review required", nil, nil)
	if sess.Status != models.ReviewStatusCreated {
		t.Fatalf("expected CREATED, got %s", sess.Status)
	}
	if sess.ID == "" {
		t.Fatal("expected non-empty session ID")
	}

	// Allowed: CREATED -> REVIEWING
	if err := Transition(sess, models.ReviewStatusReviewing); err != nil {
		t.Fatalf("expected transition to REVIEWING, got error: %v", err)
	}

	// Disallowed: REVIEWING -> RUNNING_BISYNC
	if err := Transition(sess, models.ReviewStatusRunningBisync); err == nil {
		t.Fatal("expected error transitioning from REVIEWING to RUNNING_BISYNC")
	}

	// Allowed: REVIEWING -> APPLYING
	if err := Transition(sess, models.ReviewStatusApplying); err != nil {
		t.Fatalf("expected transition to APPLYING, got error: %v", err)
	}

	// Allowed: APPLYING -> READY
	if err := Transition(sess, models.ReviewStatusReady); err != nil {
		t.Fatalf("expected transition to READY, got error: %v", err)
	}

	// Allowed: READY -> RUNNING_BISYNC
	if err := Transition(sess, models.ReviewStatusRunningBisync); err != nil {
		t.Fatalf("expected transition to RUNNING_BISYNC, got error: %v", err)
	}

	// Allowed: RUNNING_BISYNC -> COMPLETE
	if err := Transition(sess, models.ReviewStatusComplete); err != nil {
		t.Fatalf("expected transition to COMPLETE, got error: %v", err)
	}

	// Terminal: COMPLETE -> anything fails
	if err := Transition(sess, models.ReviewStatusReviewing); err == nil {
		t.Fatal("expected error transitioning out of terminal COMPLETE")
	}

	// Nil session check
	if err := Transition(nil, models.ReviewStatusReady); err == nil {
		t.Fatal("expected error with nil session")
	}
}

func TestSessionFSM_FailureAndRollback(t *testing.T) {
	sess := NewSession("ERROR", "Review", nil, nil)
	_ = Transition(sess, models.ReviewStatusReviewing)
	_ = Transition(sess, models.ReviewStatusApplying)

	// APPLYING -> FAILED
	if err := Transition(sess, models.ReviewStatusFailed); err != nil {
		t.Fatalf("APPLYING -> FAILED failed: %v", err)
	}

	// FAILED -> ROLLBACK
	if err := Transition(sess, models.ReviewStatusRollback); err != nil {
		t.Fatalf("FAILED -> ROLLBACK failed: %v", err)
	}

	// ROLLBACK -> REVIEWING
	if err := Transition(sess, models.ReviewStatusReviewing); err != nil {
		t.Fatalf("ROLLBACK -> REVIEWING failed: %v", err)
	}

	// REVIEWING -> DISMISSED
	if err := Transition(sess, models.ReviewStatusDismissed); err != nil {
		t.Fatalf("REVIEWING -> DISMISSED failed: %v", err)
	}

	// DISMISSED is terminal
	if CanTransition(models.ReviewStatusDismissed, models.ReviewStatusReviewing) {
		t.Fatal("DISMISSED should not transition to REVIEWING")
	}
}

func TestStageDecision_DeletionsAndConflicts(t *testing.T) {
	deletions := []models.DeletionCandidate{
		{
			ID:       "del-1",
			Path:     "docs/old.pdf",
			Resolved: false,
		},
	}
	conflicts := []models.ConflictCandidate{
		{
			ID:       "conf-1",
			Path:     "notes.txt",
			Resolved: false,
		},
	}

	sess := NewSession("ERROR", "msg", deletions, conflicts)

	// Stage decision for deletion
	if err := StageDecision(sess, "del-1", models.DecisionAcceptDelete); err != nil {
		t.Fatalf("failed to stage deletion decision: %v", err)
	}
	if !sess.Deletions[0].Resolved || sess.Deletions[0].Decision != models.DecisionAcceptDelete {
		t.Fatalf("deletion candidate not updated: %+v", sess.Deletions[0])
	}
	// Session should have transitioned to REVIEWING
	if sess.Status != models.ReviewStatusReviewing {
		t.Fatalf("expected status REVIEWING, got %s", sess.Status)
	}

	// Invalid decision for deletion
	if err := StageDecision(sess, "del-1", models.DecisionKeepLocal); err == nil {
		t.Fatal("expected error for keep-local on deletion candidate")
	}

	// Stage decision for conflict
	if err := StageDecision(sess, "conf-1", models.DecisionKeepLocal); err != nil {
		t.Fatalf("failed to stage conflict decision: %v", err)
	}
	if !sess.Conflicts[0].Resolved || sess.Conflicts[0].Decision != models.DecisionKeepLocal {
		t.Fatalf("conflict candidate not updated: %+v", sess.Conflicts[0])
	}

	// Invalid decision for conflict
	if err := StageDecision(sess, "conf-1", models.DecisionAcceptDelete); err == nil {
		t.Fatal("expected error for accept-delete on conflict candidate")
	}

	// Unknown candidate
	if err := StageDecision(sess, "unknown", models.DecisionKeepBoth); err == nil {
		t.Fatal("expected error for unknown candidate ID")
	}

	// AllResolved check
	if !AllResolved(sess) {
		t.Fatal("expected AllResolved true")
	}
	if count := PendingCount(sess); count != 0 {
		t.Fatalf("expected 0 pending, got %d", count)
	}
	if total := TotalCandidateCount(sess); total != 2 {
		t.Fatalf("expected 2 total candidates, got %d", total)
	}
	summary := ActionCandidateSummary(sess)
	if summary == "" {
		t.Fatal("expected summary")
	}

	// Test nil session safety
	if StageDecision(nil, "del-1", models.DecisionAcceptDelete) == nil {
		t.Fatal("expected error with nil session")
	}
	if AllResolved(nil) {
		t.Fatal("expected false for AllResolved(nil)")
	}
	if PendingCount(nil) != 0 {
		t.Fatal("expected 0 for PendingCount(nil)")
	}
	if TotalCandidateCount(nil) != 0 {
		t.Fatal("expected 0 for TotalCandidateCount(nil)")
	}
	if ActionCandidateSummary(nil) != "no session" {
		t.Fatal("expected 'no session' for ActionCandidateSummary(nil)")
	}
}

func TestSession_DecisionAndTransitionHelpers(t *testing.T) {
	// IsValidDecision
	validDecisions := []models.ActionDecision{
		models.DecisionKeepLocal,
		models.DecisionKeepRemote,
		models.DecisionKeepBoth,
		models.DecisionAcceptDelete,
		models.DecisionRestoreItem,
	}
	for _, d := range validDecisions {
		if !IsValidDecision(d) {
			t.Errorf("expected %s to be valid", d)
		}
	}
	if IsValidDecision("invalid-decision") {
		t.Fatal("expected invalid-decision to be false")
	}

	// Additional CanTransition cases
	if !CanTransition(models.ReviewStatusReady, models.ReviewStatusReviewing) {
		t.Fatal("expected READY -> REVIEWING to be true")
	}
	if !CanTransition(models.ReviewStatusFailed, models.ReviewStatusDismissed) {
		t.Fatal("expected FAILED -> DISMISSED to be true")
	}
	if !CanTransition(models.ReviewStatusRollback, models.ReviewStatusDismissed) {
		t.Fatal("expected ROLLBACK -> DISMISSED to be true")
	}
	if CanTransition(models.ReviewStatusComplete, models.ReviewStatusReviewing) {
		t.Fatal("expected COMPLETE -> REVIEWING to be false")
	}

	// Empty session AllResolved check
	emptySess := &models.ReviewSession{}
	if !AllResolved(emptySess) {
		t.Fatal("expected AllResolved true for empty candidates")
	}
}

