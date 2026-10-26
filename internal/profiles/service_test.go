package profiles

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/audit"
)

type mockRemoteChecker struct {
	remotes []string
	err     error
}

func (m *mockRemoteChecker) ListRemotes(ctx context.Context) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.remotes, nil
}

type mockAuditRecorder struct {
	events []struct {
		profileID string
		action    audit.Action
	}
}

func (m *mockAuditRecorder) RecordEvent(profileID string, action audit.Action, actor string, details map[string]any) (*audit.AuditRecord, error) {
	m.events = append(m.events, struct {
		profileID string
		action    audit.Action
	}{profileID, action})
	return &audit.AuditRecord{ProfileID: profileID, Action: action}, nil
}

func TestProfileService_CRUDAndAudit(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(filepath.Join(tmpDir, "p.json"), filepath.Join(tmpDir, "h.json"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	remotes := &mockRemoteChecker{remotes: []string{"gdrive:", "onedrive:"}}
	auditRec := &mockAuditRecorder{}
	sched := NewScheduler(store, nil)
	_ = sched.Start(context.Background())
	defer sched.Stop()

	svc := NewService(store, remotes, auditRec, sched)

	// Test List before create
	emptyList := svc.List(false)
	if len(emptyList) != 0 {
		t.Fatalf("expected empty list, got %d", len(emptyList))
	}

	ctx := context.Background()

	// 1. Remote list
	rList, err := svc.ListRemotes(ctx)
	if err != nil || len(rList) != 2 {
		t.Fatalf("expected 2 remotes, got %v (err: %v)", rList, err)
	}

	// 2. Create with invalid remote
	badRemoteProfile := &Profile{
		ID:        "p-invalid",
		Name:      "Invalid Remote",
		Remote:    "dropbox:",
		LocalPath: "/tmp",
	}
	if _, err := svc.Create(ctx, badRemoteProfile); err == nil {
		t.Fatal("expected error creating profile with unconfigured remote")
	}

	// 3. Create valid profile
	validProfile := &Profile{
		ID:        "work-sync",
		Name:      "Work Google Drive",
		Remote:    "gdrive",
		LocalPath: "/tmp/work",
		Enabled:   true,
	}
	created, err := svc.Create(ctx, validProfile)
	if err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}
	if created.ID != "work-sync" || created.Version != 1 {
		t.Fatalf("unexpected created profile: %+v", created)
	}

	// 4. Duplicate create error
	if _, err := svc.Create(ctx, validProfile); err != ErrProfileAlreadyExists {
		t.Fatalf("expected ErrProfileAlreadyExists, got %v", err)
	}

	// 5. Update
	created.Schedule = "30m"
	updated, err := svc.Update(ctx, created, "Interval change")
	if err != nil {
		t.Fatalf("failed to update profile: %v", err)
	}
	if updated.Version != 2 || updated.Schedule != "30m" {
		t.Fatalf("unexpected updated profile: %+v", updated)
	}

	// 6. Clone
	cloned, err := svc.Clone(ctx, "work-sync", "work-clone", "Cloned Work Sync")
	if err != nil {
		t.Fatalf("failed to clone profile: %v", err)
	}
	if cloned.ID != "work-clone" || cloned.Version != 1 || cloned.Name != "Cloned Work Sync" {
		t.Fatalf("unexpected cloned profile: %+v", cloned)
	}

	// Clone already exists error
	if _, err := svc.Clone(ctx, "work-sync", "work-clone", "Another Clone"); err != ErrProfileAlreadyExists {
		t.Fatalf("expected ErrProfileAlreadyExists, got: %v", err)
	}

	// 7. Rollback
	rolledBack, err := svc.Rollback(ctx, "work-sync", 1)
	if err != nil {
		t.Fatalf("failed to rollback profile: %v", err)
	}
	if rolledBack.Version != 3 || rolledBack.Schedule != "manual" {
		t.Fatalf("expected rolled back profile version 3 with manual schedule, got: %+v", rolledBack)
	}

	// 8. TriggerSync
	if err := svc.TriggerSync(ctx, "work-sync", false); err != nil {
		t.Fatalf("expected trigger sync success, got: %v", err)
	}

	// 9. Delete
	if err := svc.Delete(ctx, "work-clone"); err != nil {
		t.Fatalf("failed to delete cloned profile: %v", err)
	}
	if _, err := svc.Get("work-clone"); err != ErrProfileNotFound {
		t.Fatalf("expected ErrProfileNotFound for deleted profile, got %v", err)
	}

	// 10. Auto-generated ID and directory provisioning on Create
	nestedDir := filepath.Join(tmpDir, "auto_provision_dir", "sub")
	autoProfile := &Profile{
		Name:      "Auto Generated ID Profile",
		Remote:    "gdrive",
		LocalPath: nestedDir,
		Enabled:   true,
	}
	autoCreated, err := svc.Create(ctx, autoProfile)
	if err != nil {
		t.Fatalf("failed to create profile with auto-id: %v", err)
	}
	if !strings.HasPrefix(autoCreated.ID, "prof_") {
		t.Fatalf("expected auto-generated ID to start with prof_, got: %s", autoCreated.ID)
	}
	if fi, err := os.Stat(nestedDir); err != nil || !fi.IsDir() {
		t.Fatalf("expected directory %s to be auto-provisioned, err: %v", nestedDir, err)
	}

	// 11. Auto-generated ID on Clone
	clonedAuto, err := svc.Clone(ctx, autoCreated.ID, "", "Cloned Auto")
	if err != nil {
		t.Fatalf("failed to clone with empty target id: %v", err)
	}
	if !strings.HasPrefix(clonedAuto.ID, "prof_") || clonedAuto.ID == autoCreated.ID {
		t.Fatalf("expected unique auto-generated clone ID, got: %s", clonedAuto.ID)
	}

	// 12. Check audit logs
	if len(auditRec.events) < 5 {
		t.Fatalf("expected at least 5 audit events, got %d", len(auditRec.events))
	}
}
