package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestReviewSession_JSON(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	sess := ReviewSession{
		ID:        "session-123",
		CreatedAt: now,
		UpdatedAt: now,
		Status:    ReviewStatusReviewing,
		Deletions: []DeletionCandidate{
			{
				ID:           "del-1",
				Path:         "docs/old.txt",
				Name:         "old.txt",
				IsDir:        false,
				Source:       SideLocal,
				TargetAction: "Would delete 1 file on remote",
				Size:         1024,
				ModTime:      now.Format(time.RFC3339),
				Condition:    ConditionTooManyDeletes,
				Decision:     DecisionAcceptDelete,
				Resolved:     true,
			},
		},
		Conflicts: []ConflictCandidate{
			{
				ID:            "conf-1",
				Path:          "notes.md",
				Name:          "notes.md",
				LocalSize:     2048,
				LocalModTime:  now.Format(time.RFC3339),
				RemoteSize:    2048,
				RemoteModTime: now.Add(-time.Hour).Format(time.RFC3339),
				Condition:     ConditionConflict,
				Decision:      DecisionKeepLocal,
				Resolved:      true,
			},
		},
		Actions: []ResolutionAction{
			{
				ID:         "act-1",
				SessionID:  "session-123",
				Type:       DecisionKeepLocal,
				ItemPath:   "notes.md",
				IsDir:      false,
				Status:     "completed",
				BackupPath: "/tmp/backup/notes.md",
				BackupSide: SideRemote,
				ExecutedAt: &now,
			},
		},
		OriginError: "Fatal bisync error: 50% deletions exceed threshold",
		Message:     "Review session required",
	}

	data, err := json.Marshal(sess)
	if err != nil {
		t.Fatalf("failed to marshal ReviewSession: %v", err)
	}

	var decoded ReviewSession
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal ReviewSession: %v", err)
	}

	if decoded.ID != sess.ID {
		t.Errorf("expected ID %s, got %s", sess.ID, decoded.ID)
	}
	if decoded.Status != ReviewStatusReviewing {
		t.Errorf("expected status %s, got %s", ReviewStatusReviewing, decoded.Status)
	}
	if len(decoded.Deletions) != 1 || decoded.Deletions[0].Decision != DecisionAcceptDelete {
		t.Errorf("unexpected deletions payload: %+v", decoded.Deletions)
	}
	if len(decoded.Conflicts) != 1 || decoded.Conflicts[0].Decision != DecisionKeepLocal {
		t.Errorf("unexpected conflicts payload: %+v", decoded.Conflicts)
	}
	if len(decoded.Actions) != 1 || decoded.Actions[0].Status != "completed" {
		t.Errorf("unexpected actions payload: %+v", decoded.Actions)
	}
}

func TestWALEntry_JSON(t *testing.T) {
	entry := WALEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		SessionID: "session-123",
		ActionID:  "act-1",
		Step:      "MUTATION_COMPLETED",
		ItemPath:  "notes.md",
		Type:      DecisionKeepLocal,
		BackupRef: "/tmp/backup/notes.md",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal WALEntry: %v", err)
	}

	var decoded WALEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal WALEntry: %v", err)
	}

	if decoded.ActionID != entry.ActionID || decoded.Step != "MUTATION_COMPLETED" {
		t.Errorf("unexpected decoded WALEntry: %+v", decoded)
	}
}

func TestDTOs_JSON(t *testing.T) {
	req := StageDecisionRequest{
		CandidateID: "cand-1",
		Decision:    DecisionKeepBoth,
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal StageDecisionRequest: %v", err)
	}
	var decodedReq StageDecisionRequest
	if err := json.Unmarshal(data, &decodedReq); err != nil || decodedReq.Decision != DecisionKeepBoth {
		t.Errorf("unexpected decoded StageDecisionRequest: %+v", decodedReq)
	}

	execResp := ExecuteReviewResponse{
		Status:    "success",
		SessionID: "sess-1",
		Applied:   2,
		Failed:    0,
	}
	data, err = json.Marshal(execResp)
	if err != nil {
		t.Fatalf("failed to marshal ExecuteReviewResponse: %v", err)
	}
	var decodedExec ExecuteReviewResponse
	if err := json.Unmarshal(data, &decodedExec); err != nil || decodedExec.Applied != 2 {
		t.Errorf("unexpected decoded ExecuteReviewResponse: %+v", decodedExec)
	}

	rollbackResp := RollbackReviewResponse{
		Status:    "success",
		SessionID: "sess-1",
		Restored:  2,
	}
	data, err = json.Marshal(rollbackResp)
	if err != nil {
		t.Fatalf("failed to marshal RollbackReviewResponse: %v", err)
	}
	var decodedRollback RollbackReviewResponse
	if err := json.Unmarshal(data, &decodedRollback); err != nil || decodedRollback.Restored != 2 {
		t.Errorf("unexpected decoded RollbackReviewResponse: %+v", decodedRollback)
	}
}
