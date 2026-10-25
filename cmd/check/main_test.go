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

	if err := checkLOC(tmpDir, 10); err != nil {
		t.Fatalf("expected nil error for short file, got: %v", err)
	}

	// 2. File exceeding threshold
	longContent := strings.Repeat("line\n", 20)
	longFile := filepath.Join(tmpDir, "long.go")
	if err := os.WriteFile(longFile, []byte(longContent), 0644); err != nil {
		t.Fatal(err)
	}

	err := checkLOC(tmpDir, 10)
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

	if err := checkLOC(tmpDir, 10); err != nil {
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

	if err := checkFmtAndScripts(tmpDir); err != nil {
		t.Fatalf("expected nil error for formatted Go file, got: %v", err)
	}

	// Unformatted Go file
	invalidFile := filepath.Join(tmpDir, "unformatted.go")
	invalidContent := "package sample\nfunc BadFormat( )   string{return \"bad\"}\n"
	if err := os.WriteFile(invalidFile, []byte(invalidContent), 0644); err != nil {
		t.Fatal(err)
	}

	err := checkFmtAndScripts(tmpDir)
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
