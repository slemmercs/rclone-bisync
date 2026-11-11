package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

func (s *Server) handleDeleteItem(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.cfg.ReadOnly {
		return mcp.NewToolResultError("operation rejected: MCP server is running in read-only mode"), nil
	}
	if s.container == nil || s.container.ExplorerService == nil {
		return mcp.NewToolResultError("ExplorerService unavailable"), nil
	}

	remote, err := req.RequireString("remote")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'remote'"), nil
	}
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'path'"), nil
	}
	cleanPath := strings.Trim(strings.TrimSpace(path), "/")
	if cleanPath == "" {
		return mcp.NewToolResultError("path cannot be empty"), nil
	}
	if strings.Contains(cleanPath, "..") {
		return mcp.NewToolResultError("path traversal forbidden"), nil
	}
	if strings.Contains(cleanPath, "RCLONE_TEST") {
		return mcp.NewToolResultError("operation rejected: canary validation file ('RCLONE_TEST') cannot be deleted"), nil
	}

	isDir := req.GetBool("is_dir", false)
	profileID := req.GetString("profile_id", "")
	targetKey := fmt.Sprintf("%s:%s", remote, cleanPath)
	token := req.GetString("confirmation_token", "")

	if token == "" {
		itemType := "file"
		if isDir {
			itemType = "directory"
		}
		ticket := s.tickets.CreateTicket("delete_item", targetKey, map[string]string{
			"remote": remote,
			"path":   cleanPath,
			"is_dir": fmt.Sprintf("%t", isDir),
		})
		return jsonResult(map[string]any{
			"status":       "confirmation_required",
			"message":      fmt.Sprintf("Deleting %s %q on %s requires explicit confirmation.", itemType, cleanPath, remote),
			"ticket_id":    ticket.ID,
			"token":        ticket.ID,
			"expires_at":   ticket.ExpiresAt,
			"ttl_seconds":  int(time.Until(ticket.ExpiresAt).Seconds()),
			"instructions": "Call bisync_delete_item with confirmation_token set to the provided token to proceed.",
		})
	}

	expectedParams := map[string]string{
		"is_dir": fmt.Sprintf("%t", isDir),
	}
	if _, err := s.tickets.ConsumeTicket(token, "delete_item", targetKey, expectedParams); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid or expired confirmation token: %v", err)), nil
	}

	localBase := s.resolveLocalPath(profileID)
	res, err := s.container.ExplorerService.DeleteItem(ctx, remote, cleanPath, isDir, localBase)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("delete failed: %v", err)), nil
	}

	if res.Status == "success" && s.container.AuditService != nil {
		_, _ = s.container.AuditService.RecordEvent(profileID, audit.ActionWorkspaceItemDeleted, "mcp", map[string]any{
			"remote": remote,
			"path":   cleanPath,
			"is_dir": isDir,
			"source": "MCP_TOOL",
		})
	}

	return jsonResult(res)
}

func (s *Server) handlePurgeDir(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.cfg.ReadOnly {
		return mcp.NewToolResultError("operation rejected: MCP server is running in read-only mode"), nil
	}
	if s.container == nil || s.container.ExplorerService == nil {
		return mcp.NewToolResultError("ExplorerService unavailable"), nil
	}

	remote, err := req.RequireString("remote")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'remote'"), nil
	}
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'path'"), nil
	}
	cleanPath := strings.Trim(strings.TrimSpace(path), "/")
	if cleanPath == "" || cleanPath == "." {
		return mcp.NewToolResultError("purging remote root path is prohibited"), nil
	}
	if strings.Contains(cleanPath, "..") {
		return mcp.NewToolResultError("path traversal forbidden"), nil
	}
	if strings.Contains(cleanPath, "RCLONE_TEST") {
		return mcp.NewToolResultError("operation rejected: directory containing canary validation ('RCLONE_TEST') cannot be purged"), nil
	}

	profileID := req.GetString("profile_id", "")
	targetKey := fmt.Sprintf("%s:%s", remote, cleanPath)
	token := req.GetString("confirmation_token", "")

	if token == "" {
		ticket := s.tickets.CreateTicket("purge_dir", targetKey, map[string]string{
			"remote": remote,
			"path":   cleanPath,
		})
		return jsonResult(map[string]any{
			"status":       "confirmation_required",
			"message":      fmt.Sprintf("Purging directory %q on %s is a destructive operation that recursively deletes all contents and requires explicit confirmation.", cleanPath, remote),
			"ticket_id":    ticket.ID,
			"token":        ticket.ID,
			"expires_at":   ticket.ExpiresAt,
			"ttl_seconds":  int(time.Until(ticket.ExpiresAt).Seconds()),
			"instructions": "Call bisync_purge_dir with confirmation_token set to the provided token to proceed.",
		})
	}

	if _, err := s.tickets.ConsumeTicket(token, "purge_dir", targetKey); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid or expired confirmation token: %v", err)), nil
	}

	localBase := s.resolveLocalPath(profileID)
	res, err := s.container.ExplorerService.PurgeDir(ctx, models.PurgeRequest{
		Remote:    remote,
		Path:      cleanPath,
		ProfileID: profileID,
	}, localBase)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("purge failed: %v", err)), nil
	}

	if res.Status == "success" && s.container.AuditService != nil {
		_, _ = s.container.AuditService.RecordEvent(profileID, audit.ActionWorkspaceDirPurged, "mcp", map[string]any{
			"remote": remote,
			"path":   cleanPath,
			"source": "MCP_TOOL",
		})
	}

	return jsonResult(res)
}
