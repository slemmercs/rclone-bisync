package audit

// Action represents a standardized audit event code.
type Action string

const (
	// Conflict & Deletion Decisions
	ActionConflictResolved Action = "CONFLICT_RESOLVED"
	ActionDeletionAccepted Action = "DELETION_ACCEPTED"
	ActionDeletionRestored Action = "DELETION_RESTORED"
	ActionReviewDismissed  Action = "REVIEW_SESSION_DISMISSED"

	// Sync Operations
	ActionSyncExecuted     Action = "SYNC_EXECUTED"
	ActionSyncAborted      Action = "SYNC_ABORTED"
	ActionRollbackExecuted Action = "ROLLBACK_EXECUTED"

	// Configuration & Remote Auth Lifecycle
	ActionConfigUpdated     Action = "CONFIG_UPDATED"
	ActionOAuthReconnected  Action = "OAUTH_RECONNECTED"
	ActionCanaryProvisioned Action = "CANARY_PROVISIONED"
	ActionPolicyViolated    Action = "POLICY_VIOLATED"
	ActionGenesisCheckpoint Action = "GENESIS_CHECKPOINT"

	// Profile Lifecycle
	ActionProfileCreated    Action = "PROFILE_CREATED"
	ActionProfileUpdated    Action = "PROFILE_UPDATED"
	ActionProfileDeleted    Action = "PROFILE_DELETED"
	ActionProfileCloned     Action = "PROFILE_CLONED"
	ActionProfileRolledBack Action = "PROFILE_ROLLED_BACK"
)
