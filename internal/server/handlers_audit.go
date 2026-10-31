package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

func (s *Server) resolveProfileFilterID(query string) string {
	if query == "" || s.ProfileService == nil {
		return query
	}
	profiles := s.ProfileService.List(true)
	for _, p := range profiles {
		if strings.EqualFold(p.Name, query) {
			return p.ID
		}
	}
	return query
}

func (s *Server) enrichAuditRecordProfileNames(records []audit.AuditRecord) {
	nameMap := make(map[string]string)
	if s.ProfileService != nil {
		profiles := s.ProfileService.List(true)
		for _, p := range profiles {
			nameMap[p.ID] = p.Name
		}
	}
	for i := range records {
		if name, ok := nameMap[records[i].ProfileID]; ok && name != "" {
			records[i].ProfileName = name
		} else if records[i].ProfileID == "default" {
			records[i].ProfileName = "Default"
		} else {
			records[i].ProfileName = records[i].ProfileID
		}
	}
}

func (s *Server) handleAuditRecords(w http.ResponseWriter, r *http.Request) {
	if s.AuditService == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "error",
			"error":   "Audit service is not available",
			"records": []audit.AuditRecord{},
			"total":   0,
		})
		return
	}

	q := r.URL.Query()
	limit := 50
	if lStr := q.Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	page := 1
	if pStr := q.Get("page"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}

	offset := (page - 1) * limit
	if oStr := q.Get("offset"); oStr != "" {
		if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
			offset = o
			page = (offset / limit) + 1
		}
	}

	filter := audit.Filter{
		ProfileID: s.resolveProfileFilterID(q.Get("profile_id")),
		Action:    audit.Action(q.Get("action")),
		Actor:     q.Get("actor"),
		Limit:     limit,
		Offset:    offset,
	}

	records, total, err := s.AuditService.Query(filter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{
			Status: "error",
			Error:  fmt.Sprintf("Failed to query audit records: %v", err),
		})
		return
	}

	if records == nil {
		records = []audit.AuditRecord{}
	}

	s.enrichAuditRecordProfileNames(records)

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"records": records,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

func (s *Server) handleAuditVerify(w http.ResponseWriter, r *http.Request) {
	if s.AuditService == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "error",
			"valid":  false,
			"error":  "Audit service is not available",
		})
		return
	}

	res, err := s.AuditService.Verify()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{
			Status: "error",
			Error:  fmt.Sprintf("Ledger verification check failed: %v", err),
		})
		return
	}

	statusCode := http.StatusOK
	statusText := "success"
	if !res.Valid {
		statusText = "invalid"
	}

	writeJSON(w, statusCode, map[string]any{
		"status":        statusText,
		"valid":         res.Valid,
		"total_records": res.TotalRecords,
		"checked_at":    res.CheckedAt,
		"head_hash":     res.HeadHash,
		"error":         res.Error,
	})
}

func (s *Server) handleAuditExport(w http.ResponseWriter, r *http.Request) {
	if s.AuditService == nil {
		http.Error(w, "Audit service is not available", http.StatusServiceUnavailable)
		return
	}

	q := r.URL.Query()
	format := q.Get("format")
	if format == "" {
		format = "jsonl"
	}

	filter := audit.Filter{
		ProfileID: s.resolveProfileFilterID(q.Get("profile_id")),
		Action:    audit.Action(q.Get("action")),
		Actor:     q.Get("actor"),
	}

	data, mimeType, filename, err := s.AuditService.Export(format, filter)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to export audit ledger: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.WriteHeader(http.StatusOK)
	// #nosec G705 - exported audit data is served as an attachment download with nosniff
	_, _ = w.Write(data)
}

func (s *Server) handleAuditReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.AuditService == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "error",
			"error":  "Audit service is not available",
		})
		return
	}

	backup, err := s.AuditService.Reset()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{
			Status: "error",
			Error:  fmt.Sprintf("Failed to reset audit ledger: %v", err),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"message": "Audit ledger reset successfully",
		"backup":  backup,
	})
}
