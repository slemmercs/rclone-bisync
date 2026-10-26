package e2e

import (
	"context"
	"fmt"
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

	cfg := &config.Config{
		LocalPath:   filepath.Join(tempDir, "local"),
		RemotePath:  "testremote:",
		BaseDir:     tempDir,
		MetricsFile: filepath.Join(tempDir, "metrics.json"),
		RCAPIURL:    "http://127.0.0.1:5572",
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
		chromedp.WaitVisible("header"),
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

	// 2. Audit each cockpit tab
	tabs := []struct {
		name       string
		tabPillIdx int
	}{
		{"telemetry", 0},
		{"remotes", 1},
		{"explorer", 2},
		{"audit", 3},
	}

	for _, tc := range tabs {
		t.Run("Tab_"+tc.name, func(t *testing.T) {
			clickScript := fmt.Sprintf(`(() => {
				const navButtons = Array.from(document.querySelectorAll('header button'));
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
		})
	}

	// 3. Check for any runtime JavaScript console errors or exceptions
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
