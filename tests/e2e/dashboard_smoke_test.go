package e2e

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/server"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

// findChromeBinary searches for an available Chrome/Chromium executable.
func findChromeBinary() string {
	for _, bin := range []string{
		"google-chrome",
		"google-chrome-stable",
		"chromium-browser",
		"chromium",
		"headless-shell",
	} {
		if path, err := exec.LookPath(bin); err == nil {
			return path
		}
	}
	return ""
}

// findRepoRoot locates the root directory containing go.mod.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repository root containing go.mod")
		}
		dir = parent
	}
}

// TestDashboard_HeadlessChromeSmoke launches a real headless Chrome instance
// to navigate the rendered dashboard, switch cockpit tabs, audit interactive button states,
// and assert zero runtime JavaScript exceptions or console errors.
func TestDashboard_HeadlessChromeSmoke(t *testing.T) {
	chromePath := findChromeBinary()
	if chromePath == "" {
		t.Skip("skipping E2E smoke test: Chrome/Chromium binary not found in PATH")
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
		case "/operations/about":
			_, _ = w.Write([]byte(`{"total":1000000,"used":500000,"free":500000}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer mockRCServer.Close()

	cfg := &config.Config{
		LocalPath:   filepath.Join(tempDir, "local"),
		RemotePath:  "testremote:",
		BaseDir:     tempDir,
		MetricsFile: filepath.Join(tempDir, "metrics.json"),
		RCAPIURL:    mockRCServer.URL,
	}
	_ = os.MkdirAll(cfg.LocalPath, 0755)

	syncSvc := services.NewSyncService(cfg, nil)
	srv := server.NewServer(cfg, syncSvc, os.DirFS(repoRoot))
	srv.Version = "v2.0.0-e2e"

	ts := httptest.NewServer(srv)
	defer ts.Close()

	// Configure Chrome execution with flags hardened for containerized/CI runners
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromePath),
		chromedp.DisableGPU,
		chromedp.NoSandbox,
		chromedp.Headless,
		chromedp.WindowSize(1440, 900),
		// Prevent shared memory exhaustion in container environments (redirect to /tmp)
		chromedp.Flag("disable-dev-shm-usage", true),
		// Avoid software GL fallback hang when GPU disabled
		chromedp.Flag("disable-software-rasterizer", true),
		// Avoid routing loopback test server through external proxies
		chromedp.Flag("proxy-server", "direct://"),
		chromedp.Flag("proxy-bypass-list", "*"),
		// Optimize background resource consumption and prevent throttle hangs
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

	// In virtualized CI environments (such as GitHub Actions ubuntu-latest),
	// high CPU contention during parallel 'go test ./...' runs warrants a higher timeout margin.
	testTimeout := 35 * time.Second
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		testTimeout = 60 * time.Second
	}
	ctx, cancelTimeout := context.WithTimeout(ctx, testTimeout)
	defer cancelTimeout()

	// Capture console messages & uncaught exceptions
	var consoleErrors []string
	var mu sync.Mutex

	messages := chromedp.Console(ctx)
	go func() {
		for m, err := range messages {
			if err != nil {
				break
			}
			if m.Type == "error" || m.IsException() {
				mu.Lock()
				consoleErrors = append(consoleErrors, m.String())
				mu.Unlock()
			}
		}
	}()

	// 1. Initial page load & verify Alpine initialization
	err := chromedp.Do(ctx,
		chromedp.Navigate(ts.URL),
		chromedp.WaitVisible("body[x-data]"),
		chromedp.WaitVisible("#app-header"),
	)
	if err != nil {
		mu.Lock()
		loggedErrors := strings.Join(consoleErrors, "\n")
		mu.Unlock()
		if len(loggedErrors) > 0 {
			t.Fatalf("failed to load dashboard within %v: %v\nConsole errors:\n%s", testTimeout, err, loggedErrors)
		}
		t.Fatalf("failed to load dashboard within %v: %v", testTimeout, err)
	}

	title, err := chromedp.Run(ctx, chromedp.Title())
	if err != nil {
		t.Fatalf("failed reading page title: %v", err)
	}
	if !strings.Contains(title, "Bisync Hub") {
		t.Errorf("unexpected page title: got %q, expected to contain 'Bisync Hub'", title)
	}

	// Allow Alpine.js to complete initialization
	time.Sleep(800 * time.Millisecond)

	// Verify cockpit navigation buttons exist and are enabled
	navPillsDisabled, err := chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
		const navButtons = Array.from(document.querySelectorAll('header button'));
		return navButtons.length === 0 || navButtons.some(b => b.disabled);
	})()`))
	if err != nil {
		t.Fatalf("failed evaluating header nav buttons: %v", err)
	}
	if navPillsDisabled {
		t.Errorf("header navigation buttons should exist and never be disabled")
	}

	// Verify MCP status pill in header
	mcpPillText, err := chromedp.Run(ctx, chromedp.Evaluate[string](`(() => {
		const el = document.querySelector('#app-header [x-text="mcpStatusText"]');
		return el ? el.textContent.trim() : '';
	})()`))
	if err != nil {
		t.Fatalf("failed evaluating MCP status pill: %v", err)
	}
	if !strings.HasPrefix(mcpPillText, "MCP:") {
		t.Errorf("expected MCP status pill to start with 'MCP:', got %q", mcpPillText)
	}

	// 2. Audit each cockpit tab
	tabs := []struct {
		name       string
		tabPillIdx int
	}{
		{"fleet", 0},
		{"telemetry", 1},
		{"remotes", 2},
		{"explorer", 3},
		{"audit", 4},
	}

	for _, tc := range tabs {
		t.Run("Tab_"+tc.name, func(t *testing.T) {
			clickScript := fmt.Sprintf(`(() => {
				const nav = document.querySelector('header nav');
				const navButtons = nav ? Array.from(nav.querySelectorAll('button[data-tab]')) : Array.from(document.querySelectorAll('header button'));
				if (navButtons[%d]) {
					navButtons[%d].click();
					return true;
				}
				return false;
			})()`, tc.tabPillIdx, tc.tabPillIdx)

			clicked, err := chromedp.Run(ctx, chromedp.Evaluate[bool](clickScript))
			if err != nil || !clicked {
				t.Fatalf("failed clicking tab %q: %v", tc.name, err)
			}

			// Poll until async tab data loading finishes (up to 5 seconds)
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				stillLoading, err := chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
					const el = document.querySelector('body[x-data]');
					if (!el || !window.Alpine) return false;
					const data = Alpine.$data(el);
					if (!data) return false;
					if (data.remotesData && data.remotesData.loading) return true;
					if (data.audit && data.audit.loading) return true;
					if (data.explorer && ((data.explorer.left && data.explorer.left.loading) || (data.explorer.right && data.explorer.right.loading))) return true;
					return false;
				})()`))
				if err == nil && !stillLoading {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
			time.Sleep(100 * time.Millisecond)

			type TabButtonAudit struct {
				TotalButtons    int      `json:"totalButtons"`
				DisabledButtons int      `json:"disabledButtons"`
				DisabledDetails []string `json:"disabledDetails"`
			}

			auditScript := `(() => {
				const buttons = Array.from(document.querySelectorAll('button')).filter(b => {
					return b.offsetParent !== null;
				});
				const disabled = buttons.filter(b => b.disabled);
				return {
					totalButtons: buttons.length,
					disabledButtons: disabled.length,
					disabledDetails: disabled.map(b => (b.innerText || b.getAttribute('title') || b.className).trim().slice(0, 60))
				};
			})()`

			audit, err := chromedp.Run(ctx, chromedp.Evaluate[TabButtonAudit](auditScript))
			if err != nil {
				t.Fatalf("failed auditing buttons in tab %q: %v", tc.name, err)
			}

			t.Logf("Tab [%s]: %d visible buttons, %d disabled (%v)",
				tc.name, audit.TotalButtons, audit.DisabledButtons, audit.DisabledDetails)

			if tc.name == "remotes" {
				for _, detail := range audit.DisabledDetails {
					if strings.Contains(strings.ToLower(detail), "refresh") || strings.Contains(strings.ToLower(detail), "add") {
						t.Errorf("remote action button unexpectedly disabled: %s", detail)
					}
				}
			}

			if tc.name == "explorer" {
				mkdirVisible, err := chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
					const btns = Array.from(document.querySelectorAll('button')).filter(b => b.innerText && b.innerText.includes('New Folder'));
					return btns.length >= 2;
				})()`))
				if err != nil || !mkdirVisible {
					t.Errorf("expected at least 2 visible 'New Folder' buttons in dual explorer panes")
				}
			}
		})
	}

	// 3. Test Fleet Profile Card Click -> Telemetry Navigation
	t.Run("FleetCardClickNavigation", func(t *testing.T) {
		clickFleetScript := `(() => {
			const btn = document.querySelector('nav button[data-tab="fleet"]');
			if (btn) { btn.click(); return true; }
			return false;
		})()`
		clicked, err := chromedp.Run(ctx, chromedp.Evaluate[bool](clickFleetScript))
		if err != nil || !clicked {
			t.Fatalf("failed clicking fleet nav button: %v", err)
		}

		time.Sleep(150 * time.Millisecond)

		clickCardScript := `(() => {
			const cards = document.querySelectorAll('[x-show*="fleet"] .grid div[class*="cursor-pointer"]');
			if (cards.length > 0) {
				cards[0].click();
				return true;
			}
			return false;
		})()`
		cardClicked, err := chromedp.Run(ctx, chromedp.Evaluate[bool](clickCardScript))
		if err == nil && cardClicked {
			time.Sleep(150 * time.Millisecond)
			tabCheckScript := `window.Alpine ? Alpine.$data(document.querySelector('[x-data]')).currentTab : ''`
			activeTab, err := chromedp.Run(ctx, chromedp.Evaluate[string](tabCheckScript))
			if err == nil && activeTab != "telemetry" {
				t.Errorf("expected currentTab to be 'telemetry' after clicking profile card, got %q", activeTab)
			}
		}
	})
	// 4. Test Global Fleet Settings Modal Opening & Tab Switching
	t.Run("GlobalSettingsModalTabs", func(t *testing.T) {
		openScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return false;
			app.openGlobalSettingsModal('defaults');
			return true;
		})()`
		opened, err := chromedp.Run(ctx, chromedp.Evaluate[bool](openScript))
		if err != nil || !opened {
			t.Fatalf("failed opening global settings modal: %v", err)
		}
		time.Sleep(100 * time.Millisecond)

		// Test switching to engine tab
		switchEngineScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return false;
			app.globalSettingsModal.tab = 'engine';
			return true;
		})()`
		_, err = chromedp.Run(ctx, chromedp.Evaluate[bool](switchEngineScript))
		if err != nil {
			t.Fatalf("failed switching to engine tab: %v", err)
		}
		time.Sleep(100 * time.Millisecond)

		// Verify engine input fields are visible and bound
		engineInputCheck := `(() => {
			const inputs = document.querySelectorAll('input[x-model\\.number*="daemon"]');
			return inputs.length;
		})()`
		count, err := chromedp.Run(ctx, chromedp.Evaluate[int](engineInputCheck))
		if err != nil || count == 0 {
			t.Errorf("expected daemon engine inputs to be present in DOM, got %d (err: %v)", count, err)
		}

		// Test switching to bandwidth tab
		switchBwScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return false;
			app.globalSettingsModal.tab = 'bandwidth';
			return true;
		})()`
		_, err = chromedp.Run(ctx, chromedp.Evaluate[bool](switchBwScript))
		if err != nil {
			t.Fatalf("failed switching to bandwidth tab: %v", err)
		}
		time.Sleep(100 * time.Millisecond)

		bwPresetCheck := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			return app && app.bandwidth && Array.isArray(app.bandwidth.presets) && app.bandwidth.presets.length > 0;
		})()`
		bwOk, err := chromedp.Run(ctx, chromedp.Evaluate[bool](bwPresetCheck))
		if err != nil || !bwOk {
			t.Errorf("expected bandwidth presets to be accessible on bandwidth tab, got %v (err: %v)", bwOk, err)
		}

		// Verify settings modal tabs container has responsive flex-wrap
		tabNavWrapCheck := `(() => {
			const modal = document.getElementById('modal-global-settings');
			if (!modal) return false;
			const tabNav = modal.querySelector('.flex-wrap');
			return tabNav !== null;
		})()`
		hasFlexWrap, err := chromedp.Run(ctx, chromedp.Evaluate[bool](tabNavWrapCheck))
		if err != nil || !hasFlexWrap {
			t.Errorf("expected global settings tabs container to have flex-wrap, got %v (err: %v)", hasFlexWrap, err)
		}

		// Close modal
		closeScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (app) app.closeGlobalSettingsModal();
			return true;
		})()`
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](closeScript))
	})

	// 5. Test Daemon Passphrase Warning Banner & Persistence UI
	t.Run("DaemonPassphraseWarningAndPersistence", func(t *testing.T) {
		script := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return { error: "Alpine data not found" };

			// 1. Initial status: verify default state & helper methods
			if (typeof app.isSavingDaemonPassphrase !== 'function') {
				return { error: "isSavingDaemonPassphrase missing" };
			}
			if (typeof app.saveDaemonPassphrase !== 'function') {
				return { error: "saveDaemonPassphrase missing" };
			}
			const initialSaving = app.isSavingDaemonPassphrase();

			// 2. Simulate encrypted config missing daemon passphrase
			if (!app.globalSettingsModal) app.globalSettingsModal = {};
			if (!app.globalSettingsModal.encryption) app.globalSettingsModal.encryption = {};
			app.globalSettingsModal.encryption.is_encrypted = true;
			app.globalSettingsModal.encryption.has_daemon_passphrase = false;

			// 3. Test empty daemon passphrase validation
			let warningToast = false;
			const origToast = app.showToast;
			app.showToast = (msg, type) => {
				if (type === 'warning' && typeof msg === 'string' && msg.includes('daemon passphrase')) {
					warningToast = true;
				}
				if (typeof origToast === 'function') origToast.call(app, msg, type);
			};
			app.globalSettingsModal.encryption.daemon_password = '   ';
			app.saveDaemonPassphrase();
			app.showToast = origToast;

			// 4. Test opening security settings tab
			app.openGlobalSettingsModal('security');
			const activeTab = app.globalSettingsModal.tab;
			const modalOpen = app.globalSettingsModal.show;

			// 5. Restore normal unencrypted state & close modal
			app.globalSettingsModal.encryption.is_encrypted = false;
			app.globalSettingsModal.encryption.has_daemon_passphrase = true;
			app.closeGlobalSettingsModal();

			return {
				initialSaving,
				warningToast,
				activeTab,
				modalOpen
			};
		})()`

		type DaemonPassphraseResult struct {
			Error         *string `json:"error"`
			InitialSaving bool    `json:"initialSaving"`
			WarningToast  bool    `json:"warningToast"`
			ActiveTab     string  `json:"activeTab"`
			ModalOpen     bool    `json:"modalOpen"`
		}

		res, err := chromedp.Run(ctx, chromedp.Evaluate[DaemonPassphraseResult](script))
		if err != nil {
			t.Fatalf("failed evaluating daemon passphrase test script: %v", err)
		}
		if res.Error != nil {
			t.Fatalf("daemon passphrase check failed: %s", *res.Error)
		}
		if res.InitialSaving {
			t.Errorf("expected initial isSavingDaemonPassphrase to be false")
		}
		if !res.WarningToast {
			t.Errorf("expected empty daemon passphrase submission to trigger warning toast")
		}
		if res.ActiveTab != "security" {
			t.Errorf("expected active tab 'security', got %q", res.ActiveTab)
		}
		if !res.ModalOpen {
			t.Errorf("expected global settings modal to open")
		}
	})

	// 6. Test Flags Builder Select Options (ensure browser doesn't strip <option> tags)
	t.Run("FlagsBuilderSelectOptionsNotStripped", func(t *testing.T) {
		openScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return false;
			app.startNewProfile();
			app.addFlagRow('--track-renames');
			return true;
		})()`
		opened, err := chromedp.Run(ctx, chromedp.Evaluate[bool](openScript))
		if err != nil || !opened {
			t.Fatalf("failed opening create profile modal: %v", err)
		}
		time.Sleep(200 * time.Millisecond)

		script := `(() => {
			const selects = document.querySelectorAll('select');
			let flagSelect = null;
			for (const s of selects) {
				if (s.querySelector('option[value="--track-renames"]')) {
					flagSelect = s;
					break;
				}
			}
			if (!flagSelect) {
				return { error: "flag select element with --track-renames not found", count: 0 };
			}
			const options = Array.from(flagSelect.options).map(o => o.value);
			return {
				error: null,
				count: options.length,
				hasTrackRenames: options.includes('--track-renames'),
				hasTrackRenamesStrategy: options.includes('--track-renames-strategy'),
				hasCustom: options.includes('custom'),
				options: options
			};
		})()`

		type SelectCheckResult struct {
			Error                   *string  `json:"error"`
			Count                   int      `json:"count"`
			HasTrackRenames         bool     `json:"hasTrackRenames"`
			HasTrackRenamesStrategy bool     `json:"hasTrackRenamesStrategy"`
			HasCustom               bool     `json:"hasCustom"`
			Options                 []string `json:"options"`
		}

		res, err := chromedp.Run(ctx, chromedp.Evaluate[SelectCheckResult](script))
		if err != nil {
			t.Fatalf("failed evaluating flag builder select: %v", err)
		}
		if res.Error != nil {
			t.Fatalf("flag builder check failed: %s", *res.Error)
		}
		if res.Count < 10 {
			t.Errorf("expected at least 10 options in flag select, got %d (options: %v)", res.Count, res.Options)
		}
		if !res.HasTrackRenames {
			t.Errorf("expected --track-renames in flag select options")
		}
		if !res.HasTrackRenamesStrategy {
			t.Errorf("expected --track-renames-strategy in flag select options")
		}
		if !res.HasCustom {
			t.Errorf("expected 'custom' in flag select options")
		}

		// Close modal
		closeModalScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (app && typeof app.closeProfilesModal === 'function') {
				app.closeProfilesModal();
				return true;
			}
			return false;
		})()`
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](closeModalScript))
	})

	// 6. Test MCP Modal Interaction & Tool Inspection
	t.Run("MCPModalInteraction", func(t *testing.T) {
		openScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app || typeof app.openMcpModal !== 'function') return false;
			app.openMcpModal('overview');
			return app.showMcpModal;
		})()`
		opened, err := chromedp.Run(ctx, chromedp.Evaluate[bool](openScript))
		if err != nil || !opened {
			t.Fatalf("failed opening MCP modal: %v", err)
		}
		time.Sleep(100 * time.Millisecond)

		// Switch to tools tab and verify
		switchToolsScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return '';
			app.mcpModalTab = 'tools';
			return app.mcpModalTab;
		})()`
		tab, err := chromedp.Run(ctx, chromedp.Evaluate[string](switchToolsScript))
		if err != nil || tab != "tools" {
			t.Fatalf("failed switching to MCP tools tab: %v", err)
		}

		// Close MCP modal
		closeScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app || typeof app.closeMcpModal !== 'function') return false;
			app.closeMcpModal();
			return !app.showMcpModal;
		})()`
		closed, err := chromedp.Run(ctx, chromedp.Evaluate[bool](closeScript))
		if err != nil || !closed {
			t.Fatalf("failed closing MCP modal: %v", err)
		}
	})

	// 7. Test Milestone 1 Operational Controls (Bandwidth, Config Encryption, File Preview)
	t.Run("Milestone1OperationalControls", func(t *testing.T) {
		script := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return { error: "Alpine app data not found" };

			// Test Bandwidth Popover and Presets
			if (typeof app.toggleBandwidthPopover !== 'function') return { error: "toggleBandwidthPopover missing" };
			app.toggleBandwidthPopover();
			const popoverOpen = app.isBandwidthPopoverOpen();
			const presets = app.bandwidth?.presets || [];
			const formattedSpeed = app.formatSpeed(10485760);
			const formattedBytes = app.formatBytes(5242880);
			app.closeBandwidthPopover();
			const popoverClosed = !app.isBandwidthPopoverOpen();

			// Test Config Encryption Tab in Global Settings
			if (typeof app.openGlobalSettingsModal !== 'function') return { error: "openGlobalSettingsModal missing" };
			app.openGlobalSettingsModal('encryption');
			const encryptionTabActive = app.globalSettingsModal?.tab === 'encryption';
			const encryptionLoading = typeof app.isEncryptionLoading === 'function' ? app.isEncryptionLoading() : null;
			app.closeGlobalSettingsModal();

			// Test File Preview Modal
			if (typeof app.previewFile !== 'function') return { error: "previewFile missing" };
			app.filePreviewModal = {
				show: true,
				loading: false,
				error: null,
				remote: 'local',
				path: 'docs/test.txt',
				name: 'test.txt',
				preview: {
					name: 'test.txt',
					size: 1024,
					mime_type: 'text/plain',
					is_text: true,
					content: 'Hello Bisync UI Suite'
				}
			};
			const previewModalOpen = Boolean(app.filePreviewModal?.show);
			const previewContent = app.filePreviewModal?.preview?.content;
			app.closePreviewModal();
			const previewModalClosed = !app.filePreviewModal?.show;

			// Test Context Menu
			if (typeof app.openContextMenu !== 'function') return { error: "openContextMenu missing" };
			app.openContextMenu('left', { Name: 'test.txt', Path: 'test.txt', IsDir: false }, { clientX: 100, clientY: 150, preventDefault() {}, stopPropagation() {} });
			const contextMenuOpen = Boolean(app.contextMenu?.show);
			const contextMenuItemName = app.contextMenu?.item?.Name;
			app.closeContextMenu();
			const contextMenuClosed = !app.contextMenu?.show;

			// Test Encryption Removal Safety Validation (empty passphrase)
			let encryptionWarningShown = false;
			const origToast = app.showToast;
			app.showToast = (msg, type) => {
				if (type === 'warning' && typeof msg === 'string' && msg.includes('passphrase is required')) {
					encryptionWarningShown = true;
				}
				origToast.call(app, msg, type);
			};
			if (!app.globalSettingsModal) app.globalSettingsModal = {};
			if (!app.globalSettingsModal.encryption) app.globalSettingsModal.encryption = {};
			app.globalSettingsModal.encryption.old_password = '   ';
			app.removeConfigPassword();
			app.showToast = origToast;

			// Test Explorer Row Action Buttons (Inline Preview/Copy/Delete decluttered, Download and More Actions present)
			const downloadButtonsCount = document.querySelectorAll('button[title="Download file"]').length;
			const moreActionsCount = document.querySelectorAll('button[title="More actions"]').length;
			const inlinePreviewCount = document.querySelectorAll('button[title="Preview file"]').length;
			const inlineCopyCount = document.querySelectorAll('button[title*="Copy to opposing"]').length;
			const inlineDeleteCount = document.querySelectorAll('button[title*="Delete from"]').length;

			// Test Explorer Upload Triggers
			if (typeof app.triggerUpload !== 'function') return { error: "triggerUpload missing" };
			const uploadLeftInputExists = Boolean(document.getElementById('explorer-upload-left'));
			const uploadRightInputExists = Boolean(document.getElementById('explorer-upload-right'));

			return {
				popoverOpen,
				popoverClosed,
				presetsLen: presets.length,
				formattedSpeed,
				formattedBytes,
				encryptionTabActive,
				encryptionLoading,
				previewModalOpen,
				previewContent,
				previewModalClosed,
				contextMenuOpen,
				contextMenuItemName,
				contextMenuClosed,
				encryptionWarningShown,
				downloadButtonsCount,
				moreActionsCount,
				inlinePreviewCount,
				inlineCopyCount,
				inlineDeleteCount,
				uploadLeftInputExists,
				uploadRightInputExists
			};
		})()`

		type M1Result struct {
			Error                  *string `json:"error"`
			PopoverOpen            bool    `json:"popoverOpen"`
			PopoverClosed          bool    `json:"popoverClosed"`
			PresetsLen             int     `json:"presetsLen"`
			FormattedSpeed         string  `json:"formattedSpeed"`
			FormattedBytes         string  `json:"formattedBytes"`
			EncryptionTabActive    bool    `json:"encryptionTabActive"`
			EncryptionLoading      *bool   `json:"encryptionLoading"`
			PreviewModalOpen       bool    `json:"previewModalOpen"`
			PreviewContent         string  `json:"previewContent"`
			PreviewModalClosed     bool    `json:"previewModalClosed"`
			ContextMenuOpen        bool    `json:"contextMenuOpen"`
			ContextMenuItemName    string  `json:"contextMenuItemName"`
			ContextMenuClosed      bool    `json:"contextMenuClosed"`
			EncryptionWarningShown bool    `json:"encryptionWarningShown"`
			DownloadButtonsCount   int     `json:"downloadButtonsCount"`
			MoreActionsCount       int     `json:"moreActionsCount"`
			InlinePreviewCount     int     `json:"inlinePreviewCount"`
			InlineCopyCount        int     `json:"inlineCopyCount"`
			InlineDeleteCount      int     `json:"inlineDeleteCount"`
			UploadLeftInputExists  bool    `json:"uploadLeftInputExists"`
			UploadRightInputExists bool    `json:"uploadRightInputExists"`
		}

		res, err := chromedp.Run(ctx, chromedp.Evaluate[M1Result](script))
		if err != nil {
			t.Fatalf("failed evaluating Milestone 1 controls: %v", err)
		}
		if res.Error != nil {
			t.Fatalf("Milestone 1 check failed: %s", *res.Error)
		}
		if !res.PopoverOpen || !res.PopoverClosed {
			t.Errorf("bandwidth popover open/close mismatch: open=%v closed=%v", res.PopoverOpen, res.PopoverClosed)
		}
		if res.PresetsLen < 5 {
			t.Errorf("expected at least 5 bandwidth presets, got %d", res.PresetsLen)
		}
		if res.FormattedSpeed != "10.0 MB/s" {
			t.Errorf("expected formattedSpeed '10.0 MB/s', got %q", res.FormattedSpeed)
		}
		if res.FormattedBytes != "5.0 MB" {
			t.Errorf("expected formattedBytes '5.0 MB', got %q", res.FormattedBytes)
		}
		if !res.EncryptionTabActive {
			t.Errorf("expected encryption tab to be active in global settings")
		}
		if res.EncryptionLoading == nil || *res.EncryptionLoading {
			t.Errorf("expected encryption loading to be false")
		}
		if !res.PreviewModalOpen || res.PreviewContent != "Hello Bisync UI Suite" || !res.PreviewModalClosed {
			t.Errorf("preview modal mismatch: open=%v content=%q closed=%v", res.PreviewModalOpen, res.PreviewContent, res.PreviewModalClosed)
		}
		if !res.ContextMenuOpen || res.ContextMenuItemName != "test.txt" || !res.ContextMenuClosed {
			t.Errorf("context menu mismatch: open=%v name=%q closed=%v", res.ContextMenuOpen, res.ContextMenuItemName, res.ContextMenuClosed)
		}
		if !res.EncryptionWarningShown {
			t.Errorf("expected empty passphrase encryption removal to trigger warning toast")
		}
		if res.InlinePreviewCount != 0 || res.InlineCopyCount != 0 || res.InlineDeleteCount != 0 {
			t.Errorf("expected zero inline preview/copy/delete buttons in explorer rows: preview=%d copy=%d delete=%d",
				res.InlinePreviewCount, res.InlineCopyCount, res.InlineDeleteCount)
		}
		if !res.UploadLeftInputExists || !res.UploadRightInputExists {
			t.Errorf("upload input triggers missing: left=%v right=%v", res.UploadLeftInputExists, res.UploadRightInputExists)
		}

	})
	// 7. Test Milestone 2 Operational Controls (Move/Purge Modals, Upload Dock, Review Workspace Remediation Bridge)
	t.Run("Milestone2OperationalControls", func(t *testing.T) {
		script := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return { error: "Alpine root not found" };

			// Test Move Modal opening and closing
			if (typeof app.promptMoveItem !== 'function') return { error: "promptMoveItem missing" };
			app.promptMoveItem('left', { Name: 'document.pdf', Path: 'document.pdf', IsDir: false });
			const moveModalOpen = Boolean(app.moveModal?.show);
			const moveModalItem = app.moveModal?.item?.Name;
			app.cancelMove();
			const moveModalClosed = !app.moveModal?.show;

			// Test Purge Directory prompt method presence
			if (typeof app.promptPurgeDirectory !== 'function') return { error: "promptPurgeDirectory missing" };

			// Test Upload Dock Modal opening and closing
			if (typeof app.openUploadDockModal !== 'function') return { error: "openUploadDockModal missing" };
			app.openUploadDockModal();
			const uploadDockOpen = Boolean(app.uploadDockModal?.show);
			app.closeUploadDockModal();
			const uploadDockClosed = !app.uploadDockModal?.show;

			// Test Remediation inspectInWorkspace navigation helper
			if (typeof app.inspectInWorkspace !== 'function') return { error: "inspectInWorkspace missing" };
			app.reviewState = {
				active: true,
				session: {
					id: 'sess-1',
					profile_id: 'default',
					conflicts: [
						{ id: 'c1', path: 'projects/alpha/report.docx', decision: 'pending' }
					],
					deletions: [
						{ id: 'd1', path: 'trash/old_log.txt', decision: 'pending' }
					]
				}
			};
			const cnfCandidate = app.reviewCandidates.find(c => c.type === 'conflict');
			app.inspectInWorkspace(cnfCandidate);
			const candidateInspectedTab = app.currentTab;
			const candidateHighlighted = app.explorer?.highlightName === 'report.docx';
			const candidateDirNavigated = app.explorer?.left?.path === 'projects/alpha';

			// Test Context Menu Remediation lookup
			const candidateLookup = app.getReviewCandidate('left', { Name: 'report.docx' });
			const hasCandidateMatch = Boolean(candidateLookup && candidateLookup.id === 'c1');

			return {
				moveModalOpen,
				moveModalItem,
				moveModalClosed,
				uploadDockOpen,
				uploadDockClosed,
				candidateInspectedTab,
				candidateHighlighted,
				candidateDirNavigated,
				hasCandidateMatch
			};
		})()`

		type M2Result struct {
			Error                 *string `json:"error"`
			MoveModalOpen         bool    `json:"moveModalOpen"`
			MoveModalItem         string  `json:"moveModalItem"`
			MoveModalClosed       bool    `json:"moveModalClosed"`
			UploadDockOpen        bool    `json:"uploadDockOpen"`
			UploadDockClosed      bool    `json:"uploadDockClosed"`
			CandidateInspectedTab string  `json:"candidateInspectedTab"`
			CandidateHighlighted  bool    `json:"candidateHighlighted"`
			CandidateDirNavigated bool    `json:"candidateDirNavigated"`
			HasCandidateMatch     bool    `json:"hasCandidateMatch"`
		}

		res, err := chromedp.Run(ctx, chromedp.Evaluate[M2Result](script))
		if err != nil {
			t.Fatalf("failed evaluating Milestone 2 script: %v", err)
		}
		if res.Error != nil {
			t.Fatalf("Milestone 2 check failed: %s", *res.Error)
		}
		if !res.MoveModalOpen || res.MoveModalItem != "document.pdf" || !res.MoveModalClosed {
			t.Errorf("move modal mismatch: open=%v item=%q closed=%v", res.MoveModalOpen, res.MoveModalItem, res.MoveModalClosed)
		}
		if !res.UploadDockOpen || !res.UploadDockClosed {
			t.Errorf("upload dock mismatch: open=%v closed=%v", res.UploadDockOpen, res.UploadDockClosed)
		}
		if res.CandidateInspectedTab != "explorer" || !res.CandidateHighlighted || !res.CandidateDirNavigated {
			t.Errorf("inspect in workspace mismatch: tab=%q highlighted=%v dirNavigated=%v",
				res.CandidateInspectedTab, res.CandidateHighlighted, res.CandidateDirNavigated)
		}
		if !res.HasCandidateMatch {
			t.Errorf("expected review candidate match for report.docx")
		}
	})
	// 8. Test Open on Remote Capability Filtering
	t.Run("OpenOnRemoteCapabilityFiltering", func(t *testing.T) {
		script := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return { error: "Alpine root not found" };

			if (typeof app.canOpenOnRemote !== 'function') {
				return { error: "canOpenOnRemote method missing" };
			}

			// Mock remotes data while saving originals
			const origRemotesData = app.remotesData;
			const origLeftTarget = app.explorer?.left?.target;
			const origRightTarget = app.explorer?.right?.target;

			app.remotesData = {
				remotes: [
					{ name: 'drive_remote', type: 'drive', can_public_link: true },
					{ name: 'sftp_remote', type: 'sftp', can_public_link: false },
					{ name: 'box_remote', type: 'box' } // fallback whitelist
				]
			};

			if (app.explorer?.left) app.explorer.left.target = 'local';
			if (app.explorer?.right) app.explorer.right.target = 'drive_remote:';

			const localSupported = app.canOpenOnRemote('left');
			const driveSupported = app.canOpenOnRemote('right');

			if (app.explorer?.right) app.explorer.right.target = 'sftp_remote:';
			const sftpSupported = app.canOpenOnRemote('right');

			if (app.explorer?.right) app.explorer.right.target = 'box_remote:';
			const boxSupported = app.canOpenOnRemote('right');

			// Restore originals
			app.remotesData = origRemotesData;
			if (app.explorer?.left) app.explorer.left.target = origLeftTarget;
			if (app.explorer?.right) app.explorer.right.target = origRightTarget;

			return {
				localSupported,
				driveSupported,
				sftpSupported,
				boxSupported
			};
		})()`

		type CapResult struct {
			Error          *string `json:"error"`
			LocalSupported bool    `json:"localSupported"`
			DriveSupported bool    `json:"driveSupported"`
			SftpSupported  bool    `json:"sftpSupported"`
			BoxSupported   bool    `json:"boxSupported"`
		}

		res, err := chromedp.Run(ctx, chromedp.Evaluate[CapResult](script))
		if err != nil {
			t.Fatalf("failed evaluating OpenOnRemote script: %v", err)
		}
		if res.Error != nil {
			t.Fatalf("OpenOnRemote check failed: %s", *res.Error)
		}
		if res.LocalSupported {
			t.Errorf("expected local target to return false for canOpenOnRemote")
		}
		if !res.DriveSupported {
			t.Errorf("expected drive_remote to return true for canOpenOnRemote")
		}
		if res.SftpSupported {
			t.Errorf("expected sftp_remote to return false for canOpenOnRemote")
		}
		if !res.BoxSupported {
			t.Errorf("expected box_remote to return true for canOpenOnRemote via whitelist fallback")
		}
	})
	// 9. Test Cockpit UI/UX Consolidation Controls (Quick-Create, System Health Modal, Wizard Auth Toggle, Canary Protection)
	t.Run("CockpitConsolidationControls", func(t *testing.T) {
		script := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return { error: "Alpine root not found" };

			// Test Quick-Create Menu toggle
			app.quickCreateOpen = true;
			const quickCreateOpened = app.quickCreateOpen === true;
			app.quickCreateOpen = false;
			const quickCreateClosed = app.quickCreateOpen === false;

			// Test System Health Modal tabs
			if (typeof app.openSystemHealthModal !== 'function') return { error: "openSystemHealthModal missing" };
			app.openSystemHealthModal('daemon');
			const daemonTabOpen = app.systemHealthModal?.show === true && app.systemHealthModal?.tab === 'daemon';
			app.openSystemHealthModal('mcp');
			const mcpTabOpen = app.systemHealthModal?.show === true && app.systemHealthModal?.tab === 'mcp';
			app.openSystemHealthModal('poller');
			const pollerTabOpen = app.systemHealthModal?.show === true && app.systemHealthModal?.tab === 'poller';
			app.closeSystemHealthModal();
			const healthModalClosed = app.systemHealthModal?.show === false;

			// Test First-Launch Onboarding condition
			const onboardingVisibleWithZero = (app.profilesData?.list?.length === 0);

			// Test Remote Creation Wizard Segmented Auth Toggle
			app.createRemoteModal.authMode = 'custom';
			const authTypeCustom = app.createRemoteModal?.authMode === 'custom';
			app.createRemoteModal.authMode = 'oauth';
			const authTypeOAuth = app.createRemoteModal?.authMode === 'oauth';

			// Test MCP Tool Registry Accordion & Filter
			const mcpTotalTools = (app.mcpTools || []).length;
			app.systemHealthModal.mcpFilter = 'read';
			const mcpReadCount = (app.filteredMcpTools || []).length;
			app.systemHealthModal.mcpFilter = 'write';
			const mcpWriteCount = (app.filteredMcpTools || []).length;
			app.systemHealthModal.mcpFilter = 'all';
			app.systemHealthModal.mcpSearch = 'trigger_sync';
			const mcpSearchCount = (app.filteredMcpTools || []).length;
			app.systemHealthModal.mcpSearch = '';
			if (typeof app.toggleMcpTool === 'function') {
				app.toggleMcpTool('bisync_get_status');
			}
			const mcpExpanded = app.systemHealthModal.mcpExpandedTool === 'bisync_get_status';

			// Test Onboarding Fleet Remote Navigation & Badges
			const origTab = app.currentTab;
			const hasRemotes = app.hasConfiguredRemotes;
			if (typeof app.navigateToRemotes === 'function') {
				app.navigateToRemotes();
			}
			const navigatedToRemotes = app.currentTab === 'remotes';
			if (typeof app.switchTab === 'function') {
				app.switchTab(origTab || 'fleet');
			}

			// Test Explorer Canary drag guard
			if (typeof app.onDragStart !== 'function') return { error: "onDragStart missing" };
			app.explorer.draggedItem = null;
			app.onDragStart('left', { Name: 'RCLONE_TEST', Path: 'RCLONE_TEST', IsDir: false }, { preventDefault() {} });
			const canaryDragBlocked = app.explorer.draggedItem === null;

			// Test Explorer Remote Unlock & Breadcrumb Location Bars
			let remoteLockToggled = false;
			if (typeof app.toggleExplorerRemoteLock === 'function') {
				const beforeLock = app.isExplorerRemoteLocked();
				const origSave = app.saveGlobalSettings;
				app.saveGlobalSettings = () => Promise.resolve();
				app.toggleExplorerRemoteLock();
				remoteLockToggled = app.isExplorerRemoteLocked() !== beforeLock;
				app.toggleExplorerRemoteLock();
				app.saveGlobalSettings = origSave;
			}
			const breadcrumbBarsCount = document.querySelectorAll('.no-scrollbar').length;

			// Test Directional Transfer Buttons presence
			const transferToRemoteBtn = Array.from(document.querySelectorAll('button')).some(b => b.innerText && b.innerText.includes('Transfer to Remote'));
			const transferToLocalBtn = Array.from(document.querySelectorAll('button')).some(b => b.innerText && b.innerText.includes('Transfer to Local'));

			return {
				quickCreateOpened,
				quickCreateClosed,
				daemonTabOpen,
				mcpTabOpen,
				pollerTabOpen,
				healthModalClosed,
				onboardingVisibleWithZero,
				authTypeCustom,
				authTypeOAuth,
				mcpTotalTools,
				mcpReadCount,
				mcpWriteCount,
				mcpSearchCount,
				mcpExpanded,
				hasRemotes,
				navigatedToRemotes,
				canaryDragBlocked,
				remoteLockToggled,
				breadcrumbBarsCount,
				transferToRemoteBtn,
				transferToLocalBtn
			};
		})()`

		type CockpitResult struct {
			Error                     *string `json:"error"`
			QuickCreateOpened         bool    `json:"quickCreateOpened"`
			QuickCreateClosed         bool    `json:"quickCreateClosed"`
			DaemonTabOpen             bool    `json:"daemonTabOpen"`
			McpTabOpen                bool    `json:"mcpTabOpen"`
			PollerTabOpen             bool    `json:"pollerTabOpen"`
			HealthModalClosed         bool    `json:"healthModalClosed"`
			OnboardingVisibleWithZero bool    `json:"onboardingVisibleWithZero"`
			AuthTypeCustom            bool    `json:"authTypeCustom"`
			AuthTypeOAuth             bool    `json:"authTypeOAuth"`
			McpTotalTools             int     `json:"mcpTotalTools"`
			McpReadCount              int     `json:"mcpReadCount"`
			McpWriteCount             int     `json:"mcpWriteCount"`
			McpSearchCount            int     `json:"mcpSearchCount"`
			McpExpanded               bool    `json:"mcpExpanded"`
			HasRemotes                bool    `json:"hasRemotes"`
			NavigatedToRemotes        bool    `json:"navigatedToRemotes"`
			CanaryDragBlocked         bool    `json:"canaryDragBlocked"`
			RemoteLockToggled         bool    `json:"remoteLockToggled"`
			BreadcrumbBarsCount       int     `json:"breadcrumbBarsCount"`
			TransferToRemoteBtn       bool    `json:"transferToRemoteBtn"`
			TransferToLocalBtn        bool    `json:"transferToLocalBtn"`
		}

		res, err := chromedp.Run(ctx, chromedp.Evaluate[CockpitResult](script))
		if err != nil {
			t.Fatalf("failed evaluating Cockpit Consolidation script: %v", err)
		}
		if res.Error != nil {
			t.Fatalf("Cockpit Consolidation check failed: %s", *res.Error)
		}
		if !res.QuickCreateOpened || !res.QuickCreateClosed {
			t.Errorf("quick-create menu toggle mismatch: open=%v close=%v", res.QuickCreateOpened, res.QuickCreateClosed)
		}
		if !res.DaemonTabOpen || !res.McpTabOpen || !res.PollerTabOpen || !res.HealthModalClosed {
			t.Errorf("system health modal mismatch: daemon=%v mcp=%v poller=%v closed=%v",
				res.DaemonTabOpen, res.McpTabOpen, res.PollerTabOpen, res.HealthModalClosed)
		}
		if !res.OnboardingVisibleWithZero {
			t.Errorf("expected onboarding card to be visible when profiles count is 0")
		}
		if !res.AuthTypeCustom || !res.AuthTypeOAuth {
			t.Errorf("wizard auth type toggle mismatch: custom=%v oauth=%v", res.AuthTypeCustom, res.AuthTypeOAuth)
		}
		if res.McpTotalTools != 24 {
			t.Errorf("expected 24 MCP tools registered in System Health modal, got %d", res.McpTotalTools)
		}
		if res.McpReadCount != 10 {
			t.Errorf("expected 10 read-only MCP tools, got %d", res.McpReadCount)
		}
		if res.McpWriteCount != 14 {
			t.Errorf("expected 14 mutating MCP tools, got %d", res.McpWriteCount)
		}
		if res.McpSearchCount != 1 {
			t.Errorf("expected 1 tool matching 'trigger_sync', got %d", res.McpSearchCount)
		}
		if !res.McpExpanded {
			t.Errorf("expected toggleMcpTool to expand bisync_get_status")
		}
		if !res.HasRemotes {
			t.Errorf("expected hasConfiguredRemotes to be true with testremote configured")
		}
		if !res.NavigatedToRemotes {
			t.Errorf("expected navigateToRemotes to switch currentTab to remotes")
		}
		if !res.CanaryDragBlocked {
			t.Errorf("expected dragging RCLONE_TEST canary to be blocked")
		}
		if !res.RemoteLockToggled {
			t.Errorf("expected toggleExplorerRemoteLock to toggle remote lock state")
		}
		if res.BreadcrumbBarsCount < 2 {
			t.Errorf("expected at least 2 breadcrumb location bars, got %d", res.BreadcrumbBarsCount)
		}
		if !res.TransferToRemoteBtn || !res.TransferToLocalBtn {
			t.Errorf("expected directional transfer buttons: toRemote=%v toLocal=%v", res.TransferToRemoteBtn, res.TransferToLocalBtn)
		}
	})
	// 10. Test Deep File Path Navigation & Pinned Log Audit Dock Engine
	t.Run("DeepPathNavigationAndLogDock", func(t *testing.T) {
		script := `(async () => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return { error: "Alpine root not found" };

			// Test contextual path tokenization
			const path1 = extractPathFromLogMsg("Copied (new) to: documents/reports/final.pdf");
			const path2 = extractPathFromLogMsg("Deleting: old/backup.tar.gz");
			const path3 = extractPathFromLogMsg("drive:data/photos/trip.jpg: Conflict identified");
			const tokenizationValid = (path1 === "documents/reports/final.pdf" &&
				path2 === "old/backup.tar.gz" &&
				path3 === "data/photos/trip.jpg");

			// Test getEntryPath
			const ep1 = app.getEntryPath({ object: "files/data.csv" });
			const ep2 = app.getEntryPath({ msg: "Transferred: notes/meeting.txt" });
			const entryPathValid = (ep1 === "files/data.csv" && ep2 === "notes/meeting.txt");

			// Test Log Dock toggle and height presets
			if (typeof app.toggleLogDock !== 'function') return { error: "toggleLogDock missing" };
			app.toggleLogDock(true);
			const dockOpened = app.logDock?.open === true;
			app.setLogDockHeight('expanded');
			const dockExpanded = app.logDock?.heightPreset === 'expanded';
			app.setLogDockHeight('docked');
			const dockDocked = app.logDock?.heightPreset === 'docked';
			app.setLogDockHeight('collapsed');
			const dockCollapsed = app.logDock?.heightPreset === 'collapsed';
			app.toggleLogDock(false);
			const dockClosed = app.logDock?.open === false;

			// Test Log Dock filtering with whitespace-padded levels and stringified JSON logs
			app.telemetry.bisync_logs = [
				{ timestamp: "2026-11-11T20:00:00Z", level: "NOTICE  ", msg: "Transferred: notes/a.txt" },
				'{"time":"2026-11-11T20:00:01Z","level":"ERROR   ","msg":"Error reading data"}',
				{ timestamp: "2026-11-11T20:00:02Z", level: "INFO    ", msg: "Engine idle" }
			];
			app.bisyncPaginatedLogs = [];
			app.logDock.activeFilter = 'all';
			const allLogsCount = app.getFilteredDockLogs().length;
			app.logDock.activeFilter = 'mutations';
			const mutationLogsCount = app.getFilteredDockLogs().length;
			app.logDock.activeFilter = 'file_ops';
			const fileOpsLogsCount = app.getFilteredDockLogs().length;
			app.logDock.activeFilter = 'errors';
			const errorLogsCount = app.getFilteredDockLogs().length;
			const fetchDaemonLogsExists = typeof app.fetchDaemonLogs === 'function';

			// Verify deep subfolder path navigation
			await app.jumpToExplorerPath('documents/reports/q4/summary.pdf');
			const deepNavDirValid = (app.explorer?.left?.path === 'documents/reports/q4');
			const deepNavHighlightValid = (app.explorer?.highlightName === 'summary.pdf');

			// Verify multi-file input attributes and onDrop handler
			const uploadLeftMultiple = document.getElementById('explorer-upload-left')?.hasAttribute('multiple') === true;
			const uploadRightMultiple = document.getElementById('explorer-upload-right')?.hasAttribute('multiple') === true;
			const hasOnDrop = typeof app.onDrop === 'function';

			// Test pulse row highlight styling
			app.explorer.highlightName = 'sample.txt';
			const highlightClass = app.getItemReviewRowClass('left', { Name: 'sample.txt' });
			const nonHighlightClass = app.getItemReviewRowClass('left', { Name: 'other.txt' });
			const highlightPulseActive = highlightClass.includes('animate-pulse');
			const nonHighlightPulseInactive = !nonHighlightClass.includes('animate-pulse');
			app.explorer.highlightName = null;

			return {
				tokenizationValid,
				entryPathValid,
				dockOpened,
				dockExpanded,
				dockDocked,
				dockCollapsed,
				dockClosed,
				allLogsCount,
				mutationLogsCount,
				fileOpsLogsCount,
				errorLogsCount,
				deepNavDirValid,
				deepNavHighlightValid,
				uploadLeftMultiple,
				uploadRightMultiple,
				hasOnDrop,
				fetchDaemonLogsExists,
				highlightPulseActive,
				nonHighlightPulseInactive
			};
		})()`

		type DeepNavResult struct {
			Error                     *string `json:"error"`
			TokenizationValid         bool    `json:"tokenizationValid"`
			EntryPathValid            bool    `json:"entryPathValid"`
			DockOpened                bool    `json:"dockOpened"`
			DockExpanded              bool    `json:"dockExpanded"`
			DockDocked                bool    `json:"dockDocked"`
			DockCollapsed             bool    `json:"dockCollapsed"`
			DockClosed                bool    `json:"dockClosed"`
			AllLogsCount              int     `json:"allLogsCount"`
			MutationLogsCount         int     `json:"mutationLogsCount"`
			FileOpsLogsCount          int     `json:"fileOpsLogsCount"`
			ErrorLogsCount            int     `json:"errorLogsCount"`
			DeepNavDirValid           bool    `json:"deepNavDirValid"`
			DeepNavHighlightValid     bool    `json:"deepNavHighlightValid"`
			UploadLeftMultiple        bool    `json:"uploadLeftMultiple"`
			UploadRightMultiple       bool    `json:"uploadRightMultiple"`
			HasOnDrop                 bool    `json:"hasOnDrop"`
			FetchDaemonLogsExists     bool    `json:"fetchDaemonLogsExists"`
			HighlightPulseActive      bool    `json:"highlightPulseActive"`
			NonHighlightPulseInactive bool    `json:"nonHighlightPulseInactive"`
		}

		res, err := chromedp.Run(ctx, chromedp.Evaluate[DeepNavResult](script, chromedp.EvalAwaitPromise))
		if err != nil {
			t.Fatalf("failed evaluating Deep Path Nav script: %v", err)
		}
		if res.Error != nil {
			t.Fatalf("Deep Path Nav check failed: %s", *res.Error)
		}
		if !res.TokenizationValid {
			t.Errorf("expected extractPathFromLogMsg to properly tokenize paths")
		}
		if !res.EntryPathValid {
			t.Errorf("expected getEntryPath to extract paths from object and msg")
		}
		if !res.DockOpened || !res.DockExpanded || !res.DockDocked || !res.DockCollapsed || !res.DockClosed {
			t.Errorf("log dock toggle/preset mismatch: open=%v expanded=%v docked=%v collapsed=%v closed=%v",
				res.DockOpened, res.DockExpanded, res.DockDocked, res.DockCollapsed, res.DockClosed)
		}
		if res.AllLogsCount != 3 || res.MutationLogsCount != 2 || res.ErrorLogsCount != 1 {
			t.Errorf("unexpected filtered log dock counts: all=%d mutations=%d errors=%d",
				res.AllLogsCount, res.MutationLogsCount, res.ErrorLogsCount)
		}
		if res.FileOpsLogsCount != 1 {
			t.Errorf("unexpected file_ops log dock count: got %d, want 1", res.FileOpsLogsCount)
		}
		if !res.DeepNavDirValid || !res.DeepNavHighlightValid {
			t.Errorf("deep path navigation failed: dirValid=%v, highlightValid=%v", res.DeepNavDirValid, res.DeepNavHighlightValid)
		}
		if !res.UploadLeftMultiple || !res.UploadRightMultiple {
			t.Errorf("multi-file input attribute missing: left=%v, right=%v", res.UploadLeftMultiple, res.UploadRightMultiple)
		}
		if !res.HasOnDrop {
			t.Errorf("expected onDrop drag-and-drop handler to exist on Alpine instance")
		}
		if !res.FetchDaemonLogsExists {
			t.Errorf("expected fetchDaemonLogs method to exist on Alpine instance")
		}
		if !res.HighlightPulseActive || !res.NonHighlightPulseInactive {
			t.Errorf("row pulse class mismatch: active=%v inactive=%v", res.HighlightPulseActive, res.NonHighlightPulseInactive)
		}
	})

	// 11. Test Timeline Ledger Clustering & Auto-Refresh Hook
	t.Run("AuditTimelineLedgerAndClustering", func(t *testing.T) {
		script := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return { error: "Alpine root not found" };

			// Test clusterLogEntries keeps distinct objects uncollapsed
			const testAuditEntries = [
				{ id: 1, timestamp: "2026-11-13T10:00:00Z", level: "NOTICE", msg: "[profile1] UPLOAD: fileA.txt", action: "UPLOAD", object: "fileA.txt", details: {} },
				{ id: 2, timestamp: "2026-11-13T10:00:01Z", level: "NOTICE", msg: "[profile1] UPLOAD: fileB.txt", action: "UPLOAD", object: "fileB.txt", details: {} }
			];
			const clusters = app.clusterLogEntries(testAuditEntries);
			const distinctCount = clusters.length;

			// Test same object/action clusters
			const testDuplicateEntries = [
				{ id: 3, timestamp: "2026-11-13T10:00:00Z", level: "NOTICE", msg: "[profile1] UPLOAD: fileA.txt", action: "UPLOAD", object: "fileA.txt", details: {} },
				{ id: 4, timestamp: "2026-11-13T10:00:02Z", level: "NOTICE", msg: "[profile1] UPLOAD: fileA.txt", action: "UPLOAD", object: "fileA.txt", details: {} }
			];
			const dupClusters = app.clusterLogEntries(testDuplicateEntries);
			const dupCount = dupClusters.length;

			// Test fetchBisyncLogs and fetchAuditRecords methods exist
			const hasFetchBisyncLogs = typeof app.fetchBisyncLogs === 'function';
			const hasFetchAuditRecords = typeof app.fetchAuditRecords === 'function';

			return {
				distinctCount,
				dupCount,
				hasFetchBisyncLogs,
				hasFetchAuditRecords
			};
		})()`

		type ClusterResult struct {
			Error                *string `json:"error"`
			DistinctCount        int     `json:"distinctCount"`
			DupCount             int     `json:"dupCount"`
			HasFetchBisyncLogs   bool    `json:"hasFetchBisyncLogs"`
			HasFetchAuditRecords bool    `json:"hasFetchAuditRecords"`
		}

		res, err := chromedp.Run(ctx, chromedp.Evaluate[ClusterResult](script))
		if err != nil {
			t.Fatalf("failed evaluating Audit Timeline Ledger script: %v", err)
		}
		if res.Error != nil {
			t.Fatalf("Audit Timeline Ledger check failed: %s", *res.Error)
		}
		if res.DistinctCount != 2 {
			t.Errorf("expected 2 distinct clusters for different objects, got %d", res.DistinctCount)
		}
		if res.DupCount != 1 {
			t.Errorf("expected 1 cluster for identical object and action, got %d", res.DupCount)
		}
		if !res.HasFetchBisyncLogs || !res.HasFetchAuditRecords {
			t.Errorf("expected fetchBisyncLogs and fetchAuditRecords to exist on Alpine instance")
		}
	})

	// 12. Test Log Dock Pagination, Sync File Ops & Deep Navigation
	t.Run("LogDockPaginationAndSyncFiles", func(t *testing.T) {
		script := `(async () => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return { error: "Alpine root not found" };

			const syncFileLog = {
				timestamp: "2026-11-13T10:00:00Z",
				level: "INFO",
				type: "sync_file",
				action: "COPY",
				object: "docs/report.pdf",
				msg: "Transferred: docs/report.pdf"
			};
			const syncOverflowLog = {
				timestamp: "2026-11-13T10:00:01Z",
				level: "NOTICE",
				type: "sync_overflow",
				msg: "Transferred files exceed display limit (50). View detailed logs in Audit."
			};
			const deleteFileLog = {
				timestamp: "2026-11-13T10:00:02Z",
				level: "INFO",
				type: "sync_file",
				action: "DELETE",
				object: "trash/old.txt",
				msg: "Deleted: trash/old.txt"
			};

			const syncFileIsOp = app.isFileOperationLog(syncFileLog);
			const overflowIsOp = app.isFileOperationLog(syncOverflowLog);
			const syncFilePath = app.getEntryPath(syncFileLog);

			const copyBadgeClass = app.getDockActionBadgeClass(syncFileLog);
			const copyLabel = app.getDockActionLabel(syncFileLog);
			const deleteBadgeClass = app.getDockActionBadgeClass(deleteFileLog);
			const deleteLabel = app.getDockActionLabel(deleteFileLog);

			const badgeClassValid = copyBadgeClass.includes("emerald") && deleteBadgeClass.includes("rose");
			const badgeLabelValid = copyLabel === "SYNC: COPY" && deleteLabel === "SYNC: DELETE";

			const mockLogs = [];
			for (let i = 1; i <= 65; i++) {
				mockLogs.push({
					id: "mock_" + i,
					timestamp: "2026-11-13T10:00:" + String(i).padStart(2, "0") + "Z",
					level: "INFO",
					type: "sync_file",
					action: "COPY",
					object: "items/item_" + i + ".dat",
					msg: "Copied items/item_" + i + ".dat"
				});
			}
			app.telemetry.bisync_logs = mockLogs;
			app.setLogDockFilter('all');
			app.logDock.pageSize = 50;

			const totalEntries = app.getDockTotalEntries();
			const totalPages = app.getDockTotalPages();
			const p1PrevDisabled = app.isDockPrevDisabled();
			const p1NextDisabled = app.isDockNextDisabled();
			const p1Count = app.getFilteredDockLogs().length;

			app.loadDockNextPage();
			const curPage = app.logDock.page;
			const p2PrevDisabled = app.isDockPrevDisabled();
			const p2NextDisabled = app.isDockNextDisabled();
			const p2Count = app.getFilteredDockLogs().length;

			app.loadDockPrevPage();
			const backToP1 = app.logDock.page === 1;

			app.logDock.page = 1;
			app.logDock.selectedIdx = 49;
			app.logDockStep(1);
			const steppedToPage2 = app.logDock.page === 2 && app.logDock.selectedIdx === 0;

			app.logDockStep(-1);
			const steppedBackToPage1 = app.logDock.page === 1 && app.logDock.selectedIdx === 49;
			const hasNavigateToExplorer = typeof app.navigateToExplorer === "function";

			return {
				syncFileIsOp, overflowIsOp, syncFilePath, badgeClassValid, badgeLabelValid,
				totalEntries, totalPages, p1PrevDisabled, p1NextDisabled, p1Count,
				curPage, p2PrevDisabled, p2NextDisabled, p2Count, backToP1,
				steppedToPage2, steppedBackToPage1, hasNavigateToExplorer
			};
		})()`
		type DockPaginationResult struct {
			Error                 *string `json:"error"`
			SyncFileIsOp          bool    `json:"syncFileIsOp"`
			OverflowIsOp          bool    `json:"overflowIsOp"`
			SyncFilePath          string  `json:"syncFilePath"`
			BadgeClassValid       bool    `json:"badgeClassValid"`
			BadgeLabelValid       bool    `json:"badgeLabelValid"`
			TotalEntries          int     `json:"totalEntries"`
			TotalPages            int     `json:"totalPages"`
			P1PrevDisabled        bool    `json:"p1PrevDisabled"`
			P1NextDisabled        bool    `json:"p1NextDisabled"`
			P1Count               int     `json:"p1Count"`
			CurPage               int     `json:"curPage"`
			P2PrevDisabled        bool    `json:"p2PrevDisabled"`
			P2NextDisabled        bool    `json:"p2NextDisabled"`
			P2Count               int     `json:"p2Count"`
			BackToP1              bool    `json:"backToP1"`
			SteppedToPage2        bool    `json:"steppedToPage2"`
			SteppedBackToPage1    bool    `json:"steppedBackToPage1"`
			HasNavigateToExplorer bool    `json:"hasNavigateToExplorer"`
		}

		res, err := chromedp.Run(ctx, chromedp.Evaluate[DockPaginationResult](script, chromedp.EvalAwaitPromise))
		if err != nil {
			t.Fatalf("failed evaluating Log Dock Pagination script: %v", err)
		}
		if res.Error != nil {
			t.Fatalf("Log Dock Pagination check failed: %s", *res.Error)
		}
		if !res.SyncFileIsOp || !res.OverflowIsOp {
			t.Errorf("expected sync_file and sync_overflow to be recognized as file operations")
		}
		if res.SyncFilePath != "docs/report.pdf" {
			t.Errorf("expected getEntryPath to return 'docs/report.pdf', got %q", res.SyncFilePath)
		}
		if !res.BadgeClassValid || !res.BadgeLabelValid {
			t.Errorf("action badge class or label invalid: class=%v label=%v", res.BadgeClassValid, res.BadgeLabelValid)
		}
		if res.TotalEntries != 65 || res.TotalPages != 2 {
			t.Errorf("pagination count mismatch: totalEntries=%d, totalPages=%d (expected 65 and 2)", res.TotalEntries, res.TotalPages)
		}
		if !res.P1PrevDisabled || res.P1NextDisabled || res.P1Count != 50 {
			t.Errorf("page 1 state invalid: prevDisabled=%v, nextDisabled=%v, count=%d", res.P1PrevDisabled, res.P1NextDisabled, res.P1Count)
		}
		if res.CurPage != 2 || res.P2PrevDisabled || !res.P2NextDisabled || res.P2Count != 15 {
			t.Errorf("page 2 state invalid: page=%d, prevDisabled=%v, nextDisabled=%v, count=%d", res.CurPage, res.P2PrevDisabled, res.P2NextDisabled, res.P2Count)
		}
		if !res.BackToP1 {
			t.Errorf("expected loadDockPrevPage to navigate back to page 1")
		}
		if !res.SteppedToPage2 || !res.SteppedBackToPage1 {
			t.Errorf("boundary step mismatch: steppedToPage2=%v steppedBackToPage1=%v", res.SteppedToPage2, res.SteppedBackToPage1)
		}
		if !res.HasNavigateToExplorer {
			t.Errorf("expected navigateToExplorer helper method to exist")
		}
	})

	// 6. Check for any runtime JavaScript console errors or exceptions
	mu.Lock()
	defer mu.Unlock()

	var significantErrors []string
	for _, ce := range consoleErrors {
		if strings.Contains(ce, "Telemetry poll failed") {
			continue
		}
		significantErrors = append(significantErrors, ce)
	}

	if len(significantErrors) > 0 {
		t.Errorf("encountered %d JavaScript console errors/exceptions during smoke test:\n%s",
			len(significantErrors), strings.Join(significantErrors, "\n"))
	}
}

// TestDashboard_HelpDrawerAndGuidance validates the slide-over reference manual drawer,
// contextual guidance, live diagnostic checks query, tab switching, and keyboard triggers.
func TestDashboard_HelpDrawerAndGuidance(t *testing.T) {
	chromePath := findChromeBinary()
	if chromePath == "" {
		t.Skip("skipping E2E smoke test: Chrome/Chromium binary not found in PATH")
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
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer mockRCServer.Close()

	cfg := &config.Config{
		LocalPath:   filepath.Join(tempDir, "local"),
		RemotePath:  "testremote:",
		BaseDir:     tempDir,
		MetricsFile: filepath.Join(tempDir, "metrics.json"),
		RCAPIURL:    mockRCServer.URL,
	}
	_ = os.MkdirAll(cfg.LocalPath, 0755)

	syncSvc := services.NewSyncService(cfg, nil)
	srv := server.NewServer(cfg, syncSvc, os.DirFS(repoRoot))
	srv.Version = "v2.0.0-e2e"

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
			if m.Type == "error" || m.IsException() {
				mu.Lock()
				consoleErrors = append(consoleErrors, m.String())
				mu.Unlock()
			}
		}
	}()

	err := chromedp.Do(ctx,
		chromedp.Navigate(ts.URL),
		chromedp.WaitVisible("body[x-data]"),
		chromedp.WaitVisible("#app-header"),
	)
	if err != nil {
		t.Fatalf("failed to load dashboard: %v", err)
	}

	time.Sleep(800 * time.Millisecond)

	// Sub-test 1: Header Help Button & Initial Drawer Opening
	t.Run("HelpHeaderButtonOpening", func(t *testing.T) {
		openScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app || typeof app.openHelpDrawer !== 'function') return false;
			app.openHelpDrawer('diagnostics');
			return app.helpDrawer.open;
		})()`
		opened, err := chromedp.Run(ctx, chromedp.Evaluate[bool](openScript))
		if err != nil || !opened {
			t.Fatalf("failed to open help drawer via Alpine controller: %v", err)
		}
	})

	// Sub-test 2: Live Diagnostics Audit Fetching
	t.Run("DiagnosticsAuditFetch", func(t *testing.T) {
		fetchScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return false;
			app.fetchDiagnostics();
			return true;
		})()`
		_, err := chromedp.Run(ctx, chromedp.Evaluate[bool](fetchScript))
		if err != nil {
			t.Fatalf("failed calling fetchDiagnostics: %v", err)
		}

		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			done, err := chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
				const app = Alpine.$data(document.querySelector('[x-data]'));
				return app && !app.helpDrawer.loading && app.helpDrawer.diagnostics !== null;
			})()`))
			if err == nil && done {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		type DiagStatus struct {
			Status     string `json:"status"`
			CheckCount int    `json:"checkCount"`
			HasSysInfo bool   `json:"hasSysInfo"`
		}
		statusScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			const d = app?.helpDrawer?.diagnostics;
			if (!d) return { status: "", checkCount: 0, hasSysInfo: false };
			return {
				status: d.status || "",
				checkCount: Array.isArray(d.checks) ? d.checks.length : 0,
				hasSysInfo: Boolean(d.system_info && d.system_info.go_version)
			};
		})()`
		ds, err := chromedp.Run(ctx, chromedp.Evaluate[DiagStatus](statusScript))
		if err != nil {
			t.Fatalf("failed reading diagnostic state: %v", err)
		}
		if ds.Status == "" {
			t.Errorf("expected diagnostic status to be populated, got empty")
		}
		if ds.CheckCount == 0 {
			t.Errorf("expected at least 1 diagnostic check returned, got 0")
		}
		if !ds.HasSysInfo {
			t.Errorf("expected diagnostic system_info to contain runtime details")
		}
	})

	// Sub-test 3: Drawer Tab Navigation & Search Filters
	t.Run("DrawerTabsAndSearch", func(t *testing.T) {
		tabsAuditScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return null;

			app.setHelpTab('safety');
			const safetyCount = app.filteredPolicies().length;

			app.helpDrawer.search = 'circuit';
			const safetyFiltered = app.filteredPolicies().length;
			app.helpDrawer.search = '';

			app.setHelpTab('manual');
			const flagCount = app.filteredFlags().length;

			app.helpDrawer.search = '--resync';
			const flagFiltered = app.filteredFlags().length;
			app.helpDrawer.search = '';

			app.setHelpTab('runbooks');
			const runbookCount = app.filteredRunbooks().length;

			app.helpDrawer.search = 'OAuth';
			const runbookFiltered = app.filteredRunbooks().length;
			app.helpDrawer.search = '';

			return {
				safetyCount,
				safetyFiltered,
				flagCount,
				flagFiltered,
				runbookCount,
				runbookFiltered
			};
		})()`

		type TabAuditResult struct {
			SafetyCount     int `json:"safetyCount"`
			SafetyFiltered  int `json:"safetyFiltered"`
			FlagCount       int `json:"flagCount"`
			FlagFiltered    int `json:"flagFiltered"`
			RunbookCount    int `json:"runbookCount"`
			RunbookFiltered int `json:"runbookFiltered"`
		}

		res, err := chromedp.Run(ctx, chromedp.Evaluate[TabAuditResult](tabsAuditScript))
		if err != nil {
			t.Fatalf("failed auditing drawer tabs and search: %v", err)
		}
		if res.SafetyCount < 3 || res.SafetyFiltered != 1 {
			t.Errorf("unexpected safety policy counts: total=%d, filtered=%d", res.SafetyCount, res.SafetyFiltered)
		}
		if res.FlagCount < 18 || res.FlagFiltered != 1 {
			t.Errorf("unexpected flag counts: total=%d, filtered=%d (expected >= 18 flags)", res.FlagCount, res.FlagFiltered)
		}
		if res.RunbookCount < 3 || res.RunbookFiltered != 1 {
			t.Errorf("unexpected runbook counts: total=%d, filtered=%d", res.RunbookCount, res.RunbookFiltered)
		}
	})

	// Sub-test 4: Drawer Closing and Global Window Trigger
	t.Run("DrawerCloseAndGlobalTrigger", func(t *testing.T) {
		closeAndTriggerScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return false;

			app.closeHelpDrawer();
			if (app.helpDrawer.open) return false;

			if (typeof window.openHelpDrawer === 'function') {
				window.openHelpDrawer('manual', '--dry-run');
			}
			return app.helpDrawer.open && app.helpDrawer.tab === 'manual' && app.helpDrawer.search === '--dry-run';
		})()`
		success, err := chromedp.Run(ctx, chromedp.Evaluate[bool](closeAndTriggerScript))
		if err != nil || !success {
			t.Fatalf("failed testing drawer close and global trigger: %v", err)
		}

		// Close after test
		_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (app) app.closeHelpDrawer();
			return true;
		})()`))
	})

	// Assert no console errors
	// Sub-test 5: Context-Aware Detection, Scoped Filtering & Scoped Toggle
	t.Run("ContextAwareDetectionAndFiltering", func(t *testing.T) {
		contextScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return null;

			// Close any open modals and drawer
			app.closeHelpDrawer();
			app.currentTab = 'remotes';

			// 1. Detect active tab context
			const remotesCtx = app.detectActiveContext();

			// 2. Open drawer in context of remotes tab
			app.openHelpDrawer();
			const activeCtxId = app.helpDrawer.contextId || '';
			const isScoped = app.helpDrawer.contextOnly;
			const initialTab = app.helpDrawer.tab;
			const scopedFlagCount = app.filteredFlags().length;

			// 3. Toggle off scopedOnly
			app.toggleContextFilter();
			const allFlagCount = app.filteredFlags().length;

			// 4. Reset drawer
			app.closeHelpDrawer();

			return {
				remotesCtxId: remotesCtx ? remotesCtx.id : '',
				activeCtxId,
				isScoped,
				initialTab,
				scopedFlagCount,
				allFlagCount
			};
		})()`

		type ContextAuditResult struct {
			RemotesCtxId    string `json:"remotesCtxId"`
			ActiveCtxId     string `json:"activeCtxId"`
			IsScoped        bool   `json:"isScoped"`
			InitialTab      string `json:"initialTab"`
			ScopedFlagCount int    `json:"scopedFlagCount"`
			AllFlagCount    int    `json:"allFlagCount"`
		}

		res, err := chromedp.Run(ctx, chromedp.Evaluate[ContextAuditResult](contextScript))
		if err != nil {
			t.Fatalf("failed testing context detection and filtering: %v", err)
		}
		if res.RemotesCtxId != "view_remotes" {
			t.Errorf("expected remotesCtxId to be 'view_remotes', got %q", res.RemotesCtxId)
		}
		if res.ActiveCtxId != "view_remotes" {
			t.Errorf("expected activeCtxId to be 'view_remotes', got %q", res.ActiveCtxId)
		}
		if !res.IsScoped {
			t.Errorf("expected isScoped to be true upon opening drawer")
		}
		if res.InitialTab != "runbooks" {
			t.Errorf("expected initialTab for view_remotes to be 'runbooks', got %q", res.InitialTab)
		}
		if res.ScopedFlagCount >= res.AllFlagCount || res.ScopedFlagCount == 0 {
			t.Errorf("expected scopedFlagCount (%d) to be > 0 and < allFlagCount (%d)", res.ScopedFlagCount, res.AllFlagCount)
		}
	})

	// Sub-test 6: Modal Header Contextual Help Triggers
	t.Run("ModalHeaderContextTriggers", func(t *testing.T) {
		modalTriggerScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return null;

			// Verify presence of modal header buttons in DOM
			const profileBtn = Array.from(document.querySelectorAll('button')).find(b => b.getAttribute('@click')?.includes("openHelpDrawer('modal_profiles')"));
			const reviewBtn = Array.from(document.querySelectorAll('button')).find(b => b.getAttribute('@click')?.includes("openHelpDrawer('modal_review')"));
			const settingsBtn = Array.from(document.querySelectorAll('button')).find(b => b.getAttribute('@click')?.includes("openHelpDrawer('modal_global_settings')"));

			// Test Profiles Modal Trigger
			app.closeHelpDrawer();
			app.profileModal.show = true;
			const detectedProfileCtx = app.detectActiveContext();

			app.openHelpDrawer('modal_profiles');
			const profileDrawerOpen = app.helpDrawer.open;
			const profileActiveId = app.helpDrawer.contextId || '';
			const profileTab = app.helpDrawer.tab;
			const profilePolicies = app.filteredPolicies().length;

			// Test Review Modal Trigger
			app.closeHelpDrawer();
			app.profileModal.show = false;
			app.reviewState.showModal = true;
			const detectedReviewCtx = app.detectActiveContext();

			app.openHelpDrawer('modal_review');
			const reviewDrawerOpen = app.helpDrawer.open;
			const reviewActiveId = app.helpDrawer.contextId || '';
			const reviewTab = app.helpDrawer.tab;

			// Test Global Settings Modal Trigger
			app.closeHelpDrawer();
			app.reviewState.showModal = false;
			app.globalSettingsModal.show = true;
			const detectedSettingsCtx = app.detectActiveContext();

			app.openHelpDrawer('modal_global_settings');
			const settingsDrawerOpen = app.helpDrawer.open;
			const settingsActiveId = app.helpDrawer.contextId || '';
			const settingsTab = app.helpDrawer.tab;

			// Cleanup
			app.closeHelpDrawer();
			app.globalSettingsModal.show = false;

			return {
				hasProfileBtn: Boolean(profileBtn),
				hasReviewBtn: Boolean(reviewBtn),
				hasSettingsBtn: Boolean(settingsBtn),
				detectedProfileCtxId: detectedProfileCtx ? detectedProfileCtx.id : '',
				profileDrawerOpen,
				profileActiveId,
				profileTab,
				profilePolicies,
				detectedReviewCtxId: detectedReviewCtx ? detectedReviewCtx.id : '',
				reviewDrawerOpen,
				reviewActiveId,
				reviewTab,
				detectedSettingsCtxId: detectedSettingsCtx ? detectedSettingsCtx.id : '',
				settingsDrawerOpen,
				settingsActiveId,
				settingsTab
			};
		})()`

		type ModalAuditResult struct {
			HasProfileBtn         bool   `json:"hasProfileBtn"`
			HasReviewBtn          bool   `json:"hasReviewBtn"`
			HasSettingsBtn        bool   `json:"hasSettingsBtn"`
			DetectedProfileCtxId  string `json:"detectedProfileCtxId"`
			ProfileDrawerOpen     bool   `json:"profileDrawerOpen"`
			ProfileActiveId       string `json:"profileActiveId"`
			ProfileTab            string `json:"profileTab"`
			ProfilePolicies       int    `json:"profilePolicies"`
			DetectedReviewCtxId   string `json:"detectedReviewCtxId"`
			ReviewDrawerOpen      bool   `json:"reviewDrawerOpen"`
			ReviewActiveId        string `json:"reviewActiveId"`
			ReviewTab             string `json:"reviewTab"`
			DetectedSettingsCtxId string `json:"detectedSettingsCtxId"`
			SettingsDrawerOpen    bool   `json:"settingsDrawerOpen"`
			SettingsActiveId      string `json:"settingsActiveId"`
			SettingsTab           string `json:"settingsTab"`
		}

		res, err := chromedp.Run(ctx, chromedp.Evaluate[ModalAuditResult](modalTriggerScript))
		if err != nil {
			t.Fatalf("failed testing modal header context triggers: %v", err)
		}

		if !res.HasProfileBtn || !res.HasReviewBtn || !res.HasSettingsBtn {
			t.Errorf("modal contextual help buttons missing in DOM: profile=%v, review=%v, settings=%v",
				res.HasProfileBtn, res.HasReviewBtn, res.HasSettingsBtn)
		}

		if res.DetectedProfileCtxId != "modal_profiles" || res.ProfileActiveId != "modal_profiles" {
			t.Errorf("profile context mismatch: detected=%q, active=%q", res.DetectedProfileCtxId, res.ProfileActiveId)
		}
		if !res.ProfileDrawerOpen || res.ProfileTab != "manual" {
			t.Errorf("profile drawer state invalid: open=%v, tab=%q", res.ProfileDrawerOpen, res.ProfileTab)
		}
		if res.ProfilePolicies == 0 {
			t.Errorf("expected scoped profile policies, got 0")
		}

		if res.DetectedReviewCtxId != "modal_review" || res.ReviewActiveId != "modal_review" {
			t.Errorf("review context mismatch: detected=%q, active=%q", res.DetectedReviewCtxId, res.ReviewActiveId)
		}
		if !res.ReviewDrawerOpen || res.ReviewTab != "safety" {
			t.Errorf("review drawer state invalid: open=%v, tab=%q", res.ReviewDrawerOpen, res.ReviewTab)
		}

		if res.DetectedSettingsCtxId != "modal_global_settings" || res.SettingsActiveId != "modal_global_settings" {
			t.Errorf("settings context mismatch: detected=%q, active=%q", res.DetectedSettingsCtxId, res.SettingsActiveId)
		}
		if !res.SettingsDrawerOpen || res.SettingsTab != "safety" {
			t.Errorf("settings drawer state invalid: open=%v, tab=%q", res.SettingsDrawerOpen, res.SettingsTab)
		}
	})
	// Sub-test 7: Doc Link Rendering and Integrity
	t.Run("DocLinkRenderingAndIntegrity", func(t *testing.T) {
		docLinksScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return null;

			// Switch to manual tab and clear search to see all flags
			app.openHelpDrawer('manual');
			app.helpDrawer.search = '';
			app.helpDrawer.contextOnly = false;

			const flags = app.filteredFlags();
			const flagsWithDocUrl = flags.filter(f => Boolean(f.docUrl));
			const rcloneOrgUrls = flagsWithDocUrl.filter(f => f.docUrl && f.docUrl.startsWith('https://rclone.org/'));

			// Query rendered DOM anchor elements within manual tab
			const links = Array.from(document.querySelectorAll('a[href^="https://rclone.org/"]'));
			const validRelCount = links.filter(l => l.getAttribute('rel') === 'noopener noreferrer' && l.getAttribute('target') === '_blank').length;

			// Cleanup
			app.closeHelpDrawer();

			return {
				totalFlags: flags.length,
				flagsWithDocUrl: flagsWithDocUrl.length,
				rcloneOrgUrls: rcloneOrgUrls.length,
				renderedLinksCount: links.length,
				validRelCount
			};
		})()`

		type DocLinksResult struct {
			TotalFlags         int `json:"totalFlags"`
			FlagsWithDocUrl    int `json:"flagsWithDocUrl"`
			RcloneOrgUrls      int `json:"rcloneOrgUrls"`
			RenderedLinksCount int `json:"renderedLinksCount"`
			ValidRelCount      int `json:"validRelCount"`
		}

		res, err := chromedp.Run(ctx, chromedp.Evaluate[DocLinksResult](docLinksScript))
		if err != nil {
			t.Fatalf("failed verifying doc links: %v", err)
		}
		if res.TotalFlags < 18 {
			t.Errorf("expected at least 18 total flags in catalog, got %d", res.TotalFlags)
		}
		if res.FlagsWithDocUrl < 18 {
			t.Errorf("expected at least 18 flags to have docUrl, got %d", res.FlagsWithDocUrl)
		}
		if res.RcloneOrgUrls != res.FlagsWithDocUrl {
			t.Errorf("expected all docUrls to target rclone.org, got %d/%d", res.RcloneOrgUrls, res.FlagsWithDocUrl)
		}
	})

	// Sub-test 8: Provider-Aware Flag Scoping and Drawer Isolation
	t.Run("ProviderAwareFlagScopingAndDrawerIsolation", func(t *testing.T) {
		providerScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return null;

			// 1. Open profile creation modal with a OneDrive remote
			app.closeHelpDrawer();
			app.startNewProfile();
			app.profileModal.editing.remote = "work-onedrive:";

			const oneDriveType = app.getEditingRemoteType();
			const hasNoVersionsInCatalog = Boolean(app.flagsCatalog.find(c => c.flag === "--onedrive-no-versions"));
			const hasDeltaInCatalog = Boolean(app.flagsCatalog.find(c => c.flag === "--onedrive-delta"));
			const hasSharedWithMeInCatalog = Boolean(app.flagsCatalog.find(c => c.flag === "--drive-shared-with-me"));

			// 2. Open drawer in context of profiles modal (scoped by default)
			app.openHelpDrawer('modal_profiles');
			const activeProviderOne = app.getActiveProviderFilter();
			const oneDriveFlags = app.filteredFlags().map(f => f.flag);
			const hasOneDriveFlag = oneDriveFlags.includes("--onedrive-no-versions");
			const hasOneDriveDeltaFlag = oneDriveFlags.includes("--onedrive-delta");
			const hasDriveFlagOnOneDrive = oneDriveFlags.includes("--drive-shared-with-me");

			// 3. Toggle off contextOnly: all flags should now appear in help drawer
			app.toggleContextFilter();
			const allFlags = app.filteredFlags().map(f => f.flag);
			const allHasOneDrive = allFlags.includes("--onedrive-no-versions");
			const allHasOneDriveDelta = allFlags.includes("--onedrive-delta");
			const allHasDrive = allFlags.includes("--drive-shared-with-me");

			// 4. Switch to Google Drive remote with contextOnly restored
			app.toggleContextFilter(); // toggle back to scoped
			app.profileModal.editing.remote = "team-gdrive:";
			const driveType = app.getEditingRemoteType();
			const driveFlags = app.filteredFlags().map(f => f.flag);
			const hasDriveFlag = driveFlags.includes("--drive-shared-with-me");
			const hasOneDriveOnDrive = driveFlags.includes("--onedrive-no-versions");
			const hasOneDriveDeltaOnDrive = driveFlags.includes("--onedrive-delta");

			// Cleanup
			app.closeHelpDrawer();
			if (typeof app.closeProfilesModal === 'function') {
				app.closeProfilesModal();
			}

			return {
				oneDriveType,
				hasNoVersionsInCatalog,
				hasDeltaInCatalog,
				hasSharedWithMeInCatalog,
				activeProviderOne,
				hasOneDriveFlag,
				hasOneDriveDeltaFlag,
				hasDriveFlagOnOneDrive,
				allHasOneDrive,
				allHasOneDriveDelta,
				allHasDrive,
				driveType,
				hasDriveFlag,
				hasOneDriveOnDrive,
				hasOneDriveDeltaOnDrive
			};
		})()`

		type ProviderTestResult struct {
			OneDriveType             string `json:"oneDriveType"`
			HasNoVersionsInCatalog   bool   `json:"hasNoVersionsInCatalog"`
			HasDeltaInCatalog        bool   `json:"hasDeltaInCatalog"`
			HasSharedWithMeInCatalog bool   `json:"hasSharedWithMeInCatalog"`
			ActiveProviderOne        string `json:"activeProviderOne"`
			HasOneDriveFlag          bool   `json:"hasOneDriveFlag"`
			HasOneDriveDeltaFlag     bool   `json:"hasOneDriveDeltaFlag"`
			HasDriveFlagOnOneDrive   bool   `json:"hasDriveFlagOnOneDrive"`
			AllHasOneDrive           bool   `json:"allHasOneDrive"`
			AllHasOneDriveDelta      bool   `json:"allHasOneDriveDelta"`
			AllHasDrive              bool   `json:"allHasDrive"`
			DriveType                string `json:"driveType"`
			HasDriveFlag             bool   `json:"hasDriveFlag"`
			HasOneDriveOnDrive       bool   `json:"hasOneDriveOnDrive"`
			HasOneDriveDeltaOnDrive  bool   `json:"hasOneDriveDeltaOnDrive"`
		}

		res, err := chromedp.Run(ctx, chromedp.Evaluate[ProviderTestResult](providerScript))
		if err != nil {
			t.Fatalf("failed evaluating provider-aware flag scoping: %v", err)
		}

		if res.OneDriveType != "onedrive" {
			t.Errorf("expected oneDriveType to be 'onedrive', got %q", res.OneDriveType)
		}
		if res.HasNoVersionsInCatalog {
			t.Errorf("expected --onedrive-no-versions to be omitted from custom flag builder (curated in Target Provider Options)")
		}
		if res.HasDeltaInCatalog {
			t.Errorf("expected --onedrive-delta to be omitted from custom flag builder (curated in Target Provider Options)")
		}
		if res.HasSharedWithMeInCatalog {
			t.Errorf("expected --drive-shared-with-me to be omitted from custom flag builder (curated in Target Provider Options)")
		}
		if res.ActiveProviderOne != "onedrive" {
			t.Errorf("expected activeProviderOne to be 'onedrive', got %q", res.ActiveProviderOne)
		}
		if !res.HasOneDriveFlag || !res.HasOneDriveDeltaFlag {
			t.Errorf("expected help drawer filteredFlags() to include OneDrive flags for onedrive")
		}
		if res.HasDriveFlagOnOneDrive {
			t.Errorf("expected help drawer filteredFlags() to exclude --drive-shared-with-me for onedrive")
		}
		if !res.AllHasOneDrive || !res.AllHasOneDriveDelta || !res.AllHasDrive {
			t.Errorf("expected all provider flags to be visible in help drawer when scoped filter is disabled")
		}
		if res.DriveType != "drive" {
			t.Errorf("expected driveType to be 'drive', got %q", res.DriveType)
		}
		if !res.HasDriveFlag {
			t.Errorf("expected help drawer filteredFlags() to include --drive-shared-with-me for drive")
		}
		if res.HasOneDriveOnDrive || res.HasOneDriveDeltaOnDrive {
			t.Errorf("expected help drawer filteredFlags() to exclude OneDrive flags for drive")
		}
	})

	// Sub-test 9: Declarative Provider Options Engine Parity
	t.Run("DeclarativeProviderOptionsEngineParity", func(t *testing.T) {
		declarativeScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return null;

			// Verify schema presence
			const schemaLen = Array.isArray(window.PROVIDER_OPTIONS_SCHEMA) ? window.PROVIDER_OPTIONS_SCHEMA.length : 0;

			// Test onedrive
			app.startNewProfile();
			app.profileModal.editing.remote = "my-onedrive:";
			const isOneDriveTier1 = app.isTier1Provider();
			const odBoolOpts = app.getProviderBoolOptionsList().map(o => o.key);
			const odNonBoolOpts = app.getProviderNonBoolOptionsList().map(o => o.key);

			// Test drive
			app.profileModal.editing.remote = "my-gdrive:";
			const isDriveTier1 = app.isTier1Provider();
			const driveBoolOpts = app.getProviderBoolOptionsList().map(o => o.key);
			const driveNonBoolOpts = app.getProviderNonBoolOptionsList().map(o => o.key);

			// Test s3
			app.profileModal.editing.remote = "my-s3:";
			const isS3Tier1 = app.isTier1Provider();
			const s3NonBoolOpts = app.getProviderNonBoolOptionsList().map(o => o.key);

			// Test generic remote
			app.profileModal.editing.remote = "sftp-backup:";
			const isGenericTier1 = app.isTier1Provider();

			// Test command synthesis with provider options
			app.profileModal.editing.remote = "work-onedrive:";
			app.setProviderOption('onedrive_delta', 'true');
			app.setProviderOption('onedrive_chunk_size', '20M');
			const synthesized = app.generateCLIPreview();

			// Cleanup
			if (typeof app.closeProfilesModal === 'function') {
				app.closeProfilesModal();
			}

			return {
				schemaLen,
				isOneDriveTier1,
				hasOneDriveDeltaBool: odBoolOpts.includes('onedrive_delta'),
				hasOneDriveChunkNonBool: odNonBoolOpts.includes('onedrive_chunk_size'),
				isDriveTier1,
				hasDriveSkipShortcutsBool: driveBoolOpts.includes('drive_skip_shortcuts'),
				hasDriveExportNonBool: driveNonBoolOpts.includes('drive_export_formats'),
				isS3Tier1,
				hasS3StorageClassNonBool: s3NonBoolOpts.includes('s3_storage_class'),
				isGenericTier1,
				synthesizedHasDelta: synthesized.includes('--onedrive-delta'),
				synthesizedHasChunk: synthesized.includes('--onedrive-chunk-size 20M')
			};
		})()`

		type DeclarativeResult struct {
			SchemaLen                 int  `json:"schemaLen"`
			IsOneDriveTier1           bool `json:"isOneDriveTier1"`
			HasOneDriveDeltaBool      bool `json:"hasOneDriveDeltaBool"`
			HasOneDriveChunkNonBool   bool `json:"hasOneDriveChunkNonBool"`
			IsDriveTier1              bool `json:"isDriveTier1"`
			HasDriveSkipShortcutsBool bool `json:"hasDriveSkipShortcutsBool"`
			HasDriveExportNonBool     bool `json:"hasDriveExportNonBool"`
			IsS3Tier1                 bool `json:"isS3Tier1"`
			HasS3StorageClassNonBool  bool `json:"hasS3StorageClassNonBool"`
			IsGenericTier1            bool `json:"isGenericTier1"`
			SynthesizedHasDelta       bool `json:"synthesizedHasDelta"`
			SynthesizedHasChunk       bool `json:"synthesizedHasChunk"`
		}

		dRes, err := chromedp.Run(ctx, chromedp.Evaluate[DeclarativeResult](declarativeScript))
		if err != nil {
			t.Fatalf("failed evaluating declarative provider options: %v", err)
		}

		if dRes.SchemaLen == 0 {
			t.Errorf("expected non-empty PROVIDER_OPTIONS_SCHEMA")
		}
		if !dRes.IsOneDriveTier1 || !dRes.HasOneDriveDeltaBool || !dRes.HasOneDriveChunkNonBool {
			t.Errorf("onedrive declarative options mismatch: %+v", dRes)
		}
		if !dRes.IsDriveTier1 || !dRes.HasDriveSkipShortcutsBool || !dRes.HasDriveExportNonBool {
			t.Errorf("drive declarative options mismatch: %+v", dRes)
		}
		if !dRes.IsS3Tier1 || !dRes.HasS3StorageClassNonBool {
			t.Errorf("s3 declarative options mismatch: %+v", dRes)
		}
		if dRes.IsGenericTier1 {
			t.Errorf("expected sftp-backup to not be a Tier 1 provider")
		}
		if !dRes.SynthesizedHasDelta || !dRes.SynthesizedHasChunk {
			t.Errorf("synthesized bisync command missing provider flags: %+v", dRes)
		}
	})

	mu.Lock()
	defer mu.Unlock()
	var significantErrors []string
	for _, ce := range consoleErrors {
		if strings.Contains(ce, "Telemetry poll failed") {
			continue
		}
		significantErrors = append(significantErrors, ce)
	}
	if len(significantErrors) > 0 {
		t.Errorf("encountered %d JavaScript console errors during help drawer E2E test:\n%s",
			len(significantErrors), strings.Join(significantErrors, "\n"))
	}
}
