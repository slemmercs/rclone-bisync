package review

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// ErrorClass defines the operational severity and handling category for errors.
type ErrorClass string

const (
	ErrorClassNone       ErrorClass = "none"
	ErrorClassIdempotent ErrorClass = "idempotent"
	ErrorClassRetryable  ErrorClass = "retryable"
	ErrorClassFatal      ErrorClass = "fatal"
)

// ClassifyError inspects an error and categorizes it for resilience handling.
func ClassifyError(err error) ErrorClass {
	if err == nil {
		return ErrorClassNone
	}

	if os.IsNotExist(err) || os.IsExist(err) {
		return ErrorClassIdempotent
	}

	errStr := strings.ToLower(err.Error())

	// Idempotent errors: resource is already in the target state
	if strings.Contains(errStr, "404") ||
		strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "already absent") ||
		strings.Contains(errStr, "does not exist") ||
		strings.Contains(errStr, "409") ||
		strings.Contains(errStr, "already exists") ||
		strings.Contains(errStr, "already present") {
		return ErrorClassIdempotent
	}

	// Retryable transient errors
	if strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "too many requests") ||
		strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "service unavailable") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "bad gateway") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "connection refused") {
		return ErrorClassRetryable
	}

	return ErrorClassFatal
}

// WorkItemKind distinguishes candidate origins in the queue.
type WorkItemKind string

const (
	WorkItemKindDeletion WorkItemKind = "deletion"
	WorkItemKindConflict WorkItemKind = "conflict"
)

// WorkItem wraps a candidate ready for dependency-ordered execution.
type WorkItem struct {
	ID       string
	Kind     WorkItemKind
	Path     string
	IsDir    bool
	Decision models.ActionDecision
	Source   models.CandidateSide
	Depth    int
	Deletion *models.DeletionCandidate
	Conflict *models.ConflictCandidate
}

// CalculateDepth computes the hierarchical path nesting level.
func CalculateDepth(path string) int {
	clean := filepath.ToSlash(filepath.Clean(path))
	clean = strings.Trim(clean, "/")
	if clean == "" || clean == "." {
		return 0
	}
	return strings.Count(clean, "/")
}

// BuildWorkItems constructs work items from review session candidates.
func BuildWorkItems(deletions []models.DeletionCandidate, conflicts []models.ConflictCandidate) []WorkItem {
	items := make([]WorkItem, 0, len(deletions)+len(conflicts))

	for i := range deletions {
		cand := &deletions[i]
		if !cand.Resolved || cand.Decision == "" || cand.Decision == models.DecisionSkip {
			continue
		}
		items = append(items, WorkItem{
			ID:       cand.ID,
			Kind:     WorkItemKindDeletion,
			Path:     cand.Path,
			IsDir:    cand.IsDir,
			Decision: cand.Decision,
			Source:   cand.Source,
			Depth:    CalculateDepth(cand.Path),
			Deletion: cand,
		})
	}

	for i := range conflicts {
		cand := &conflicts[i]
		if !cand.Resolved || cand.Decision == "" || cand.Decision == models.DecisionSkip {
			continue
		}
		items = append(items, WorkItem{
			ID:       cand.ID,
			Kind:     WorkItemKindConflict,
			Path:     cand.Path,
			IsDir:    false,
			Decision: cand.Decision,
			Depth:    CalculateDepth(cand.Path),
			Conflict: cand,
		})
	}

	return items
}

// SortWorkItems topological sorts items so directory prerequisites and leaf deletions are safely ordered.
// Execution order:
//  1. Restores: directories top-down (depth ascending) -> files top-down (depth ascending)
//  2. Conflicts: depth ascending
//  3. Deletions: files bottom-up (depth descending) -> directories bottom-up (depth descending)
func SortWorkItems(items []WorkItem) []WorkItem {
	var restoreDirs []WorkItem
	var restoreFiles []WorkItem
	var conflicts []WorkItem
	var deleteFiles []WorkItem
	var deleteDirs []WorkItem
	var others []WorkItem

	for _, item := range items {
		switch item.Decision {
		case models.DecisionRestoreItem:
			if item.IsDir {
				restoreDirs = append(restoreDirs, item)
			} else {
				restoreFiles = append(restoreFiles, item)
			}
		case models.DecisionAcceptDelete:
			if item.IsDir {
				deleteDirs = append(deleteDirs, item)
			} else {
				deleteFiles = append(deleteFiles, item)
			}
		case models.DecisionKeepLocal, models.DecisionKeepRemote, models.DecisionKeepBoth:
			conflicts = append(conflicts, item)
		default:
			others = append(others, item)
		}
	}

	// Restores: shallowest first (depth ascending)
	sort.SliceStable(restoreDirs, func(i, j int) bool {
		if restoreDirs[i].Depth != restoreDirs[j].Depth {
			return restoreDirs[i].Depth < restoreDirs[j].Depth
		}
		return restoreDirs[i].Path < restoreDirs[j].Path
	})
	sort.SliceStable(restoreFiles, func(i, j int) bool {
		if restoreFiles[i].Depth != restoreFiles[j].Depth {
			return restoreFiles[i].Depth < restoreFiles[j].Depth
		}
		return restoreFiles[i].Path < restoreFiles[j].Path
	})

	// Conflicts: shallowest first (depth ascending)
	sort.SliceStable(conflicts, func(i, j int) bool {
		if conflicts[i].Depth != conflicts[j].Depth {
			return conflicts[i].Depth < conflicts[j].Depth
		}
		return conflicts[i].Path < conflicts[j].Path
	})

	// Deletions: deepest first (depth descending) to delete children before parents
	sort.SliceStable(deleteFiles, func(i, j int) bool {
		if deleteFiles[i].Depth != deleteFiles[j].Depth {
			return deleteFiles[i].Depth > deleteFiles[j].Depth
		}
		return deleteFiles[i].Path < deleteFiles[j].Path
	})
	sort.SliceStable(deleteDirs, func(i, j int) bool {
		if deleteDirs[i].Depth != deleteDirs[j].Depth {
			return deleteDirs[i].Depth > deleteDirs[j].Depth
		}
		return deleteDirs[i].Path < deleteDirs[j].Path
	})

	result := make([]WorkItem, 0, len(items))
	result = append(result, restoreDirs...)
	result = append(result, restoreFiles...)
	result = append(result, conflicts...)
	result = append(result, deleteFiles...)
	result = append(result, deleteDirs...)
	result = append(result, others...)

	return result
}

// BatchSummary summarizes execution results of a work queue.
type BatchSummary struct {
	Total     int               `json:"total"`
	Succeeded int               `json:"succeeded"`
	Failed    int               `json:"failed"`
	Skipped   int               `json:"skipped"`
	Duration  time.Duration     `json:"duration"`
	Errors    map[string]string `json:"errors,omitempty"`
}

// HasErrors returns true if any item failed.
func (s *BatchSummary) HasErrors() bool {
	return s.Failed > 0
}

// FormatSummary produces a clean human-readable execution summary.
func (s BatchSummary) FormatSummary() string {
	if s.Failed == 0 {
		return fmt.Sprintf("Batch complete: %d/%d items succeeded in %v", s.Succeeded, s.Total, s.Duration.Round(time.Millisecond))
	}
	return fmt.Sprintf("Batch finished with errors: %d succeeded, %d failed out of %d in %v", s.Succeeded, s.Failed, s.Total, s.Duration.Round(time.Millisecond))
}

// QueueConfig tunes execution parameters and backoff.
type QueueConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

// DefaultQueueConfig returns safe default queue retry settings.
func DefaultQueueConfig() QueueConfig {
	return QueueConfig{
		MaxRetries:     3,
		InitialBackoff: 50 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
	}
}

// ExecuteWithRetry runs an operation with exponential backoff on retryable errors.
func ExecuteWithRetry(ctx context.Context, cfg QueueConfig, op func(ctx context.Context) error) error {
	var lastErr error
	backoff := cfg.InitialBackoff

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := op(ctx)
		if err == nil {
			return nil
		}

		class := ClassifyError(err)
		if class == ErrorClassIdempotent {
			return nil
		}

		if class != ErrorClassRetryable || attempt == cfg.MaxRetries {
			return err
		}

		lastErr = err

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > cfg.MaxBackoff {
			backoff = cfg.MaxBackoff
		}
	}

	if lastErr != nil {
		return lastErr
	}
	return errors.New("operation failed after retries")
}

// ReviewQueue coordinates topological staging and fault-tolerant execution.
type ReviewQueue struct {
	Config QueueConfig
}

// NewReviewQueue creates a review queue with given or default configuration.
func NewReviewQueue(cfg *QueueConfig) *ReviewQueue {
	if cfg == nil {
		defaultCfg := DefaultQueueConfig()
		cfg = &defaultCfg
	}
	return &ReviewQueue{Config: *cfg}
}

// Process executes all work items through the provided executor, isolating errors per candidate.
func (q *ReviewQueue) Process(
	ctx context.Context,
	items []WorkItem,
	executor func(ctx context.Context, item WorkItem) error,
) BatchSummary {
	startTime := time.Now()
	summary := BatchSummary{
		Total:  len(items),
		Errors: make(map[string]string),
	}

	ordered := SortWorkItems(items)

	for _, item := range ordered {
		if ctx.Err() != nil {
			summary.Failed++
			summary.Errors[item.Path] = ctx.Err().Error()
			continue
		}

		err := ExecuteWithRetry(ctx, q.Config, func(opCtx context.Context) error {
			return executor(opCtx, item)
		})

		if err != nil {
			summary.Failed++
			summary.Errors[item.Path] = err.Error()
		} else {
			summary.Succeeded++
		}
	}

	summary.Duration = time.Since(startTime)
	if len(summary.Errors) == 0 {
		summary.Errors = nil
	}
	return summary
}


