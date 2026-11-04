package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/core/client"
)

// startRcloneSandbox spawns an ephemeral real `rclone rcd` process bound to an ephemeral port
// with isolated cache and null config, guaranteeing zero impact on user cloud remotes or state.
func startRcloneSandbox(t *testing.T) (*client.RcloneClient, string) {
	t.Helper()

	if _, err := exec.LookPath("rclone"); err != nil {
		t.Skip("rclone binary not found in PATH, skipping Tier 4 integration test")
	}

	sandboxDir := t.TempDir()
	cacheDir := filepath.Join(sandboxDir, "cache")
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		t.Fatalf("failed to create sandbox cache dir: %v", err)
	}

	// Allocate free TCP port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate free port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	cmd := exec.Command("rclone", "rcd",
		"--rc-addr", addr,
		"--rc-no-auth",
		"--config", "/dev/null",
		"--cache-dir", cacheDir,
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start ephemeral rclone rcd process: %v", err)
	}

	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	})

	rcClient := client.NewRcloneClient(fmt.Sprintf("http://%s", addr), "", "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ok, err := rcClient.WaitForHealthy(ctx, 10*time.Second)
	if !ok || err != nil {
		t.Fatalf("ephemeral rclone daemon failed to reach healthy state: %v", err)
	}

	return rcClient, sandboxDir
}

func TestRcloneSandbox_HealthAndStats(t *testing.T) {
	rcClient, _ := startRcloneSandbox(t)
	ctx := context.Background()

	// 1. Health check verification against real Rclone RC
	healthy, latency, err := rcClient.CheckHealth(ctx, 2, 50*time.Millisecond)
	if !healthy || err != nil {
		t.Fatalf("expected healthy daemon, got healthy=%v err=%v", healthy, err)
	}
	if latency <= 0 {
		t.Errorf("expected positive latency, got %f ms", latency)
	}

	// 2. Query /core/stats on real daemon
	statsBytes, err := rcClient.GetCoreStats(ctx)
	if err != nil {
		t.Fatalf("unexpected error fetching core stats: %v", err)
	}
	var stats map[string]interface{}
	if err := json.Unmarshal(statsBytes, &stats); err != nil {
		t.Fatalf("failed to unmarshal real core stats: %v", err)
	}
	if _, ok := stats["bytes"]; !ok {
		t.Errorf("expected 'bytes' metric in core stats response: %s", string(statsBytes))
	}
}

func TestRcloneSandbox_FileAndDirectoryOperations(t *testing.T) {
	rcClient, sandboxDir := startRcloneSandbox(t)
	ctx := context.Background()

	srcDir := filepath.Join(sandboxDir, "src")
	dstDir := filepath.Join(sandboxDir, "dst")

	// 1. Mkdir on real filesystem via RC
	if _, err := rcClient.Mkdir(ctx, sandboxDir, "src"); err != nil {
		t.Fatalf("failed to Mkdir src: %v", err)
	}
	if fi, err := os.Stat(srcDir); err != nil || !fi.IsDir() {
		t.Fatalf("srcDir not created as directory on disk: %v", err)
	}

	// 2. TouchFile
	fileName := "sample.txt"
	if _, err := rcClient.TouchFile(ctx, srcDir, fileName); err != nil {
		t.Fatalf("failed to TouchFile: %v", err)
	}
	filePath := filepath.Join(srcDir, fileName)
	if err := os.WriteFile(filePath, []byte("sandbox-content-verified"), 0600); err != nil {
		t.Fatalf("failed to populate sample file: %v", err)
	}

	// 3. ListDirectory
	listing := rcClient.ListDirectory(ctx, sandboxDir, "src")
	if listing.Error != "" {
		t.Fatalf("ListDirectory failed: %v", listing.Error)
	}
	if listing.HasCanary {
		t.Errorf("expected HasCanary=false when no canary file exists")
	}
	if len(listing.Items) != 1 {
		t.Fatalf("unexpected items listed: %+v", listing.Items)
	}

	// Plant canary file and verify detection
	canaryPath := filepath.Join(srcDir, "RCLONE_TEST")
	if err := os.WriteFile(canaryPath, []byte("canary"), 0600); err != nil {
		t.Fatalf("failed to write canary: %v", err)
	}
	listingWithCanary := rcClient.ListDirectory(ctx, sandboxDir, "src")
	if listingWithCanary.Error != "" || !listingWithCanary.HasCanary {
		t.Fatalf("expected HasCanary=true, got %v (err: %v)", listingWithCanary.HasCanary, listingWithCanary.Error)
	}
	if len(listingWithCanary.Items) != 2 {
		t.Errorf("expected 2 items with canary, got %d", len(listingWithCanary.Items))
	}

	// 4. CopyFile via RC
	if _, err := rcClient.Mkdir(ctx, sandboxDir, "dst"); err != nil {
		t.Fatalf("failed to Mkdir dst: %v", err)
	}
	if _, err := rcClient.CopyFile(ctx, sandboxDir, "src/sample.txt", sandboxDir, "dst/sample_copy.txt"); err != nil {
		t.Fatalf("failed to CopyFile: %v", err)
	}
	copiedData, err := os.ReadFile(filepath.Join(dstDir, "sample_copy.txt"))
	if err != nil || string(copiedData) != "sandbox-content-verified" {
		t.Fatalf("copied content mismatch: got %q, err: %v", string(copiedData), err)
	}

	// 5. DeleteFile
	if _, err := rcClient.DeleteFile(ctx, sandboxDir, "src/sample.txt"); err != nil {
		t.Fatalf("failed to DeleteFile: %v", err)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("expected sample.txt to be deleted, stat err: %v", err)
	}

	// 6. PurgeDir
	if _, err := rcClient.PurgeDir(ctx, sandboxDir, "dst"); err != nil {
		t.Fatalf("failed to PurgeDir: %v", err)
	}
	if _, err := os.Stat(dstDir); !os.IsNotExist(err) {
		t.Errorf("expected dstDir to be purged, stat err: %v", err)
	}
}

func TestRcloneSandbox_RealBisyncLifecycle(t *testing.T) {
	rcClient, sandboxDir := startRcloneSandbox(t)
	ctx := context.Background()

	srcDir := filepath.Join(sandboxDir, "sync_src")
	dstDir := filepath.Join(sandboxDir, "sync_dst")
	if err := os.MkdirAll(srcDir, 0700); err != nil {
		t.Fatalf("failed to create sync_src: %v", err)
	}
	if err := os.MkdirAll(dstDir, 0700); err != nil {
		t.Fatalf("failed to create sync_dst: %v", err)
	}

	// Step 1: Seed initial file in src
	seedFile := filepath.Join(srcDir, "seed.txt")
	if err := os.WriteFile(seedFile, []byte("bisync-seed-data"), 0600); err != nil {
		t.Fatalf("failed to write seed file: %v", err)
	}

	// Step 2: Trigger initial resync via RC
	jobID, err := rcClient.TriggerBisync(ctx, map[string]interface{}{
		"path1":  srcDir,
		"path2":  dstDir,
		"resync": true,
	}, "sandbox-resync")
	if err != nil {
		t.Fatalf("TriggerBisync resync failed: %v", err)
	}

	// Step 3: Monitor job completion
	res := rcClient.MonitorJob(ctx, jobID, 100*time.Millisecond, 1*time.Second, 1.5, 3, 15*time.Second)
	if !res.Success {
		t.Fatalf("resync job %d failed or timed out: success=%v err=%v", jobID, res.Success, res.ErrorMessage)
	}

	// Verify file arrived at destination
	dstSeed := filepath.Join(dstDir, "seed.txt")
	gotData, err := os.ReadFile(dstSeed)
	if err != nil || string(gotData) != "bisync-seed-data" {
		t.Fatalf("seed file not synced to destination: got %q, err: %v", string(gotData), err)
	}

	// Step 4: Write secondary file in dst for bidirectional test
	twoWayFile := filepath.Join(dstDir, "reply.txt")
	if err := os.WriteFile(twoWayFile, []byte("two-way-response"), 0600); err != nil {
		t.Fatalf("failed to write two-way reply file: %v", err)
	}

	// Step 5: Trigger normal bidirectional sync (without resync)
	twoWayJobID, err := rcClient.TriggerBisync(ctx, map[string]interface{}{
		"path1": srcDir,
		"path2": dstDir,
	}, "sandbox-bidirectional")
	if err != nil {
		t.Fatalf("TriggerBisync bidirectional failed: %v", err)
	}

	res2 := rcClient.MonitorJob(ctx, twoWayJobID, 100*time.Millisecond, 1*time.Second, 1.5, 3, 15*time.Second)
	if !res2.Success {
		t.Fatalf("bidirectional sync job %d failed: success=%v err=%v", twoWayJobID, res2.Success, res2.ErrorMessage)
	}

	// Verify reply file arrived back at source
	srcReply := filepath.Join(srcDir, "reply.txt")
	replyData, err := os.ReadFile(srcReply)
	if err != nil || string(replyData) != "two-way-response" {
		t.Fatalf("reply file not synced back to src: got %q, err: %v", string(replyData), err)
	}
}
