package models

// DedupeStageAction represents an individual staged resolution action for a duplicate item.
type DedupeStageAction struct {
	ID        string `json:"id,omitempty"`        // Remote item ID (especially for Google Drive)
	Path      string `json:"path"`                // Current relative file path
	Action    string `json:"action"`              // "delete", "rename", "keep"
	NewName   string `json:"new_name,omitempty"`  // Target name when action is "rename"
	Size      int64  `json:"size,omitempty"`
	ModTime   string `json:"mod_time,omitempty"`
	Candidate int    `json:"candidate,omitempty"` // Candidate index
}

// DedupeBatchApplyRequest represents the batch payload to execute staged duplicate resolutions.
type DedupeBatchApplyRequest struct {
	Remote            string              `json:"remote"`
	Path              string              `json:"path"`
	ProfileID         string              `json:"profile_id,omitempty"`
	Actions           []DedupeStageAction `json:"actions"`
	TriggerResync     bool                `json:"trigger_resync,omitempty"`
	ConfirmationToken string              `json:"confirmation_token,omitempty"`
}

// DedupeBatchApplyResponse represents the outcome of executing staged duplicate resolutions.
type DedupeBatchApplyResponse struct {
	Status        string   `json:"status"`
	RunID         string   `json:"run_id,omitempty"`
	SuccessCount  int      `json:"success_count"`
	FailureCount  int      `json:"failure_count"`
	Errors        []string `json:"errors,omitempty"`
	ResyncQueued  bool     `json:"resync_queued,omitempty"`
	ResyncMessage string   `json:"resync_message,omitempty"`
	Error         string   `json:"error,omitempty"`
}
