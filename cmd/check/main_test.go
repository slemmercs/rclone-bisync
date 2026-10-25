package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanHTMLForAlpineViolations_Valid(t *testing.T) {
	html := []byte(`
<!DOCTYPE html>
<html>
<body>
	<!-- Safe top-level booleans and helper methods -->
	<button :disabled="isLoading" class="btn">Click</button>
	<input type="checkbox" :checked="isChecked" />
	<button :disabled="isItemCopying(item)">Copy</button>
	<button x-bind:disabled="canProceed()">Proceed</button>

	<!-- Non-boolean attributes with dotted paths are permitted -->
	<span :class="theme.activeClass">Text</span>
	<input :value="user.profile.name" />
	<a :href="item.url">Link</a>

	<!-- Comments should be ignored even if containing violations -->
	<!-- <button :disabled="item.loading">Old code</button> -->
	<!--
		<input :checked="model.checked" />
	-->
</body>
</html>
`)

	violations := ScanHTMLForAlpineViolations("test.html", html)
	if len(violations) != 0 {
		t.Fatalf("expected 0 violations for valid HTML, got %d: %+v", len(violations), violations)
	}
}

func TestScanHTMLForAlpineViolations_DottedViolations(t *testing.T) {
	tests := []struct {
		name      string
		snippet   string
		wantAttr  string
		wantVal   string
		wantCount int
	}{
		{
			name:      "simple dotted disabled",
			snippet:   `<button :disabled="item.loading">Save</button>`,
			wantAttr:  ":disabled",
			wantVal:   "item.loading",
			wantCount: 1,
		},
		{
			name:      "x-bind dotted checked",
			snippet:   `<input type="checkbox" x-bind:checked="audit.active" />`,
			wantAttr:  "x-bind:checked",
			wantVal:   "audit.active",
			wantCount: 1,
		},
		{
			name: "multiline dotted disabled",
			snippet: `<button
				type="button"
				:disabled="
					state.isProcessing
				">Submit</button>`,
			wantAttr:  ":disabled",
			wantVal:   "state.isProcessing",
			wantCount: 1,
		},
		{
			name:      "single quoted dotted readonly",
			snippet:   `<input :readonly='doc.isLocked' />`,
			wantAttr:  ":readonly",
			wantVal:   "doc.isLocked",
			wantCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			violations := ScanHTMLForAlpineViolations("sample.html", []byte(tc.snippet))
			if len(violations) != tc.wantCount {
				t.Fatalf("expected %d violation, got %d: %+v", tc.wantCount, len(violations), violations)
			}
			v := violations[0]
			if v.Attribute != tc.wantAttr {
				t.Errorf("expected attribute %q, got %q", tc.wantAttr, v.Attribute)
			}
			if v.Value != tc.wantVal {
				t.Errorf("expected value %q, got %q", tc.wantVal, v.Value)
			}
		})
	}
}

func TestCheckLOC(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. File under threshold
	shortFile := filepath.Join(tmpDir, "short.go")
	if err := os.WriteFile(shortFile, []byte("package foo\n\nfunc Bar() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := checkLOC(tmpDir, 10, false); err != nil {
		t.Fatalf("expected nil error for short file, got: %v", err)
	}

	// 2. File exceeding threshold
	longContent := strings.Repeat("line\n", 20)
	longFile := filepath.Join(tmpDir, "long.go")
	if err := os.WriteFile(longFile, []byte(longContent), 0644); err != nil {
		t.Fatal(err)
	}

	err := checkLOC(tmpDir, 10, false)
	if err == nil {
		t.Fatalf("expected error for file exceeding LOC threshold 10, got nil")
	}
	if !strings.Contains(err.Error(), "long.go") {
		t.Errorf("expected error message to mention 'long.go', got: %v", err)
	}

	// 3. Test files should be ignored even if exceeding threshold
	_ = os.Remove(longFile)
	longTestFile := filepath.Join(tmpDir, "long_test.go")
	if err := os.WriteFile(longTestFile, []byte(longContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := checkLOC(tmpDir, 10, false); err != nil {
		t.Fatalf("expected _test.go files to be ignored, got: %v", err)
	}
}

func TestCheckFmtAndScripts(t *testing.T) {
	tmpDir := t.TempDir()

	// Formatted Go file
	validFile := filepath.Join(tmpDir, "valid.go")
	validContent := "package sample\n\nfunc Hello() string {\n\treturn \"world\"\n}\n"
	if err := os.WriteFile(validFile, []byte(validContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := checkFmtAndScripts(tmpDir, false); err != nil {
		t.Fatalf("expected nil error for formatted Go file, got: %v", err)
	}

	// Unformatted Go file
	invalidFile := filepath.Join(tmpDir, "unformatted.go")
	invalidContent := "package sample\nfunc BadFormat( )   string{return \"bad\"}\n"
	if err := os.WriteFile(invalidFile, []byte(invalidContent), 0644); err != nil {
		t.Fatal(err)
	}

	err := checkFmtAndScripts(tmpDir, false)
	if err == nil {
		t.Fatalf("expected error for unformatted Go file, got nil")
	}
	if !strings.Contains(err.Error(), "unformatted.go") {
		t.Errorf("expected error message to mention unformatted.go, got: %v", err)
	}
}

func TestShouldSkip(t *testing.T) {
	cases := []struct {
		path     string
		expected bool
	}{
		{".git/config", true},
		{"node_modules/pkg/index.js", true},
		{"site/index.html", true},
		{"_site/assets/style.css", true},
		{".build/bin", true},
		{"internal/services/service.go", false},
		{"dashboard/frontend/app.js", false},
	}

	for _, c := range cases {
		got := shouldSkip(c.path)
		if got != c.expected {
			t.Errorf("shouldSkip(%q) = %v, expected %v", c.path, got, c.expected)
		}
	}
}
func TestPartitionArgs(t *testing.T) {
	cases := []struct {
		name         string
		args         []string
		wantFlagArgs []string
		wantPosArgs  []string
	}{
		{
			name:         "stage with trailing verbose flag",
			args:         []string{"all", "-v"},
			wantFlagArgs: []string{"-v"},
			wantPosArgs:  []string{"all"},
		},
		{
			name:         "stage with leading verbose flag",
			args:         []string{"-v", "fmt"},
			wantFlagArgs: []string{"-v"},
			wantPosArgs:  []string{"fmt"},
		},
		{
			name:         "loc with positional limit and trailing flag",
			args:         []string{"loc", "350", "-v"},
			wantFlagArgs: []string{"-v"},
			wantPosArgs:  []string{"loc", "350"},
		},
		{
			name:         "loc with flag limit separated by space",
			args:         []string{"-loc", "350", "loc"},
			wantFlagArgs: []string{"-loc", "350"},
			wantPosArgs:  []string{"loc"},
		},
		{
			name:         "loc with flag limit joined by equal",
			args:         []string{"loc", "-loc=250", "--verbose"},
			wantFlagArgs: []string{"-loc=250", "--verbose"},
			wantPosArgs:  []string{"loc"},
		},
		{
			name:         "test with race and min-coverage",
			args:         []string{"test", "-race", "--min-coverage", "88.5"},
			wantFlagArgs: []string{"-race", "--min-coverage", "88.5"},
			wantPosArgs:  []string{"test"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			flags, pos := partitionArgs(tc.args)
			if strings.Join(flags, " ") != strings.Join(tc.wantFlagArgs, " ") {
				t.Errorf("flags mismatch: got %v, want %v", flags, tc.wantFlagArgs)
			}
			if strings.Join(pos, " ") != strings.Join(tc.wantPosArgs, " ") {
				t.Errorf("pos mismatch: got %v, want %v", pos, tc.wantPosArgs)
			}
		})
	}
}

func TestParseConfigWithEnv(t *testing.T) {
	noEnv := func(k string) string { return "" }

	t.Run("default empty args", func(t *testing.T) {
		cfg, isHelp, err := parseConfigWithEnv([]string{}, noEnv)
		if err != nil || isHelp {
			t.Fatalf("unexpected err: %v or isHelp: %v", err, isHelp)
		}
		if cfg.Stage != "all" || cfg.Verbose || cfg.LocThreshold != 400 || cfg.MinCoverage != 80.0 {
			t.Errorf("unexpected defaults: %+v", cfg)
		}
	})

	t.Run("stage all with trailing verbose", func(t *testing.T) {
		cfg, isHelp, err := parseConfigWithEnv([]string{"all", "-v"}, noEnv)
		if err != nil || isHelp {
			t.Fatalf("unexpected err: %v or isHelp: %v", err, isHelp)
		}
		if cfg.Stage != "all" || !cfg.Verbose {
			t.Errorf("expected Stage=all, Verbose=true, got %+v", cfg)
		}
	})

	t.Run("stage loc with positional threshold and verbose", func(t *testing.T) {
		cfg, isHelp, err := parseConfigWithEnv([]string{"loc", "320", "-v"}, noEnv)
		if err != nil || isHelp {
			t.Fatalf("unexpected err: %v or isHelp: %v", err, isHelp)
		}
		if cfg.Stage != "loc" || cfg.LocThreshold != 320 || !cfg.Verbose {
			t.Errorf("expected Stage=loc, Loc=320, Verbose=true, got %+v", cfg)
		}
	})

	t.Run("stage test with race and min-coverage", func(t *testing.T) {
		cfg, isHelp, err := parseConfigWithEnv([]string{"test", "-race", "--min-coverage", "92.5"}, noEnv)
		if err != nil || isHelp {
			t.Fatalf("unexpected err: %v or isHelp: %v", err, isHelp)
		}
		if cfg.Stage != "test" || !cfg.RunRace || cfg.MinCoverage != 92.5 {
			t.Errorf("expected Stage=test, Race=true, MinCov=92.5, got %+v", cfg)
		}
	})

	t.Run("help flag and positional", func(t *testing.T) {
		for _, arg := range [][]string{{"help"}, {"-h"}, {"--help"}} {
			cfg, isHelp, err := parseConfigWithEnv(arg, noEnv)
			if err != nil || !isHelp {
				t.Errorf("expected isHelp=true for %v, got isHelp=%v, err=%v, cfg=%+v", arg, isHelp, err, cfg)
			}
		}
	})

	t.Run("environment variable fallback", func(t *testing.T) {
		mockEnv := map[string]string{
			"VERBOSE":       "1",
			"LOC_THRESHOLD": "300",
			"MIN_COVERAGE":  "85.0",
			"RUN_RACE":      "true",
		}
		cfg, isHelp, err := parseConfigWithEnv([]string{"test"}, func(k string) string {
			return mockEnv[k]
		})
		if err != nil || isHelp {
			t.Fatalf("unexpected err: %v or isHelp: %v", err, isHelp)
		}
		if !cfg.Verbose || cfg.LocThreshold != 300 || cfg.MinCoverage != 85.0 || !cfg.RunRace {
			t.Errorf("expected env overrides to take effect, got %+v", cfg)
		}
	})
}
