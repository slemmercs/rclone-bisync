package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
)

func setupTestServerWithQueue(t *testing.T) (*Server, *profiles.Service, *profiles.Scheduler, *profiles.Store) {
	t.Helper()
	tmpDir := t.TempDir()
	profPath := filepath.Join(tmpDir, "profiles.json")
	histPath := filepath.Join(tmpDir, "profiles_history.json")

	store, err := profiles.NewStore(profPath, histPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	executor := func(ctx context.Context, p *profiles.Profile, opts profiles.SyncOptions) error {
		return nil
	}

	sched := profiles.NewScheduler(store, executor, profiles.WithMaxConcurrency(2))
	_ = sched.Start(context.Background())
	t.Cleanup(func() {
		sched.Stop()
	})

	svc := profiles.NewService(store, nil, sched)

	cfg := &config.Config{
		BaseDir:             tmpDir,
		ProfilesFile:        profPath,
		ProfilesHistoryFile: histPath,
		LogFile:             filepath.Join(tmpDir, "bisync.log"),
	}

	s := NewServer(cfg, nil, nil)
	s.ProfileService = svc
	return s, svc, sched, store
}

func doQueueReq(s *Server, method, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Dashboard-Request", "1")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	return w
}

func TestQueue_Snapshot_And_Lifecycle(t *testing.T) {
	s, _, sched, store := setupTestServerWithQueue(t)

	p1 := &profiles.Profile{
		ID:        "prof-q-1",
		Name:      "Queue Profile 1",
		Remote:    "remote1",
		LocalPath: "/tmp",
		Enabled:   true,
		Schedule:  "manual",
	}
	p2 := &profiles.Profile{
		ID:        "prof-q-2",
		Name:      "Queue Profile 2",
		Remote:    "remote2",
		LocalPath: "/tmp",
		Enabled:   true,
		Schedule:  "manual",
	}
	_ = store.Save(p1, "Init 1")
	_ = store.Save(p2, "Init 2")
	sched.RestartProfile(p1)
	sched.RestartProfile(p2)

	// 1. Initial snapshot: empty queue
	w := doQueueReq(s, "GET", "/api/queue")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var snapResp models.QueueSnapshotResponse
	if err := json.Unmarshal(w.Body.Bytes(), &snapResp); err != nil {
		t.Fatalf("failed to parse queue snapshot response: %v", err)
	}
	if snapResp.Status != "success" || snapResp.Snapshot.Paused || snapResp.Snapshot.QueuedCount != 0 {
		t.Fatalf("unexpected snapshot: %+v", snapResp)
	}

	// 2. Pause queue admissions
	pauseW := doQueueReq(s, "POST", "/api/queue/pause")
	if pauseW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", pauseW.Code, pauseW.Body.String())
	}

	// Verify snapshot reports paused
	w = doQueueReq(s, "GET", "/api/queue")
	_ = json.Unmarshal(w.Body.Bytes(), &snapResp)
	if !snapResp.Snapshot.Paused {
		t.Fatalf("expected snapshot Paused to be true")
	}

	// 3. Enqueue profile while paused
	go func() {
		_ = sched.TriggerSync(context.Background(), "prof-q-1", false)
	}()
	time.Sleep(50 * time.Millisecond)

	// Check GET /api/queue has 1 queued job
	w = doQueueReq(s, "GET", "/api/queue")
	_ = json.Unmarshal(w.Body.Bytes(), &snapResp)
	if snapResp.Snapshot.QueuedCount != 1 || len(snapResp.Snapshot.QueuedJobs) != 1 {
		t.Fatalf("expected 1 queued job, got %+v", snapResp.Snapshot)
	}
	if snapResp.Snapshot.QueuedJobs[0].ProfileName != "Queue Profile 1" {
		t.Fatalf("expected profile name 'Queue Profile 1', got %s", snapResp.Snapshot.QueuedJobs[0].ProfileName)
	}

	// 4. Cancel single queued job DELETE /api/queue/prof-q-1
	delW := doQueueReq(s, "DELETE", "/api/queue/prof-q-1")
	if delW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", delW.Code, delW.Body.String())
	}

	// Check queue is empty
	w = doQueueReq(s, "GET", "/api/queue")
	_ = json.Unmarshal(w.Body.Bytes(), &snapResp)
	if snapResp.Snapshot.QueuedCount != 0 {
		t.Fatalf("expected 0 queued jobs after delete, got %d", snapResp.Snapshot.QueuedCount)
	}

	// 5. Cancel non-existent job returns 404
	delUnknownW := doQueueReq(s, "DELETE", "/api/queue/prof-q-unknown")
	if delUnknownW.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown job, got %d: %s", delUnknownW.Code, delUnknownW.Body.String())
	}

	// 6. Bulk clear DELETE /api/queue
	go func() { _ = sched.TriggerSync(context.Background(), "prof-q-1", false) }()
	go func() { _ = sched.TriggerSync(context.Background(), "prof-q-2", false) }()
	time.Sleep(50 * time.Millisecond)

	clearW := doQueueReq(s, "DELETE", "/api/queue")
	if clearW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", clearW.Code, clearW.Body.String())
	}

	var clearResp models.QueueActionResponse
	if err := json.Unmarshal(clearW.Body.Bytes(), &clearResp); err != nil {
		t.Fatalf("failed to decode clear response: %v", err)
	}
	if clearResp.Count != 2 {
		t.Fatalf("expected 2 cleared jobs, got %d", clearResp.Count)
	}

	// 7. Resume queue POST /api/queue/resume
	resW := doQueueReq(s, "POST", "/api/queue/resume")
	if resW.Code != http.StatusOK {
		t.Fatalf("expected 200 on resume, got %d", resW.Code)
	}

	w = doQueueReq(s, "GET", "/api/queue")
	_ = json.Unmarshal(w.Body.Bytes(), &snapResp)
	if snapResp.Snapshot.Paused {
		t.Fatalf("expected paused to be false after resume")
	}
}

func TestQueue_ServiceUnavailable(t *testing.T) {
	s := NewServer(&config.Config{}, nil, nil)
	s.ProfileService = nil

	// GET /api/queue returns 200 with empty snapshot even when ProfileService is nil
	w := doQueueReq(s, "GET", "/api/queue")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/queue: expected 200, got %d", w.Code)
	}

	mutationEndpoints := []struct {
		method string
		target string
	}{
		{"POST", "/api/queue/pause"},
		{"POST", "/api/queue/resume"},
		{"DELETE", "/api/queue/prof-1"},
		{"DELETE", "/api/queue"},
	}

	for _, ep := range mutationEndpoints {
		w := doQueueReq(s, ep.method, ep.target)
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("%s %s: expected 503, got %d", ep.method, ep.target, w.Code)
		}
	}
}
