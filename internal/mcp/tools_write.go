package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
)

func (s *Server) registerMutatingTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("bisync_trigger_sync",
			mcp.WithDescription("Trigger synchronization on a profile with blast-radius protection and dry-run safety"),
			mcp.WithString("profile_id", mcp.Required(), mcp.Description("Target profile ID")),
			mcp.WithBoolean("dry_run", mcp.DefaultBool(true), mcp.Description("Perform a dry-run without writing changes")),
			mcp.WithBoolean("resync", mcp.DefaultBool(false), mcp.Description("Force full resync (rebuilds bisync cache; requires confirmation)")),
			mcp.WithBoolean("force", mcp.DefaultBool(false), mcp.Description("Bypass lock checks and override conflict safety (requires confirmation)")),
			mcp.WithString("confirmation_token", mcp.Description("Confirmation token returned by previous invocation for high blast-radius operations")),
		),
		s.handleTriggerSync,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_resolve_review",
			mcp.WithDescription("Stage or apply conflict resolution decisions on a review session"),
			mcp.WithString("session_id", mcp.Description("Conflict review session ID (defaults to active session)")),
			mcp.WithString("candidate_id", mcp.Description("Optional specific candidate file ID to stage decision for")),
			mcp.WithString("decision", mcp.Required(), mcp.Description("Resolution decision: KEEP_LOCAL, KEEP_REMOTE, KEEP_BOTH, MANUAL, DISMISS")),
			mcp.WithBoolean("execute", mcp.DefaultBool(false), mcp.Description("Apply and execute staged decisions to disk (requires confirmation)")),
			mcp.WithString("confirmation_token", mcp.Description("Confirmation token required when execute=true")),
		),
		s.handleResolveReview,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_dedupe_by_hash",
			mcp.WithDescription("Execute Rclone deduplication on a remote or profile path"),
			mcp.WithString("remote", mcp.Required(), mcp.Description("Target remote or local path")),
			mcp.WithString("path", mcp.Description("Subdirectory path within the remote")),
			mcp.WithString("mode", mcp.DefaultString("oldest"), mcp.Description("Deduplication resolution strategy: 'list', 'oldest', or 'newest'")),
			mcp.WithBoolean("by_hash", mcp.DefaultBool(true), mcp.Description("Find duplicates by cryptographic hash instead of name/size only")),
			mcp.WithString("confirmation_token", mcp.Description("Confirmation token required when mode != 'list'")),
		),
		s.handleDedupeByHash,
	)
}

func (s *Server) handleTriggerSync(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.cfg.ReadOnly {
		return mcp.NewToolResultError("operation rejected: MCP server is running in read-only mode"), nil
	}

	profileID, err := req.RequireString("profile_id")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'profile_id'"), nil
	}

	dryRun := req.GetBool("dry_run", true)
	resync := req.GetBool("resync", false)
	force := req.GetBool("force", false)
	token := req.GetString("confirmation_token", "")

	// High blast-radius operations require confirmation
	if (resync || force) && !dryRun {
		if token == "" {
			desc := fmt.Sprintf("Trigger full resync on profile %s (blast radius: cache rebuilt, files re-compared)", profileID)
			if force {
				desc = fmt.Sprintf("Trigger forced sync on profile %s (blast radius: lock/safety overrides active)", profileID)
			}
			ticket := s.tickets.CreateTicket("trigger_sync", profileID, map[string]string{
				"description": desc,
				"profile_id":  profileID,
				"resync":      fmt.Sprintf("%t", resync),
				"force":       fmt.Sprintf("%t", force),
			})
			return jsonResult(map[string]any{
				"status":       "confirmation_required",
				"message":      "High blast-radius operation requires explicit confirmation.",
				"ticket_id":    ticket.ID,
				"token":        ticket.ID,
				"expires_at":   ticket.ExpiresAt,
				"ttl_seconds":  int(time.Until(ticket.ExpiresAt).Seconds()),
				"instructions": "Call bisync_trigger_sync with confirmation_token set to the provided token to proceed.",
			})
		}

		if _, err := s.tickets.ConsumeTicket(token, "trigger_sync", profileID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid or expired confirmation token: %v", err)), nil
		}
	}

	if s.container == nil || s.container.ProfileService == nil {
		return mcp.NewToolResultError("ProfileService is not initialized"), nil
	}

	opts := profiles.SyncOptions{
		Force:  force,
		Resync: resync,
		DryRun: dryRun,
		Origin: "mcp",
	}

	if err := s.container.ProfileService.TriggerSyncWithOptions(ctx, profileID, opts); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("sync execution failed: %v", err)), nil
	}

	return jsonResult(map[string]any{
		"status":     "success",
		"profile_id": profileID,
		"dry_run":    dryRun,
		"resync":     resync,
		"force":      force,
		"message":    fmt.Sprintf("Sync triggered successfully for profile %s", profileID),
	})
}

func (s *Server) handleResolveReview(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.cfg.ReadOnly {
		return mcp.NewToolResultError("operation rejected: MCP server is running in read-only mode"), nil
	}
	if s.container == nil || s.container.ReviewService == nil {
		return mcp.NewToolResultError("ReviewService is not initialized"), nil
	}

	decision, err := req.RequireString("decision")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'decision'"), nil
	}

	sessionID := req.GetString("session_id", "")
	execute := req.GetBool("execute", false)
	token := req.GetString("confirmation_token", "")

	if sessionID == "" {
		active, err := s.container.ReviewService.GetActiveSession(ctx)
		if err != nil || active == nil {
			return mcp.NewToolResultError("no active conflict review session found"), nil
		}
		sessionID = active.ID
	}

	if execute {
		if token == "" {
			ticket := s.tickets.CreateTicket("execute_review", sessionID, map[string]string{
				"session_id": sessionID,
				"decision":   decision,
			})
			return jsonResult(map[string]any{
				"status":       "confirmation_required",
				"message":      "Applying review decisions executes disk mutations and requires confirmation.",
				"ticket_id":    ticket.ID,
				"token":        ticket.ID,
				"expires_at":   ticket.ExpiresAt,
				"ttl_seconds":  int(time.Until(ticket.ExpiresAt).Seconds()),
				"instructions": "Call bisync_resolve_review with confirmation_token set to the provided token to execute.",
			})
		}

		if _, err := s.tickets.ConsumeTicket(token, "execute_review", sessionID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid or expired confirmation token: %v", err)), nil
		}

		sess, err := s.container.ReviewService.ApplyReviewSession(ctx, sessionID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to apply review session: %v", err)), nil
		}
		return jsonResult(map[string]any{
			"status":     "success",
			"session_id": sessionID,
			"applied":    len(sess.Actions),
			"actions":    sess.Actions,
		})
	}

	candidateID := req.GetString("candidate_id", "")
	var candidateIDs []string
	if candidateID != "" {
		candidateIDs = []string{candidateID}
	} else {
		active, err := s.container.ReviewService.GetActiveSession(ctx)
		if err == nil && active != nil {
			for _, d := range active.Deletions {
				candidateIDs = append(candidateIDs, d.ID)
			}
			for _, c := range active.Conflicts {
				candidateIDs = append(candidateIDs, c.ID)
			}
		}
	}

	sess, err := s.container.ReviewService.StageCandidateDecisions(sessionID, candidateIDs, models.ActionDecision(decision))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to stage decision: %v", err)), nil
	}

	return jsonResult(map[string]any{
		"status":     "success",
		"session_id": sess.ID,
		"decision":   decision,
		"staged":     len(candidateIDs),
	})
}

func (s *Server) handleDedupeByHash(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.cfg.ReadOnly {
		return mcp.NewToolResultError("operation rejected: MCP server is running in read-only mode"), nil
	}

	remote, err := req.RequireString("remote")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'remote'"), nil
	}

	path := req.GetString("path", "")
	mode := strings.ToLower(strings.TrimSpace(req.GetString("mode", "oldest")))
	if mode == "" {
		mode = "oldest"
	}
	if mode != "list" && mode != "oldest" && mode != "newest" {
		return mcp.NewToolResultError("invalid mode: must be 'list', 'oldest', or 'newest'"), nil
	}

	byHash := req.GetBool("by_hash", true)
	token := req.GetString("confirmation_token", "")

	if mode != "list" {
		if token == "" {
			ticket := s.tickets.CreateTicket("dedupe", remote, map[string]string{
				"remote":  remote,
				"path":    path,
				"mode":    mode,
				"by_hash": fmt.Sprintf("%t", byHash),
			})
			return jsonResult(map[string]any{
				"status":       "confirmation_required",
				"message":      "Deduplication can permanently delete duplicate files and requires confirmation.",
				"ticket_id":    ticket.ID,
				"token":        ticket.ID,
				"expires_at":   ticket.ExpiresAt,
				"ttl_seconds":  int(time.Until(ticket.ExpiresAt).Seconds()),
				"instructions": "Call bisync_dedupe_by_hash with confirmation_token set to the provided token to proceed.",
			})
		}

		if _, err := s.tickets.ConsumeTicket(token, "dedupe", remote); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid or expired confirmation token: %v", err)), nil
		}
	}

	var rcClient *client.RcloneClient
	if s.container != nil {
		if s.container.Client != nil {
			rcClient = s.container.Client
		} else if s.container.SyncService != nil && s.container.SyncService.Client != nil {
			rcClient = s.container.SyncService.Client
		}
	}

	var output string
	if rcClient != nil {
		out, err := rcClient.Dedupe(ctx, remote, path, mode, byHash)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("dedupe execution failed: %v", err)), nil
		}
		output = out
	} else {
		output = fmt.Sprintf("[SIMULATED] Deduplication completed for %s:%s (mode: %s, by_hash: %t)", remote, path, mode, byHash)
	}

	if s.container != nil && s.container.AuditService != nil {
		_, _ = s.container.AuditService.RecordEvent(remote, audit.ActionDedupeExecuted, "mcp", map[string]any{
			"remote":     remote,
			"path":       path,
			"mode":       mode,
			"by_hash":    byHash,
			"raw_output": output,
		})
	}

	return jsonResult(map[string]any{
		"status":  "success",
		"remote":  remote,
		"path":    path,
		"mode":    mode,
		"by_hash": byHash,
		"output":  output,
	})
}
