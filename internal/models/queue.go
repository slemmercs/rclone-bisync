package models

import "time"

// QueueJobDTO represents a running or queued profile execution in API responses.
type QueueJobDTO struct {
	ProfileID   string     `json:"profile_id"`
	ProfileName string     `json:"profile_name"`
	Origin      string     `json:"origin"`
	DryRun      bool       `json:"dry_run"`
	Resync      bool       `json:"resync"`
	Force       bool       `json:"force"`
	Signature   string     `json:"signature"`
	EnqueuedAt  time.Time  `json:"enqueued_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	WaitSeconds float64    `json:"wait_seconds"`
}

// QueueSnapshotDTO represents the state of the execution queue and running cohorts.
type QueueSnapshotDTO struct {
	Paused          bool          `json:"paused"`
	ActiveSignature string        `json:"active_signature"`
	MaxConcurrency  int           `json:"max_concurrency"`
	RunningCount    int           `json:"running_count"`
	QueuedCount     int           `json:"queued_count"`
	ActiveJobs      []QueueJobDTO `json:"active_jobs"`
	QueuedJobs      []QueueJobDTO `json:"queued_jobs"`
}

// QueueSnapshotResponse models the response for GET /api/queue.
type QueueSnapshotResponse struct {
	Status   string           `json:"status"`
	Snapshot QueueSnapshotDTO `json:"snapshot"`
	Error    string           `json:"error,omitempty"`
}

// QueueActionResponse models responses for queue pause, resume, cancel, and clear actions.
type QueueActionResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Count   int    `json:"count,omitempty"`
	Error   string `json:"error,omitempty"`
}
