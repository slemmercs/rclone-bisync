package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
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
			mcp.WithString("decision", mcp.Required(), mcp.Description("Resolution decision: 'keep-local', 'keep-remote', 'keep-both', 'accept-delete', 'restore-item', 'skip', or 'dismiss'")),
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

		expectedParams := map[string]string{
			"resync": fmt.Sprintf("%t", resync),
			"force":  fmt.Sprintf("%t", force),
		}
		if _, err := s.tickets.ConsumeTicket(token, "trigger_sync", profileID, expectedParams); err != nil {
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
		errMsg := fmt.Sprintf("sync execution failed: %v", err)
		if s.container.ReviewService != nil {
			if sess, sessErr := s.container.ReviewService.GetActiveSession(ctx); sessErr == nil && sess != nil {
				errMsg += fmt.Sprintf("\n[Review Barrier Active]: Conflict/deletion review session '%s' created (%d deletions, %d conflicts). Use bisync_inspect_conflict_session and bisync_resolve_review to inspect and resolve.", sess.ID, len(sess.Deletions), len(sess.Conflicts))
			}
		}
		return mcp.NewToolResultError(errMsg), nil
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



