package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
)

func (s *Server) registerReadTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("bisync_list_profiles",
			mcp.WithDescription("List all configured bisync profiles with current status, schedules, and path mappings"),
			mcp.WithBoolean("include_archived", mcp.DefaultBool(false), mcp.Description("Include archived profiles")),
		),
		s.handleListProfiles,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_get_status",
			mcp.WithDescription("Get overall system health, daemon status, active sync operations, and canary validity"),
		),
		s.handleGetStatus,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_list_remotes",
			mcp.WithDescription("List configured Rclone remotes, their storage types, and connectivity status"),
			mcp.WithBoolean("refresh", mcp.DefaultBool(false), mcp.Description("Force fresh probe of remote status")),
		),
		s.handleListRemotes,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_get_sync_history",
			mcp.WithDescription("Query historical sync execution records and audit ledger"),
			mcp.WithString("profile_id", mcp.Description("Optional profile ID to filter history")),
			mcp.WithNumber("limit", mcp.Description("Maximum number of records to return (default: 20)")),
		),
		s.handleGetSyncHistory,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_inspect_conflict_session",
			mcp.WithDescription("Retrieve pending conflict review session details, file differences, and candidate resolutions"),
			mcp.WithString("session_id", mcp.Description("Optional conflict session ID; returns active session if omitted")),
		),
		s.handleInspectConflictSession,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_explore_path",
			mcp.WithDescription("Safely explore files and directories under a configured profile or local path with jail containment"),
			mcp.WithString("remote", mcp.Required(), mcp.Description("Storage target, e.g. 'local' or 'remote:'")),
			mcp.WithString("path", mcp.Description("Relative path within the remote to explore")),
			mcp.WithString("profile_id", mcp.Description("Profile ID to infer local jail root from")),
		),
		s.handleExplorePath,
	)
}

func (s *Server) handleListProfiles(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.container == nil || s.container.ProfileService == nil {
		return jsonResult(map[string]any{"profiles": []*profiles.Profile{}, "total": 0})
	}
	includeArchived := req.GetBool("include_archived", false)
	list := s.container.ProfileService.List(includeArchived)
	return jsonResult(map[string]any{
		"profiles": list,
		"total":    len(list),
	})
}

func (s *Server) handleGetStatus(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.container == nil || s.container.TelemetryService == nil {
		return jsonResult(map[string]any{"status": "idle", "daemon": "unknown"})
	}
	isSyncing := false
	if s.container.ProfileService != nil {
		isSyncing = s.container.ProfileService.IsAnySyncing()
	}
	payload := s.container.TelemetryService.GetTelemetryPayload(ctx, isSyncing, 10)
	return jsonResult(payload)
}

func (s *Server) handleListRemotes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.container == nil || s.container.TelemetryService == nil {
		return jsonResult(map[string]any{"remotes": []any{}, "count": 0})
	}
	refresh := req.GetBool("refresh", false)
	summary, err := s.container.TelemetryService.GetRemotesSummaryCached(ctx, refresh)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to retrieve remotes: %v", err)), nil
	}
	return jsonResult(summary)
}

func (s *Server) handleGetSyncHistory(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	profileID := req.GetString("profile_id", "")
	limit := req.GetInt("limit", 20)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	if s.container != nil && s.container.AuditService != nil {
		filter := audit.Filter{
			ProfileID: profileID,
			Action:    audit.ActionSyncExecuted,
			Limit:     limit,
		}
		records, total, err := s.container.AuditService.Query(filter)
		if err == nil {
			return jsonResult(map[string]any{
				"total":   total,
				"records": records,
			})
		}
	}

	if s.container != nil && s.container.ProfileService != nil && profileID != "" {
		hist, err := s.container.ProfileService.GetHistory(profileID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to retrieve profile history: %v", err)), nil
		}
		return jsonResult(map[string]any{"history": hist})
	}

	return jsonResult(map[string]any{"records": []any{}, "total": 0})
}

func (s *Server) handleInspectConflictSession(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.container == nil || s.container.ReviewService == nil {
		return jsonResult(map[string]any{"active": false, "message": "ReviewService unavailable"})
	}
	sessionID := req.GetString("session_id", "")
	active, err := s.container.ReviewService.GetActiveSession(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get active review session: %v", err)), nil
	}
	if active == nil {
		return jsonResult(map[string]any{"active": false, "message": "No active conflict review session"})
	}
	if sessionID != "" && active.ID != sessionID {
		return jsonResult(map[string]any{"active": false, "message": "Session ID mismatch"})
	}
	return jsonResult(map[string]any{
		"active":  true,
		"session": active,
	})
}

func (s *Server) handleExplorePath(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.container == nil || s.container.ExplorerService == nil {
		return mcp.NewToolResultError("ExplorerService unavailable"), nil
	}
	remote, err := req.RequireString("remote")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'remote'"), nil
	}
	relPath := req.GetString("path", "")
	profileID := req.GetString("profile_id", "")

	var localBase string
	if s.container.ProfileService != nil && profileID != "" {
		if p, err := s.container.ProfileService.Get(profileID); err == nil && p != nil && p.LocalPath != "" {
			localBase = p.LocalPath
		}
	}
	if localBase == "" && s.container.Config != nil {
		localBase = s.container.Config.LocalPath
	}

	listing := s.container.ExplorerService.BrowsePath(ctx, remote, relPath, localBase)
	if listing.Error != "" {
		return mcp.NewToolResultError(fmt.Sprintf("explore path failed: %s", listing.Error)), nil
	}

	return jsonResult(listing)
}

func jsonResult(data any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("serialization failed: %v", err)), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}
