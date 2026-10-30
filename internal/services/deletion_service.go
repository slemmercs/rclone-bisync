package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/state"
)

// DeletionService handles scanning and retrieval of pending deletions.
type DeletionService struct {
	Config *config.Config
}

// NewDeletionService initializes a new DeletionService.
func NewDeletionService(cfg *config.Config) *DeletionService {
	return &DeletionService{
		Config: cfg,
	}
}

// ParsePendingDeletionsFromLog scans daemon log for aborted deletions.
func (d *DeletionService) ParsePendingDeletionsFromLog() state.PendingDeletionsData {
	if d.Config == nil {
		return state.PendingDeletionsData{HasDeletions: false, TotalCount: 0, Files: []string{}, Source: "unknown"}
	}

	daemonLog := filepath.Join(d.Config.BaseDir, "rclone-daemon.log")
	if _, err := os.Stat(daemonLog); os.IsNotExist(err) {
		return state.PendingDeletionsData{HasDeletions: false, TotalCount: 0, Files: []string{}, Source: "unknown"}
	}

	var deletedFiles []string
	abortMsg := ""
	source := "Google Drive (remote)"
	targetAction := "Will be deleted from local disk (~/gdrive)"

	lines := TailFile(daemonLog, 2000)
	for i := len(lines) - 1; i >= 0; i-- {
		lineStr := strings.TrimSpace(lines[i])
		if lineStr == "" {
			continue
		}

		// Fast check: if a sync completed successfully before any abort, no deletions can be pending
		if abortMsg == "" && strings.Contains(strings.ToLower(lineStr), "bisync successful") {
			return state.PendingDeletionsData{HasDeletions: false, TotalCount: 0, Files: []string{}}
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(lineStr), &data); err != nil {
			continue
		}

		rawMsg, _ := data["msg"].(string)
		sublines := strings.Split(rawMsg, "\n")
		for j := len(sublines) - 1; j >= 0; j-- {
			imsg := strings.TrimSpace(sublines[j])
			if strings.HasPrefix(imsg, "{") && strings.HasSuffix(imsg, "}") {
				var inner map[string]interface{}
				if err := json.Unmarshal([]byte(imsg), &inner); err == nil {
					if innerMsg, ok := inner["msg"].(string); ok {
						imsg = innerMsg
					}
				}
			}

			lower := strings.ToLower(imsg)
			if abortMsg == "" && strings.Contains(lower, "bisync successful") {
				return state.PendingDeletionsData{HasDeletions: false, TotalCount: 0, Files: []string{}}
			}

			if strings.Contains(lower, "too many deletes") || strings.Contains(lower, "safety abort") {
				abortMsg = imsg
				if strings.Contains(lower, "path2") {
					source = "Local (~/gdrive)"
					targetAction = "Will be deleted from Google Drive"
				} else {
					source = "Google Drive (remote)"
					targetAction = "Will be deleted from local disk (~/gdrive)"
				}
			}

			if strings.Contains(imsg, "File was deleted") {
				parts := strings.SplitN(imsg, "File was deleted", 2)
				if len(parts) == 2 {
					fp := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(parts[1]), "-"))
					if fp != "" {
						found := false
						for _, f := range deletedFiles {
							if f == fp {
								found = true
								break
							}
						}
						if !found {
							deletedFiles = append(deletedFiles, fp)
						}
					}
				}
			}
		}

		if (abortMsg != "" || len(deletedFiles) > 0) && (strings.Contains(lineStr, "Synching Path1") || strings.Contains(strings.ToLower(lineStr), "bisync successful")) {
			break
		}
	}

	if abortMsg != "" && len(deletedFiles) > 0 {
		// Reverse deletedFiles to preserve chronological order
		for i, j := 0, len(deletedFiles)-1; i < j; i, j = i+1, j-1 {
			deletedFiles[i], deletedFiles[j] = deletedFiles[j], deletedFiles[i]
		}
		return state.PendingDeletionsData{
			HasDeletions: true,
			TotalCount:   len(deletedFiles),
			Source:       source,
			TargetAction: targetAction,
			Message:      abortMsg,
			Files:        deletedFiles,
		}
	}

	return state.PendingDeletionsData{HasDeletions: false, TotalCount: 0, Files: []string{}}
}

// GetPendingDeletions returns structured pending deletions or falls back to log parsing.
func (d *DeletionService) GetPendingDeletions() state.PendingDeletionsData {
	if d.Config != nil && d.Config.PendingDeletionsFile != "" {
		structured, err := state.ReadPendingDeletions(d.Config.PendingDeletionsFile)
		if err == nil && structured != nil {
			return *structured
		}
	}
	return d.ParsePendingDeletionsFromLog()
}
