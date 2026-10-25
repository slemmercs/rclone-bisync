package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config encapsulates execution flags and thresholds for check stages.
type Config struct {
	Stage        string
	Verbose      bool
	LocThreshold int
	MinCoverage  float64
	RunRace      bool
	BuildTarget  string
	CoverageFile string
}

// CheckResult contains outcome data for a single check stage.
type CheckResult struct {
	Name   string
	Passed bool
	ErrLog string
}

var valueFlags = map[string]bool{
	"loc":           true,
	"min-coverage":  true,
	"target":        true,
	"coverage-file": true,
}

func partitionArgs(args []string) ([]string, []string) {
	var flagArgs, posArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			clean := strings.TrimLeft(arg, "-")
			if strings.Contains(clean, "=") {
				flagArgs = append(flagArgs, arg)
				continue
			}
			if valueFlags[clean] && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flagArgs = append(flagArgs, arg, args[i+1])
				i++
				continue
			}
			flagArgs = append(flagArgs, arg)
		} else {
			posArgs = append(posArgs, arg)
		}
	}
	return flagArgs, posArgs
}

func parseConfig(args []string) (Config, bool, error) {
	return parseConfigWithEnv(args, os.Getenv)
}

func parseConfigWithEnv(args []string, getenv func(string) string) (Config, bool, error) {
	cfg := Config{
		LocThreshold: 400,
		MinCoverage:  80.0,
		BuildTarget:  ".",
		CoverageFile: "coverage.out",
	}

	flagArgs, posArgs := partitionArgs(args)

	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&cfg.Verbose, "v", false, "Stream live output")
	fs.BoolVar(&cfg.Verbose, "verbose", false, "Stream live output")
	fs.IntVar(&cfg.LocThreshold, "loc", 400, "LOC threshold (default 400)")
	fs.Float64Var(&cfg.MinCoverage, "min-coverage", 80.0, "Required test coverage %")
	fs.BoolVar(&cfg.RunRace, "race", false, "Enable race detection during tests")
	fs.StringVar(&cfg.BuildTarget, "target", ".", "Build target path")
	fs.StringVar(&cfg.CoverageFile, "coverage-file", "coverage.out", "Coverage output file")

	if err := fs.Parse(flagArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return cfg, true, nil
		}
		return cfg, false, err
	}

	if getenv != nil {
		if envV := getenv("VERBOSE"); (envV == "1" || envV == "true") && !cfg.Verbose {
			cfg.Verbose = true
		}
		if envLoc := getenv("LOC_THRESHOLD"); envLoc != "" {
			if val, err := strconv.Atoi(envLoc); err == nil && cfg.LocThreshold == 400 {
				cfg.LocThreshold = val
			}
		}
		if envCov := getenv("MIN_COVERAGE"); envCov != "" {
			if val, err := strconv.ParseFloat(envCov, 64); err == nil && cfg.MinCoverage == 80.0 {
				cfg.MinCoverage = val
			}
		}
		if envRace := getenv("RUN_RACE"); (envRace == "true" || envRace == "1") && !cfg.RunRace {
			cfg.RunRace = true
		}
		if envTarget := getenv("BUILD_TARGET"); envTarget != "" && cfg.BuildTarget == "." {
			cfg.BuildTarget = envTarget
		}
		if envFile := getenv("COVERAGE_FILE"); envFile != "" && cfg.CoverageFile == "coverage.out" {
			cfg.CoverageFile = envFile
		}
	}

	cfg.Stage = "all"
	if len(posArgs) > 0 {
		cfg.Stage = posArgs[0]
	}

	if cfg.Stage == "help" {
		return cfg, true, nil
	}

	if (cfg.Stage == "loc" || cfg.Stage == "lines") && len(posArgs) > 1 {
		if val, err := strconv.Atoi(posArgs[1]); err == nil {
			cfg.LocThreshold = val
		}
	}

	return cfg, false, nil
}

func main() {
	cfg, isHelp, err := parseConfig(os.Args[1:])
	if isHelp {
		printHelp()
		os.Exit(0)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing arguments: %v\n\n", err)
		printHelp()
		os.Exit(1)
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Printf("Error locating repo root: %v\n", err)
		os.Exit(1)
	}
	_ = os.Chdir(repoRoot)

	if cfg.Verbose {
		fmt.Printf("==> [check] stage=%s loc=%d min-coverage=%.1f%% race=%v\n",
			cfg.Stage, cfg.LocThreshold, cfg.MinCoverage, cfg.RunRace)
	}

	start := time.Now()
	var results []CheckResult

	switch cfg.Stage {
	case "fmt", "format":
		results = append(results, runStage("Go Formatting & Shell Check", cfg.Verbose, func() error {
			return checkFmtAndScripts(repoRoot, cfg.Verbose)
		}))
	case "templates", "template":
		results = append(results, runStage("Alpine Template Safety", cfg.Verbose, func() error {
			return checkAlpineTemplates(repoRoot, cfg.Verbose)
		}))
	case "loc", "lines":
		results = append(results, runStage("Lines of Code Audit", cfg.Verbose, func() error {
			return checkLOC(repoRoot, cfg.LocThreshold, cfg.Verbose)
		}))
	case "lint", "vet", "security":
		results = append(results, runStage("Static Analysis & Security", cfg.Verbose, func() error {
			return runLintAndSecurity(cfg)
		}))
	case "test", "tests":
		results = append(results, runStage("Unit Tests & Coverage", cfg.Verbose, func() error {
			return runTests(cfg)
		}))
	case "build", "builds":
		results = append(results, runStage("Build Verification", cfg.Verbose, func() error {
			return runBuilds(cfg)
		}))
	case "all":
		results = runAllStages(cfg, repoRoot)
	default:
		fmt.Printf("Unknown stage '%s'. Available stages: all, fmt, lint, templates, loc [limit], test, build\n", cfg.Stage)
		os.Exit(1)
	}

	failed := 0
	for _, res := range results {
		if !res.Passed {
			failed++
		}
	}

	if failed == 0 {
		fmt.Printf("OK: All quality gate checks passed (%d/%d) [%v]\n", len(results), len(results), time.Since(start).Round(time.Millisecond))
		os.Exit(0)
	}

	fmt.Printf("FAIL: Quality gate failed (%d/%d checks failed).\n", failed, len(results))
	for _, res := range results {
		if !res.Passed {
			fmt.Printf("\n--- Error log for '%s' ---\n%s\n", res.Name, res.ErrLog)
		}
	}
	os.Exit(1)
}

func runStage(name string, verbose bool, fn func() error) CheckResult {
	if verbose {
		fmt.Printf("==> [stage:start] %s\n", name)
	}
	start := time.Now()
	err := fn()
	elapsed := time.Since(start).Round(time.Millisecond)
	if err != nil {
		if verbose {
			fmt.Printf("==> [stage:fail]  %s (%v)\n", name, elapsed)
		}
		return CheckResult{Name: name, Passed: false, ErrLog: err.Error()}
	}
	if verbose {
		fmt.Printf("==> [stage:pass]  %s (%v)\n", name, elapsed)
	}
	return CheckResult{Name: name, Passed: true}
}

func runAllStages(cfg Config, repoRoot string) []CheckResult {
	results := make([]CheckResult, 6)
	var wg sync.WaitGroup

	// Run fast static scans in parallel
	wg.Add(3)
	go func() {
		defer wg.Done()
		results[0] = runStage("Go Formatting & Shell Check", cfg.Verbose, func() error {
			return checkFmtAndScripts(repoRoot, cfg.Verbose)
		})
	}()
	go func() {
		defer wg.Done()
		results[1] = runStage("Alpine Template Safety", cfg.Verbose, func() error {
			return checkAlpineTemplates(repoRoot, cfg.Verbose)
		})
	}()
	go func() {
		defer wg.Done()
		results[2] = runStage("Lines of Code Audit", cfg.Verbose, func() error {
			return checkLOC(repoRoot, cfg.LocThreshold, cfg.Verbose)
		})
	}()
	wg.Wait()

	// Heavy / build operations run sequentially to prevent resource exhaustion
	results[3] = runStage("Static Analysis & Security", cfg.Verbose, func() error {
		return runLintAndSecurity(cfg)
	})
	results[4] = runStage("Unit Tests & Coverage", cfg.Verbose, func() error {
		return runTests(cfg)
	})
	results[5] = runStage("Build Verification", cfg.Verbose, func() error {
		return runBuilds(cfg)
	})

	return results
}

func printHelp() {
	fmt.Println(`rclone-bisync Quality Gate Tool

Usage:
  check [stage] [flags]

Stages:
  all                Run all quality checks (default)
  fmt                Check Go formatting and shell script syntax
  lint               Run static analysis, linters, and secret scans
  templates          Audit Alpine.js templates for unsafe dotted boolean bindings
  loc [limit]        Audit line-of-code thresholds across Go, JS, and HTML files
  test               Run test suite with coverage enforcement
  build              Verify native and cross-compilation builds

Flags:
  -v, --verbose      Enable live verbose output
  -loc int           Line count limit for loc stage (default: 400)
  -min-coverage float Required test coverage percentage (default: 80.0)
  -race              Enable Go race detector during tests
  -target string     Build target directory (default: .)
  -coverage-file str Path for coverage profile (default: coverage.out)`)
}
