package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/compiler"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
)

func (s *Server) handleValidateFlags(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var argsList []string

	// 1. Parse raw flags if provided (accepts space-separated string or array of strings)
	if argsMap, ok := req.Params.Arguments.(map[string]any); ok {
		if rawFlags, found := argsMap["flags"]; found && rawFlags != nil {
			switch v := rawFlags.(type) {
			case string:
				trimmed := strings.TrimSpace(v)
				if trimmed != "" {
					argsList = strings.Fields(trimmed)
				}
			case []any:
				for _, item := range v {
					if str, ok := item.(string); ok && strings.TrimSpace(str) != "" {
						argsList = append(argsList, strings.TrimSpace(str))
					}
				}
			case []string:
				argsList = v
			}
		}
	}

	// 2. Parse profile_id if provided and merge or load flags
	profileID := strings.TrimSpace(req.GetString("profile_id", ""))
	if profileID != "" {
		if s.container == nil || s.container.ProfileService == nil {
			return mcp.NewToolResultError("ProfileService unavailable"), nil
		}
		p, err := s.container.ProfileService.Get(profileID)
		if err != nil || p == nil {
			return mcp.NewToolResultError(fmt.Sprintf("profile %q not found", profileID)), nil
		}
		globals := config.DefaultGlobalDefaults()
		if s.container.Config != nil {
			globals = s.container.Config.GlobalDefaults
		}
		profileArgs, err := profiles.BuildCLIArgs(p, globals, nil)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to compile profile flags: %v", err)), nil
		}
		if len(argsList) == 0 {
			argsList = profileArgs
		} else {
			argsList = append(profileArgs, argsList...)
		}
	}

	if len(argsList) == 0 {
		return mcp.NewToolResultError("either 'flags' or 'profile_id' must be provided"), nil
	}

	res, _ := compiler.CompileFlagsToParams(argsList)

	response := map[string]any{
		"valid":              res.Valid,
		"params":             res.Params,
		"config":             res.Config,
		"backend_options":    res.BackendOptions,
		"unrecognized_flags": res.UnrecognizedFlags,
	}
	if profileID != "" {
		response["profile_id"] = profileID
	}
	if len(res.Filter) > 0 {
		response["filter"] = res.Filter
	}
	if len(res.Errors) > 0 {
		response["errors"] = res.Errors
	}

	return jsonResult(response)
}
