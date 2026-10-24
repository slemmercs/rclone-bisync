package review

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/core/state"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

// hashID produces a short deterministic ID from string inputs.
func hashID(prefix, input string) string {
	sum := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(sum[:6]))
}

// DiscoverCandidates scans structured pending files, log lines, and filesystem to extract review candidates.
func DiscoverCandidates(logLines []string, pendingFile, localRoot string) ([]models.DeletionCandidate, []models.ConflictCandidate, error) {
	deletions := discoverDeletions(logLines, pendingFile, localRoot)
	conflicts := discoverConflicts(logLines, localRoot)
	return deletions, conflicts, nil
}

// discoverDeletions extracts deletion candidates from structured state or log lines.
func discoverDeletions(logLines []string, pendingFile, localRoot string) []models.DeletionCandidate {
	var candidates []models.DeletionCandidate
	seen := make(map[string]bool)

	// 1. Attempt structured pending deletions file first
	if pendingFile != "" {
		if structured, err := state.ReadPendingDeletions(pendingFile); err == nil && structured != nil && structured.HasDeletions {
			source := models.SideRemote
			targetAction := "Will be deleted from local disk"
			if strings.Contains(strings.ToLower(structured.Source), "local") || strings.Contains(strings.ToLower(structured.TargetAction), "remote") {
				source = models.SideLocal
				targetAction = "Will be deleted from remote"
			}

			for _, f := range structured.Files {
				cleanPath := strings.TrimSpace(f)
				if idx := strings.Index(cleanPath, `","`); idx != -1 {
					cleanPath = cleanPath[:idx]
				}
				if idx := strings.Index(cleanPath, `", "`); idx != -1 {
					cleanPath = cleanPath[:idx]
				}
				if idx := strings.Index(cleanPath, `","source"`); idx != -1 {
					cleanPath = cleanPath[:idx]
				}
				if idx := strings.Index(cleanPath, `", "source"`); idx != -1 {
					cleanPath = cleanPath[:idx]
				}
				cleanPath = strings.Trim(cleanPath, `"',{}\`)
				cleanPath = filepath.Clean(strings.TrimSpace(cleanPath))
				if cleanPath == "" || cleanPath == "." || strings.Contains(cleanPath, ".review_backups") || seen[cleanPath] {
					continue
				}
				seen[cleanPath] = true

				cand := buildDeletionCandidate(cleanPath, source, targetAction, localRoot)
				candidates = append(candidates, cand)
			}
			if len(candidates) > 0 {
				return candidates
			}
		}
	}

	// 2. Parse from log lines
	source := models.SideRemote
	targetAction := "Will be deleted from local disk"

	for i := len(logLines) - 1; i >= 0; i-- {
		lower := strings.ToLower(logLines[i])
		if strings.Contains(lower, "too many deletes") {
			if strings.Contains(lower, "path2") {
				source = models.SideLocal
				targetAction = "Will be deleted from remote"
			}
			break
		}
	}

	for i := len(logLines) - 1; i >= 0; i-- {
		lineStr := strings.TrimSpace(logLines[i])
		if lineStr == "" {
			continue
		}

		rawMsg := lineStr
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(lineStr), &data); err == nil {
			if m, ok := data["msg"].(string); ok {
				rawMsg = m
			}
		} else {
			// Check for embedded JSON substring {...}
			start := strings.Index(lineStr, "{")
			end := strings.LastIndex(lineStr, "}")
			if start != -1 && end > start {
				if err := json.Unmarshal([]byte(lineStr[start:end+1]), &data); err == nil {
					if m, ok := data["msg"].(string); ok {
						rawMsg = m
					}
				}
			}
		}

		// Normalize literal escaped newlines
		normalized := strings.ReplaceAll(rawMsg, `\n`, "\n")
		sublines := strings.Split(normalized, "\n")
		for _, sub := range sublines {
			imsg := strings.TrimSpace(sub)
			if imsg == "" {
				continue
			}

			// If subline is an embedded JSON object, extract the inner "msg"
			var subData map[string]interface{}
			if err := json.Unmarshal([]byte(imsg), &subData); err == nil {
				if m, ok := subData["msg"].(string); ok {
					imsg = strings.TrimSpace(m)
				}
			}

			if strings.Contains(imsg, "File was deleted") {
				lineSource := source
				lineAction := targetAction
				if strings.Contains(imsg, "Path1") {
					lineSource = models.SideRemote
					lineAction = "Will be deleted from local disk"
				} else if strings.Contains(imsg, "Path2") {
					lineSource = models.SideLocal
					lineAction = "Will be deleted from remote"
				}

				parts := strings.Split(imsg, "File was deleted")
				for _, part := range parts[1:] {
					fp := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(part), "-:"))
					// Strip any trailing JSON delimiters if nested unmarshaling missed them
					if idx := strings.Index(fp, `","`); idx != -1 {
						fp = fp[:idx]
					}
					if idx := strings.Index(fp, `", "`); idx != -1 {
						fp = fp[:idx]
					}
					if idx := strings.Index(fp, `","source"`); idx != -1 {
						fp = fp[:idx]
					}
					if idx := strings.Index(fp, `", "source"`); idx != -1 {
						fp = fp[:idx]
					}
					fp = strings.Trim(fp, `"',{}\`)
					fp = strings.TrimSpace(fp)
					if fp != "" {
						cleanPath := filepath.Clean(fp)
						if cleanPath != "" && cleanPath != "." && !strings.Contains(cleanPath, ".review_backups") && !seen[cleanPath] {
							seen[cleanPath] = true
							candidates = append(candidates, buildDeletionCandidate(cleanPath, lineSource, lineAction, localRoot))
						}
					}
				}
			}

			// Boundary check: If we have extracted candidates from the current run and reach the run start banner, stop scanning further backwards
			if strings.Contains(imsg, "Synching Path1") ||
				strings.Contains(imsg, "Building Path1 and Path2 listings") ||
				strings.Contains(imsg, "Bisyncing with Comparison Settings") {
				if len(candidates) > 0 {
					return candidates
				}
			}
		}
	}

	return candidates
}

// buildDeletionCandidate constructs a DeletionCandidate with metadata if file exists on disk.
func buildDeletionCandidate(path string, source models.CandidateSide, targetAction, localRoot string) models.DeletionCandidate {
	cand := models.DeletionCandidate{
		ID:           hashID("del", path+":"+string(source)),
		Path:         path,
		Name:         filepath.Base(path),
		IsDir:        false,
		Source:       source,
		TargetAction: targetAction,
		Condition:    models.ConditionTooManyDeletes,
		Resolved:     false,
	}

	if localRoot != "" {
		fullPath := filepath.Join(localRoot, path)
		if fi, err := os.Stat(fullPath); err == nil {
			cand.IsDir = fi.IsDir()
			cand.Size = fi.Size()
			cand.ModTime = fi.ModTime().UTC().Format(time.RFC3339)
		}
	}

	return cand
}

// discoverConflicts extracts conflict candidates from logs and filesystem patterns.
func discoverConflicts(logLines []string, localRoot string) []models.ConflictCandidate {
	var candidates []models.ConflictCandidate
	seen := make(map[string]bool)

	// 1. Scan log lines for conflict markers
	for _, line := range logLines {
		clean := strings.TrimSpace(line)
		if clean == "" {
			continue
		}
		lower := strings.ToLower(clean)
		if strings.Contains(lower, "conflict") {
			path := extractConflictPath(clean)
			if path != "" && !seen[path] {
				seen[path] = true
				candidates = append(candidates, buildConflictCandidate(path, localRoot))
			}
		}
	}

	// 2. Scan local filesystem for conflict file artifacts (.conflict-, sync-conflict)
	if localRoot != "" {
		_ = filepath.Walk(localRoot, func(p string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			rel, relErr := filepath.Rel(localRoot, p)
			if relErr != nil || rel == "." {
				return nil
			}
			lower := strings.ToLower(info.Name())
			if strings.Contains(lower, ".conflict") || strings.Contains(lower, "sync-conflict") {
				base := normalizeConflictPath(rel)
				if !seen[base] {
					seen[base] = true
					candidates = append(candidates, buildConflictCandidate(base, localRoot))
				}
			}
			return nil
		})
	}

	return candidates
}

// extractConflictPath extracts candidate file path from error log lines.
func extractConflictPath(line string) string {
	idx := strings.Index(line, "Conflict: ")
	if idx != -1 {
		part := strings.TrimSpace(line[idx+len("Conflict: "):])
		fields := strings.Fields(part)
		if len(fields) > 0 {
			return filepath.Clean(strings.Trim(fields[0], `",'`))
		}
	}
	idx = strings.Index(line, "conflict on ")
	if idx != -1 {
		part := strings.TrimSpace(line[idx+len("conflict on "):])
		fields := strings.Fields(part)
		if len(fields) > 0 {
			return filepath.Clean(strings.Trim(fields[0], `",'`))
		}
	}
	return ""
}

// normalizeConflictPath extracts base file path from a conflict artifact filename.
func normalizeConflictPath(path string) string {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	if idx := strings.Index(base, ".conflict"); idx != -1 {
		ext := filepath.Ext(base)
		cleanedBase := base[:idx] + ext
		if dir == "." {
			return cleanedBase
		}
		return filepath.Join(dir, cleanedBase)
	}
	return path
}

// buildConflictCandidate constructs a ConflictCandidate with local filesystem metadata.
func buildConflictCandidate(path, localRoot string) models.ConflictCandidate {
	cand := models.ConflictCandidate{
		ID:        hashID("conf", path+":conflict"),
		Path:      path,
		Name:      filepath.Base(path),
		Condition: models.ConditionConflict,
		Resolved:  false,
	}

	if localRoot != "" {
		fullPath := filepath.Join(localRoot, path)
		if fi, err := os.Stat(fullPath); err == nil {
			cand.LocalSize = fi.Size()
			cand.LocalModTime = fi.ModTime().UTC().Format(time.RFC3339)
		}
	}

	return cand
}
