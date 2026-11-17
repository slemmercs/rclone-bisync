package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
)

func (s *Server) handleGetQueue(w http.ResponseWriter, r *http.Request) {
	if s.ProfileService == nil {
		writeJSON(w, http.StatusServiceUnavailable, models.ErrorResponse{Status: "error", Error: "Profile service not initialized"})
		return
	}

	snap := s.ProfileService.GetQueueSnapshot()
	resp := models.QueueSnapshotResponse{
		Status: "success",
		Snapshot: models.QueueSnapshotDTO{
			Paused:          snap.Paused,
			ActiveSignature: string(snap.ActiveSignature),
			MaxConcurrency:  snap.MaxConcurrency,
			RunningCount:    snap.RunningCount,
			QueuedCount:     snap.QueuedCount,
			ActiveJobs:      make([]models.QueueJobDTO, 0, len(snap.ActiveJobs)),
			QueuedJobs:      make([]models.QueueJobDTO, 0, len(snap.QueuedJobs)),
		},
	}

	for _, item := range snap.ActiveJobs {
		resp.Snapshot.ActiveJobs = append(resp.Snapshot.ActiveJobs, models.QueueJobDTO{
			ProfileID:   item.ProfileID,
			ProfileName: item.ProfileName,
			Origin:      item.Origin,
			DryRun:      item.DryRun,
			Resync:      item.Resync,
			Force:       item.Force,
			Signature:   string(item.Signature),
			EnqueuedAt:  item.EnqueuedAt,
			StartedAt:   item.StartedAt,
			WaitSeconds: item.WaitSeconds,
		})
	}

	for _, item := range snap.QueuedJobs {
		resp.Snapshot.QueuedJobs = append(resp.Snapshot.QueuedJobs, models.QueueJobDTO{
			ProfileID:   item.ProfileID,
			ProfileName: item.ProfileName,
			Origin:      item.Origin,
			DryRun:      item.DryRun,
			Resync:      item.Resync,
			Force:       item.Force,
			Signature:   string(item.Signature),
			EnqueuedAt:  item.EnqueuedAt,
			StartedAt:   item.StartedAt,
			WaitSeconds: item.WaitSeconds,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handlePauseQueue(w http.ResponseWriter, r *http.Request) {
	if s.ProfileService == nil {
		writeJSON(w, http.StatusServiceUnavailable, models.ErrorResponse{Status: "error", Error: "Profile service not initialized"})
		return
	}

	s.ProfileService.PauseQueue()
	writeJSON(w, http.StatusOK, models.QueueActionResponse{
		Status:  "success",
		Message: "queue admissions paused",
	})
}

func (s *Server) handleResumeQueue(w http.ResponseWriter, r *http.Request) {
	if s.ProfileService == nil {
		writeJSON(w, http.StatusServiceUnavailable, models.ErrorResponse{Status: "error", Error: "Profile service not initialized"})
		return
	}

	s.ProfileService.ResumeQueue()
	writeJSON(w, http.StatusOK, models.QueueActionResponse{
		Status:  "success",
		Message: "queue admissions resumed",
	})
}

func (s *Server) handleCancelQueuedJob(w http.ResponseWriter, r *http.Request) {
	if s.ProfileService == nil {
		writeJSON(w, http.StatusServiceUnavailable, models.ErrorResponse{Status: "error", Error: "Profile service not initialized"})
		return
	}

	profileID := r.PathValue("id")
	if profileID == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Status: "error", Error: "Profile ID required"})
		return
	}

	err := s.ProfileService.CancelQueuedJob(profileID)
	if err != nil {
		if errors.Is(err, profiles.ErrJobNotQueued) {
			writeJSON(w, http.StatusNotFound, models.ErrorResponse{Status: "error", Error: "Job is not in queue"})
			return
		}
		if errors.Is(err, profiles.ErrJobAlreadyRunning) {
			writeJSON(w, http.StatusConflict, models.ErrorResponse{Status: "error", Error: "Job is already running and cannot be cancelled from queue"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Status: "error", Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, models.QueueActionResponse{
		Status:  "success",
		Message: fmt.Sprintf("queued profile %s cancelled", profileID),
	})
}

func (s *Server) handleClearQueue(w http.ResponseWriter, r *http.Request) {
	if s.ProfileService == nil {
		writeJSON(w, http.StatusServiceUnavailable, models.ErrorResponse{Status: "error", Error: "Profile service not initialized"})
		return
	}

	cleared := s.ProfileService.ClearQueue()
	writeJSON(w, http.StatusOK, models.QueueActionResponse{
		Status:  "success",
		Message: fmt.Sprintf("cleared %d queued jobs", cleared),
		Count:   cleared,
	})
}
