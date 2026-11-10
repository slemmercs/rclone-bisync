package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
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
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
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

	// Write isolated rclone.conf defining a local alias remote "sandrem"
	confContent := fmt.Sprintf("[sandrem]\ntype = alias\nremote = %s\n", remoteDir)
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
			RemotePath: "",
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
		RemotePath:          "sandrem:",
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

	t.Run("RealExploration", func(t *testing.T) {
		currentURL, _ := chromedp.Run(ctx, chromedp.Location())
		if !strings.HasPrefix(currentURL, "http") {
			_ = chromedp.Do(ctx,
				chromedp.Navigate(harness.serverURL),
				chromedp.WaitVisible("body[x-data]"),
				chromedp.WaitVisible("#app-header"),
			)
			time.Sleep(500 * time.Millisecond)
		}

		err := chromedp.Do(ctx,
			chromedp.Click(`button[data-tab="explorer"]`),
			chromedp.WaitVisible(`div[x-show="currentTab === 'explorer'"]`),
		)
		if err != nil {
			t.Fatalf("failed switching to explorer tab: %v", err)
		}

		time.Sleep(1 * time.Second)

		type PaneStats struct {
			LeftTarget  string   `json:"leftTarget"`
			LeftItems   []string `json:"leftItems"`
			RightTarget string   `json:"rightTarget"`
			RightItems  []string `json:"rightItems"`
			RightCanary bool     `json:"rightCanary"`
		}

		queryPanesScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			if (!app) return { error: "Alpine root not initialized" };
			return {
				leftTarget: app.explorer.left.target,
				leftItems: (app.explorer.left.items || []).map(i => i.Name),
				rightTarget: app.explorer.right.target,
				rightItems: (app.explorer.right.items || []).map(i => i.Name),
				rightCanary: Boolean(app.explorer.right.hasCanary)
			};
		})()`

		stats, err := chromedp.Run(ctx, chromedp.Evaluate[PaneStats](queryPanesScript))
		if err != nil {
			t.Fatalf("failed querying explorer pane stats: %v", err)
		}

		if stats.LeftTarget != "local" {
			t.Errorf("expected left target to be 'local', got %q", stats.LeftTarget)
		}
		if stats.RightTarget != "sandrem" {
			t.Errorf("expected right target to be 'sandrem', got %q", stats.RightTarget)
		}

		// Verify local pane items
		hasAlpha := false
		hasRename := false
		for _, name := range stats.LeftItems {
			if name == "alpha.txt" {
				hasAlpha = true
			}
			if name == "to_rename.txt" {
				hasRename = true
			}
		}
		if !hasAlpha || !hasRename {
			t.Errorf("expected left pane to list alpha.txt and to_rename.txt, got %+v", stats.LeftItems)
		}

		// Verify remote pane items fetched through live rclone RC
		hasCloudData := false
		hasPurgeDir := false
		for _, name := range stats.RightItems {
			if name == "cloud_data.txt" {
				hasCloudData = true
			}
			if name == "purge_target" {
				hasPurgeDir = true
			}
		}
		if !hasCloudData || !hasPurgeDir {
			t.Errorf("expected right pane to list cloud_data.txt and purge_target via live RC, got %+v", stats.RightItems)
		}
	})

	t.Run("RealMoveRename", func(t *testing.T) {
		currentURL, _ := chromedp.Run(ctx, chromedp.Location())
		if !strings.HasPrefix(currentURL, "http") {
			_ = chromedp.Do(ctx,
				chromedp.Navigate(harness.serverURL),
				chromedp.WaitVisible("body[x-data]"),
				chromedp.WaitVisible("#app-header"),
				chromedp.Click(`button[data-tab="explorer"]`),
				chromedp.WaitVisible(`div[x-show="currentTab === 'explorer'"]`),
			)
			time.Sleep(500 * time.Millisecond)
		}

		// Verify initial disk state
		initialSrcPath := filepath.Join(harness.localDir, "to_rename.txt")
		destPath := filepath.Join(harness.localDir, "renamed_success.txt")
		if _, err := os.Stat(initialSrcPath); err != nil {
			t.Fatalf("expected initial source file to exist: %v", err)
		}
		if _, err := os.Stat(destPath); !os.IsNotExist(err) {
			t.Fatalf("expected dest file to not exist prior to move: %v", err)
		}

		// Trigger promptMoveItem via UI/Alpine
		openPromptScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			const item = (app.explorer.left.items || []).find(i => i.Name === 'to_rename.txt');
			if (!item) return { error: "to_rename.txt not found in left pane" };
			app.promptMoveItem('left', item);
			return { ok: true, name: item.Name };
		})()`

		type PromptRes struct {
			Error *string `json:"error"`
			OK    bool    `json:"ok"`
			Name  string  `json:"name"`
		}
		promptRes, err := chromedp.Run(ctx, chromedp.Evaluate[PromptRes](openPromptScript))
		if err != nil {
			t.Fatalf("failed opening move modal: %v", err)
		}
		if promptRes.Error != nil {
			t.Fatalf("failed prompting move: %s", *promptRes.Error)
		}

		// Modal should now be visible
		err = chromedp.Do(ctx,
			chromedp.WaitVisible("#move-new-name-input"),
		)
		if err != nil {
			t.Fatalf("move-new-name-input not visible: %v", err)
		}

		// Enter new name and trigger execution
		executeScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			app.moveModal.newName = "renamed_success.txt";
			return app.executeMoveItem();
		})()`
		_, err = chromedp.Run(ctx, chromedp.Evaluate[any](executeScript))
		if err != nil {
			t.Fatalf("failed executing move item: %v", err)
		}

		// Wait briefly for filesystem mutation and UI directory refresh
		time.Sleep(800 * time.Millisecond)

		// Assert physical disk mutation
		if _, err := os.Stat(destPath); err != nil {
			t.Fatalf("expected renamed_success.txt on physical disk after UI move, got error: %v", err)
		}
		if _, err := os.Stat(initialSrcPath); !os.IsNotExist(err) {
			t.Fatalf("expected to_rename.txt to be removed from physical disk, got err: %v", err)
		}

		// Verify UI reflects the change
		queryLeftPane := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			return (app.explorer.left.items || []).map(i => i.Name);
		})()`
		leftItems, err := chromedp.Run(ctx, chromedp.Evaluate[[]string](queryLeftPane))
		if err != nil {
			t.Fatalf("failed querying left pane items: %v", err)
		}

		hasNewName := false
		hasOldName := false
		for _, name := range leftItems {
			if name == "renamed_success.txt" {
				hasNewName = true
			}
			if name == "to_rename.txt" {
				hasOldName = true
			}
		}
		if !hasNewName {
			t.Errorf("expected left pane to contain 'renamed_success.txt', got %+v", leftItems)
		}
		if hasOldName {
			t.Errorf("expected left pane to NOT contain 'to_rename.txt', got %+v", leftItems)
		}
	})

	t.Run("RealPurgeAndCanary", func(t *testing.T) {
		currentURL, _ := chromedp.Run(ctx, chromedp.Location())
		if !strings.HasPrefix(currentURL, "http") {
			_ = chromedp.Do(ctx,
				chromedp.Navigate(harness.serverURL),
				chromedp.WaitVisible("body[x-data]"),
				chromedp.WaitVisible("#app-header"),
				chromedp.Click(`button[data-tab="explorer"]`),
				chromedp.WaitVisible(`div[x-show="currentTab === 'explorer'"]`),
			)
			time.Sleep(500 * time.Millisecond)
		}

		// Initial physical disk state
		purgeTargetDir := filepath.Join(harness.remoteDir, "purge_target")
		canaryFile := filepath.Join(harness.remoteDir, "protected_canary", "RCLONE_TEST")

		if _, err := os.Stat(purgeTargetDir); err != nil {
			t.Fatalf("expected remote purge_target directory to exist: %v", err)
		}
		if _, err := os.Stat(canaryFile); err != nil {
			t.Fatalf("expected canary file to exist in protected_canary: %v", err)
		}

		// Part A: Execute real purge of non-canary directory via UI modal confirmation
		purgePromptScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			const item = (app.explorer.right.items || []).find(i => i.Name === 'purge_target');
			if (!item) return { error: "purge_target not found in right pane" };
			app.promptPurgeDirectory('right', item);
			return { ok: true, modalShown: Boolean(app.confirmModal && app.confirmModal.show) };
		})()`

		type PurgePromptRes struct {
			Error      *string `json:"error"`
			OK         bool    `json:"ok"`
			ModalShown bool    `json:"modalShown"`
		}
		pRes, err := chromedp.Run(ctx, chromedp.Evaluate[PurgePromptRes](purgePromptScript))
		if err != nil {
			t.Fatalf("failed prompting purge directory: %v", err)
		}
		if pRes.Error != nil {
			t.Fatalf("prompt purge returned error: %s", *pRes.Error)
		}
		if !pRes.ModalShown {
			t.Fatalf("expected confirmModal.show to be true after promptPurgeDirectory")
		}

		// Confirm directory purge
		confirmScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			app.handleConfirmModal();
			return { confirmed: true };
		})()`
		_, err = chromedp.Run(ctx, chromedp.Evaluate[any](confirmScript))
		if err != nil {
			t.Fatalf("failed executing handleConfirmModal: %v", err)
		}

		// Wait for remote operation and explorer UI refresh
		time.Sleep(1000 * time.Millisecond)

		// Assert physical disk deletion on remote alias directory
		if _, err := os.Stat(purgeTargetDir); !os.IsNotExist(err) {
			t.Fatalf("expected purge_target to be physically deleted from remote disk, got err: %v", err)
		}

		// Query right pane items to verify purge_target is removed
		queryRightPane := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			return (app.explorer.right.items || []).map(i => i.Name);
		})()`
		rightItems, err := chromedp.Run(ctx, chromedp.Evaluate[[]string](queryRightPane))
		if err != nil {
			t.Fatalf("failed querying right pane items: %v", err)
		}
		for _, name := range rightItems {
			if name == "purge_target" {
				t.Fatalf("expected right pane to not contain 'purge_target' after purge, got %+v", rightItems)
			}
		}

		// Part B: Attempt to purge directory containing canary (RCLONE_TEST)
		canaryPurgePromptScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			const item = (app.explorer.right.items || []).find(i => i.Name === 'protected_canary');
			if (!item) return { error: "protected_canary not found in right pane" };
			app.promptPurgeDirectory('right', item);
			return { ok: true, modalShown: Boolean(app.confirmModal && app.confirmModal.show) };
		})()`
		pRes, err = chromedp.Run(ctx, chromedp.Evaluate[PurgePromptRes](canaryPurgePromptScript))
		if err != nil {
			t.Fatalf("failed prompting purge on canary dir: %v", err)
		}
		if pRes.Error != nil {
			t.Fatalf("prompt canary purge returned error: %s", *pRes.Error)
		}

		// Confirm attempt to purge canary directory
		_, err = chromedp.Run(ctx, chromedp.Evaluate[any](confirmScript))
		if err != nil {
			t.Fatalf("failed confirming canary dir purge: %v", err)
		}

		time.Sleep(800 * time.Millisecond)

		// Assert physical canary file is STILL intact on disk
		if _, err := os.Stat(canaryFile); err != nil {
			t.Fatalf("expected canary file to remain intact on disk after purge attempt, got error: %v", err)
		}

		// Check UI toast message for refusal
		type ToastState struct {
			Show    bool   `json:"show"`
			Message string `json:"message"`
			Type    string `json:"type"`
		}
		queryToast := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			return {
				show: Boolean(app.toast && app.toast.show),
				message: app.toast ? app.toast.message : "",
				type: app.toast ? app.toast.type : ""
			};
		})()`
		toast, err := chromedp.Run(ctx, chromedp.Evaluate[ToastState](queryToast))
		if err != nil {
			t.Fatalf("failed querying toast state: %v", err)
		}
		if !strings.Contains(strings.ToLower(toast.Message), "canary") && !strings.Contains(strings.ToLower(toast.Message), "purge failed") {
			t.Errorf("expected canary purge rejection toast message, got: %q", toast.Message)
		}

		// Part C: Direct canary file rename attempt rejected
		canaryMoveScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			app.promptMoveItem('right', { Name: 'RCLONE_TEST', Path: 'RCLONE_TEST', IsDir: false });
			return {
				modalShown: Boolean(app.moveModal && app.moveModal.show),
				message: app.toast ? app.toast.message : ""
			};
		})()`
		type MoveCanaryRes struct {
			ModalShown bool   `json:"modalShown"`
			Message    string `json:"message"`
		}
		canaryMoveRes, err := chromedp.Run(ctx, chromedp.Evaluate[MoveCanaryRes](canaryMoveScript))
		if err != nil {
			t.Fatalf("failed calling promptMoveItem on canary: %v", err)
		}
		if canaryMoveRes.ModalShown {
			t.Errorf("expected move modal to NOT open for canary item, but modalShown=true")
		}
		if !strings.Contains(strings.ToLower(canaryMoveRes.Message), "canary") {
			t.Errorf("expected canary warning toast, got: %q", canaryMoveRes.Message)
		}
	})

	t.Run("UploadDockJourney", func(t *testing.T) {
		currentURL, _ := chromedp.Run(ctx, chromedp.Location())
		if !strings.HasPrefix(currentURL, "http") {
			_ = chromedp.Do(ctx,
				chromedp.Navigate(harness.serverURL),
				chromedp.WaitVisible("body[x-data]"),
				chromedp.WaitVisible("#app-header"),
				chromedp.Click(`button[data-tab="explorer"]`),
				chromedp.WaitVisible(`div[x-show="currentTab === 'explorer'"]`),
			)
			time.Sleep(500 * time.Millisecond)
		}

		// Step 1: Initial upload dock state assertion
		initScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			return {
				pendingCount: app.uploadDock ? app.uploadDock.pending_count : 0,
				modalShown: Boolean(app.uploadDockModal && app.uploadDockModal.show)
			};
		})()`
		type DockState struct {
			PendingCount int  `json:"pendingCount"`
			ModalShown   bool `json:"modalShown"`
		}
		initRes, err := chromedp.Run(ctx, chromedp.Evaluate[DockState](initScript))
		if err != nil {
			t.Fatalf("failed querying initial dock state: %v", err)
		}
		if initRes.ModalShown {
			t.Errorf("expected uploadDockModal to be initially hidden")
		}

		// Step 2: Enqueue items into upload dock via HTTP API
		postDockItem := func(item models.UploadQueueItem) models.UploadQueueItem {
			b, _ := json.Marshal(item)
			req, err := http.NewRequest(http.MethodPost, harness.serverURL+"/api/explorer/dock", bytes.NewReader(b))
			if err != nil {
				t.Fatalf("failed creating dock request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Dashboard-Request", "1")
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("failed enqueuing item: %v", err)
			}
			defer res.Body.Close()
			var enqueued models.UploadQueueItem
			_ = json.NewDecoder(res.Body).Decode(&enqueued)
			return enqueued
		}

		item1 := postDockItem(models.UploadQueueItem{
			ProfileID: "sandbox-prof",
			Remote:    "local",
			Path:      "uploads",
			FileName:  "staged_dataset_1.csv",
			Size:      1048576,
		})
		item2 := postDockItem(models.UploadQueueItem{
			ProfileID: "sandbox-prof",
			Remote:    "sandrem:",
			Path:      "backups",
			FileName:  "archive_snapshot.tar.gz",
			Size:      2097152,
		})

		if item1.ID == "" || item2.ID == "" {
			t.Fatalf("expected non-empty IDs for enqueued items, got item1=%+v item2=%+v", item1, item2)
		}

		// Refresh upload dock in UI and assert pending count
		refreshScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			app.fetchUploadDockStatus();
			return true;
		})()`
		_, err = chromedp.Run(ctx, chromedp.Evaluate[bool](refreshScript))
		if err != nil {
			t.Fatalf("failed triggering fetchUploadDockStatus: %v", err)
		}
		time.Sleep(500 * time.Millisecond)

		pollDockScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			return {
				pendingCount: app.uploadDock ? app.uploadDock.pending_count : 0,
				itemCount: (app.uploadDock && app.uploadDock.items) ? app.uploadDock.items.length : 0
			};
		})()`
		type PollRes struct {
			PendingCount int `json:"pendingCount"`
			ItemCount    int `json:"itemCount"`
		}
		pRes, err := chromedp.Run(ctx, chromedp.Evaluate[PollRes](pollDockScript))
		if err != nil {
			t.Fatalf("failed polling dock: %v", err)
		}
		if pRes.PendingCount < 2 || pRes.ItemCount < 2 {
			t.Fatalf("expected at least 2 pending items, got pending=%d items=%d", pRes.PendingCount, pRes.ItemCount)
		}

		// Step 3: Open upload dock modal via UI interaction
		openModalScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			app.openUploadDockModal();
			return {
				modalShown: Boolean(app.uploadDockModal && app.uploadDockModal.show),
				renderedFiles: (app.uploadDock.items || []).map(i => i.file_name)
			};
		})()`
		type ModalOpenRes struct {
			ModalShown    bool     `json:"modalShown"`
			RenderedFiles []string `json:"renderedFiles"`
		}
		openRes, err := chromedp.Run(ctx, chromedp.Evaluate[ModalOpenRes](openModalScript))
		if err != nil {
			t.Fatalf("failed opening upload dock modal: %v", err)
		}
		if !openRes.ModalShown {
			t.Fatalf("expected uploadDockModal.show to be true after openUploadDockModal")
		}
		hasFile1 := false
		hasFile2 := false
		for _, f := range openRes.RenderedFiles {
			if f == "staged_dataset_1.csv" {
				hasFile1 = true
			}
			if f == "archive_snapshot.tar.gz" {
				hasFile2 = true
			}
		}
		if !hasFile1 || !hasFile2 {
			t.Fatalf("expected modal to list both queued files, got %+v", openRes.RenderedFiles)
		}

		// Step 4: Delete one item from dock via API and refresh status
		deleteDockItem := func(id string) {
			req, err := http.NewRequest(http.MethodDelete, harness.serverURL+"/api/explorer/dock?id="+id, nil)
			if err != nil {
				t.Fatalf("failed creating delete dock request: %v", err)
			}
			req.Header.Set("X-Dashboard-Request", "1")
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("failed deleting item: %v", err)
			}
			res.Body.Close()
		}
		deleteDockItem(item1.ID)

		_, err = chromedp.Run(ctx, chromedp.Evaluate[bool](refreshScript))
		if err != nil {
			t.Fatalf("failed triggering refresh after delete: %v", err)
		}
		time.Sleep(500 * time.Millisecond)

		pRes, err = chromedp.Run(ctx, chromedp.Evaluate[PollRes](pollDockScript))
		if err != nil {
			t.Fatalf("failed polling dock after delete: %v", err)
		}
		if pRes.PendingCount != 1 {
			t.Errorf("expected pending count to decrease to 1, got %d", pRes.PendingCount)
		}

		// Step 5: Close upload dock modal
		closeModalScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			app.closeUploadDockModal();
			return Boolean(app.uploadDockModal && app.uploadDockModal.show);
		})()`
		modalShown, err := chromedp.Run(ctx, chromedp.Evaluate[bool](closeModalScript))
		if err != nil {
			t.Fatalf("failed closing upload dock modal: %v", err)
		}
		if modalShown {
			t.Fatalf("expected uploadDockModal.show to be false after closeUploadDockModal")
		}

		// Step 6: Clear all remaining items and verify dock is empty
		deleteDockItem(item2.ID)
		_, err = chromedp.Run(ctx, chromedp.Evaluate[bool](refreshScript))
		if err != nil {
			t.Fatalf("failed triggering refresh after clearing remaining: %v", err)
		}
		time.Sleep(500 * time.Millisecond)

		pRes, err = chromedp.Run(ctx, chromedp.Evaluate[PollRes](pollDockScript))
		if err != nil {
			t.Fatalf("failed polling dock after final clear: %v", err)
		}
		if pRes.PendingCount != 0 {
			t.Errorf("expected dock pending count to be 0 after clear, got %d", pRes.PendingCount)
		}
	})
	t.Run("RemoteWizardAndProfileBridgeJourney", func(t *testing.T) {
		currentURL, _ := chromedp.Run(ctx, chromedp.Location())
		if !strings.HasPrefix(currentURL, "http") {
			_ = chromedp.Do(ctx,
				chromedp.Navigate(harness.serverURL),
				chromedp.WaitVisible("body[x-data]"),
				chromedp.WaitVisible("#app-header"),
			)
			time.Sleep(500 * time.Millisecond)
		}
		// Step 1: Switch to remotes tab and verify initial remotes
		err := chromedp.Do(ctx,
			chromedp.Click(`button[data-tab="remotes"]`),
			chromedp.WaitVisible(`div[x-show="currentTab === 'remotes'"]`),
		)
		if err != nil {
			t.Fatalf("failed switching to remotes tab: %v", err)
		}
		time.Sleep(500 * time.Millisecond)

		// Step 2: Open Remote Creation Wizard
		openWizardScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			app.openCreateRemoteWizard();
			return Boolean(app.createRemoteModal && app.createRemoteModal.show);
		})()`
		shown, err := chromedp.Run(ctx, chromedp.Evaluate[bool](openWizardScript))
		if err != nil {
			t.Fatalf("failed opening remote wizard: %v", err)
		}
		if !shown {
			t.Fatalf("expected createRemoteModal.show to be true")
		}

		// Wait for providers catalog to load
		time.Sleep(1000 * time.Millisecond)

		// Step 3: Verify provider catalog loaded and filter categories
		type ProviderCatalogStats struct {
			TotalCount    int  `json:"totalCount"`
			FilteredCount int  `json:"filteredCount"`
			HasDrive      bool `json:"hasDrive"`
		}
		evalCatalogScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			const list = app.createRemoteModal.providers || [];
			const filtered = typeof app.getFilteredWizardProviders === 'function' ? app.getFilteredWizardProviders() : [];
			const hasDrive = list.some(p => p.prefix === 'drive');
			return {
				totalCount: list.length,
				filteredCount: filtered.length,
				hasDrive: hasDrive
			};
		})()`
		catStats, err := chromedp.Run(ctx, chromedp.Evaluate[ProviderCatalogStats](evalCatalogScript))
		if err != nil {
			t.Fatalf("failed querying provider catalog: %v", err)
		}
		if catStats.TotalCount == 0 {
			t.Fatalf("expected non-empty provider catalog from live rclone RC, got 0")
		}
		if !catStats.HasDrive {
			t.Fatalf("expected provider catalog to include Google Drive ('drive')")
		}

		// Step 4: Select 'alias' provider and configure remote & profile bridge
		targetLocal2 := filepath.Join(harness.sandboxDir, "local2")
		_ = os.MkdirAll(targetLocal2, 0750)

		configureScript := fmt.Sprintf(`(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			const provider = (app.createRemoteModal.providers || []).find(p => p.prefix === 'alias') || { prefix: 'alias', name: 'Alias for an existing remote' };
			app.selectWizardProvider(provider);
			app.createRemoteModal.remoteName = 'sandalias';
			app.onWizardRemoteNameChange();
			app.createRemoteModal.parameters = { remote: '%s' };
			app.createRemoteModal.createProfile = true;
			app.createRemoteModal.profile.id = 'sandalias-prof';
			app.createRemoteModal.profile.name = 'SandAlias Sync';
			app.createRemoteModal.profile.local_path = '%s';
			return {
				step: app.createRemoteModal.step,
				nameError: app.createRemoteModal.nameError
			};
		})()`, harness.remoteDir, targetLocal2)

		type ConfigRes struct {
			Step      int    `json:"step"`
			NameError string `json:"nameError"`
		}
		cfgRes, err := chromedp.Run(ctx, chromedp.Evaluate[ConfigRes](configureScript))
		if err != nil {
			t.Fatalf("failed configuring wizard: %v", err)
		}
		if cfgRes.NameError != "" {
			t.Fatalf("unexpected nameError: %s", cfgRes.NameError)
		}
		if cfgRes.Step != 2 {
			t.Fatalf("expected wizard to advance to step 2, got step %d", cfgRes.Step)
		}
		// Step 5: Advance to step 3 and submit creation
		advanceAndSubmitScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			app.nextWizardStep();
			return app.createRemoteModal.step;
		})()`
		step3, err := chromedp.Run(ctx, chromedp.Evaluate[int](advanceAndSubmitScript))
		if err != nil || step3 != 3 {
			t.Fatalf("expected advance to step 3, got step %d (err: %v)", step3, err)
		}

		// Submit creation
		submitScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			app.submitCreateRemote();
			return true;
		})()`
		_, err = chromedp.Run(ctx, chromedp.Evaluate[bool](submitScript))
		if err != nil {
			t.Fatalf("failed submitting create remote: %v", err)
		}

		// Wait for remote and profile creation
		time.Sleep(1500 * time.Millisecond)

		// Step 6: Verify result and created profile
		verifyCreationScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			const created = Boolean(app.createRemoteModal.result && (app.createRemoteModal.result.success || app.createRemoteModal.result.status === 'created'));
			const profExists = (app.profilesData?.list || []).some(p => p.remote === 'sandalias' || p.name === 'SandAlias Sync');
			const remoteExists = (app.remotesData?.remotes || []).some(r => r.name === 'sandalias' || r.name === 'sandalias:');
			return {
				created: created,
				profExists: profExists,
				remoteExists: remoteExists,
				error: app.createRemoteModal.error
			};
		})()`
		type VerifyRes struct {
			Created      bool    `json:"created"`
			ProfExists   bool    `json:"profExists"`
			RemoteExists bool    `json:"remoteExists"`
			Error        *string `json:"error"`
		}
		vRes, err := chromedp.Run(ctx, chromedp.Evaluate[VerifyRes](verifyCreationScript))
		if err != nil {
			t.Fatalf("failed querying creation result: %v", err)
		}
		if vRes.Error != nil && *vRes.Error != "" {
			t.Fatalf("wizard creation returned error: %s", *vRes.Error)
		}
		if !vRes.Created {
			t.Fatalf("expected createRemoteModal.result.status === 'success'")
		}
		if !vRes.ProfExists {
			t.Fatalf("expected newly provisioned profile 'sandalias-prof' in profilesData.list")
		}

		// Step 7: Close wizard
		closeScript := `(() => {
			const app = Alpine.$data(document.querySelector('[x-data]'));
			app.closeCreateRemoteWizard();
			return Boolean(app.createRemoteModal.show);
		})()`
		isShown, err := chromedp.Run(ctx, chromedp.Evaluate[bool](closeScript))
		if err != nil || isShown {
			t.Fatalf("expected wizard modal to close, got show=%v (err: %v)", isShown, err)
		}

		// Step 8: Clean up by verifying conflict guard and deleting remote via HTTP API
		delReq, err := http.NewRequest(http.MethodDelete, harness.serverURL+"/api/remotes/sandalias", nil)
		if err != nil {
			t.Fatalf("failed creating delete request: %v", err)
		}
		delReq.Header.Set("X-Dashboard-Request", "1")
		delRes, err := http.DefaultClient.Do(delReq)
		if err != nil {
			t.Fatalf("failed executing unforced delete: %v", err)
		}
		if delRes.StatusCode != http.StatusConflict {
			t.Errorf("expected 409 Conflict due to active profile, got %d", delRes.StatusCode)
		}
		delRes.Body.Close()

		// Force delete
		forceDelReq, err := http.NewRequest(http.MethodDelete, harness.serverURL+"/api/remotes/sandalias?force=true", nil)
		if err != nil {
			t.Fatalf("failed creating force delete request: %v", err)
		}
		forceDelReq.Header.Set("X-Dashboard-Request", "1")
		forceDelRes, err := http.DefaultClient.Do(forceDelReq)
		if err != nil {
			t.Fatalf("failed executing force delete: %v", err)
		}
		if forceDelRes.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK for force delete, got %d", forceDelRes.StatusCode)
		}
		forceDelRes.Body.Close()
	})

}
