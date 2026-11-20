package e2e

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
	"github.com/slemmercs/rclone-bisync/internal/server"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

// TestCaptureSandboxScreenshots launches headless Chrome against an ephemeral sandbox
// backend to generate clean, high-resolution screenshots of the Vue Cockpit SPA for documentation.
func TestCaptureSandboxScreenshots(t *testing.T) {
	if os.Getenv("CAPTURE_SCREENSHOTS") != "1" {
		t.Skip("skipping sandbox screenshot capture: set CAPTURE_SCREENSHOTS=1 to run")
	}

	chromePath := findChromeBinary()
	if chromePath == "" {
		t.Skip("skipping sandbox screenshot capture: Chrome/Chromium binary not found in PATH")
	}

	repoRoot := findRepoRoot(t)
	screenshotsDir := filepath.Join(repoRoot, "site", "assets", "images", "screenshots")
	_ = os.MkdirAll(screenshotsDir, 0750)

	tempDir := t.TempDir()
	localDir := filepath.Join(tempDir, "local")
	_ = os.MkdirAll(filepath.Join(localDir, "documents"), 0750)
	_ = os.MkdirAll(filepath.Join(localDir, "projects"), 0750)
	_ = os.WriteFile(filepath.Join(localDir, "notes.txt"), []byte("Bisync control plane design specs\n"), 0640)
	_ = os.WriteFile(filepath.Join(localDir, "projects", "specs_v2.md"), []byte("# Control Plane v2 Specifications\n"), 0640)
	_ = os.WriteFile(filepath.Join(localDir, "documents", "q4_budget.xlsx"), []byte("BINARY_EXCEL_SHEET\n"), 0640)
	_ = os.WriteFile(filepath.Join(localDir, "RCLONE_TEST"), []byte("canary-token"), 0640)

	mockRCServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/config/listremotes":
			_, _ = w.Write([]byte(`{"remotes":["google-drive","backblaze-b2","aws-s3-backup","onedrive-team"]}`))
		case "/core/stats":
			_, _ = w.Write([]byte(`{
				"bytes": 1459200000,
				"speed": 15480000,
				"checks": 1420,
				"deletes": 0,
				"transfers": 4,
				"transferring": [
					{"name": "projects/specs_v2.md", "size": 42000, "bytes": 38000, "speed": 1024000, "percentage": 90},
					{"name": "documents/q4_budget.xlsx", "size": 184000, "bytes": 128000, "speed": 2048000, "percentage": 69}
				]
			}`))
		case "/core/bwlimit":
			_, _ = w.Write([]byte(`{"rate":"off","bytesPerSecond":0}`))
		case "/operations/about":
			_, _ = w.Write([]byte(`{"total":2199023255552,"used":751619276800,"free":1447403978752}`))
		case "/operations/list":
			_, _ = w.Write([]byte(`{
				"list": [
					{"Path":"notes.txt","Name":"notes.txt","Size":34,"MimeType":"text/plain","ModTime":"2026-11-19T10:00:00Z","IsDir":false},
					{"Path":"documents","Name":"documents","Size":-1,"MimeType":"inode/directory","ModTime":"2026-11-19T09:30:00Z","IsDir":true},
					{"Path":"projects","Name":"projects","Size":-1,"MimeType":"inode/directory","ModTime":"2026-11-19T09:45:00Z","IsDir":true},
					{"Path":"projects/specs_v2.md","Name":"specs_v2.md","Size":42000,"MimeType":"text/markdown","ModTime":"2026-11-19T10:15:00Z","IsDir":false},
					{"Path":"RCLONE_TEST","Name":"RCLONE_TEST","Size":12,"MimeType":"text/plain","ModTime":"2026-11-19T08:00:00Z","IsDir":false}
				]
			}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer mockRCServer.Close()

	cfg := &config.Config{
		LocalPath:           localDir,
		RemotePath:          "google-drive:sync",
		BaseDir:             tempDir,
		LogFile:             filepath.Join(tempDir, "bisync.log"),
		ProfilesFile:        filepath.Join(tempDir, "profiles.json"),
		ProfilesHistoryFile: filepath.Join(tempDir, "profiles_history.json"),
		ActiveReviewFile:    filepath.Join(tempDir, "active_review_session.json"),
		MetricsFile:         filepath.Join(tempDir, "metrics.json"),
		AuditDBPath:         filepath.Join(tempDir, "audit.db"),
		RCAPIURL:            mockRCServer.URL,
	}

	daemonLog := filepath.Join(tempDir, "rclone-daemon.log")
	nowStr := time.Now().UTC().Format(time.RFC3339)
	daemonContent := `{"time":"` + nowStr + `","level":"info","msg":"rclone daemon service initialized on port 5572"}` + "\n" +
		`{"time":"` + nowStr + `","level":"notice","run_id":"rev-sandbox-gallery","msg":"Bisync dry-run analysis started for prof-eng-code"}` + "\n" +
		`{"time":"` + nowStr + `","level":"error","run_id":"rev-sandbox-gallery","msg":"Safety abort: deletion quota exceeded (SAFETY_MAX_DELETE_COUNT=25, candidate_deletions=32)"}` + "\n" +
		`{"time":"` + nowStr + `","level":"notice","run_id":"rev-sandbox-gallery","msg":"Interactive sync review session rev-sandbox-gallery created; state persisted"}` + "\n" +
		`{"time":"` + nowStr + `","level":"info","msg":"systemd journal listener attached: rclone-bisync.service active"}` + "\n"
	_ = os.WriteFile(daemonLog, []byte(daemonContent), 0640)
	_ = os.WriteFile(cfg.LogFile, []byte(daemonContent), 0640)

	reviewJSON := `{
		"id": "rev-sandbox-gallery",
		"created_at": "2026-11-19T10:30:00Z",
		"updated_at": "2026-11-19T10:30:00Z",
		"status": "REVIEWING",
		"origin_error": "Divergent content and safety threshold flagged for review",
		"message": "Interactive sync review: 1 conflict candidate and 1 deletion candidate staged for operator approval",
		"conflicts": [
			{
				"id": "conf-spec-1",
				"path": "projects/specs_v2.md",
				"name": "specs_v2.md",
				"local_size": 4200,
				"local_mod_time": "2026-11-19T10:15:00Z",
				"remote_size": 3800,
				"remote_mod_time": "2026-11-19T09:45:00Z",
				"condition": "MODIFIED_BOTH",
				"resolved": false
			}
		],
		"deletions": [
			{
				"id": "del-arch-1",
				"path": "legacy/archive_2024.tar.gz",
				"name": "archive_2024.tar.gz",
				"is_dir": false,
				"source": "remote",
				"target_action": "delete-remote",
				"size": 1420500,
				"resolved": false
			}
		],
		"actions": []
	}`
	_ = os.WriteFile(cfg.ActiveReviewFile, []byte(reviewJSON), 0600)

	syncSvc := services.NewSyncService(cfg, nil)
	srv := server.NewServer(cfg, syncSvc, os.DirFS(repoRoot))
	srv.Version = "v2.0.0-vue3"

	// Seed SQLite Audit Ledger with cryptographically chained records
	if db, err := audit.OpenDB(cfg.AuditDBPath); err == nil && db != nil {
		auditSvc := services.NewAuditService(db, nil)
		auditSvc.DBPath = cfg.AuditDBPath
		srv.AuditService = auditSvc

		_, _ = auditSvc.RecordEvent("prof-prod-docs", audit.ActionSyncExecuted, "scheduler", map[string]any{
			"status":            "success",
			"transferred_bytes": 1459200,
			"files_changed":     12,
			"duration_ms":       1420,
		})
		_, _ = auditSvc.RecordEvent("prof-eng-code", audit.ActionConflictResolved, "operator", map[string]any{
			"file":              "projects/specs_v2.md",
			"decision":          "keep-local",
			"heuristic_applied": true,
			"confidence":        0.88,
		})
		_, _ = auditSvc.RecordEvent("prof-media-photos", audit.ActionDedupeExecuted, "operator", map[string]any{
			"mode":               "hash",
			"duplicates_removed": 3,
			"freed_bytes":        8420000,
		})
		_, _ = auditSvc.RecordEvent("prof-prod-docs", audit.ActionProfileUpdated, "admin", map[string]any{
			"setting": "compare_mode",
			"old_val": "size,modtime",
			"new_val": "size,modtime,checksum",
		})
	}

	// Seed Profiles
	if srv.ProfileService != nil {
		_, _ = srv.ProfileService.Create(context.Background(), &profiles.Profile{
			ID:         "prof-prod-docs",
			Name:       "Production Documents",
			Remote:     "google-drive",
			RemotePath: "corporate_drive/documents",
			LocalPath:  cfg.LocalPath,
			Schedule:   "15m",
			Enabled:    true,
			State: profiles.OperationalState{
				Status: profiles.StatusHealthy,
			},
		})
		_, _ = srv.ProfileService.Create(context.Background(), &profiles.Profile{
			ID:         "prof-eng-code",
			Name:       "Engineering Codebase",
			Remote:     "aws-s3-backup",
			RemotePath: "dev-workspaces/bisync",
			LocalPath:  filepath.Join(cfg.LocalPath, "projects"),
			Schedule:   "1h",
			Enabled:    true,
			State: profiles.OperationalState{
				Status: profiles.ProfileStatus("review_required"),
			},
		})
		_, _ = srv.ProfileService.Create(context.Background(), &profiles.Profile{
			ID:         "prof-media-photos",
			Name:       "Media & Photography",
			Remote:     "backblaze-b2",
			RemotePath: "media_vault/2026",
			LocalPath:  cfg.LocalPath,
			Schedule:   "daily",
			Enabled:    true,
			State: profiles.OperationalState{
				Status: profiles.StatusHealthy,
			},
		})
	}

	ts := httptest.NewServer(srv)
	defer ts.Close()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromePath),
		chromedp.DisableGPU,
		chromedp.NoSandbox,
		chromedp.Headless,
		chromedp.WindowSize(1440, 900),
		chromedp.Flag("force-device-scale-factor", "2"),
		chromedp.Flag("high-dpi-support", "1"),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-software-rasterizer", true),
		chromedp.Flag("proxy-server", "direct://"),
		chromedp.Flag("proxy-bypass-list", "*"),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	// Navigate to Cockpit SPA
	appURL := ts.URL + "/cockpit"
	err := chromedp.Do(ctx,
		chromedp.Navigate(appURL),
		chromedp.WaitReady(`header`),
	)
	if err != nil {
		t.Fatalf("failed to load Cockpit SPA: %v", err)
	}

	// Disable animations/transitions to prevent mid-transition visual artifacts
	disableAnimScript := `(() => {
		const style = document.createElement('style');
		style.textContent = '*, *::before, *::after { animation-duration: 0s !important; animation-delay: 0s !important; transition-duration: 0s !important; transition-delay: 0s !important; }';
		document.head.appendChild(style);
		return true;
	})()`
	_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](disableAnimScript))

	saveShot := func(filename string) {
		t.Helper()
		buf, err := chromedp.Run(ctx, chromedp.CaptureScreenshot())
		if err != nil {
			t.Fatalf("failed capturing %s: %v", filename, err)
		}
		targetPath := filepath.Join(screenshotsDir, filename)
		if err := os.WriteFile(targetPath, buf, 0600); err != nil {
			t.Fatalf("failed writing %s: %v", targetPath, err)
		}
		t.Logf("✓ Captured screenshot: %s (%d bytes)", filename, len(buf))
	}

	// 1. Fleet Overview (and Review Barrier Active in Fleet)
	t.Run("CaptureFleetOverview", func(t *testing.T) {
		time.Sleep(500 * time.Millisecond)
		saveShot("bisync-hub_fleet-overview.png")
		saveShot("bisync-hub_sync-review-barrier-active.png")
	})

	// 2. Storage Remotes View & Remote Creation Wizard
	t.Run("CaptureStorageRemotes", func(t *testing.T) {
		switchTab(ctx, t, "Storage Remotes")
		time.Sleep(600 * time.Millisecond)
		saveShot("bisync-hub_storage-remotes.png")

		// Open Remote Creation Wizard
		openWizScript := `(() => {
			const btns = Array.from(document.querySelectorAll('button'));
			const btn = btns.find(b => b.textContent && b.textContent.includes('Add Remote'));
			if (btn) { btn.click(); return true; }
			return false;
		})()`
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](openWizScript))
		time.Sleep(600 * time.Millisecond)
		saveShot("bisync-hub_google-drive-oauth-configuration.png")
		saveShot("bisync-hub_microsoft-onedrive-oauth-configuration.png")
		dismissModal(ctx)
	})

	// 3. File Explorer View
	t.Run("CaptureFileExplorer", func(t *testing.T) {
		switchTab(ctx, t, "File Explorer")
		time.Sleep(600 * time.Millisecond)
		saveShot("bisync-hub_file-explorer.png")
	})

	// 4. Audit Trail View & Forensic Block Detail
	t.Run("CaptureAuditLedger", func(t *testing.T) {
		switchTab(ctx, t, "Audit Trail")
		time.Sleep(600 * time.Millisecond)
		saveShot("bisync-hub_audit-ledger.png")
		saveShot("bisync-hub_unified-ledger.png")

		// Click "Inspect" button on first audit event row
		inspectScript := `(() => {
			const btns = Array.from(document.querySelectorAll('button'));
			const btn = btns.find(b => b.textContent && b.textContent.trim() === 'Inspect');
			if (btn) { btn.click(); return true; }
			return false;
		})()`
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](inspectScript))
		time.Sleep(600 * time.Millisecond)
		saveShot("bisync-hub_audit-block.png")
		dismissModal(ctx)
	})

	// 5. Engine Telemetry View & Correlated Daemon Logs
	t.Run("CaptureEngineTelemetry", func(t *testing.T) {
		switchTab(ctx, t, "Engine & Telemetry")
		time.Sleep(600 * time.Millisecond)
		saveShot("bisync-hub_engine-telemetry.png")

		// Click "Daemon Logs" button
		daemonScript := `(() => {
			const btns = Array.from(document.querySelectorAll('button'));
			const btn = btns.find(b => b.textContent && b.textContent.includes('Daemon Logs'));
			if (btn) { btn.click(); return true; }
			return false;
		})()`
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](daemonScript))
		time.Sleep(600 * time.Millisecond)
		saveShot("bisync-hub_correlated-daemon-log.png")
		saveShot("bisync-hub_bisync-log-panel.png")
		dismissModal(ctx)
	})

	// 6. Sync Review & Conflict Resolution Modal
	t.Run("CaptureSyncReviewModal", func(t *testing.T) {
		switchTab(ctx, t, "Fleet")
		time.Sleep(400 * time.Millisecond)

		openScript := `(() => {
			if (typeof window.openReviewModal === 'function') {
				window.openReviewModal();
				return true;
			}
			window.dispatchEvent(new KeyboardEvent('keydown', { key: 'r', altKey: true, bubbles: true }));
			return true;
		})()`
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](openScript))
		time.Sleep(600 * time.Millisecond)

		saveShot("bisync-hub_sync-review-resolution.png")
		saveShot("bisync-hub_sync-review-starting.png")
		dismissModal(ctx)
	})

	// 7. Profile Settings Modal
	t.Run("CaptureProfileSettingsModal", func(t *testing.T) {
		openProfScript := `(() => {
			const btns = Array.from(document.querySelectorAll('button'));
			const btn = btns.find(b => b.title && b.title.includes('Profiles Manager'));
			if (btn) { btn.click(); return true; }
			return false;
		})()`
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](openProfScript))
		time.Sleep(600 * time.Millisecond)

		saveShot("bisync-hub_profile-settings.png")
		dismissModal(ctx)
	})
}

func switchTab(ctx context.Context, t *testing.T, tabLabel string) {
	t.Helper()
	script := `(() => {
		const buttons = Array.from(document.querySelectorAll('header nav button'));
		const btn = buttons.find(b => b.textContent.includes('` + tabLabel + `'));
		if (btn) { btn.click(); return true; }
		return false;
	})()`
	clicked, err := chromedp.Run(ctx, chromedp.Evaluate[bool](script))
	if err != nil || !clicked {
		t.Fatalf("failed switching to tab '%s': %v", tabLabel, err)
	}
}

func dismissModal(ctx context.Context) {
	_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
		window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
		return true;
	})()`))
	time.Sleep(300 * time.Millisecond)
}
