package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
	"github.com/slemmercs/rclone-bisync/internal/server"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

type sandboxHarness struct {
	sandboxDir string
	localDir   string
	remoteDir  string
	rcClient   *client.RcloneClient
	rcAddr     string
	serverURL  string
	ts         *httptest.Server
	cfg        *config.Config
}

func setupSandboxHarness(t *testing.T) *sandboxHarness {
	t.Helper()

	if _, err := exec.LookPath("rclone"); err != nil {
		t.Skip("rclone binary not found in PATH, skipping sandbox UI journey test")
	}

	sandboxDir := t.TempDir()
	cacheDir := filepath.Join(sandboxDir, "cache")
	localDir := filepath.Join(sandboxDir, "local")
	remoteDir := filepath.Join(sandboxDir, "remote")
	confPath := filepath.Join(sandboxDir, "rclone.conf")

	_ = os.MkdirAll(cacheDir, 0700)
	_ = os.MkdirAll(localDir, 0750)
	_ = os.MkdirAll(remoteDir, 0750)

	// Write isolated rclone.conf defining a local remote "sandrem"
	confContent := "[sandrem]\ntype = local\n"
	if err := os.WriteFile(confPath, []byte(confContent), 0600); err != nil {
		t.Fatalf("failed to write sandbox rclone.conf: %v", err)
	}

	// Seed local directory items
	_ = os.WriteFile(filepath.Join(localDir, "alpha.txt"), []byte("alpha payload"), 0600)
	_ = os.WriteFile(filepath.Join(localDir, "to_rename.txt"), []byte("rename payload"), 0600)
	_ = os.MkdirAll(filepath.Join(localDir, "docs"), 0750)
	_ = os.WriteFile(filepath.Join(localDir, "docs", "guide.pdf"), []byte("guide content"), 0600)

	// Seed remote directory items
	_ = os.WriteFile(filepath.Join(remoteDir, "cloud_data.txt"), []byte("cloud payload"), 0600)
	purgeTargetDir := filepath.Join(remoteDir, "purge_target")
	_ = os.MkdirAll(purgeTargetDir, 0750)
	_ = os.WriteFile(filepath.Join(purgeTargetDir, "obsolete.log"), []byte("obsolete"), 0600)
	canaryDir := filepath.Join(remoteDir, "protected_canary")
	_ = os.MkdirAll(canaryDir, 0750)
	_ = os.WriteFile(filepath.Join(canaryDir, "RCLONE_TEST"), []byte("canary-guard"), 0600)

	// Allocate dynamic loopback TCP port for rclone rcd
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate free port: %v", err)
	}
	rcPort := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	rcAddr := fmt.Sprintf("127.0.0.1:%d", rcPort)
	cmd := exec.Command("rclone", "rcd",
		"--rc-addr", rcAddr,
		"--rc-no-auth",
		"--config", confPath,
		"--cache-dir", cacheDir,
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start sandbox rclone rcd: %v", err)
	}

	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	})

	rcClient := client.NewRcloneClient(fmt.Sprintf("http://%s", rcAddr), "", "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ok, err := rcClient.WaitForHealthy(ctx, 10*time.Second)
	if !ok || err != nil {
		t.Fatalf("sandbox rclone daemon failed to reach healthy state: %v", err)
	}

	// Configure Bisync Hub profiles
	profilesPath := filepath.Join(sandboxDir, "profiles.json")
	initialProfiles := []*profiles.Profile{
		{
			ID:         "sandbox-prof",
			Name:       "Sandbox Profile",
			Remote:     "sandrem:",
			RemotePath: remoteDir,
			LocalPath:  localDir,
			Enabled:    true,
			Schedule:   "manual",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
	}
	profBytes, _ := json.MarshalIndent(initialProfiles, "", "  ")
	_ = os.WriteFile(profilesPath, profBytes, 0600)

	cfg := &config.Config{
		LocalPath:           localDir,
		RemotePath:          fmt.Sprintf("sandrem:%s", remoteDir),
		BaseDir:             sandboxDir,
		ProfilesFile:        profilesPath,
		ProfilesHistoryFile: filepath.Join(sandboxDir, "profiles_history.json"),
		AuditDBPath:         filepath.Join(sandboxDir, "audit.db"),
		MetricsFile:         filepath.Join(sandboxDir, "metrics.json"),
		RCAPIURL:            fmt.Sprintf("http://%s", rcAddr),
	}

	repoRoot := findRepoRoot(t)
	syncSvc := services.NewSyncService(cfg, rcClient)
	srv := server.NewServer(cfg, syncSvc, os.DirFS(repoRoot))
	srv.Version = "v2.0.0-sandbox"

	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)

	return &sandboxHarness{
		sandboxDir: sandboxDir,
		localDir:   localDir,
		remoteDir:  remoteDir,
		rcClient:   rcClient,
		rcAddr:     rcAddr,
		serverURL:  ts.URL,
		ts:         ts,
		cfg:        cfg,
	}
}

// TestDashboard_SandboxUIJourney runs the end-to-end browser journey using real local folders
// and a live ephemeral rclone rcd sandbox.
func TestDashboard_SandboxUIJourney(t *testing.T) {
	chromePath := findChromeBinary()
	if chromePath == "" {
		t.Skip("skipping sandbox UI journey test: Chrome/Chromium binary not found in PATH")
	}

	harness := setupSandboxHarness(t)

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

	t.Run("HarnessSetup", func(t *testing.T) {
		err := chromedp.Do(ctx,
			chromedp.Navigate(harness.serverURL),
			chromedp.WaitVisible("body[x-data]"),
			chromedp.WaitVisible("#app-header"),
		)
		if err != nil {
			t.Fatalf("failed loading dashboard on real sandbox server: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		checkInitScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return { error: "Alpine root not initialized" };
			return {
				profilesCount: app.profilesData?.list?.length || 0,
				activeProfileId: app.profilesData?.activeId || (app.activeProfile ? app.activeProfile.id : null)
			};
		})()`

		type InitResult struct {
			Error           *string `json:"error"`
			ProfilesCount   int     `json:"profilesCount"`
			ActiveProfileID *string `json:"activeProfileId"`
		}

		res, err := chromedp.Run(ctx, chromedp.Evaluate[InitResult](checkInitScript))
		if err != nil {
			t.Fatalf("failed evaluating initial harness state: %v", err)
		}
		if res.Error != nil {
			t.Fatalf("harness init check failed: %s", *res.Error)
		}
		if res.ProfilesCount != 1 {
			t.Errorf("expected 1 loaded sandbox profile, got %d", res.ProfilesCount)
		}
	})
}
