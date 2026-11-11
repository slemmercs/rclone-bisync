package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

func (s *Server) registerExplorerOpsTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("bisync_preview_file",
			mcp.WithDescription("Inspect safe text or image preview metadata for a file under remote or local path"),
			mcp.WithString("remote", mcp.Required(), mcp.Description("Target remote name or 'local'")),
			mcp.WithString("path", mcp.Required(), mcp.Description("Relative file path to preview")),
			mcp.WithString("profile_id", mcp.Description("Optional profile ID to infer local jail root")),
		),
		s.handlePreviewFile,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_mkdir",
			mcp.WithDescription("Create a directory under local or remote workspace path"),
			mcp.WithString("remote", mcp.Required(), mcp.Description("Target remote name or 'local'")),
			mcp.WithString("path", mcp.Required(), mcp.Description("Directory path to create")),
			mcp.WithString("profile_id", mcp.Description("Optional profile ID to infer local jail root")),
		),
		s.handleMakeDir,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_copy_item",
			mcp.WithDescription("Copy a file or directory between remote and local storage targets"),
			mcp.WithString("src_remote", mcp.Required(), mcp.Description("Source remote name or 'local'")),
			mcp.WithString("src_path", mcp.Required(), mcp.Description("Source relative path")),
			mcp.WithString("dst_remote", mcp.Required(), mcp.Description("Destination remote name or 'local'")),
			mcp.WithString("dst_path", mcp.Required(), mcp.Description("Destination relative path")),
			mcp.WithBoolean("is_dir", mcp.DefaultBool(false), mcp.Description("True if copying a directory")),
			mcp.WithString("profile_id", mcp.Description("Optional profile ID to infer local jail root and excludes")),
		),
		s.handleCopyItem,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_move_item",
			mcp.WithDescription("Move or rename a file or directory between remote and local storage targets"),
			mcp.WithString("src_remote", mcp.Required(), mcp.Description("Source remote name or 'local'")),
			mcp.WithString("src_path", mcp.Required(), mcp.Description("Source relative path")),
			mcp.WithString("dst_remote", mcp.Required(), mcp.Description("Destination remote name or 'local'")),
			mcp.WithString("dst_path", mcp.Required(), mcp.Description("Destination relative path")),
			mcp.WithBoolean("is_dir", mcp.DefaultBool(false), mcp.Description("True if moving a directory")),
			mcp.WithString("profile_id", mcp.Description("Optional profile ID to infer local jail root and excludes")),
		),
		s.handleMoveItem,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_delete_item",
			mcp.WithDescription("Safely delete a file or directory with canary protection and confirmation safeguards"),
			mcp.WithString("remote", mcp.Required(), mcp.Description("Target remote name or 'local'")),
			mcp.WithString("path", mcp.Required(), mcp.Description("Relative path to delete")),
			mcp.WithBoolean("is_dir", mcp.DefaultBool(false), mcp.Description("True if deleting a directory")),
			mcp.WithString("profile_id", mcp.Description("Optional profile ID to infer local jail root")),
			mcp.WithString("confirmation_token", mcp.Description("Confirmation token required for deletion")),
		),
		s.handleDeleteItem,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("bisync_purge_dir",
			mcp.WithDescription("Recursively delete a directory and all contents with high blast-radius safeguards"),
			mcp.WithString("remote", mcp.Required(), mcp.Description("Target remote name or 'local'")),
			mcp.WithString("path", mcp.Required(), mcp.Description("Directory relative path to purge")),
			mcp.WithString("profile_id", mcp.Description("Optional profile ID to infer local jail root")),
			mcp.WithString("confirmation_token", mcp.Description("Confirmation token required for directory purge")),
		),
		s.handlePurgeDir,
	)
}

func (s *Server) resolveLocalPath(profileID string) string {
	if s.container != nil {
		if s.container.ProfileService != nil && profileID != "" {
			if p, err := s.container.ProfileService.Get(profileID); err == nil && p != nil && p.LocalPath != "" {
				return p.LocalPath
			}
		}
		if s.container.Config != nil {
			return s.container.Config.LocalPath
		}
	}
	return ""
}

func (s *Server) resolveExcludePatterns(profileID string) []string {
	if s.container != nil && s.container.ProfileService != nil && profileID != "" {
		if p, err := s.container.ProfileService.Get(profileID); err == nil && p != nil {
			return p.ExcludePatterns
		}
	}
	return nil
}

func (s *Server) handlePreviewFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	if strings.Contains(path, "..") {
		return mcp.NewToolResultError("path traversal forbidden"), nil
	}

	profileID := req.GetString("profile_id", "")
	localBase := s.resolveLocalPath(profileID)

	prev, err := s.container.ExplorerService.PreviewItem(ctx, remote, path, localBase)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to preview file: %v", err)), nil
	}

	return jsonResult(prev)
}

func (s *Server) handleMakeDir(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	if strings.Contains(path, "..") {
		return mcp.NewToolResultError("path traversal forbidden"), nil
	}

	profileID := req.GetString("profile_id", "")
	localBase := s.resolveLocalPath(profileID)

	res, err := s.container.ExplorerService.MakeDir(ctx, models.MkdirRequest{
		Remote:    remote,
		Path:      path,
		ProfileID: profileID,
	}, localBase)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("mkdir failed: %v", err)), nil
	}

	if res.Status == "success" && s.container.AuditService != nil {
		_, _ = s.container.AuditService.RecordEvent(profileID, audit.ActionWorkspaceDirCreated, "mcp", map[string]any{
			"remote": remote,
			"path":   path,
			"source": "MCP_TOOL",
		})
	}

	return jsonResult(res)
}

func (s *Server) handleCopyItem(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.cfg.ReadOnly {
		return mcp.NewToolResultError("operation rejected: MCP server is running in read-only mode"), nil
	}
	if s.container == nil || s.container.ExplorerService == nil {
		return mcp.NewToolResultError("ExplorerService unavailable"), nil
	}

	srcRemote, err := req.RequireString("src_remote")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'src_remote'"), nil
	}
	srcPath, err := req.RequireString("src_path")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'src_path'"), nil
	}
	dstRemote, err := req.RequireString("dst_remote")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'dst_remote'"), nil
	}
	dstPath, err := req.RequireString("dst_path")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'dst_path'"), nil
	}
	if strings.Contains(srcPath, "..") || strings.Contains(dstPath, "..") {
		return mcp.NewToolResultError("path traversal forbidden"), nil
	}

	isDir := req.GetBool("is_dir", false)
	profileID := req.GetString("profile_id", "")
	localBase := s.resolveLocalPath(profileID)
	excludes := s.resolveExcludePatterns(profileID)

	res, err := s.container.ExplorerService.CopyItemWithOptions(ctx, srcRemote, srcPath, dstRemote, dstPath, isDir, localBase, excludes)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("copy failed: %v", err)), nil
	}

	if res.Status == "success" && s.container.AuditService != nil {
		_, _ = s.container.AuditService.RecordEvent(profileID, audit.ActionWorkspaceItemCopied, "mcp", map[string]any{
			"src_remote": srcRemote,
			"src_path":   srcPath,
			"dst_remote": dstRemote,
			"dst_path":   dstPath,
			"is_dir":     isDir,
			"source":     "MCP_TOOL",
		})
	}

	return jsonResult(res)
}

func (s *Server) handleMoveItem(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.cfg.ReadOnly {
		return mcp.NewToolResultError("operation rejected: MCP server is running in read-only mode"), nil
	}
	if s.container == nil || s.container.ExplorerService == nil {
		return mcp.NewToolResultError("ExplorerService unavailable"), nil
	}

	srcRemote, err := req.RequireString("src_remote")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'src_remote'"), nil
	}
	srcPath, err := req.RequireString("src_path")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'src_path'"), nil
	}
	dstRemote, err := req.RequireString("dst_remote")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'dst_remote'"), nil
	}
	dstPath, err := req.RequireString("dst_path")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'dst_path'"), nil
	}
	if strings.Contains(srcPath, "..") || strings.Contains(dstPath, "..") {
		return mcp.NewToolResultError("path traversal forbidden"), nil
	}
	if strings.Contains(srcPath, "RCLONE_TEST") || strings.Contains(dstPath, "RCLONE_TEST") {
		return mcp.NewToolResultError("operation rejected: canary validation file ('RCLONE_TEST') cannot be moved or overwritten"), nil
	}

	isDir := req.GetBool("is_dir", false)
	profileID := req.GetString("profile_id", "")
	localBase := s.resolveLocalPath(profileID)

	res, err := s.container.ExplorerService.MoveItem(ctx, models.MoveRequest{
		ProfileID: profileID,
		SrcRemote: srcRemote,
		SrcPath:   srcPath,
		DstRemote: dstRemote,
		DstPath:   dstPath,
		IsDir:     isDir,
	}, localBase)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("move failed: %v", err)), nil
	}

	if res.Status == "success" && s.container.AuditService != nil {
		_, _ = s.container.AuditService.RecordEvent(profileID, audit.ActionWorkspaceItemMoved, "mcp", map[string]any{
			"src_remote": srcRemote,
			"src_path":   srcPath,
			"dst_remote": dstRemote,
			"dst_path":   dstPath,
			"is_dir":     isDir,
			"source":     "MCP_TOOL",
		})
	}

	return jsonResult(res)
}
