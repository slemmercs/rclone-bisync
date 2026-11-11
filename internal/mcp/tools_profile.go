package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
)

func (s *Server) registerProfileTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("bisync_create_profile",
			mcp.WithDescription("Create a new bidirectional synchronization profile with path validation"),
			mcp.WithString("name", mcp.Required(), mcp.Description("Human-readable profile name")),
			mcp.WithString("remote", mcp.Required(), mcp.Description("Configured remote target name (or 'local', e.g. 'drive', 'backup')")),
			mcp.WithString("local_path", mcp.Required(), mcp.Description("Local filesystem directory path")),
			mcp.WithString("remote_path", mcp.Description("Optional remote subdirectory path")),
			mcp.WithString("schedule", mcp.Description("Sync schedule interval preset: 'manual', '15m', '30m', '1h', '4h', 'daily' (default: 'manual')")),
			mcp.WithString("conflict_mode", mcp.Description("Conflict resolution mode: 'path1', 'path2', 'newer', 'older', 'larger', 'smaller'")),
			mcp.WithString("review_mode", mcp.Description("Review mode for operations: 'disabled', 'deletions', 'conflicts', 'all' (default: 'disabled')")),
			mcp.WithString("compare_mode", mcp.Description("Comparison mode: 'size,modtime', 'size,modtime,checksum', 'checksum', 'modtime'")),
			mcp.WithNumber("max_delete", mcp.Description("Max allowable deletion percentage before triggering safety abort")),
		),
		s.handleCreateProfile,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_update_profile",
			mcp.WithDescription("Update configuration settings on an existing sync profile"),
			mcp.WithString("profile_id", mcp.Required(), mcp.Description("ID of the profile to update")),
			mcp.WithString("name", mcp.Description("Updated human-readable name")),
			mcp.WithString("remote", mcp.Description("Updated remote target name")),
			mcp.WithString("local_path", mcp.Description("Updated local directory path")),
			mcp.WithString("remote_path", mcp.Description("Updated remote subdirectory path")),
			mcp.WithString("schedule", mcp.Description("Updated sync schedule interval preset: 'manual', '15m', '30m', '1h', '4h', 'daily'")),
			mcp.WithString("conflict_mode", mcp.Description("Updated conflict resolution mode: 'path1', 'path2', 'newer', 'older', 'larger', 'smaller'")),
			mcp.WithString("review_mode", mcp.Description("Updated review mode: 'disabled', 'deletions', 'conflicts', 'all'")),
			mcp.WithString("compare_mode", mcp.Description("Updated comparison mode: 'size,modtime', 'size,modtime,checksum', 'checksum', 'modtime'")),
			mcp.WithNumber("max_delete", mcp.Description("Updated max delete percentage threshold")),
			mcp.WithBoolean("enabled", mcp.Description("Enable or disable scheduled execution")),
		),
		s.handleUpdateProfile,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_delete_profile",
			mcp.WithDescription("Permanently delete a sync profile with confirmation token protection"),
			mcp.WithString("profile_id", mcp.Required(), mcp.Description("Profile ID to permanently delete")),
			mcp.WithString("confirmation_token", mcp.Description("Confirmation token required for profile deletion")),
		),
		s.handleDeleteProfile,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_rollback_profile",
			mcp.WithDescription("Roll back a sync profile to a previous version snapshot from configuration history"),
			mcp.WithString("profile_id", mcp.Required(), mcp.Description("Profile ID to roll back")),
			mcp.WithNumber("version", mcp.Required(), mcp.Description("Target historical version number to restore")),
			mcp.WithString("confirmation_token", mcp.Description("Confirmation token required for profile rollback")),
		),
		s.handleRollbackProfile,
	)
}

func (s *Server) handleCreateProfile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.cfg.ReadOnly {
		return mcp.NewToolResultError("operation rejected: MCP server is running in read-only mode"), nil
	}
	if s.container == nil || s.container.ProfileService == nil {
		return mcp.NewToolResultError("ProfileService unavailable"), nil
	}

	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'name'"), nil
	}
	remote, err := req.RequireString("remote")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'remote'"), nil
	}
	localPath, err := req.RequireString("local_path")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'local_path'"), nil
	}

	if strings.Contains(localPath, "..") {
		return mcp.NewToolResultError("local_path contains prohibited path traversal sequence ('..')"), nil
	}
	cleanLocal := filepath.Clean(localPath)

	cleanRemote := strings.TrimRight(strings.TrimSpace(remote), ":")
	if cleanRemote == "" {
		return mcp.NewToolResultError("remote cannot be empty"), nil
	}

	sched := strings.TrimSpace(req.GetString("schedule", ""))
	if sched != "" && !profiles.IsValidSchedule(sched) {
		return mcp.NewToolResultError(fmt.Sprintf("invalid schedule %q: must be one of 'manual', '15m', '30m', '1h', '4h', 'daily'", sched)), nil
	}

	cMode := strings.TrimSpace(req.GetString("conflict_mode", ""))
	if cMode != "" && !profiles.IsValidConflictMode(cMode) {
		return mcp.NewToolResultError(fmt.Sprintf("invalid conflict_mode %q: must be one of 'path1', 'path2', 'newer', 'older', 'larger', 'smaller'", cMode)), nil
	}

	rMode := strings.TrimSpace(req.GetString("review_mode", ""))
	if rMode != "" && !profiles.IsValidReviewMode(profiles.ReviewMode(rMode)) {
		return mcp.NewToolResultError(fmt.Sprintf("invalid review_mode %q: must be one of 'disabled', 'deletions', 'conflicts', 'all'", rMode)), nil
	}

	cmpMode := strings.TrimSpace(req.GetString("compare_mode", ""))
	if cmpMode != "" && !profiles.IsValidCompareMode(cmpMode) {
		return mcp.NewToolResultError(fmt.Sprintf("invalid compare_mode %q: must be one of 'size,modtime', 'size,modtime,checksum', 'checksum', 'modtime'", cmpMode)), nil
	}

	prof := &profiles.Profile{
		Name:       strings.TrimSpace(name),
		Remote:     cleanRemote,
		LocalPath:  cleanLocal,
		RemotePath: strings.Trim(req.GetString("remote_path", ""), "/\\ "),
		Schedule:   sched,
		ReviewMode: profiles.ReviewMode(rMode),
		Enabled:    true,
	}

	if cMode != "" {
		prof.Flags.ConflictResolve = &cMode
	}
	if cmpMode != "" {
		prof.Flags.CompareMode = &cmpMode
	}
	if maxDel := req.GetInt("max_delete", -1); maxDel >= 0 {
		val := profiles.MaxDeleteValue(fmt.Sprintf("%d", maxDel))
		prof.Flags.MaxDelete = &val
	}

	created, err := s.container.ProfileService.Create(ctx, prof)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create profile: %v", err)), nil
	}

	return jsonResult(created)
}

func (s *Server) handleUpdateProfile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.cfg.ReadOnly {
		return mcp.NewToolResultError("operation rejected: MCP server is running in read-only mode"), nil
	}
	if s.container == nil || s.container.ProfileService == nil {
		return mcp.NewToolResultError("ProfileService unavailable"), nil
	}

	profileID, err := req.RequireString("profile_id")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'profile_id'"), nil
	}

	existing, err := s.container.ProfileService.Get(profileID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("profile %q not found: %v", profileID, err)), nil
	}

	if name := strings.TrimSpace(req.GetString("name", "")); name != "" {
		existing.Name = name
	}
	if rem := strings.TrimSpace(req.GetString("remote", "")); rem != "" {
		cleanRem := strings.TrimRight(rem, ":")
		if cleanRem == "" {
			return mcp.NewToolResultError("remote cannot be empty"), nil
		}
		existing.Remote = cleanRem
	}
	if loc := strings.TrimSpace(req.GetString("local_path", "")); loc != "" {
		if strings.Contains(loc, "..") {
			return mcp.NewToolResultError("local_path contains prohibited path traversal sequence ('..')"), nil
		}
		cleanLoc := filepath.Clean(loc)
		existing.LocalPath = cleanLoc
	}
	if remPath := req.GetString("remote_path", ""); remPath != "" {
		existing.RemotePath = strings.Trim(remPath, "/\\ ")
	}
	if sched := strings.TrimSpace(req.GetString("schedule", "")); sched != "" {
		if !profiles.IsValidSchedule(sched) {
			return mcp.NewToolResultError(fmt.Sprintf("invalid schedule %q: must be one of 'manual', '15m', '30m', '1h', '4h', 'daily'", sched)), nil
		}
		existing.Schedule = sched
	}
	if cMode := strings.TrimSpace(req.GetString("conflict_mode", "")); cMode != "" {
		if !profiles.IsValidConflictMode(cMode) {
			return mcp.NewToolResultError(fmt.Sprintf("invalid conflict_mode %q: must be one of 'path1', 'path2', 'newer', 'older', 'larger', 'smaller'", cMode)), nil
		}
		existing.Flags.ConflictResolve = &cMode
	}
	if rMode := strings.TrimSpace(req.GetString("review_mode", "")); rMode != "" {
		if !profiles.IsValidReviewMode(profiles.ReviewMode(rMode)) {
			return mcp.NewToolResultError(fmt.Sprintf("invalid review_mode %q: must be one of 'disabled', 'deletions', 'conflicts', 'all'", rMode)), nil
		}
		existing.ReviewMode = profiles.ReviewMode(rMode)
	}
	if cmpMode := strings.TrimSpace(req.GetString("compare_mode", "")); cmpMode != "" {
		if !profiles.IsValidCompareMode(cmpMode) {
			return mcp.NewToolResultError(fmt.Sprintf("invalid compare_mode %q: must be one of 'size,modtime', 'size,modtime,checksum', 'checksum', 'modtime'", cmpMode)), nil
		}
		existing.Flags.CompareMode = &cmpMode
	}
	if maxDel := req.GetInt("max_delete", -1); maxDel >= 0 {
		val := profiles.MaxDeleteValue(fmt.Sprintf("%d", maxDel))
		existing.Flags.MaxDelete = &val
	}
	if argsMap, ok := req.Params.Arguments.(map[string]any); ok {
		if _, exists := argsMap["enabled"]; exists {
			existing.Enabled = req.GetBool("enabled", true)
		}
	}

	updated, err := s.container.ProfileService.Update(ctx, existing, "Updated via MCP")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update profile %q: %v", profileID, err)), nil
	}

	return jsonResult(updated)
}

func (s *Server) handleDeleteProfile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.cfg.ReadOnly {
		return mcp.NewToolResultError("operation rejected: MCP server is running in read-only mode"), nil
	}
	if s.container == nil || s.container.ProfileService == nil {
		return mcp.NewToolResultError("ProfileService unavailable"), nil
	}

	profileID, err := req.RequireString("profile_id")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'profile_id'"), nil
	}

	// Verify profile exists before issuing confirmation
	if _, err := s.container.ProfileService.Get(profileID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("profile %q not found: %v", profileID, err)), nil
	}

	token := req.GetString("confirmation_token", "")
	if token == "" {
		ticket := s.tickets.CreateTicket("delete_profile", profileID, map[string]string{
			"profile_id": profileID,
		})
		return jsonResult(map[string]any{
			"status":       "confirmation_required",
			"message":      fmt.Sprintf("Permanently deleting profile %q is a destructive operation that requires explicit confirmation.", profileID),
			"ticket_id":    ticket.ID,
			"token":        ticket.ID,
			"expires_at":   ticket.ExpiresAt,
			"ttl_seconds":  int(time.Until(ticket.ExpiresAt).Seconds()),
			"instructions": "Call bisync_delete_profile with confirmation_token set to the provided token to proceed.",
		})
	}

	if _, err := s.tickets.ConsumeTicket(token, "delete_profile", profileID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid or expired confirmation token: %v", err)), nil
	}

	if err := s.container.ProfileService.Delete(ctx, profileID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete profile %q: %v", profileID, err)), nil
	}

	return jsonResult(map[string]any{
		"status":     "success",
		"profile_id": profileID,
		"message":    fmt.Sprintf("Profile %q deleted successfully", profileID),
	})
}

func (s *Server) handleRollbackProfile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.cfg.ReadOnly {
		return mcp.NewToolResultError("operation rejected: MCP server is running in read-only mode"), nil
	}
	if s.container == nil || s.container.ProfileService == nil {
		return mcp.NewToolResultError("ProfileService unavailable"), nil
	}

	profileID, err := req.RequireString("profile_id")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'profile_id'"), nil
	}

	version := req.GetInt("version", 0)
	if version <= 0 {
		return mcp.NewToolResultError("version must be a positive integer"), nil
	}

	token := req.GetString("confirmation_token", "")
	if token == "" {
		ticket := s.tickets.CreateTicket("rollback_profile", profileID, map[string]string{
			"profile_id": profileID,
			"version":    fmt.Sprintf("%d", version),
		})
		return jsonResult(map[string]any{
			"status":       "confirmation_required",
			"message":      fmt.Sprintf("Rolling back profile %q to version %d requires explicit confirmation.", profileID, version),
			"ticket_id":    ticket.ID,
			"token":        ticket.ID,
			"expires_at":   ticket.ExpiresAt,
			"ttl_seconds":  int(time.Until(ticket.ExpiresAt).Seconds()),
			"instructions": "Call bisync_rollback_profile with confirmation_token set to the provided token to proceed.",
		})
	}

	expectedParams := map[string]string{
		"version": fmt.Sprintf("%d", version),
	}
	if _, err := s.tickets.ConsumeTicket(token, "rollback_profile", profileID, expectedParams); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid or expired confirmation token: %v", err)), nil
	}

	rolledBack, err := s.container.ProfileService.Rollback(ctx, profileID, version)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to rollback profile %q: %v", profileID, err)), nil
	}

	return jsonResult(rolledBack)
}
