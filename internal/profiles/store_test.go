package profiles

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestProfileModel_Validate(t *testing.T) {
	tests := []struct {
		name    string
		p       Profile
		wantErr error
	}{
		{
			name:    "empty id",
			p:       Profile{Name: "Test", Remote: "gdrive", LocalPath: "/tmp"},
			wantErr: ErrInvalidProfileID,
		},
		{
			name:    "invalid chars in id",
			p:       Profile{ID: "bad id!", Name: "Test", Remote: "gdrive", LocalPath: "/tmp"},
			wantErr: ErrInvalidProfileID,
		},
		{
			name:    "empty name",
			p:       Profile{ID: "p1", Name: " ", Remote: "gdrive", LocalPath: "/tmp"},
			wantErr: ErrInvalidName,
		},
		{
			name:    "empty remote",
			p:       Profile{ID: "p1", Name: "Test", Remote: " ", LocalPath: "/tmp"},
			wantErr: ErrInvalidRemote,
		},
		{
			name:    "empty local path",
			p:       Profile{ID: "p1", Name: "Test", Remote: "gdrive", LocalPath: " "},
			wantErr: ErrInvalidLocalPath,
		},
		{
			name: "valid profile with defaults applied",
			p: Profile{
				ID:        "p_1-valid",
				Name:      "Valid Workload",
				Remote:    "gdrive",
				LocalPath: "/tmp/data",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.p.Validate()
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.p.Schedule != "manual" {
					t.Errorf("expected default schedule manual, got %s", tt.p.Schedule)
				}
				if tt.p.ReviewMode != ReviewDisabled {
					t.Errorf("expected default review mode disabled, got %s", tt.p.ReviewMode)
				}
				if tt.p.Safety.FailureThreshold != 3 {
					t.Errorf("expected default failure threshold 3, got %d", tt.p.Safety.FailureThreshold)
				}
			}
		})
	}
}

func TestProfile_Helpers(t *testing.T) {
	p := Profile{
		ID:         "work-docs",
		Remote:     "workdrive:",
		RemotePath: "/Projects/2026",
	}
	if p.FullRemoteTarget() != "workdrive:Projects/2026" {
		t.Errorf("unexpected FullRemoteTarget: %s", p.FullRemoteTarget())
	}
	p.RemotePath = ""
	if p.FullRemoteTarget() != "workdrive:" {
		t.Errorf("unexpected FullRemoteTarget: %s", p.FullRemoteTarget())
	}
	if p.LockFilePath() != "/tmp/rclone-bisync-work-docs.lock" {
		t.Errorf("unexpected LockFilePath: %s", p.LockFilePath())
	}
}

func TestStore_LifecycleAndRollback(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "profiles.json")
	historyPath := filepath.Join(tmpDir, "history.json")

	// 1. Invalid empty path
	if _, err := NewStore("", ""); err == nil {
		t.Fatal("expected error with empty storePath")
	}

	store, err := NewStore(storePath, historyPath)
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// 2. Bootstrap default
	boot, err := store.BootstrapDefault("/home/user/docs", "gdrive:")
	if err != nil {
		t.Fatalf("failed to bootstrap default: %v", err)
	}
	if boot == nil || boot.ID != "default" {
		t.Fatalf("expected default profile, got: %+v", boot)
	}

	// Secondary bootstrap should be a no-op
	boot2, err := store.BootstrapDefault("/other", "other:")
	if err != nil || boot2 != nil {
		t.Fatalf("expected nil second bootstrap, got: %v, err: %v", boot2, err)
	}

	// 3. List
	list := store.List(false)
	if len(list) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(list))
	}

	// 4. Update Profile
	p := list[0]
	p.Schedule = "30m"
	p.Safety.DeleteThresholdPct = 15.0
	if err := store.Save(p, "Updated schedule and delete threshold"); err != nil {
		t.Fatalf("failed to save profile update: %v", err)
	}

	updated, err := store.Get("default")
	if err != nil {
		t.Fatalf("failed to get updated profile: %v", err)
	}
	if updated.Version != 2 || updated.Schedule != "30m" {
		t.Fatalf("expected version 2 and schedule 30m, got version %d schedule %s", updated.Version, updated.Schedule)
	}

	// 5. Update State
	now := time.Now().UTC()
	err = store.UpdateState("default", OperationalState{
		Status:              StatusRunning,
		LastSync:            &now,
		ConsecutiveFailures: 0,
	})
	if err != nil {
		t.Fatalf("failed to update state: %v", err)
	}
	stateP, _ := store.Get("default")
	if stateP.State.Status != StatusRunning {
		t.Fatalf("expected status running, got %s", stateP.State.Status)
	}

	// 6. Rollback to version 1
	hist, err := store.GetHistory("default")
	if err != nil || len(hist) != 2 {
		t.Fatalf("expected 2 history entries, got %d (err: %v)", len(hist), err)
	}

	rolledBack, err := store.Rollback("default", 1)
	if err != nil {
		t.Fatalf("failed to rollback: %v", err)
	}
	if rolledBack.Version != 3 || rolledBack.Schedule != "15m" {
		t.Fatalf("expected version 3 and restored schedule 15m, got version %d schedule %s", rolledBack.Version, rolledBack.Schedule)
	}

	// Non-existent version rollback
	if _, err := store.Rollback("default", 999); err != ErrVersionNotFound {
		t.Fatalf("expected ErrVersionNotFound, got %v", err)
	}

	// 7. Reload from disk
	storeReloaded, err := NewStore(storePath, historyPath)
	if err != nil {
		t.Fatalf("failed to reload store: %v", err)
	}
	reloadedP, err := storeReloaded.Get("default")
	if err != nil || reloadedP.Version != 3 {
		t.Fatalf("expected reloaded profile version 3, got %+v", reloadedP)
	}

	// 8. Delete
	if err := storeReloaded.Delete("default"); err != nil {
		t.Fatalf("failed to delete profile: %v", err)
	}
	if _, err := storeReloaded.Get("default"); err != ErrProfileNotFound {
		t.Fatalf("expected ErrProfileNotFound after deletion, got %v", err)
	}
	if err := storeReloaded.Delete("non-existent"); err != ErrProfileNotFound {
		t.Fatalf("expected ErrProfileNotFound for missing profile, got %v", err)
	}
}

func TestStore_List_DeterministicSorting(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "profiles.json")
	historyPath := filepath.Join(tmpDir, "history.json")

	store, err := NewStore(storePath, historyPath)
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	baseTime := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)

	// Save multiple profiles with different CreatedAt times in random order
	names := []string{"Delta", "Alpha", "Charlie", "Bravo"}
	for i, name := range names {
		p := &Profile{
			ID:        fmt.Sprintf("prof_%d", i),
			Name:      name,
			LocalPath: "/tmp/local",
			Remote:    "remote:",
			Safety: SafetySettings{
				DeleteThresholdPct: 10.0,
				FailureThreshold:   3,
			},
			CreatedAt: baseTime.Add(time.Duration(i) * time.Hour),
		}
		if err := store.Save(p, "Initial creation"); err != nil {
			t.Fatalf("failed to save profile %s: %v", name, err)
		}
	}

	// Repeatedly list and verify order is strictly deterministic (sorted by CreatedAt)
	for iter := 0; iter < 10; iter++ {
		list := store.List(false)
		if len(list) != 4 {
			t.Fatalf("expected 4 profiles, got %d", len(list))
		}
		for i, expectedID := range []string{"prof_0", "prof_1", "prof_2", "prof_3"} {
			if list[i].ID != expectedID {
				t.Fatalf("iter %d, index %d: expected %s, got %s", iter, i, expectedID, list[i].ID)
			}
		}
	}
}
