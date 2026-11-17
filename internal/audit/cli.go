package audit

import (
	"fmt"
	"strings"
)

// RunCLI executes an audit CLI action ("seed" or "verify") against the specified database.
func RunCLI(action string, dbPath string, localPath string) error {
	db, err := OpenDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open audit ledger (%s): %w", dbPath, err)
	}
	defer func() {
		_ = db.Close()
	}()

	switch strings.ToLower(strings.TrimSpace(action)) {
	case "seed":
		seedEntries := []struct {
			profileID string
			action    Action
			actor     string
			details   map[string]any
		}{
			{
				profileID: "prof_sandbox",
				action:    ActionProfileCreated,
				actor:     "admin",
				details: map[string]any{
					"name":        "Sandbox Test Remote",
					"remote":      "testremote:",
					"local_path":  localPath,
					"compare":     "size,modtime",
					"review_mode": "all",
				},
			},
			{
				profileID: "prof_sandbox",
				action:    ActionSyncExecuted,
				actor:     "timer",
				details: map[string]any{
					"dry_run":           true,
					"origin":            "scheduler",
					"status":            "completed",
					"transferred_files": 2,
				},
			},
			{
				profileID: "prof_backup",
				action:    ActionConfigUpdated,
				actor:     "admin",
				details: map[string]any{
					"remote": "backup_remote",
					"type":   "alias",
				},
			},
			{
				profileID: "prof_sandbox",
				action:    ActionProfileUpdated,
				actor:     "admin",
				details: map[string]any{
					"version_before": 1,
					"version_after":  2,
					"max_delete":     15,
				},
			},
			{
				profileID: "prof_sandbox",
				action:    ActionDedupeExecuted,
				actor:     "mcp_agent",
				details: map[string]any{
					"mode":          "oldest",
					"by_hash":       true,
					"removed_count": 1,
				},
			},
		}

		for _, e := range seedEntries {
			rec, err := Append(db, e.profileID, e.action, e.actor, e.details)
			if err != nil {
				return fmt.Errorf("failed to seed audit entry: %w", err)
			}
			fmt.Printf("Seeded audit block #%d [%s] %s (%s...)\n", rec.SeqID, rec.Action, rec.Actor, rec.Hash[:10])
		}

		valid, count, err := VerifyLedger(db)
		if err != nil || !valid {
			return fmt.Errorf("cryptographic verification failed after seeding: %w", err)
		}
		fmt.Printf("✔ Successfully seeded and verified %d audit ledger blocks in %s\n", count, dbPath)
		return nil

	case "verify":
		valid, count, err := VerifyLedger(db)
		if err != nil {
			return fmt.Errorf("audit ledger verification error: %w", err)
		}
		if !valid {
			return fmt.Errorf("audit ledger verification FAILED! Integrity compromised at block %d", count)
		}
		fmt.Printf("✔ Audit ledger integrity verified: %d records intact.\n", count)
		return nil

	default:
		return fmt.Errorf("unknown audit action. Valid actions: 'verify', 'seed'")
	}
}
