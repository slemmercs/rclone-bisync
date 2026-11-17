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
	srv.Version = "v2.0.0-vue3-e2e"

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

	// 9. Final Console Error Audit
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
