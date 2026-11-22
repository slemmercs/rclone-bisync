package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
)

func (s *Server) registerRemoteTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("bisync_list_providers",
			mcp.WithDescription("List supported Rclone cloud and object storage providers categorized by type"),
			mcp.WithString("category", mcp.Description("Optional filter by category: cloud, object, protocol, virtual, other")),
			mcp.WithBoolean("popular_only", mcp.DefaultBool(false), mcp.Description("Filter to only popular providers (e.g., drive, onedrive, s3, b2, dropbox)")),
			mcp.WithBoolean("force_refresh", mcp.DefaultBool(false), mcp.Description("Bypass internal 5-minute cache and query rclone RC fresh")),
		),
		s.handleListProviders,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_create_remote",
			mcp.WithDescription("Provision a new Rclone remote target with optional automated sync profile provisioning bridge"),
			mcp.WithString("name", mcp.Required(), mcp.Description("Unique name for the remote (alphanumeric, dash, underscore; colons stripped)")),
			mcp.WithString("type", mcp.Required(), mcp.Description("Provider backend type (e.g. s3, drive, b2, sftp)")),
			mcp.WithObject("parameters", mcp.Description("Configuration key-value pairs for the remote")),
			mcp.WithBoolean("create_profile", mcp.DefaultBool(false), mcp.Description("Whether to automatically create a linked sync profile")),
			mcp.WithString("local_path", mcp.Description("Local filesystem path if creating a linked profile")),
			mcp.WithString("remote_subpath", mcp.Description("Optional subdirectory path within the remote")),
			mcp.WithString("profile_name", mcp.Description("Optional display name for the linked profile")),
			mcp.WithString("schedule", mcp.Description("Sync schedule interval preset: 'manual', '15m', '30m', '1h', '4h', 'daily' (default: 'manual')")),
		),
		s.handleCreateRemote,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_delete_remote",
			mcp.WithDescription("Delete an existing Rclone remote with blast-radius protection and profile-link safeguards"),
			mcp.WithString("name", mcp.Required(), mcp.Description("Target remote name to delete")),
			mcp.WithBoolean("force", mcp.DefaultBool(false), mcp.Description("Force deletion even if active profiles reference this remote")),
			mcp.WithString("confirmation_token", mcp.Description("Confirmation token required for destructive remote deletion")),
		),
		s.handleDeleteRemote,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_hasher_command",
			mcp.WithDescription("Execute Hasher cache maintenance command (dump, fulldump, drop, import, stickyimport)"),
			mcp.WithString("remote", mcp.Required(), mcp.Description("Hasher remote target name")),
			mcp.WithString("command", mcp.Required(), mcp.Description("Hasher backend command: dump, fulldump, drop, import, stickyimport")),
			mcp.WithString("args", mcp.Description("Optional positional arguments or space-separated string (e.g. hash types)")),
			mcp.WithObject("opt", mcp.Description("Optional key-value options for the hasher command")),
		),
		s.handleHasherCommandTool,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_create_crypt_remote",
			mcp.WithDescription("Wrap an existing remote with client-side Crypt encryption overlay"),
			mcp.WithString("name", mcp.Required(), mcp.Description("Unique name for the encrypted remote (alphanumeric, dash, underscore)")),
			mcp.WithString("remote", mcp.Required(), mcp.Description("Base remote target to wrap (e.g. 'myremote:path')")),
			mcp.WithString("password", mcp.Required(), mcp.Description("Primary encryption password")),
			mcp.WithString("password2", mcp.Description("Optional secondary password / salt")),
			mcp.WithString("filename_encryption", mcp.Description("Filename encryption mode: 'standard', 'obfuscate', 'off' (default: 'standard')")),
			mcp.WithBoolean("create_profile", mcp.DefaultBool(false), mcp.Description("Whether to automatically create a linked sync profile")),
			mcp.WithString("local_path", mcp.Description("Local filesystem path if creating a linked profile")),
			mcp.WithString("schedule", mcp.Description("Sync schedule interval preset: 'manual', '15m', '30m', '1h', '4h', 'daily' (default: 'manual')")),
		),
		s.handleCreateCryptRemoteTool,
	)
}

func (s *Server) handleListProviders(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.container == nil || s.container.RemoteWizardService == nil {
		return mcp.NewToolResultError("RemoteWizardService unavailable"), nil
	}

	categoryFilter := strings.ToLower(strings.TrimSpace(req.GetString("category", "")))
	popularOnly := req.GetBool("popular_only", false)
	forceRefresh := req.GetBool("force_refresh", false)

	catalog, err := s.container.RemoteWizardService.GetProvidersCatalog(ctx, forceRefresh)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get provider catalog: %v", err)), nil
	}

	filtered := make([]models.ProviderCatalogItem, 0, len(catalog.Providers))
	for _, p := range catalog.Providers {
		if categoryFilter != "" && string(p.Category) != categoryFilter {
			continue
		}
		if popularOnly && !p.Popular {
			continue
		}
		filtered = append(filtered, p)
	}

	return jsonResult(map[string]any{
		"providers":  filtered,
		"categories": catalog.Categories,
		"total":      len(filtered),
	})
}

func (s *Server) handleCreateRemote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.cfg.ReadOnly {
		return mcp.NewToolResultError("operation rejected: MCP server is running in read-only mode"), nil
	}
	if s.container == nil || s.container.RemoteWizardService == nil {
		return mcp.NewToolResultError("RemoteWizardService unavailable"), nil
	}

	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'name'"), nil
	}
	cleanName := strings.TrimRight(strings.TrimSpace(name), ":")
	if cleanName == "" {
		return mcp.NewToolResultError("remote name cannot be empty"), nil
	}

	remType, err := req.RequireString("type")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'type'"), nil
	}

	params := extractMapArg(req.Params.Arguments, "parameters")
	createProfile := req.GetBool("create_profile", false)
	localPath := req.GetString("local_path", "")
	remoteSubpath := req.GetString("remote_subpath", "")
	profileName := req.GetString("profile_name", "")
	schedule := strings.TrimSpace(req.GetString("schedule", ""))

	if createProfile && schedule != "" && !profiles.IsValidSchedule(schedule) {
		return mcp.NewToolResultError(fmt.Sprintf("invalid schedule %q: must be one of 'manual', '15m', '30m', '1h', '4h', 'daily'", schedule)), nil
	}

	bridgeReq := models.CreateRemoteBridgeRequest{
		Name:          cleanName,
		Type:          remType,
		Parameters:    params,
		CreateProfile: createProfile,
		LocalPath:     localPath,
		RemoteSubpath: remoteSubpath,
		ProfileName:   profileName,
		Schedule:      schedule,
	}

	result, err := s.container.RemoteWizardService.CreateRemote(ctx, bridgeReq, "mcp")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create remote: %v", err)), nil
	}

	return jsonResult(result)
}

func (s *Server) handleDeleteRemote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.cfg.ReadOnly {
		return mcp.NewToolResultError("operation rejected: MCP server is running in read-only mode"), nil
	}
	if s.container == nil || s.container.RemoteWizardService == nil {
		return mcp.NewToolResultError("RemoteWizardService unavailable"), nil
	}

	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'name'"), nil
	}
	cleanName := strings.TrimRight(strings.TrimSpace(name), ":")

	force := req.GetBool("force", false)
	token := req.GetString("confirmation_token", "")

	// Destructive remote deletion requires confirmation ticket
	if token == "" {
		desc := fmt.Sprintf("Delete remote %q from Rclone configuration", cleanName)
		if force {
			desc += " (force override active: profiles referencing this remote may break)"
		}
		ticket := s.tickets.CreateTicket("delete_remote", cleanName, map[string]string{
			"description": desc,
			"remote":      cleanName,
			"force":       fmt.Sprintf("%t", force),
		})
		return jsonResult(map[string]any{
			"status":       "confirmation_required",
			"message":      fmt.Sprintf("Deleting remote %q is a destructive operation that requires explicit confirmation.", cleanName),
			"ticket_id":    ticket.ID,
			"token":        ticket.ID,
			"expires_at":   ticket.ExpiresAt,
			"ttl_seconds":  int(time.Until(ticket.ExpiresAt).Seconds()),
			"instructions": "Call bisync_delete_remote with confirmation_token set to the provided token to proceed.",
		})
	}

	expectedParams := map[string]string{
		"force": fmt.Sprintf("%t", force),
	}
	if _, err := s.tickets.ConsumeTicket(token, "delete_remote", cleanName, expectedParams); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid or expired confirmation token: %v", err)), nil
	}

	if err := s.container.RemoteWizardService.DeleteRemote(ctx, cleanName, force, "mcp"); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete remote %q: %v", cleanName, err)), nil
	}

	return jsonResult(map[string]any{
		"status":  "success",
		"remote":  cleanName,
		"message": fmt.Sprintf("Remote %q deleted successfully", cleanName),
	})
}

func extractMapArg(args any, key string) map[string]any {
	argsMap, ok := args.(map[string]any)
	if !ok || argsMap == nil {
		return nil
	}
	if raw, ok := argsMap[key]; ok && raw != nil {
		switch v := raw.(type) {
		case map[string]any:
			return v
		case string:
			if strings.TrimSpace(v) != "" {
				var parsed map[string]any
				if err := json.Unmarshal([]byte(v), &parsed); err == nil {
					return parsed
				}
			}
		}
	}
	return nil
}

func (s *Server) handleHasherCommandTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cmd := strings.ToLower(strings.TrimSpace(req.GetString("command", "")))
	if s.cfg.ReadOnly && cmd != "dump" && cmd != "fulldump" {
		return mcp.NewToolResultError("operation rejected: MCP server is running in read-only mode"), nil
	}

	if s.container == nil || s.container.HasherService == nil {
		return mcp.NewToolResultError("HasherService unavailable"), nil
	}

	remote, err := req.RequireString("remote")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'remote'"), nil
	}
	cleanRemote := strings.TrimRight(strings.TrimSpace(remote), ":")
	if cleanRemote == "" {
		return mcp.NewToolResultError("remote cannot be empty"), nil
	}

	if cmd == "" {
		c, err := req.RequireString("command")
		if err != nil {
			return mcp.NewToolResultError("missing required parameter 'command'"), nil
		}
		cmd = strings.ToLower(strings.TrimSpace(c))
	}

	args := extractStringSliceArg(req.Params.Arguments, "args")
	opt := extractStringMapArg(req.Params.Arguments, "opt")

	cmdReq := models.HasherCommandRequest{
		Remote:  cleanRemote,
		Command: cmd,
		Args:    args,
		Opt:     opt,
	}

	res, err := s.container.HasherService.ExecuteHasherCommand(ctx, cmdReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("hasher command %q failed: %v", cmd, err)), nil
	}

	return jsonResult(res)
}

func (s *Server) handleCreateCryptRemoteTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.cfg.ReadOnly {
		return mcp.NewToolResultError("operation rejected: MCP server is running in read-only mode"), nil
	}
	if s.container == nil || s.container.CryptService == nil {
		return mcp.NewToolResultError("CryptService unavailable"), nil
	}

	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'name'"), nil
	}
	cleanName := strings.TrimRight(strings.TrimSpace(name), ":")
	if cleanName == "" {
		return mcp.NewToolResultError("remote name cannot be empty"), nil
	}

	remote, err := req.RequireString("remote")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'remote'"), nil
	}
	cleanRemote := strings.TrimSpace(remote)
	if cleanRemote == "" {
		return mcp.NewToolResultError("target remote cannot be empty"), nil
	}

	password, err := req.RequireString("password")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'password'"), nil
	}

	password2 := req.GetString("password2", "")
	fnEnc := req.GetString("filename_encryption", "standard")
	createProfile := req.GetBool("create_profile", false)
	localPath := req.GetString("local_path", "")
	schedule := strings.TrimSpace(req.GetString("schedule", ""))

	if createProfile && schedule != "" && !profiles.IsValidSchedule(schedule) {
		return mcp.NewToolResultError(fmt.Sprintf("invalid schedule %q: must be one of 'manual', '15m', '30m', '1h', '4h', 'daily'", schedule)), nil
	}

	cryptReq := models.CryptWrapRequest{
		Name:                    cleanName,
		Remote:                  cleanRemote,
		Password:                password,
		Password2:               password2,
		FilenameEncryption:      fnEnc,
		DirectoryNameEncryption: true,
		CreateProfile:           createProfile,
		LocalPath:               localPath,
		Schedule:                schedule,
	}

	res, err := s.container.CryptService.WrapRemoteInCrypt(ctx, cryptReq, "mcp")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create crypt remote: %v", err)), nil
	}

	return jsonResult(res)
}

func extractStringMapArg(args any, key string) map[string]string {
	argsMap, ok := args.(map[string]any)
	if !ok || argsMap == nil {
		return nil
	}
	res := make(map[string]string)
	if raw, ok := argsMap[key]; ok && raw != nil {
		switch v := raw.(type) {
		case map[string]any:
			for k, val := range v {
				res[k] = fmt.Sprint(val)
			}
			return res
		case map[string]string:
			return v
		case string:
			if strings.TrimSpace(v) != "" {
				var parsed map[string]string
				if err := json.Unmarshal([]byte(v), &parsed); err == nil {
					return parsed
				}
			}
		}
	}
	if len(res) == 0 {
		return nil
	}
	return res
}

func extractStringSliceArg(args any, key string) []string {
	argsMap, ok := args.(map[string]any)
	if !ok || argsMap == nil {
		return nil
	}
	var res []string
	if raw, ok := argsMap[key]; ok && raw != nil {
		switch v := raw.(type) {
		case string:
			trimmed := strings.TrimSpace(v)
			if trimmed != "" {
				res = strings.Fields(trimmed)
			}
		case []any:
			for _, item := range v {
				if str, ok := item.(string); ok && strings.TrimSpace(str) != "" {
					res = append(res, strings.TrimSpace(str))
				}
			}
		case []string:
			res = v
		}
	}
	return res
}

