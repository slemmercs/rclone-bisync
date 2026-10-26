package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// checkFmtAndScripts checks that Go files are formatted and shell scripts pass syntax checks.
func checkFmtAndScripts(root string, verbose bool) error {
	var unformatted []string
	var shellScripts []string
	var goFilesCount int

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || shouldSkip(path) {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if strings.HasSuffix(path, ".go") {
			goFilesCount++
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			formatted, err := format.Source(src)
			if err != nil || !bytes.Equal(src, formatted) {
				unformatted = append(unformatted, rel)
			}
		} else if strings.HasSuffix(path, ".sh") {
			shellScripts = append(shellScripts, rel)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if verbose {
		fmt.Printf("    Checked %d Go files and %d shell scripts\n", goFilesCount, len(shellScripts))
	}

	if len(unformatted) > 0 {
		return fmt.Errorf("unformatted Go files:\n  %s", strings.Join(unformatted, "\n  "))
	}

	// Validate shell script syntax if bash is available
	if bashPath, err := exec.LookPath("bash"); err == nil && len(shellScripts) > 0 {
		var invalidShell []string
		for _, sh := range shellScripts {
			cmd := exec.Command(bashPath, "-n", filepath.Join(root, sh))
			if out, err := cmd.CombinedOutput(); err != nil {
				invalidShell = append(invalidShell, fmt.Sprintf("%s:\n%s", sh, strings.TrimSpace(string(out))))
			}
		}
		if len(invalidShell) > 0 {
			return fmt.Errorf("shell script syntax errors:\n  %s", strings.Join(invalidShell, "\n  "))
		}
	}

	return nil
}

// checkLOC audits Go, JS, and HTML files against a line count threshold.
func checkLOC(root string, threshold int, verbose bool) error {
	type locEntry struct {
		file  string
		lines int
	}
	var oversized []locEntry
	var totalScanned int

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || shouldSkip(path) {
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".go" && strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if ext == ".go" || ext == ".js" || ext == ".html" {
			totalScanned++
			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			lines := bytes.Count(content, []byte("\n"))
			if lines >= threshold {
				rel, _ := filepath.Rel(root, path)
				oversized = append(oversized, locEntry{file: rel, lines: lines})
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	if verbose {
		fmt.Printf("    Audited %d files under threshold %d LOC\n", totalScanned, threshold)
	}

	if len(oversized) > 0 {
		sort.Slice(oversized, func(i, j int) bool {
			return oversized[i].lines > oversized[j].lines
		})
		var msgs []string
		for _, o := range oversized {
			msgs = append(msgs, fmt.Sprintf("%5d lines  %s", o.lines, o.file))
		}
		return fmt.Errorf("found %d file(s) exceeding %d lines of code:\n  %s", len(oversized), threshold, strings.Join(msgs, "\n  "))
	}
	return nil
}

// runLintAndSecurity executes static analysis, secret scanning, and optional linter tools.
func runLintAndSecurity(cfg Config) error {
	var errors []string

	// 1. go vet ./...
	if cfg.Verbose {
		fmt.Println("    Running go vet ./...")
	}
	cmd := exec.Command("go", "vet", "./...")
	if out, err := cmd.CombinedOutput(); err != nil {
		errors = append(errors, fmt.Sprintf("go vet failed:\n%s", strings.TrimSpace(string(out))))
	}

	// 2. staticcheck ./... (optional)
	if path, err := exec.LookPath("staticcheck"); err == nil {
		if cfg.Verbose {
			fmt.Println("    Running staticcheck ./...")
		}
		cmd := exec.Command(path, "./...")
		if out, err := cmd.CombinedOutput(); err != nil {
			errors = append(errors, fmt.Sprintf("staticcheck failed:\n%s", strings.TrimSpace(string(out))))
		}
	} else if cfg.Verbose {
		fmt.Println("    staticcheck not found (skipped)")
	}

	// 3. govulncheck ./... (optional)
	if path, err := exec.LookPath("govulncheck"); err == nil {
		if cfg.Verbose {
			fmt.Println("    Running govulncheck ./...")
		}
		cmd := exec.Command(path, "./...")
		if out, err := cmd.CombinedOutput(); err != nil {
			errors = append(errors, fmt.Sprintf("govulncheck failed:\n%s", strings.TrimSpace(string(out))))
		}
	} else if cfg.Verbose {
		fmt.Println("    govulncheck not found (skipped)")
	}

	// 4. betterleaks dir . (optional)
	if path, err := exec.LookPath("betterleaks"); err == nil {
		if cfg.Verbose {
			fmt.Println("    Running betterleaks secret scan...")
		}
		var args []string
		if _, err := os.Stat(".betterleaks.toml"); err == nil {
			args = []string{"dir", "--config", ".betterleaks.toml", "."}
		} else if _, err := os.Stat(".gitleaks.toml"); err == nil {
			args = []string{"dir", "--config", ".gitleaks.toml", "."}
		} else {
			args = []string{"dir", "."}
		}
		cmd := exec.Command(path, args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			errors = append(errors, fmt.Sprintf("betterleaks failed:\n%s", strings.TrimSpace(string(out))))
		}
	} else if cfg.Verbose {
		fmt.Println("    betterleaks not found (skipped)")
	}

	// 5. esbuild check for frontend syntax (optional)
	if path, err := exec.LookPath("esbuild"); err == nil {
		appJS := filepath.Join("dashboard", "frontend", "app.js")
		if _, err := os.Stat(appJS); err == nil {
			if cfg.Verbose {
				fmt.Println("    Running esbuild bundle syntax check...")
			}
			cmd := exec.Command(path, "--bundle", appJS, "--outfile=/dev/null")
			if out, err := cmd.CombinedOutput(); err != nil {
				errors = append(errors, fmt.Sprintf("esbuild JS check failed:\n%s", strings.TrimSpace(string(out))))
			}
		}
	} else if cfg.Verbose {
		fmt.Println("    esbuild not found (skipped)")
	}

	// 6. tidy HTML structure (optional)
	if path, err := exec.LookPath("tidy"); err == nil {
		if cfg.Verbose {
			fmt.Println("    Running HTML tidy structure check...")
		}
		var htmlFiles []string
		_ = filepath.WalkDir("dashboard/frontend", func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(p, ".html") {
				htmlFiles = append(htmlFiles, p)
			}
			return nil
		})
		for _, hf := range htmlFiles {
			cmd := exec.Command(path, "-errors", "-quiet", "-utf8", hf)
			_ = cmd.Run() // tidy returns 1 for warnings, 2 for errors
			if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() > 1 {
				errors = append(errors, fmt.Sprintf("HTML tidy errors in %s", hf))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "\n\n"))
	}
	return nil
}

// runTests runs the Go test suite and verifies statement coverage against MinCoverage.
func runTests(cfg Config) error {
	coverMode := "count"
	if cfg.RunRace {
		coverMode = "atomic"
	}

	args := []string{
		"test",
		"-count=1",
		"-covermode=" + coverMode,
		"-coverpkg=./internal/...",
		"-coverprofile=" + cfg.CoverageFile,
	}
	if cfg.RunRace {
		args = append(args, "-race", "-shuffle=on")
	}
	
	if cfg.Verbose {
		args = append(args, "-v")
	}
	args = append(args, "./...")

	if cfg.Verbose {
		fmt.Printf("    Running: go %s\n", strings.Join(args, " "))
	}
	cmd := exec.Command("go", args...)
	if cfg.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("go test failed: %w", err)
		}
	} else {
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("go test failed:\n%s", strings.TrimSpace(string(out)))
		}
	}

	// Verify coverage percentage if coverage file was generated
	if _, statErr := os.Stat(cfg.CoverageFile); statErr == nil {
		covCmd := exec.Command("go", "tool", "cover", "-func="+cfg.CoverageFile)
		covOut, covErr := covCmd.CombinedOutput()
		if covErr == nil {
			covLines := strings.Split(string(covOut), "\n")
			for _, line := range covLines {
				if strings.HasPrefix(line, "total:") {
					fields := strings.Fields(line)
					if len(fields) >= 3 {
						pctStr := strings.TrimSuffix(fields[len(fields)-1], "%")
						if pct, parseErr := strconv.ParseFloat(pctStr, 64); parseErr == nil {
							if cfg.Verbose {
								fmt.Printf("    Coverage total: %.1f%% (minimum threshold: %.1f%%)\n", pct, cfg.MinCoverage)
							}
							if pct < cfg.MinCoverage {
								return fmt.Errorf("test coverage %.1f%% is below required threshold %.1f%%", pct, cfg.MinCoverage)
							}
						}
					}
					break
				}
			}
		}
	}

	return nil
}

// runBuilds verifies native and cross-compilation targets.
func runBuilds(cfg Config) error {
	tmpDir, err := os.MkdirTemp("", "check-build-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if cfg.Verbose {
		fmt.Printf("    Building native binary from %s...\n", cfg.BuildTarget)
	}
	nativeBinary := filepath.Join(tmpDir, "rclone-bisync")
	cmd := exec.Command("go", "build", "-trimpath", "-o", nativeBinary, cfg.BuildTarget)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("native build failed:\n%s", strings.TrimSpace(string(out)))
	}

	targets := []struct{ os, arch string }{
		{"linux", "amd64"},
		{"linux", "arm64"},
	}
	for _, t := range targets {
		if cfg.Verbose {
			fmt.Printf("    Cross-compiling for %s/%s...\n", t.os, t.arch)
		}
		outPath := filepath.Join(tmpDir, fmt.Sprintf("rclone-bisync-%s-%s", t.os, t.arch))
		c := exec.Command("go", "build", "-trimpath", "-o", outPath, cfg.BuildTarget)
		c.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+t.os, "GOARCH="+t.arch)
		if out, err := c.CombinedOutput(); err != nil {
			return fmt.Errorf("cross-build %s/%s failed:\n%s", t.os, t.arch, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

var skipDirs = []string{".git", "node_modules", "site", "_site", ".build", ".tmp"}

func shouldSkip(path string) bool {
	normalized := filepath.ToSlash(path)
	for _, skip := range skipDirs {
		if strings.Contains(normalized, "/"+skip+"/") || strings.HasPrefix(normalized, skip+"/") || normalized == skip {
			return true
		}
	}
	return false
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod not found")
}
