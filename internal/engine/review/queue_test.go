package review

import (
	"context"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorClass
	}{
		{"nil", nil, ErrorClassNone},
		{"os.ErrNotExist", os.ErrNotExist, ErrorClassIdempotent},
		{"os.ErrExist", os.ErrExist, ErrorClassIdempotent},
		{"rclone 404", errors.New("HTTP 404: object not found"), ErrorClassIdempotent},
		{"already absent", errors.New("item already absent"), ErrorClassIdempotent},
		{"already exists", errors.New("directory already exists"), ErrorClassIdempotent},
		{"rate limit 429", errors.New("HTTP 429: too many requests"), ErrorClassRetryable},
		{"service unavailable 503", errors.New("503 Service Unavailable"), ErrorClassRetryable},
		{"gateway 502", errors.New("502 Bad Gateway"), ErrorClassRetryable},
		{"timeout", errors.New("i/o timeout"), ErrorClassRetryable},
		{"connection reset", errors.New("connection reset by peer"), ErrorClassRetryable},
		{"fatal permission denied", errors.New("permission denied"), ErrorClassFatal},
		{"fatal path traversal", errors.New("path escapes root jail"), ErrorClassFatal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyError(tt.err)
			if got != tt.expected {
				t.Errorf("ClassifyError(%v) = %v; want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestCalculateDepth(t *testing.T) {
	cases := map[string]int{
		"":              0,
		".":             0,
		"file.txt":      0,
		"dir/file.txt":  1,
		"a/b/c.txt":     2,
		"/a/b/c/d.txt/": 3,
	}

	for path, expected := range cases {
		got := CalculateDepth(path)
		if got != expected {
			t.Errorf("CalculateDepth(%q) = %d; want %d", path, got, expected)
		}
	}
}

func TestSortWorkItems(t *testing.T) {
	rawItems := []WorkItem{
		// Deletions
		{ID: "d1", Kind: WorkItemKindDeletion, Path: "root_dir", IsDir: true, Decision: models.DecisionAcceptDelete, Depth: 0},
		{ID: "d2", Kind: WorkItemKindDeletion, Path: "root_dir/sub/file.txt", IsDir: false, Decision: models.DecisionAcceptDelete, Depth: 2},
		{ID: "d3", Kind: WorkItemKindDeletion, Path: "root_dir/sub", IsDir: true, Decision: models.DecisionAcceptDelete, Depth: 1},

		// Restores
		{ID: "r1", Kind: WorkItemKindDeletion, Path: "restore_dir/sub/item.txt", IsDir: false, Decision: models.DecisionRestoreItem, Depth: 2},
		{ID: "r2", Kind: WorkItemKindDeletion, Path: "restore_dir", IsDir: true, Decision: models.DecisionRestoreItem, Depth: 0},
		{ID: "r3", Kind: WorkItemKindDeletion, Path: "restore_dir/sub", IsDir: true, Decision: models.DecisionRestoreItem, Depth: 1},

		// Conflict
		{ID: "c1", Kind: WorkItemKindConflict, Path: "conflict.txt", IsDir: false, Decision: models.DecisionKeepLocal, Depth: 0},
	}

	sorted := SortWorkItems(rawItems)

	expectedIDs := []string{"r2", "r3", "r1", "c1", "d2", "d3", "d1"}
	if len(sorted) != len(expectedIDs) {
		t.Fatalf("expected %d items, got %d", len(expectedIDs), len(sorted))
	}

	for i, expID := range expectedIDs {
		if sorted[i].ID != expID {
			t.Errorf("position %d: expected ID %s, got %s (Path: %s)", i, expID, sorted[i].ID, sorted[i].Path)
		}
	}
}

func TestExecuteWithRetry(t *testing.T) {
	cfg := QueueConfig{
		MaxRetries:     2,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     5 * time.Millisecond,
	}

	t.Run("ImmediateSuccess", func(t *testing.T) {
		calls := 0
		err := ExecuteWithRetry(context.Background(), cfg, func(ctx context.Context) error {
			calls++
			return nil
		})
		if err != nil || calls != 1 {
			t.Fatalf("expected nil err and 1 call, got err=%v, calls=%d", err, calls)
		}
	})

	t.Run("IdempotentTreatedAsSuccess", func(t *testing.T) {
		calls := 0
		err := ExecuteWithRetry(context.Background(), cfg, func(ctx context.Context) error {
			calls++
			return errors.New("HTTP 404: Not Found")
		})
		if err != nil || calls != 1 {
			t.Fatalf("expected idempotent to succeed without retry, got err=%v, calls=%d", err, calls)
		}
	})

	t.Run("RetryableSucceedsOnRetry", func(t *testing.T) {
		calls := 0
		err := ExecuteWithRetry(context.Background(), cfg, func(ctx context.Context) error {
			calls++
			if calls < 2 {
				return errors.New("HTTP 429: Too Many Requests")
			}
			return nil
		})
		if err != nil || calls != 2 {
			t.Fatalf("expected retry to succeed on 2nd attempt, got err=%v, calls=%d", err, calls)
		}
	})

	t.Run("FatalFailsImmediately", func(t *testing.T) {
		calls := 0
		fatalErr := errors.New("permission denied")
		err := ExecuteWithRetry(context.Background(), cfg, func(ctx context.Context) error {
			calls++
			return fatalErr
		})
		if !errors.Is(err, fatalErr) || calls != 1 {
			t.Fatalf("expected immediate failure without retries, got err=%v, calls=%d", err, calls)
		}
	})
}

func TestReviewQueue_Process(t *testing.T) {
	q := NewReviewQueue(&QueueConfig{
		MaxRetries:     1,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     2 * time.Millisecond,
	})

	items := []WorkItem{
		{ID: "item1", Path: "dir1", Decision: models.DecisionRestoreItem, IsDir: true, Depth: 0},
		{ID: "item2", Path: "dir1/fail.txt", Decision: models.DecisionRestoreItem, IsDir: false, Depth: 1},
		{ID: "item3", Path: "dir1/ok.txt", Decision: models.DecisionRestoreItem, IsDir: false, Depth: 1},
	}

	var executedOrder []string
	var callCount int32

	summary := q.Process(context.Background(), items, func(ctx context.Context, item WorkItem) error {
		atomic.AddInt32(&callCount, 1)
		executedOrder = append(executedOrder, item.ID)
		if item.ID == "item2" {
			return errors.New("disk full")
		}
		return nil
	})

	if summary.Total != 3 {
		t.Errorf("expected Total 3, got %d", summary.Total)
	}
	if summary.Succeeded != 2 {
		t.Errorf("expected Succeeded 2, got %d", summary.Succeeded)
	}
	if summary.Failed != 1 {
		t.Errorf("expected Failed 1, got %d", summary.Failed)
	}
	if !summary.HasErrors() {
		t.Errorf("expected HasErrors to be true")
	}
	if _, ok := summary.Errors["dir1/fail.txt"]; !ok {
		t.Errorf("expected error for dir1/fail.txt, got %v", summary.Errors)
	}

	if len(executedOrder) != 3 || executedOrder[0] != "item1" {
		t.Errorf("unexpected execution order: %v", executedOrder)
	}

	formatted := summary.FormatSummary()
	if formatted == "" {
		t.Errorf("expected formatted summary string")
	}
}
