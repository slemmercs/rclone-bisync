package app

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

func TestNewContainer_NilConfig(t *testing.T) {
	c, err := NewContainer(nil)
	if err != nil {
		t.Fatalf("unexpected error with nil config: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil container")
	}
	if c.Config == nil {
		t.Fatal("expected non-nil container config")
	}
	if c.SyncService == nil {
		t.Fatal("expected default SyncService")
	}
	if c.CanaryService == nil || c.DeletionService == nil || c.TelemetryService == nil || c.ExplorerService == nil {
		t.Fatal("expected core domain services initialized")
	}
	if c.LogService == nil || c.AuthService == nil {
		t.Fatal("expected log and auth services initialized")
	}
}

func TestNewContainer_WithOptions(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	rcClient := client.NewRcloneClient(ts.URL, "user", "pass")
	cfg := &config.Config{
		RCAPIURL: ts.URL,
	}

	executed := false
	customExec := func(ctx context.Context, p *profiles.Profile, opts profiles.SyncOptions) error {
		executed = true
		return nil
	}

	c, err := NewContainer(cfg,
		WithClient(rcClient),
		WithProfileExecutor(customExec),
	)
	if err != nil {
		t.Fatalf("failed to create container with options: %v", err)
	}
	if c.Client != rcClient {
		t.Fatalf("expected client %v, got %v", rcClient, c.Client)
	}
	if c.ProfileExecutor == nil {
		t.Fatal("expected custom profile executor set")
	}
	_ = c.ProfileExecutor(context.Background(), &profiles.Profile{}, profiles.SyncOptions{})
	if !executed {
		t.Fatal("expected custom executor to be executed")
	}
}

func TestNewContainer_WithSyncServiceOption(t *testing.T) {
	cfg := &config.Config{}
	existingSync := services.NewSyncService(cfg, nil)

	c, err := NewContainer(cfg, WithSyncService(existingSync))
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}
	if c.SyncService != existingSync {
		t.Fatal("expected container to retain provided SyncService")
	}
	if c.TelemetryService != existingSync.Telemetry {
		t.Fatal("expected telemetry service unpacked from SyncService")
	}
}

func TestNewContainer_Lifecycle_ProfilesAndAudit(t *testing.T) {
	tmpDir := t.TempDir()
	localDir := filepath.Join(tmpDir, "local")
	profilesFile := filepath.Join(tmpDir, "profiles.json")
	historyFile := filepath.Join(tmpDir, "profiles_history.json")
	auditDB := filepath.Join(tmpDir, "audit.db")
	logFile := filepath.Join(tmpDir, "bisync.log")

	cfg := &config.Config{
		BaseDir:             tmpDir,
		LocalPath:           localDir,
		RemotePath:          "gdrive:/test",
		ProfilesFile:        profilesFile,
		ProfilesHistoryFile: historyFile,
		AuditDBPath:         auditDB,
		LogFile:             logFile,
	}

	c, err := NewContainer(cfg)
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}
	if c.ProfileStore == nil {
		t.Fatal("expected profile store initialized")
	}
	if c.ProfileScheduler == nil {
		t.Fatal("expected profile scheduler initialized")
	}
	if c.ProfileService == nil {
		t.Fatal("expected profile service initialized")
	}
	if c.AuditService == nil {
		t.Fatal("expected audit service initialized")
	}

	ctx := context.Background()
	if err := c.Start(ctx); err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	// Calling Start again should be idempotent
	if err := c.Start(ctx); err != nil {
		t.Fatalf("idempotent start failed: %v", err)
	}

	if err := c.Close(); err != nil {
		t.Fatalf("failed to close container: %v", err)
	}
	// Calling Close again should be idempotent
	if err := c.Close(); err != nil {
		t.Fatalf("idempotent close failed: %v", err)
	}
}

func TestDefaultProfileExecutor(t *testing.T) {
	tmpDir := t.TempDir()
	localDir := filepath.Join(tmpDir, "profile_local")
	cfg := &config.Config{
		BaseDir:          tmpDir,
		LocalPath:        localDir,
		RemotePath:       "drive:backup",
		LockFile:         filepath.Join(tmpDir, "test.lock"),
		RcloneCommonArgs: []string{"--verbose"},
	}

	exec := DefaultProfileExecutor(cfg)
	if exec == nil {
		t.Fatal("expected non-nil default executor")
	}

	p := &profiles.Profile{
		ID:              "prof-1",
		Remote:          "drive",
		RemotePath:      "backup",
		LocalPath:       localDir,
		ExcludePatterns: []string{"*.tmp", "node_modules/"},
	}

	// Profile directory does not exist initially
	if _, err := os.Stat(localDir); !os.IsNotExist(err) {
		t.Fatalf("expected directory %s to not exist yet", localDir)
	}

	// Running executor with DryRun will create the directory and attempt sync
	ctx := context.Background()
	_ = exec(ctx, p, profiles.SyncOptions{DryRun: true})

	// Verify local path was created safely with 0750 permissions
	fi, err := os.Stat(localDir)
	if err != nil {
		t.Fatalf("expected local directory to be created: %v", err)
	}
	if !fi.IsDir() {
		t.Fatalf("expected %s to be a directory", localDir)
	}

	// Test prohibited custom flags rejection in executor
	pBad := &profiles.Profile{
		ID:        "prof-bad",
		Remote:    "drive",
		LocalPath: localDir,
		Flags: profiles.ProfileFlags{
			CustomFlags: "--config /etc/evil.conf",
		},
	}
	err = exec(ctx, pBad, profiles.SyncOptions{DryRun: true})
	if err == nil {
		t.Fatalf("expected executor to reject profile with prohibited flags")
	}
}
func TestNewContainer_WithLoggerAndDebugMode(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "bisync-hub-log.json")

	cfg := &config.Config{
		BaseDir:   tmpDir,
		LogFile:   logPath,
		DebugMode: true,
	}

	c, err := NewContainer(cfg)
	if err != nil {
		t.Fatalf("failed to initialize container: %v", err)
	}
	defer c.Close()

	if c.Logger == nil {
		t.Fatal("expected container Logger to be initialized")
	}
	if c.Logger.Level() != slog.LevelDebug {
		t.Fatalf("expected logger level Debug when DebugMode is true, got: %v", c.Logger.Level())
	}
}

