package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

func (s *Server) handleGetBandwidthLimit(w http.ResponseWriter, r *http.Request) {
	if s.CoreService == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "Core service not available")
		return
	}

	limit, err := s.CoreService.GetBandwidthLimit(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, limit)
}

func (s *Server) handleSetBandwidthLimit(w http.ResponseWriter, r *http.Request) {
	if s.CoreService == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "Core service not available")
		return
	}

	var req models.BandwidthLimitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.Rate = strings.TrimSpace(req.Rate)
	if req.Rate == "" {
		writeJSONError(w, http.StatusBadRequest, "Bandwidth rate limit is required")
		return
	}

	res, err := s.CoreService.SetBandwidthLimit(r.Context(), req.Rate)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleGetCoreStats(w http.ResponseWriter, r *http.Request) {
	if s.CoreService == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "Core service not available")
		return
	}

	stats, err := s.CoreService.GetCoreStats(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, stats)
}
