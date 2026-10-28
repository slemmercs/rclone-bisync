package main

import (
	"bytes"
	"flag"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedFrontendFS(t *testing.T) {
	// Verify embedded files exist in frontendFS
	data, err := fs.ReadFile(frontendFS, "dashboard/frontend/index.html")
	if err != nil {
		t.Fatalf("expected dashboard/frontend/index.html in embedded FS, got: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("embedded index.html is empty")
	}

	dataApp, err := fs.ReadFile(frontendFS, "dashboard/frontend/app.js")
	if err != nil {
		t.Fatalf("expected dashboard/frontend/app.js in embedded FS, got: %v", err)
	}
	if len(dataApp) == 0 {
		t.Fatal("embedded app.js is empty")
	}

	dataRemotes, err := fs.ReadFile(frontendFS, "dashboard/frontend/js/remotes.js")
	if err != nil {
		t.Fatalf("expected dashboard/frontend/js/remotes.js in embedded FS, got: %v", err)
	}
	if len(dataRemotes) == 0 {
		t.Fatal("embedded remotes.js is empty")
	}

	dataReview, err := fs.ReadFile(frontendFS, "dashboard/frontend/js/review.js")
	if err != nil {
		t.Fatalf("expected dashboard/frontend/js/review.js in embedded FS, got: %v", err)
	}
	if len(dataReview) == 0 {
		t.Fatal("embedded review.js is empty")
	}

	dataFleet, err := fs.ReadFile(frontendFS, "dashboard/frontend/js/fleet.js")
	if err != nil {
		t.Fatalf("expected dashboard/frontend/js/fleet.js in embedded FS, got: %v", err)
	}
	if len(dataFleet) == 0 {
		t.Fatal("embedded fleet.js is empty")
	}

	dataHistory, err := fs.ReadFile(frontendFS, "dashboard/frontend/js/profiles_history.js")
	if err != nil {
		t.Fatalf("expected dashboard/frontend/js/profiles_history.js in embedded FS, got: %v", err)
	}
	if len(dataHistory) == 0 {
		t.Fatal("embedded profiles_history.js is empty")
	}

	subFS, err := fs.Sub(frontendFS, "dashboard/frontend")
	if err != nil {
		t.Fatalf("failed sub fs: %v", err)
	}
	tmpl, tmplErr := template.ParseFS(subFS, "index.html", "partials/*.html", "partials/modals/*.html")
	if tmplErr != nil {
		t.Fatalf("ParseFS failed on embedded frontendFS: %v", tmplErr)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		t.Fatalf("Execute failed on assembled embedded template: %v", err)
	}
	rendered := buf.String()
	if strings.Contains(rendered, "1 1 0 .696") {
		t.Fatal("corrupted SVG arc found in rendered embedded template")
	}
	if !strings.Contains(rendered, "1 1 0 0 1 0 .696") {
		t.Fatal("expected valid SVG arc in rendered embedded template")
	}
}

func TestTemplateHTMLBalancedTags(t *testing.T) {
	err := fs.WalkDir(frontendFS, "dashboard/frontend", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}
		contentBytes, err := fs.ReadFile(frontendFS, path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}
		content := string(contentBytes)

		openDivs := strings.Count(content, "<div")
		closeDivs := strings.Count(content, "</div>")
		if openDivs != closeDivs {
			t.Errorf("unbalanced <div> tags in %s: %d open vs %d close", path, openDivs, closeDivs)
		}

		openMains := strings.Count(content, "<main")
		closeMains := strings.Count(content, "</main>")
		if openMains != closeMains {
			t.Errorf("unbalanced <main> tags in %s: %d open vs %d close", path, openMains, closeMains)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir failed: %v", err)
	}
}

func TestSubcommandFlagParsing(t *testing.T) {
	// 1. Dashboard subcommand flagset
	dashCmd := flag.NewFlagSet("dashboard", flag.ContinueOnError)
	host := dashCmd.String("host", "127.0.0.1", "")
	port := dashCmd.Int("port", 8080, "")
	envFile := dashCmd.String("env-file", "", "")

	err := dashCmd.Parse([]string{"--host", "0.0.0.0", "--port", "9090", "--env-file", "/tmp/bisync.env"})
	if err != nil {
		t.Fatalf("failed to parse dashboard flags: %v", err)
	}
	if *host != "0.0.0.0" || *port != 9090 || *envFile != "/tmp/bisync.env" {
		t.Fatalf("unexpected dashboard parsed flags: host=%s, port=%d, env=%s", *host, *port, *envFile)
	}

	// 2. Sync subcommand flagset
	syncCmd := flag.NewFlagSet("sync", flag.ContinueOnError)
	resync := syncCmd.Bool("resync", false, "")
	force := syncCmd.Bool("force", false, "")
	dryRun := syncCmd.Bool("dry-run", false, "")
	noNotify := syncCmd.Bool("no-notify", false, "")
	origin := syncCmd.String("origin", "cli", "")
	syncEnv := syncCmd.String("env-file", "", "")

	err = syncCmd.Parse([]string{"--resync", "--force", "--dry-run", "--no-notify", "--origin", "timer", "--env-file", "/tmp/test.env"})
	if err != nil {
		t.Fatalf("failed to parse sync flags: %v", err)
	}
	if !*resync || !*force || !*dryRun || !*noNotify || *origin != "timer" || *syncEnv != "/tmp/test.env" {
		t.Fatalf("unexpected sync parsed flags: resync=%v force=%v dryRun=%v noNotify=%v origin=%s env=%s",
			*resync, *force, *dryRun, *noNotify, *origin, *syncEnv)
	}

	// 3. Canary subcommand flagset
	canaryCmd := flag.NewFlagSet("canary", flag.ContinueOnError)
	remote := canaryCmd.String("remote", "local", "")
	path := canaryCmd.String("path", "", "")
	canaryEnv := canaryCmd.String("env-file", "", "")

	err = canaryCmd.Parse([]string{"--remote", "gdrive", "--path", "sub/folder", "--env-file", "/tmp/c.env"})
	if err != nil {
		t.Fatalf("failed to parse canary flags: %v", err)
	}
	if *remote != "gdrive" || *path != "sub/folder" || *canaryEnv != "/tmp/c.env" {
		t.Fatalf("unexpected canary flags: remote=%s path=%s env=%s", *remote, *path, *canaryEnv)
	}
}

func TestRunCanaryLocalExecution(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "main-canary-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	envContent := "LOCAL_PATH=" + tmpDir + "\nRC_PASS=testpass123\n"
	envPath := filepath.Join(tmpDir, "bisync.env")
	_ = os.WriteFile(envPath, []byte(envContent), 0644)

	// Invoke runCanary with local target
	runCanary("local", "", envPath)

	// Verify RCLONE_TEST created
	canaryFile := filepath.Join(tmpDir, "RCLONE_TEST")
	if _, err := os.Stat(canaryFile); os.IsNotExist(err) {
		t.Fatalf("expected canary file to be created at %s", canaryFile)
	}
}

func TestVersionDefault(t *testing.T) {
	if Version == "" {
		t.Fatal("expected Version to have a default value")
	}
}
