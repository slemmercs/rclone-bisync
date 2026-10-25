package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
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

func main() {
	cfg := Config{}
	flag.Usage = printHelp
	flag.BoolVar(&cfg.Verbose, "v", false, "Stream live output")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "Stream live output")
	flag.IntVar(&cfg.LocThreshold, "loc", 400, "LOC threshold (default 400)")
	flag.Float64Var(&cfg.MinCoverage, "min-coverage", 80.0, "Required test coverage %")
	flag.BoolVar(&cfg.RunRace, "race", false, "Enable race detection during tests")
	flag.StringVar(&cfg.BuildTarget, "target", ".", "Build target path")
	flag.StringVar(&cfg.CoverageFile, "coverage-file", "coverage.out", "Coverage output file")
	flag.Parse()

	if envLoc := os.Getenv("LOC_THRESHOLD"); envLoc != "" {
		if val, err := strconv.Atoi(envLoc); err == nil && cfg.LocThreshold == 400 {
			cfg.LocThreshold = val
		}
	}
	if envCov := os.Getenv("MIN_COVERAGE"); envCov != "" {
		if val, err := strconv.ParseFloat(envCov, 64); err == nil && cfg.MinCoverage == 80.0 {
			cfg.MinCoverage = val
		}
	}
	if envRace := os.Getenv("RUN_RACE"); (envRace == "true" || envRace == "1") && !cfg.RunRace {
		cfg.RunRace = true
	}
	if envTarget := os.Getenv("BUILD_TARGET"); envTarget != "" && cfg.BuildTarget == "." {
		cfg.BuildTarget = envTarget
	}
	if envFile := os.Getenv("COVERAGE_FILE"); envFile != "" && cfg.CoverageFile == "coverage.out" {
		cfg.CoverageFile = envFile
	}

	cfg.Stage = "all"
	if flag.NArg() > 0 {
		cfg.Stage = flag.Arg(0)
	}

	if (cfg.Stage == "loc" || cfg.Stage == "lines") && flag.NArg() > 1 {
		if val, err := strconv.Atoi(flag.Arg(1)); err == nil {
			cfg.LocThreshold = val
		}
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Printf("Error locating repo root: %v\n", err)
		os.Exit(1)
	}
	_ = os.Chdir(repoRoot)

	start := time.Now()
	var results []CheckResult

	switch cfg.Stage {
	case "fmt", "format":
		results = append(results, runStage("Go Formatting & Shell Check", func() error {
			return checkFmtAndScripts(repoRoot)
		}))
	case "templates", "template":
		results = append(results, runStage("Alpine Template Safety", func() error {
			return checkAlpineTemplates(repoRoot)
		}))
	case "loc", "lines":
		results = append(results, runStage("Lines of Code Audit", func() error {
			return checkLOC(repoRoot, cfg.LocThreshold)
		}))
	case "lint", "vet", "security":
		results = append(results, runStage("Static Analysis & Security", func() error {
			return runLintAndSecurity(cfg)
		}))
	case "test", "tests":
		results = append(results, runStage("Unit Tests & Coverage", func() error {
			return runTests(cfg)
		}))
	case "build", "builds":
		results = append(results, runStage("Build Verification", func() error {
			return runBuilds(cfg)
		}))
	case "all":
		results = runAllStages(cfg, repoRoot)
	case "help", "-h", "--help":
		printHelp()
		os.Exit(0)
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

func runStage(name string, fn func() error) CheckResult {
	err := fn()
	if err != nil {
		return CheckResult{Name: name, Passed: false, ErrLog: err.Error()}
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
		results[0] = runStage("Go Formatting & Shell Check", func() error {
			return checkFmtAndScripts(repoRoot)
		})
	}()
	go func() {
		defer wg.Done()
		results[1] = runStage("Alpine Template Safety", func() error {
			return checkAlpineTemplates(repoRoot)
		})
	}()
	go func() {
		defer wg.Done()
		results[2] = runStage("Lines of Code Audit", func() error {
			return checkLOC(repoRoot, cfg.LocThreshold)
		})
	}()
	wg.Wait()

	// Heavy / build operations run sequentially to prevent resource exhaustion
	results[3] = runStage("Static Analysis & Security", func() error {
		return runLintAndSecurity(cfg)
	})
	results[4] = runStage("Unit Tests & Coverage", func() error {
		return runTests(cfg)
	})
	results[5] = runStage("Build Verification", func() error {
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
