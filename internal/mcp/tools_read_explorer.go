package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

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
	limit := req.GetInt("limit", 50)
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := req.GetInt("offset", 0)
	if offset < 0 {
		offset = 0
	}
	filterStr := strings.ToLower(strings.TrimSpace(req.GetString("filter", "")))

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

	var matchedItems []map[string]any
	for _, rawItem := range listing.Items {
		name, _ := rawItem["Name"].(string)
		itemPath, _ := rawItem["Path"].(string)
		if filterStr != "" && !strings.Contains(strings.ToLower(name), filterStr) && !strings.Contains(strings.ToLower(itemPath), filterStr) {
			continue
		}
		isDir, _ := rawItem["IsDir"].(bool)
		var size int64
		switch v := rawItem["Size"].(type) {
		case float64:
			size = int64(v)
		case int64:
			size = v
		case int:
			size = int64(v)
		}
		modTime, _ := rawItem["ModTime"].(string)

		item := map[string]any{
			"name":   name,
			"path":   itemPath,
			"is_dir": isDir,
		}
		if size > 0 {
			item["size"] = size
		}
		if modTime != "" {
			item["mod_time"] = modTime
		}
		matchedItems = append(matchedItems, item)
	}

	totalItems := len(matchedItems)
	pagedItems := []map[string]any{}
	if offset < totalItems {
		end := offset + limit
		if end > totalItems {
			end = totalItems
		}
		pagedItems = matchedItems[offset:end]
	}

	return jsonResult(map[string]any{
		"remote":         listing.Remote,
		"path":           listing.Path,
		"has_canary":     listing.HasCanary,
		"total_items":    totalItems,
		"returned_items": len(pagedItems),
		"offset":         offset,
		"limit":          limit,
		"has_more":       offset+len(pagedItems) < totalItems,
		"items":          pagedItems,
	})
}
