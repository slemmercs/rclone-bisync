package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
)

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

		expectedParams := map[string]string{
			"remote":  remote,
			"path":    path,
			"mode":    mode,
			"by_hash": fmt.Sprintf("%t", byHash),
		}
		if _, err := s.tickets.ConsumeTicket(token, "dedupe", remote, expectedParams); err != nil {
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

	lines := strings.Split(output, "\n")
	var collisionsCount, deletedCount int
	for _, line := range lines {
		l := strings.TrimSpace(line)
		if l == "" {
			continue
		}
		if strings.Contains(strings.ToLower(l), "duplicate") {
			collisionsCount++
		}
		if strings.Contains(strings.ToLower(l), "deleted") || strings.Contains(strings.ToLower(l), "removed") || strings.Contains(strings.ToLower(l), "renamed") {
			deletedCount++
		}
	}

	// Invariant (ADR-004): Audit ledger strictly records mutations (mode != "list" and deletedCount > 0).
	// Bounded fields prevent database bloat; raw terminal output is omitted from audit.db.
	if s.container != nil && s.container.AuditService != nil && mode != "list" && deletedCount > 0 {
		_, _ = s.container.AuditService.RecordEvent(remote, audit.ActionDedupeExecuted, "mcp", map[string]any{
			"remote":     remote,
			"path":       path,
			"mode":       mode,
			"by_hash":    byHash,
			"collisions": collisionsCount,
			"deleted":    deletedCount,
		})
	}

	truncatedOutput := output
	isTruncated := false
	if len(truncatedOutput) > 2000 {
		truncatedOutput = truncatedOutput[:2000] + "\n...[output truncated for LLM context]"
		isTruncated = true
	}

	return jsonResult(map[string]any{
		"status":     "success",
		"remote":     remote,
		"path":       path,
		"mode":       mode,
		"by_hash":    byHash,
		"collisions": collisionsCount,
		"deleted":    deletedCount,
		"output":     truncatedOutput,
		"truncated":  isTruncated,
	})
}
