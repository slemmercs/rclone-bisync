package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
)

func (s *Server) registerResources() {
	// Static Resources
	s.mcpServer.AddResource(
		mcp.NewResource("bisync://profiles", "Profiles",
			mcp.WithResourceDescription("JSON list of all configured profiles"),
			mcp.WithMIMEType("application/json"),
		),
		s.handleResourceProfiles,
	)

	s.mcpServer.AddResource(
		mcp.NewResource("bisync://status", "Status",
			mcp.WithResourceDescription("JSON daemon health and telemetry summary"),
			mcp.WithMIMEType("application/json"),
		),
		s.handleResourceStatus,
	)

	// Resource Templates
	s.mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate("bisync://profile/{id}", "Profile Details",
			mcp.WithTemplateDescription("Detailed profile configuration by ID"),
			mcp.WithTemplateMIMEType("application/json"),
		),
		s.handleResourceProfileByID,
	)

	s.mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate("bisync://history/{id}", "Profile History",
			mcp.WithTemplateDescription("Recent sync execution history for a profile"),
			mcp.WithTemplateMIMEType("application/json"),
		),
		s.handleResourceHistoryByID,
	)

	s.mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate("bisync://conflicts/{session_id}", "Conflict Session",
			mcp.WithTemplateDescription("Conflict review session data by session ID"),
			mcp.WithTemplateMIMEType("application/json"),
		),
		s.handleResourceConflictSession,
	)
}

func (s *Server) handleResourceProfiles(_ context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	var list []*profiles.Profile
	if s.container != nil && s.container.ProfileService != nil {
		list = s.container.ProfileService.List(false)
	}
	if list == nil {
		list = []*profiles.Profile{}
	}

	payload := map[string]any{"profiles": list, "total": len(list)}
	return jsonResourceContents(req.Params.URI, payload)
}

func (s *Server) handleResourceStatus(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	if s.container == nil || s.container.TelemetryService == nil {
		return jsonResourceContents(req.Params.URI, map[string]any{"status": "idle", "daemon": "unknown"})
	}
	isSyncing := false
	if s.container.ProfileService != nil {
		isSyncing = s.container.ProfileService.IsAnySyncing()
	}
	payload := s.container.TelemetryService.GetTelemetryPayload(ctx, isSyncing, 10)
	return jsonResourceContents(req.Params.URI, payload)
}

func (s *Server) handleResourceProfileByID(_ context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	id := strings.TrimPrefix(req.Params.URI, "bisync://profile/")
	if id == "" {
		return nil, fmt.Errorf("profile ID is required in URI")
	}

	if s.container == nil || s.container.ProfileService == nil {
		return nil, fmt.Errorf("ProfileService is not initialized")
	}

	p, err := s.container.ProfileService.Get(id)
	if err != nil {
		return nil, fmt.Errorf("profile '%s' not found: %w", id, err)
	}

	return jsonResourceContents(req.Params.URI, p)
}

func (s *Server) handleResourceHistoryByID(_ context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	id := strings.TrimPrefix(req.Params.URI, "bisync://history/")
	if id == "" {
		return nil, fmt.Errorf("profile ID is required in URI")
	}

	if s.container != nil && s.container.AuditService != nil {
		filter := audit.Filter{
			ProfileID: id,
			Action:    audit.ActionSyncExecuted,
			Limit:     20,
		}
		records, total, err := s.container.AuditService.Query(filter)
		if err == nil {
			return jsonResourceContents(req.Params.URI, map[string]any{"profile_id": id, "total": total, "records": records})
		}
	}

	if s.container != nil && s.container.ProfileService != nil {
		hist, err := s.container.ProfileService.GetHistory(id)
		if err == nil {
			return jsonResourceContents(req.Params.URI, map[string]any{"profile_id": id, "history": hist})
		}
	}

	return jsonResourceContents(req.Params.URI, map[string]any{"profile_id": id, "records": []any{}, "total": 0})
}

func (s *Server) handleResourceConflictSession(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	sessionID := strings.TrimPrefix(req.Params.URI, "bisync://conflicts/")
	if s.container == nil || s.container.ReviewService == nil {
		return nil, fmt.Errorf("ReviewService is not initialized")
	}

	active, err := s.container.ReviewService.GetActiveSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve conflict session: %w", err)
	}
	if active == nil || (sessionID != "" && active.ID != sessionID) {
		return nil, fmt.Errorf("conflict session '%s' not found or inactive", sessionID)
	}

	return jsonResourceContents(req.Params.URI, active)
}

func jsonResourceContents(uri string, data any) ([]mcp.ResourceContents, error) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to encode resource content: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      uri,
			MIMEType: "application/json",
			Text:     string(b),
		},
	}, nil
}
