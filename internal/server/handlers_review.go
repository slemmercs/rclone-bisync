package server

import (
	"encoding/json"
	"net/http"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

func (s *Server) handleReviewSession(w http.ResponseWriter, r *http.Request) {
	if s.ReviewService == nil {
		writeJSON(w, http.StatusOK, models.ReviewSessionResponse{Active: false, Error: "ReviewService not initialized"})
		return
	}
	sess, err := s.ReviewService.GetActiveSession(r.Context())
	if err != nil {
		writeJSON(w, http.StatusOK, models.ReviewSessionResponse{Active: false, Error: err.Error()})
		return
	}
	if sess == nil {
		writeJSON(w, http.StatusOK, models.ReviewSessionResponse{Active: false})
		return
	}
	writeJSON(w, http.StatusOK, models.ReviewSessionResponse{Active: true, Session: sess})
}

func (s *Server) handleReviewAction(w http.ResponseWriter, r *http.Request) {
	if s.ReviewService == nil {
		writeJSON(w, http.StatusOK, models.StageDecisionResponse{Status: "error", Error: "ReviewService not initialized"})
		return
	}
	var req models.StageDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.StageDecisionResponse{Status: "error", Error: "Invalid request body"})
		return
	}
	if req.Decision == "" {
		writeJSON(w, http.StatusBadRequest, models.StageDecisionResponse{Status: "error", Error: "Decision is required"})
		return
	}

	sessionID := req.SessionID
	if sessionID == "" {
		active, err := s.ReviewService.GetActiveSession(r.Context())
		if err != nil || active == nil {
			writeJSON(w, http.StatusOK, models.StageDecisionResponse{Status: "error", Error: "No active review session found"})
			return
		}
		sessionID = active.ID
	}

	candidateIDs := req.CandidateIDs
	if len(candidateIDs) == 0 && req.CandidateID != "" {
		candidateIDs = []string{req.CandidateID}
	}
	if len(candidateIDs) == 0 {
		writeJSON(w, http.StatusBadRequest, models.StageDecisionResponse{Status: "error", Error: "CandidateID or CandidateIDs required"})
		return
	}

	var lastSession *models.ReviewSession
	var err error
	for _, cid := range candidateIDs {
		lastSession, err = s.ReviewService.StageCandidateDecision(sessionID, cid, req.Decision)
		if err != nil {
			writeJSON(w, http.StatusOK, models.StageDecisionResponse{Status: "error", Error: err.Error()})
			return
		}
	}

	writeJSON(w, http.StatusOK, models.StageDecisionResponse{Status: "success", Session: lastSession})
}

func (s *Server) handleReviewExecute(w http.ResponseWriter, r *http.Request) {
	if s.ReviewService == nil {
		writeJSON(w, http.StatusOK, models.ExecuteReviewResponse{Status: "error", Error: "ReviewService not initialized"})
		return
	}
	var req models.ExecuteReviewRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	sessionID := req.SessionID
	if sessionID == "" {
		active, err := s.ReviewService.GetActiveSession(r.Context())
		if err != nil || active == nil {
			writeJSON(w, http.StatusOK, models.ExecuteReviewResponse{Status: "error", Error: "No active review session found"})
			return
		}
		sessionID = active.ID
	}

	sess, err := s.ReviewService.ApplyReviewSession(r.Context(), sessionID)
	if err != nil {
		applied := 0
		var actions []models.ResolutionAction
		if sess != nil {
			applied = len(sess.Actions)
			actions = sess.Actions
		}
		writeJSON(w, http.StatusOK, models.ExecuteReviewResponse{
			Status:    "error",
			SessionID: sessionID,
			Applied:   applied,
			Actions:   actions,
			Error:     err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, models.ExecuteReviewResponse{
		Status:    "success",
		SessionID: sess.ID,
		Applied:   len(sess.Actions),
		Actions:   sess.Actions,
	})
}

func (s *Server) handleReviewRollback(w http.ResponseWriter, r *http.Request) {
	if s.ReviewService == nil {
		writeJSON(w, http.StatusOK, models.RollbackReviewResponse{Status: "error", Error: "ReviewService not initialized"})
		return
	}
	var req models.RollbackReviewRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	sessionID := req.SessionID
	if sessionID == "" {
		active, err := s.ReviewService.GetActiveSession(r.Context())
		if err != nil || active == nil {
			writeJSON(w, http.StatusOK, models.RollbackReviewResponse{Status: "error", Error: "No active review session found"})
			return
		}
		sessionID = active.ID
	}

	sess, err := s.ReviewService.RollbackReviewSession(r.Context(), sessionID)
	if err != nil {
		writeJSON(w, http.StatusOK, models.RollbackReviewResponse{
			Status:    "error",
			SessionID: sessionID,
			Error:     err.Error(),
		})
		return
	}

	restored := 0
	if sess != nil {
		restored = len(sess.Actions)
	}
	writeJSON(w, http.StatusOK, models.RollbackReviewResponse{
		Status:    "success",
		SessionID: sessionID,
		Restored:  restored,
	})
}

func (s *Server) handleReviewDismiss(w http.ResponseWriter, r *http.Request) {
	if s.ReviewService == nil {
		writeJSON(w, http.StatusOK, models.ActionResponse{Status: "error", Error: "ReviewService not initialized"})
		return
	}
	var req models.DismissReviewRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	sessionID := req.SessionID
	if sessionID == "" {
		active, err := s.ReviewService.GetActiveSession(r.Context())
		if err != nil || active == nil {
			writeJSON(w, http.StatusOK, models.ActionResponse{Status: "error", Error: "No active review session found"})
			return
		}
		sessionID = active.ID
	}

	if err := s.ReviewService.DismissReviewSession(sessionID); err != nil {
		writeJSON(w, http.StatusOK, models.ActionResponse{Status: "error", Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, models.ActionResponse{Status: "success", Msg: "Review session dismissed"})
}

func (s *Server) handleReviewActionBatch(w http.ResponseWriter, r *http.Request) {
	if s.ReviewService == nil {
		writeJSON(w, http.StatusOK, models.BatchReviewActionResponse{Status: "error", Error: "ReviewService not initialized"})
		return
	}

	var req models.BatchReviewActionRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	sessionID := req.SessionID
	if sessionID == "" {
		active, err := s.ReviewService.GetActiveSession(r.Context())
		if err != nil || active == nil {
			writeJSON(w, http.StatusOK, models.BatchReviewActionResponse{Status: "error", Error: "No active review session found"})
			return
		}
		sessionID = active.ID
	}

	stagedCount, err := s.ReviewService.ApplyBatchRecommendations(r.Context(), sessionID, req.MinConfidence, req.MaxAllowedRisk)
	if err != nil {
		writeJSON(w, http.StatusOK, models.BatchReviewActionResponse{Status: "error", Error: err.Error()})
		return
	}

	active, _ := s.ReviewService.GetActiveSession(r.Context())
	writeJSON(w, http.StatusOK, models.BatchReviewActionResponse{
		Status:      "success",
		Session:     active,
		StagedCount: stagedCount,
	})
}
