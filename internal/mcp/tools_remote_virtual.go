package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
)

func (s *Server) registerVirtualRemoteTools() {
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
