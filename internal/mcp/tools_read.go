package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/engine/review"
	"github.com/slemmercs/rclone-bisync/internal/models"
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
			mcp.WithDescription("Get compact system health summary (status, active syncs, daemon health, canary status)"),
			mcp.WithBoolean("include_logs", mcp.DefaultBool(false), mcp.Description("Include recent bisync and daemon logs")),
			mcp.WithBoolean("include_config", mcp.DefaultBool(false), mcp.Description("Include sanitized runtime configuration")),
			mcp.WithBoolean("include_metrics", mcp.DefaultBool(false), mcp.Description("Include execution metrics timeseries")),
		),
		s.handleGetStatus,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_get_logs",
			mcp.WithDescription("Retrieve recent bisync execution logs with tail limit, severity level, profile, and run ID filtering"),
			mcp.WithNumber("limit", mcp.DefaultNumber(20), mcp.Description("Number of recent log entries to retrieve (max 100)")),
			mcp.WithString("level", mcp.Description("Optional severity filter (e.g. ERROR, NOTICE, INFO, DEBUG)")),
			mcp.WithString("profile_id", mcp.Description("Optional profile ID filter to scope log output to a specific profile")),
			mcp.WithString("run_id", mcp.Description("Optional run ID filter to inspect logs for a specific sync execution")),
		),
		s.handleGetLogs,
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
			mcp.WithNumber("limit", mcp.DefaultNumber(20), mcp.Description("Maximum number of records to return (default: 20, max: 100)")),
			mcp.WithBoolean("include_hashes", mcp.DefaultBool(false), mcp.Description("Include cryptographic prev_hash and hash fields")),
		),
		s.handleGetSyncHistory,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_inspect_conflict_session",
			mcp.WithDescription("Retrieve pending conflict review session details, file differences, and candidate resolutions"),
			mcp.WithString("session_id", mcp.Description("Optional conflict session ID; returns active session if omitted")),
			mcp.WithBoolean("unresolved_only", mcp.DefaultBool(false), mcp.Description("Only return unresolved conflict and deletion candidates")),
		),
		s.handleInspectConflictSession,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_explore_path",
			mcp.WithDescription("Safely explore files and directories under a configured profile or local path with jail containment"),
			mcp.WithString("remote", mcp.Required(), mcp.Description("Storage target, e.g. 'local' or 'remote:'")),
			mcp.WithString("path", mcp.Description("Relative path within the remote to explore")),
			mcp.WithString("profile_id", mcp.Description("Profile ID to infer local jail root from")),
			mcp.WithNumber("limit", mcp.DefaultNumber(50), mcp.Description("Max items to return (default: 50, max: 200)")),
			mcp.WithNumber("offset", mcp.DefaultNumber(0), mcp.Description("Item offset for pagination")),
			mcp.WithString("filter", mcp.Description("Optional substring to filter item names/paths")),
		),
		s.handleExplorePath,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_validate_flags",
			mcp.WithDescription("Validate CLI flags and preview compiled Rclone bisync parameters and engine config"),
			mcp.WithString("flags", mcp.Description("CLI flags to compile and validate (space-separated string or array)")),
			mcp.WithString("profile_id", mcp.Description("Optional profile ID to compile and validate its effective configured flags")),
		),
		s.handleValidateFlags,
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

func buildCompactStatus(payload map[string]interface{}, includeLogs, includeConfig, includeMetrics bool) map[string]any {
	res := map[string]any{
		"status":             payload["execution_status"],
		"is_syncing":         payload["is_syncing"],
		"has_review":         payload["has_review"],
		"daemon_health":      payload["daemon_health"],
		"notification_state": payload["notification_state"],
	}
	if payload["transfer_stats"] != nil {
		res["transfer_stats"] = payload["transfer_stats"]
	}
	if includeLogs {
		res["bisync_logs"] = payload["bisync_logs"]
		res["daemon_logs"] = payload["daemon_logs"]
	}
	if includeMetrics {
		res["metrics"] = payload["metrics"]
	}
	if includeConfig {
		res["config"] = payload["config"]
	}
	return res
}

func (s *Server) handleGetStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.container == nil || s.container.TelemetryService == nil {
		return jsonResult(map[string]any{"status": "idle", "daemon": "unknown"})
	}
	includeLogs := req.GetBool("include_logs", false)
	includeConfig := req.GetBool("include_config", false)
	includeMetrics := req.GetBool("include_metrics", false)

	isSyncing := false
	if s.container.ProfileService != nil {
		isSyncing = s.container.ProfileService.IsAnySyncing()
	}
	logLimit := 0
	if includeLogs {
		logLimit = 10
	}
	payload := s.container.TelemetryService.GetTelemetryPayload(ctx, isSyncing, logLimit)
	return jsonResult(buildCompactStatus(payload, includeLogs, includeConfig, includeMetrics))
}

func (s *Server) handleGetLogs(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.container == nil || s.container.TelemetryService == nil {
		return jsonResult(map[string]any{"logs": []string{}, "total": 0})
	}
	limit := req.GetInt("limit", 20)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	level := strings.ToUpper(strings.TrimSpace(req.GetString("level", "")))
	profileID := strings.TrimSpace(req.GetString("profile_id", ""))
	runID := strings.TrimSpace(req.GetString("run_id", ""))

	fetchLimit := 100
	if level != "" || profileID != "" || runID != "" {
		fetchLimit = 500
	}

	rawLogs := s.container.TelemetryService.GetCachedBisyncLogs(fetchLimit)
	var filtered []string
	for _, line := range rawLogs {
		var entry struct {
			Level     string `json:"level"`
			ProfileID string `json:"profile_id"`
			RunID     string `json:"run_id"`
		}
		isJSON := json.Unmarshal([]byte(line), &entry) == nil

		if level != "" {
			if isJSON && entry.Level != "" {
				if !strings.Contains(strings.ToUpper(entry.Level), level) {
					continue
				}
			} else if !strings.Contains(strings.ToUpper(line), level) {
				continue
			}
		}

		if profileID != "" {
			if isJSON && entry.ProfileID != "" {
				if !strings.EqualFold(entry.ProfileID, profileID) {
					continue
				}
			} else if !strings.Contains(line, profileID) {
				continue
			}
		}

		if runID != "" {
			if isJSON && entry.RunID != "" {
				if !strings.EqualFold(entry.RunID, runID) {
					continue
				}
			} else if !strings.Contains(line, runID) {
				continue
			}
		}

		filtered = append(filtered, line)
		if len(filtered) >= limit {
			break
		}
	}
	if filtered == nil {
		filtered = []string{}
	}
	resp := map[string]any{
		"total": len(filtered),
		"level": level,
		"logs":  filtered,
	}
	if profileID != "" {
		resp["profile_id"] = profileID
	}
	if runID != "" {
		resp["run_id"] = runID
	}
	return jsonResult(resp)
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

// SlimAuditRecord projects audit events without cryptographic hashes for token efficiency.
type SlimAuditRecord struct {
	SeqID       int64           `json:"seq_id"`
	Timestamp   string          `json:"timestamp"`
	ProfileID   string          `json:"profile_id"`
	ProfileName string          `json:"profile_name,omitempty"`
	Action      audit.Action    `json:"action"`
	Actor       string          `json:"actor"`
	Details     json.RawMessage `json:"details"`
}

func (s *Server) handleGetSyncHistory(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	profileID := req.GetString("profile_id", "")
	limit := req.GetInt("limit", 20)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	includeHashes := req.GetBool("include_hashes", false)

	if s.container != nil && s.container.AuditService != nil {
		filter := audit.Filter{
			ProfileID: profileID,
			Action:    audit.ActionSyncExecuted,
			Limit:     limit,
		}
		records, total, err := s.container.AuditService.Query(filter)
		if err == nil {
			if includeHashes {
				return jsonResult(map[string]any{"total": total, "records": records})
			}
			slim := make([]SlimAuditRecord, 0, len(records))
			for _, r := range records {
				slim = append(slim, SlimAuditRecord{
					SeqID:       r.SeqID,
					Timestamp:   r.Timestamp,
					ProfileID:   r.ProfileID,
					ProfileName: r.ProfileName,
					Action:      r.Action,
					Actor:       r.Actor,
					Details:     r.Details,
				})
			}
			return jsonResult(map[string]any{"total": total, "records": slim})
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
	if sessionID != "" && (!review.IsValidSessionID(sessionID) || strings.Contains(sessionID, "/") || strings.Contains(sessionID, "\\") || strings.Contains(sessionID, "..")) {
		return mcp.NewToolResultError("invalid session_id format"), nil
	}
	unresolvedOnly := req.GetBool("unresolved_only", false)

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

	resSession := active
	if unresolvedOnly {
		filtered := *active
		var unresConflicts []models.ConflictCandidate
		for _, c := range active.Conflicts {
			if !c.Resolved {
				unresConflicts = append(unresConflicts, c)
			}
		}
		filtered.Conflicts = unresConflicts

		var unresDeletions []models.DeletionCandidate
		for _, d := range active.Deletions {
			if !d.Resolved {
				unresDeletions = append(unresDeletions, d)
			}
		}
		filtered.Deletions = unresDeletions
		resSession = &filtered
	}

	return jsonResult(map[string]any{
		"active":  true,
		"session": resSession,
	})
}

func jsonResult(data any) (*mcp.CallToolResult, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("serialization failed: %v", err)), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}
