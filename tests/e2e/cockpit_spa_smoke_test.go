package e2e

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
	"github.com/slemmercs/rclone-bisync/internal/server"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

// TestCockpitSPA_HeadlessChromeSmoke launches headless Chrome to verify
// the Vue 3 + PrimeVue SPA served at /cockpit, including tab switching,
// Log Dock, Help Drawer, and interactive dialogs.
func TestCockpitSPA_HeadlessChromeSmoke(t *testing.T) {
	chromePath := findChromeBinary()
	if chromePath == "" {
		t.Skip("skipping SPA E2E test: Chrome/Chromium binary not found in PATH")
	}

	repoRoot := findRepoRoot(t)
	tempDir := t.TempDir()

	mockRCServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/config/listremotes":
			_, _ = w.Write([]byte(`{"remotes":["testremote"]}`))
		case "/core/stats":
			_, _ = w.Write([]byte(`{"bytes":0,"checks":0,"deletes":0,"transfers":0}`))
		case "/core/bwlimit":
			_, _ = w.Write([]byte(`{"rate":"off","bytesPerSecond":0}`))
		case "/operations/about":
			_, _ = w.Write([]byte(`{"total":1000000,"used":500000,"free":500000}`))
		case "/operations/list":
			_, _ = w.Write([]byte(`{"list":[{"Path":"notes.txt","Name":"notes.txt","Size":12,"MimeType":"text/plain","ModTime":"2026-11-18T12:00:00Z","IsDir":false},{"Path":"RCLONE_TEST","Name":"RCLONE_TEST","Size":14,"MimeType":"text/plain","ModTime":"2026-11-18T12:00:00Z","IsDir":false}]}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer mockRCServer.Close()

	cfg := &config.Config{
		LocalPath:           filepath.Join(tempDir, "local"),
		RemotePath:          "testremote:",
		BaseDir:             tempDir,
		ProfilesFile:        filepath.Join(tempDir, "profiles.json"),
		ProfilesHistoryFile: filepath.Join(tempDir, "profiles_history.json"),
		ActiveReviewFile:    filepath.Join(tempDir, "active_review_session.json"),
		MetricsFile:         filepath.Join(tempDir, "metrics.json"),
		RCAPIURL:            mockRCServer.URL,
	}
	_ = os.MkdirAll(cfg.LocalPath, 0755)

	// Seed local files for Explorer tests
	_ = os.WriteFile(filepath.Join(cfg.LocalPath, "notes.txt"), []byte("sample notes content"), 0644)
	_ = os.WriteFile(filepath.Join(cfg.LocalPath, "RCLONE_TEST"), []byte("canary test token"), 0644)

	// Seed active conflict review session with candidate
	reviewJSON := `{
		"id": "rev-test-e2e",
		"created_at": "2026-11-18T18:00:00Z",
		"updated_at": "2026-11-18T18:00:00Z",
		"status": "REVIEWING",
		"conflicts": [
			{
				"id": "conf-e2e-1",
				"path": "test_conflict.txt",
				"name": "test_conflict.txt",
				"local_size": 100,
				"local_mod_time": "2026-11-18T18:00:00Z",
				"remote_size": 200,
				"remote_mod_time": "2026-11-18T17:00:00Z",
				"condition": "CONFLICT",
				"resolved": false
			}
		],
		"deletions": [],
		"actions": []
	}`
	_ = os.WriteFile(filepath.Join(tempDir, "active_review_session.json"), []byte(reviewJSON), 0644)

	syncSvc := services.NewSyncService(cfg, nil)
	srv := server.NewServer(cfg, syncSvc, os.DirFS(repoRoot))
	srv.Version = "v2.0.0-vue3-e2e"

	// Seed sync profiles
	if srv.ProfileService != nil {
		p1, err1 := srv.ProfileService.Create(context.Background(), &profiles.Profile{
			ID:         "prof-docs",
			Name:       "Documents Sync",
			Remote:     "testremote",
			RemotePath: "docs",
			LocalPath:  cfg.LocalPath,
			Schedule:   "15m",
			Enabled:    true,
			State: profiles.OperationalState{
				Status: profiles.ProfileStatus("review_required"),
			},
		})
		if err1 != nil {
			t.Logf("Profile 1 creation failed: %v", err1)
		} else {
			t.Logf("Profile 1 created: %+v", p1)
		}
		p2, err2 := srv.ProfileService.Create(context.Background(), &profiles.Profile{
			ID:         "prof-media",
			Name:       "Media Photos",
			Remote:     "testremote",
			RemotePath: "photos",
			LocalPath:  cfg.LocalPath,
			Schedule:   "daily",
			Enabled:    true,
			State: profiles.OperationalState{
				Status: profiles.StatusHealthy,
			},
		})
		if err2 != nil {
			t.Logf("Profile 2 creation failed: %v", err2)
		} else {
			t.Logf("Profile 2 created: %+v", p2)
		}
	}
	if srv.ReviewService != nil {
		sess, errS := srv.ReviewService.GetActiveSession(context.Background())
		t.Logf("Review session on srv: sess=%+v err=%v", sess, errS)
	}

	ts := httptest.NewServer(srv)
	defer ts.Close()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromePath),
		chromedp.DisableGPU,
		chromedp.NoSandbox,
		chromedp.Headless,
		chromedp.WindowSize(1440, 900),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-software-rasterizer", true),
		chromedp.Flag("proxy-server", "direct://"),
		chromedp.Flag("proxy-bypass-list", "*"),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-breakpad", true),
		chromedp.Flag("disable-component-update", true),
		chromedp.Flag("disable-ipc-flooding-protection", true),
		chromedp.Flag("disable-hang-monitor", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("force-color-profile", "srgb"),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	testTimeout := 35 * time.Second
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		testTimeout = 60 * time.Second
	}
	ctx, cancelTimeout := context.WithTimeout(ctx, testTimeout)
	defer cancelTimeout()

	var consoleErrors []string
	var mu sync.Mutex

	messages := chromedp.Console(ctx)
	go func() {
		for m, err := range messages {
			if err != nil {
				break
			}
			t.Logf("[BROWSER CONSOLE] %s: %s", m.Type, m.Text)
			if m.Type == "error" || m.IsException() {
				mu.Lock()
				consoleErrors = append(consoleErrors, m.String())
				mu.Unlock()
			}
		}
	}()

	cockpitURL := ts.URL + "/cockpit"

	// 1. Initial Page Load & SPA Mount
	t.Run("SPAMountAndHeader", func(t *testing.T) {
		err := chromedp.Do(ctx,
			chromedp.Navigate(cockpitURL),
			chromedp.WaitReady("#app"),
			chromedp.WaitReady("header"),
		)
		if err != nil {
			t.Fatalf("failed to mount Vue SPA at %s: %v", cockpitURL, err)
		}

		brandText, err := chromedp.Run(ctx, chromedp.Evaluate[string](`(() => {
			const el = document.querySelector('header span.font-bold');
			return el ? el.textContent.trim() : '';
		})()`))
		if err != nil || !strings.Contains(brandText, "Bisync Hub") {
			t.Fatalf("expected brand 'Bisync Hub', got %q (err: %v)", brandText, err)
		}
	})

	// 2. Tab Navigation
	tabs := []string{"Fleet", "Engine & Telemetry", "Storage Remotes", "File Explorer", "Audit Trail"}
	for _, tabName := range tabs {
		t.Run("Tab_"+tabName, func(t *testing.T) {
			clickTabScript := `(() => {
				const buttons = Array.from(document.querySelectorAll('header nav button'));
				const btn = buttons.find(b => b.textContent.includes('` + tabName + `'));
				if (btn) { btn.click(); return true; }
				return false;
			})()`
			clicked, err := chromedp.Run(ctx, chromedp.Evaluate[bool](clickTabScript))
			if err != nil || !clicked {
				t.Fatalf("failed to click tab %q: %v", tabName, err)
			}
			time.Sleep(150 * time.Millisecond)
		})
	}

	// 3. Log Dock Toggle
	t.Run("LogDockInteraction", func(t *testing.T) {
		clickLogsScript := `(() => {
			const btns = Array.from(document.querySelectorAll('header button'));
			const logBtn = btns.find(b => b.textContent.includes('Logs'));
			if (logBtn) { logBtn.click(); return true; }
			return false;
		})()`
		clicked, err := chromedp.Run(ctx, chromedp.Evaluate[bool](clickLogsScript))
		if err != nil || !clicked {
			t.Fatalf("failed to toggle LogDock: %v", err)
		}
		time.Sleep(150 * time.Millisecond)

		// Close dock
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](clickLogsScript))
		time.Sleep(100 * time.Millisecond)
	})

	// 4. Help Drawer
	t.Run("HelpDrawerInteraction", func(t *testing.T) {
		clickHelpScript := `(() => {
			const btns = Array.from(document.querySelectorAll('header button'));
			const helpBtn = btns.find(b => b.textContent.includes('Help'));
			if (helpBtn) { helpBtn.click(); return true; }
			return false;
		})()`
		clicked, err := chromedp.Run(ctx, chromedp.Evaluate[bool](clickHelpScript))
		if err != nil || !clicked {
			t.Fatalf("failed to open HelpDrawer: %v", err)
		}
		time.Sleep(200 * time.Millisecond)

		// Close drawer by pressing Escape
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
			return true;
		})()`))
		time.Sleep(150 * time.Millisecond)
	})

	// 5. Global Settings Modal
	t.Run("GlobalSettingsModalInteraction", func(t *testing.T) {
		clickSettingsScript := `(() => {
			const btns = Array.from(document.querySelectorAll('header button'));
			const sBtn = btns.find(b => b.textContent.includes('Settings'));
			if (sBtn) { sBtn.click(); return true; }
			return false;
		})()`
		clicked, err := chromedp.Run(ctx, chromedp.Evaluate[bool](clickSettingsScript))
		if err != nil || !clicked {
			t.Fatalf("failed to open Settings modal: %v", err)
		}
		time.Sleep(200 * time.Millisecond)

		// Close modal by pressing Escape
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
			return true;
		})()`))
		time.Sleep(150 * time.Millisecond)
	})

	// 6. System Health Modal
	t.Run("SystemHealthModalInteraction", func(t *testing.T) {
		clickHealthScript := `(() => {
			const btns = Array.from(document.querySelectorAll('header button'));
			const hBtn = btns.find(b => (b.title && b.title.includes('System Health')) || b.textContent.includes('Daemon'));
			if (hBtn) { hBtn.click(); return true; }
			return false;
		})()`
		clicked, err := chromedp.Run(ctx, chromedp.Evaluate[bool](clickHealthScript))
		if err != nil || !clicked {
			t.Fatalf("failed to open System Health modal: %v", err)
		}
		time.Sleep(200 * time.Millisecond)

		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
			return true;
		})()`))
		time.Sleep(150 * time.Millisecond)
	})

	// 7. MCP Inspector Modal
	t.Run("McpModalInteraction", func(t *testing.T) {
		clickMcpScript := `(() => {
			const btns = Array.from(document.querySelectorAll('header button'));
			const mBtn = btns.find(b => b.textContent.includes('MCP'));
			if (mBtn) { mBtn.click(); return true; }
			return false;
		})()`
		clicked, err := chromedp.Run(ctx, chromedp.Evaluate[bool](clickMcpScript))
		if err != nil || !clicked {
			t.Fatalf("failed to open MCP modal: %v", err)
		}
		time.Sleep(200 * time.Millisecond)

		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
			return true;
		})()`))
		time.Sleep(150 * time.Millisecond)
	})

	// 8. Upload Dock Modal
	t.Run("UploadDockModalInteraction", func(t *testing.T) {
		// Switch to File Explorer tab
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			const buttons = Array.from(document.querySelectorAll('header nav button'));
			const btn = buttons.find(b => b.textContent.includes('File Explorer'));
			if (btn) { btn.click(); return true; }
			return false;
		})()`))
		time.Sleep(200 * time.Millisecond)

		// Click Uploads button in Explorer toolbar
		clickUploadsScript := `(() => {
			const btns = Array.from(document.querySelectorAll('button'));
			const uBtn = btns.find(b => b.textContent.includes('Uploads'));
			if (uBtn) { uBtn.click(); return true; }
			return false;
		})()`
		clicked, err := chromedp.Run(ctx, chromedp.Evaluate[bool](clickUploadsScript))
		if err != nil || !clicked {
			t.Fatalf("failed to open Upload Dock modal: %v", err)
		}
		time.Sleep(200 * time.Millisecond)

		// Verify modal content
		hasHeader, err := chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			return document.body.textContent.includes('Upload Dock & Transfer Queue');
		})()`))
		if err != nil || !hasHeader {
			t.Fatalf("expected Upload Dock modal header in DOM, err: %v", err)
		}

		// Close modal by pressing Escape
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
			return true;
		})()`))
		time.Sleep(150 * time.Millisecond)
	})
	// 9. Queue Manager Modal Interaction
	t.Run("QueueManagerModalInteraction", func(t *testing.T) {
		clickQueueScript := `(() => {
			const btns = Array.from(document.querySelectorAll('button'));
			const qBtn = btns.find(b => b.textContent.includes('Queue'));
			if (qBtn) { qBtn.click(); return true; }
			return false;
		})()`
		clicked, err := chromedp.Run(ctx, chromedp.Evaluate[bool](clickQueueScript))
		if err != nil || !clicked {
			t.Fatalf("failed to open Queue Manager modal: %v", err)
		}
		time.Sleep(200 * time.Millisecond)

		hasHeader, err := chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			return document.body.textContent.includes('Sync Queue & Concurrency Dispatcher');
		})()`))
		if err != nil || !hasHeader {
			t.Fatalf("expected Queue Manager modal header in DOM, err: %v", err)
		}

		// Close modal by pressing Escape
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
			return true;
		})()`))
		time.Sleep(150 * time.Millisecond)
	})

	// 10. Bandwidth Throttling Popover Interaction
	t.Run("BandwidthPopoverInteraction", func(t *testing.T) {
		clickBwScript := `(() => {
			const btns = Array.from(document.querySelectorAll('header button'));
			const bwBtn = btns.find(b => b.textContent.includes('BW:'));
			if (bwBtn) { bwBtn.click(); return true; }
			return false;
		})()`
		clicked, err := chromedp.Run(ctx, chromedp.Evaluate[bool](clickBwScript))
		if err != nil || !clicked {
			t.Fatalf("failed to open Bandwidth popover: %v", err)
		}
		time.Sleep(200 * time.Millisecond)

		hasPopover, err := chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			return document.body.textContent.includes('Bandwidth Throttling') && document.body.textContent.includes('Presets');
		})()`))
		if err != nil || !hasPopover {
			t.Fatalf("expected Bandwidth Throttling popover in DOM, err: %v", err)
		}

		// Close popover
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](clickBwScript))
		time.Sleep(150 * time.Millisecond)
	})

	// 11. Keyboard Shortcuts Modal Interaction
	t.Run("KeyboardShortcutsModalInteraction", func(t *testing.T) {
		clickShortcutsScript := `(() => {
			const btns = Array.from(document.querySelectorAll('header button'));
			const sBtn = btns.find(b => (b.title && b.title.includes('Keyboard Shortcuts')));
			if (sBtn) { sBtn.click(); return true; }
			return false;
		})()`
		clicked, err := chromedp.Run(ctx, chromedp.Evaluate[bool](clickShortcutsScript))
		if err != nil || !clicked {
			t.Fatalf("failed to open Keyboard Shortcuts modal: %v", err)
		}
		time.Sleep(200 * time.Millisecond)

		hasShortcuts, err := chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			return document.body.textContent.includes('Cockpit Keyboard Shortcuts') && document.body.textContent.includes('Operator & Polling Controls');
		})()`))
		if err != nil || !hasShortcuts {
			t.Fatalf("expected Keyboard Shortcuts modal in DOM, err: %v", err)
		}

		// Close modal by pressing Escape
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
			return true;
		})()`))
		time.Sleep(150 * time.Millisecond)
	})

	// 12. Quick Create Menu Interaction
	t.Run("QuickCreateMenuInteraction", func(t *testing.T) {
		clickCreateScript := `(() => {
			const btns = Array.from(document.querySelectorAll('header button'));
			const cBtn = btns.find(b => b.textContent.includes('Create'));
			if (cBtn) { cBtn.click(); return true; }
			return false;
		})()`
		clicked, err := chromedp.Run(ctx, chromedp.Evaluate[bool](clickCreateScript))
		if err != nil || !clicked {
			t.Fatalf("failed to open Quick Create menu: %v", err)
		}
		time.Sleep(200 * time.Millisecond)

		hasMenu, err := chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			return document.body.textContent.includes('New Sync Profile') && document.body.textContent.includes('Connect Remote');
		})()`))
		if err != nil || !hasMenu {
			t.Fatalf("expected Quick Create menu items in DOM, err: %v", err)
		}

		// Close menu
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
			return true;
		})()`))
		time.Sleep(150 * time.Millisecond)
	})

	// 13. Pending Deletions Modal Interaction
	t.Run("PendingDeletionsModalInteraction", func(t *testing.T) {
		openDeletionsScript := `(() => {
			if (typeof window.openDeletionsModal === 'function') {
				window.openDeletionsModal();
				return true;
			}
			return false;
		})()`
		opened, err := chromedp.Run(ctx, chromedp.Evaluate[bool](openDeletionsScript))
		if err != nil || !opened {
			t.Fatalf("failed to open Pending Deletions modal: %v", err)
		}
		time.Sleep(200 * time.Millisecond)

		hasDeletionsModal, err := chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			return document.body.textContent.includes('Pending Deletions Review');
		})()`))
		if err != nil || !hasDeletionsModal {
			t.Fatalf("expected Pending Deletions modal in DOM, err: %v", err)
		}

		// Close modal
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
			return true;
		})()`))
		time.Sleep(150 * time.Millisecond)
	})

	// 14. Operator Polling Hotkey Toggle
	t.Run("OperatorPollingToggle", func(t *testing.T) {
		toggleScript := `(() => {
			window.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', altKey: true }));
			return window.isPollingPaused === true;
		})()`
		paused, err := chromedp.Run(ctx, chromedp.Evaluate[bool](toggleScript))
		if err != nil || !paused {
			t.Fatalf("expected isPollingPaused=true after Alt+P, err: %v", err)
		}
		time.Sleep(150 * time.Millisecond)

		resumeScript := `(() => {
			window.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', altKey: true }));
			return window.isPollingPaused === false;
		})()`
		resumed, err := chromedp.Run(ctx, chromedp.Evaluate[bool](resumeScript))
		if err != nil || !resumed {
			t.Fatalf("expected isPollingPaused=false after second Alt+P, err: %v", err)
		}
		time.Sleep(150 * time.Millisecond)
	})

	// 15. Dedupe Workspace Modal Interaction
	t.Run("DedupeModalInteraction", func(t *testing.T) {
		// Switch to Storage Remotes tab
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			const buttons = Array.from(document.querySelectorAll('header nav button'));
			const btn = buttons.find(b => b.textContent.includes('Storage Remotes'));
			if (btn) { btn.click(); return true; }
			return false;
		})()`))
		time.Sleep(200 * time.Millisecond)

		clickDedupeScript := `(() => {
			const btns = Array.from(document.querySelectorAll('button'));
			const dBtn = btns.find(b => b.textContent.includes('Dedupe'));
			if (dBtn) { dBtn.click(); return true; }
			return false;
		})()`
		clicked, err := chromedp.Run(ctx, chromedp.Evaluate[bool](clickDedupeScript))
		if err != nil || !clicked {
			t.Fatalf("failed to open Dedupe modal: %v", err)
		}
		time.Sleep(200 * time.Millisecond)

		hasDedupeModal, err := chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			return document.body.textContent.includes('Deduplication Workspace') &&
				document.body.textContent.includes('Deduplication Strategy');
		})()`))
		if err != nil || !hasDedupeModal {
			t.Fatalf("expected Dedupe modal in DOM, err: %v", err)
		}

		// Verify collision search input and status filter pills
		hasDedupeControls, err := chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			const input = document.querySelector('input[placeholder*="filename"]');
			const text = document.body.textContent;
			const hasPills = text.includes('Unresolved') && text.includes('Staged') && text.includes('Multi-Copy');
			return !!input && hasPills;
		})()`))
		if err != nil || !hasDedupeControls {
			t.Fatalf("expected Dedupe collision search input and filter pills in DOM, err: %v", err)
		}

		// Close modal with Escape
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
			return true;
		})()`))
		time.Sleep(150 * time.Millisecond)
	})

	// 16. Fleet View Search, Filter, and Navigation Jumps
	t.Run("FleetFilterAndNavigation", func(t *testing.T) {
		// Switch to Fleet tab
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			const buttons = Array.from(document.querySelectorAll('header nav button'));
			const btn = buttons.find(b => b.textContent.includes('Fleet'));
			if (btn) { btn.click(); return true; }
			return false;
		})()`))
		time.Sleep(300 * time.Millisecond)

		// Verify profile cards render
		hasFleetCards, err := chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			const text = document.body.textContent;
			return text.includes('Documents Sync') && text.includes('Media Photos');
		})()`))
		if err != nil || !hasFleetCards {
			t.Fatalf("expected initial fleet profiles in DOM, err: %v", err)
		}

		// 1. Search input filters down to "Documents"
		searchDocScript := `(() => {
			const input = document.querySelector('input[placeholder*="Filter by profile name"]');
			if (!input) return false;
			input.value = 'Documents';
			input.dispatchEvent(new Event('input', { bubbles: true }));
			return true;
		})()`
		ok, err := chromedp.Run(ctx, chromedp.Evaluate[bool](searchDocScript))
		if err != nil || !ok {
			t.Fatalf("failed to enter Fleet search query: %v", err)
		}
		time.Sleep(200 * time.Millisecond)

		// Verify filtered card visibility
		filtered, err := chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			const cardTitles = Array.from(document.querySelectorAll('.grid span.font-bold.text-sm')).map(el => el.textContent.trim());
			return cardTitles.includes('Documents Sync') && !cardTitles.includes('Media Photos');
		})()`))
		if err != nil || !filtered {
			t.Fatalf("expected filtered fleet view showing only Documents Sync, err: %v", err)
		}

		// 2. Clear search input
		clearSearchScript := `(() => {
			const input = document.querySelector('input[placeholder*="Filter by profile name"]');
			if (!input) return false;
			input.value = '';
			input.dispatchEvent(new Event('input', { bubbles: true }));
			return true;
		})()`
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](clearSearchScript))
		time.Sleep(200 * time.Millisecond)

		// 3. Status filter pill click ("Review Required")
		clickReviewPillScript := `(() => {
			const pills = Array.from(document.querySelectorAll('button'));
			const p = pills.find(el => el.textContent.includes('Review Required'));
			if (p) { p.click(); return true; }
			return false;
		})()`
		clicked, err := chromedp.Run(ctx, chromedp.Evaluate[bool](clickReviewPillScript))
		if err != nil || !clicked {
			t.Fatalf("failed to click Review Required filter pill: %v", err)
		}
		time.Sleep(200 * time.Millisecond)

		// 4. Click All pill to reset
		clickAllPillScript := `(() => {
			const pills = Array.from(document.querySelectorAll('button'));
			const p = pills.find(el => el.textContent.includes('All'));
			if (p) { p.click(); return true; }
			return false;
		})()`
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](clickAllPillScript))
		time.Sleep(200 * time.Millisecond)

		// 5. Click Explorer jump button on Documents Sync card
		clickExplorerJumpScript := `(() => {
			const cards = Array.from(document.querySelectorAll('div.bg-carbon-900'));
			const docCard = cards.find(c => c.textContent.includes('Documents Sync'));
			if (!docCard) return false;
			const expBtn = Array.from(docCard.querySelectorAll('button')).find(b => b.title && b.title.includes('Dual Explorer'));
			if (expBtn) { expBtn.click(); return true; }
			return false;
		})()`
		jumped, err := chromedp.Run(ctx, chromedp.Evaluate[bool](clickExplorerJumpScript))
		if err != nil || !jumped {
			t.Fatalf("failed to click Explorer jump button on profile card: %v", err)
		}
		time.Sleep(300 * time.Millisecond)

		// Verify active view switched to File Explorer
		isExplorerActive, err := chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			const navBtns = Array.from(document.querySelectorAll('header nav button'));
			const expBtn = navBtns.find(b => b.textContent.includes('File Explorer'));
			return expBtn ? expBtn.classList.contains('text-white') || expBtn.classList.contains('bg-carbon-800') : false;
		})()`))
		if err != nil || !isExplorerActive {
			t.Fatalf("expected File Explorer tab to be active after jump, err: %v", err)
		}
	})

	// 17. Conflict Review Heuristics & Auto-Apply Subtest
	t.Run("ConflictReviewHeuristicsAndAutoApply", func(t *testing.T) {
		// Open Conflict Review via Alt+R shortcut or window.openReviewModal
		openReviewScript := `(() => {
			if (typeof window.openReviewModal === 'function') {
				window.openReviewModal();
				return true;
			}
			window.dispatchEvent(new KeyboardEvent('keydown', { key: 'r', altKey: true, bubbles: true }));
			return true;
		})()`
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](openReviewScript))
		time.Sleep(500 * time.Millisecond)

		// Verify Conflict Review modal is open
		hasReviewModal, err := chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			const text = document.body.textContent;
			return (text.includes('Sync Conflict Review') || text.includes('Conflict Resolution')) &&
				text.includes('test_conflict.txt');
		})()`))
		if err != nil || !hasReviewModal {
			t.Fatalf("expected Conflict Review modal in DOM, err: %v", err)
		}

		// Verify heuristic recommendation badge (Auto-Recommendation, confidence %, action chip)
		hasRecommendation, err := chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			const text = document.body.textContent;
			return text.includes('Auto-Recommendation') || text.includes('Confidence') || text.includes('Keep Local') || text.includes('Keep Remote') || text.includes('Auto-Apply');
		})()`))
		if err != nil || !hasRecommendation {
			t.Fatalf("expected heuristic recommendation badges in Conflict Review modal, err: %v", err)
		}

		// Click "Auto-Apply" button
		clickAutoApplyScript := `(() => {
			const btns = Array.from(document.querySelectorAll('button'));
			const autoBtn = btns.find(b => b.textContent.includes('Auto-Apply') || b.textContent.includes('Auto-Resolve'));
			if (autoBtn) { autoBtn.click(); return true; }
			return false;
		})()`
		applied, err := chromedp.Run(ctx, chromedp.Evaluate[bool](clickAutoApplyScript))
		if err != nil || !applied {
			t.Fatalf("failed to click Auto-Apply button: %v", err)
		}
		time.Sleep(300 * time.Millisecond)

		// Close modal by pressing Escape
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
			return true;
		})()`))
		time.Sleep(200 * time.Millisecond)
	})

	// 18. Explorer Drag-and-Drop Integrity Subtest
	t.Run("ExplorerDragAndDropIntegrity", func(t *testing.T) {
		// Ensure in File Explorer tab
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			const buttons = Array.from(document.querySelectorAll('header nav button'));
			const btn = buttons.find(b => b.textContent.includes('File Explorer'));
			if (btn) { btn.click(); return true; }
			return false;
		})()`))
		time.Sleep(300 * time.Millisecond)

		// Verify table rows have draggable="true" for normal files, draggable="false" on canary and child buttons
		integrityCheckScript := `(() => {
			const rows = Array.from(document.querySelectorAll('.divide-y > div'));
			if (rows.length === 0) return { ok: false, reason: 'no rows found in explorer table' };

			const canaryRow = rows.find(r => r.textContent.includes('RCLONE_TEST'));
			if (canaryRow && canaryRow.getAttribute('draggable') === 'true') {
				return { ok: false, reason: 'canary row must not have draggable=true' };
			}

			const fileRow = rows.find(r => r.textContent.includes('notes.txt'));
			if (fileRow && fileRow.getAttribute('draggable') !== 'true') {
				return { ok: false, reason: 'notes.txt file row must have draggable=true' };
			}

			if (fileRow) {
				const interactive = Array.from(fileRow.querySelectorAll('input, button'));
				for (const el of interactive) {
					if (el.getAttribute('draggable') !== 'false') {
						return { ok: false, reason: 'child interactive element must have draggable=false' };
					}
				}
			}

			return { ok: true };
		})()`
		res, err := chromedp.Run(ctx, chromedp.Evaluate[map[string]any](integrityCheckScript))
		if err != nil {
			t.Fatalf("failed to evaluate drag-and-drop integrity: %v", err)
		}
		if ok, _ := res["ok"].(bool); !ok {
			t.Fatalf("drag-and-drop integrity failed: %v", res["reason"])
		}
	})

	// 19. Final Console Error Audit
	t.Run("ConsoleErrorAudit", func(t *testing.T) {
		mu.Lock()
		defer mu.Unlock()

		// Filter out benign favicon or net disconnects during fast test server shutdown
		var criticalErrors []string
		for _, e := range consoleErrors {
			if strings.Contains(e, "favicon.ico") || strings.Contains(e, "net::ERR_CONNECTION_REFUSED") {
				continue
			}
			criticalErrors = append(criticalErrors, e)
		}

		if len(criticalErrors) > 0 {
			t.Errorf("encountered %d uncaught browser console errors:\n%s",
				len(criticalErrors), strings.Join(criticalErrors, "\n"))
		}
	})
}
