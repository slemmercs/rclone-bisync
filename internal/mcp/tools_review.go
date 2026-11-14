package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slemmercs/rclone-bisync/internal/engine/review"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

func normalizeDecision(d string) (models.ActionDecision, bool) {
	clean := strings.ToLower(strings.TrimSpace(d))
	clean = strings.ReplaceAll(clean, "_", "-")
	switch clean {
	case "keep-local":
		return models.DecisionKeepLocal, true
	case "keep-remote":
		return models.DecisionKeepRemote, true
	case "keep-both":
		return models.DecisionKeepBoth, true
	case "accept-delete":
		return models.DecisionAcceptDelete, true
	case "restore-item":
		return models.DecisionRestoreItem, true
	case "skip":
		return models.DecisionSkip, true
	case "dismiss":
		return "dismiss", true
	default:
		return "", false
	}
}

func (s *Server) handleResolveReview(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.cfg.ReadOnly {
		return mcp.NewToolResultError("operation rejected: MCP server is running in read-only mode"), nil
	}
	if s.container == nil || s.container.ReviewService == nil {
		return mcp.NewToolResultError("ReviewService is not initialized"), nil
	}

	decision, err := req.RequireString("decision")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'decision'"), nil
	}

	normDecision, ok := normalizeDecision(decision)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("invalid decision '%s': must be one of 'keep-local', 'keep-remote', 'keep-both', 'accept-delete', 'restore-item', 'skip', or 'dismiss'", decision)), nil
	}

	sessionID := req.GetString("session_id", "")
	if sessionID != "" && (!review.IsValidSessionID(sessionID) || strings.Contains(sessionID, "/") || strings.Contains(sessionID, "\\") || strings.Contains(sessionID, "..")) {
		return mcp.NewToolResultError("invalid session_id format"), nil
	}
	execute := req.GetBool("execute", false)
	token := req.GetString("confirmation_token", "")

	if sessionID == "" {
		active, err := s.container.ReviewService.GetActiveSession(ctx)
		if err != nil || active == nil {
			return mcp.NewToolResultError("no active conflict review session found"), nil
		}
		sessionID = active.ID
	}

	if normDecision == "dismiss" {
		if err := s.container.ReviewService.DismissReviewSession(sessionID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to dismiss review session: %v", err)), nil
		}
		return jsonResult(map[string]any{
			"status":     "success",
			"session_id": sessionID,
			"decision":   "dismiss",
			"message":    "Review session dismissed successfully without mutating files",
		})
	}

	if execute {
		if token == "" {
			ticket := s.tickets.CreateTicket("execute_review", sessionID, map[string]string{
				"session_id": sessionID,
				"decision":   string(normDecision),
			})
			return jsonResult(map[string]any{
				"status":       "confirmation_required",
				"message":      "Applying review decisions executes disk mutations and requires confirmation.",
				"ticket_id":    ticket.ID,
				"token":        ticket.ID,
				"expires_at":   ticket.ExpiresAt,
				"ttl_seconds":  int(time.Until(ticket.ExpiresAt).Seconds()),
				"instructions": "Call bisync_resolve_review with confirmation_token set to the provided token to execute.",
			})
		}

		expectedParams := map[string]string{
			"session_id": sessionID,
			"decision":   string(normDecision),
		}
		if _, err := s.tickets.ConsumeTicket(token, "execute_review", sessionID, expectedParams); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid or expired confirmation token: %v", err)), nil
		}

		sess, err := s.container.ReviewService.ApplyReviewSession(ctx, sessionID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to apply review session: %v", err)), nil
		}
		return jsonResult(map[string]any{
			"status":     "success",
			"session_id": sessionID,
			"applied":    len(sess.Actions),
			"actions":    sess.Actions,
		})
	}

	candidateID := req.GetString("candidate_id", "")
	var candidateIDs []string
	if candidateID != "" {
		candidateIDs = []string{candidateID}
	} else {
		active, err := s.container.ReviewService.GetActiveSession(ctx)
		if err == nil && active != nil {
			for _, d := range active.Deletions {
				candidateIDs = append(candidateIDs, d.ID)
			}
			for _, c := range active.Conflicts {
				candidateIDs = append(candidateIDs, c.ID)
			}
		}
	}

	sess, err := s.container.ReviewService.StageCandidateDecisions(sessionID, candidateIDs, normDecision)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to stage decision: %v", err)), nil
	}

	return jsonResult(map[string]any{
		"status":     "success",
		"session_id": sess.ID,
		"decision":   normDecision,
		"staged":     len(candidateIDs),
	})
}
