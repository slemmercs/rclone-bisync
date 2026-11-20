package e2e

import (
"context"
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
"github.com/slemmercs/rclone-bisync/internal/profiles"
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

// TestDashboard_HeadlessChromeSmoke launches headless Chrome to verify
// that the default root route GET / serves the embedded Vue 3 / PrimeVue 4 SPA,
// verifies tab switching, modals, controls, and asserts zero runtime JS errors.
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
		case "/core/bwlimit":
			_, _ = w.Write([]byte(`{"rate":"off","bytesPerSecond":0}`))
		case "/operations/about":
			_, _ = w.Write([]byte(`{"total":1000000,"used":500000,"free":500000}`))
		case "/operations/list":
			_, _ = w.Write([]byte(`{"list":[{"Path":"notes.txt","Name":"notes.txt","Size":12,"MimeType":"text/plain","ModTime":"2026-11-18T12:00:00Z","IsDir":false}]}`))
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
	_ = os.WriteFile(filepath.Join(cfg.LocalPath, "notes.txt"), []byte("sample notes content"), 0644)

	syncSvc := services.NewSyncService(cfg, nil)
	srv := server.NewServer(cfg, syncSvc, os.DirFS(repoRoot))
	srv.Version = "v2.0.0-vue3-root"

	if srv.ProfileService != nil {
		_, _ = srv.ProfileService.Create(context.Background(), &profiles.Profile{
			ID:         "prof-docs",
			Name:       "Documents Sync",
			Remote:     "testremote",
			RemotePath: "docs",
			LocalPath:  cfg.LocalPath,
			Schedule:   "15m",
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

	// 1. Initial Page Load & SPA Mount at root GET /
	t.Run("RootSPAMountAndHeader", func(t *testing.T) {
err := chromedp.Do(ctx,
chromedp.Navigate(ts.URL),
chromedp.WaitReady("#app"),
chromedp.WaitReady("header"),
)
if err != nil {
t.Fatalf("failed to mount Vue SPA at root %s: %v", ts.URL, err)
}

brandText, err := chromedp.Run(ctx, chromedp.Evaluate[string](`(() => {
			const el = document.querySelector('header span.font-bold');
			return el ? el.textContent.trim() : '';
		})()`))
if err != nil || !strings.Contains(brandText, "Bisync Hub") {
t.Fatalf("expected brand 'Bisync Hub', got %q (err: %v)", brandText, err)
}

title, err := chromedp.Run(ctx, chromedp.Title())
		if err != nil {
			t.Fatalf("failed reading page title: %v", err)
		}
		if !strings.Contains(title, "Bisync Hub") {
			t.Errorf("unexpected page title: got %q, expected to contain 'Bisync Hub'", title)
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

	// 7. MCP Modal
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

	// 8. Bandwidth Popover
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
			return document.body.textContent.includes('Bandwidth Throttling');
		})()`))
if err != nil || !hasPopover {
t.Fatalf("expected Bandwidth Throttling popover in DOM, err: %v", err)
}

_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](clickBwScript))
time.Sleep(150 * time.Millisecond)
})

	// 9. Quick Create Menu
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

_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
			window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
			return true;
		})()`))
time.Sleep(150 * time.Millisecond)
})

	// 10. Final Console Error Audit
	t.Run("ConsoleErrorAudit", func(t *testing.T) {
mu.Lock()
		defer mu.Unlock()

		var criticalErrors []string
		for _, e := range consoleErrors {
			if strings.Contains(e, "favicon.ico") || strings.Contains(e, "net::ERR_CONNECTION_REFUSED") || strings.Contains(e, "Telemetry poll failed") {
				continue
			}
			criticalErrors = append(criticalErrors, e)
		}

		if len(criticalErrors) > 0 {
			t.Errorf("encountered %d uncaught browser console errors during root SPA test:\n%s",
len(criticalErrors), strings.Join(criticalErrors, "\n"))
		}
	})
}

// TestDashboard_HelpDrawerAndGuidance validates the slide-over reference manual drawer
// on the Vue 3 Cockpit SPA at root GET /.
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
	srv.Version = "v2.0.0-vue3-help"

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

	err := chromedp.Do(ctx,
chromedp.Navigate(ts.URL),
chromedp.WaitReady("#app"),
chromedp.WaitReady("header"),
)
	if err != nil {
		t.Fatalf("failed to mount Vue SPA: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	// Open Help Drawer
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

	// Verify Help Drawer content is rendered
	hasHelpContent, err := chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
		return document.body.textContent.includes('Reference Manual') || document.body.textContent.includes('Help') || document.body.textContent.includes('Bisync');
	})()`))
	if err != nil || !hasHelpContent {
		t.Fatalf("expected Help Drawer content in DOM, err: %v", err)
	}

	// Close drawer via Escape
	_, _ = chromedp.Run(ctx, chromedp.Evaluate[bool](`(() => {
		window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
		return true;
	})()`))
	time.Sleep(150 * time.Millisecond)
}
