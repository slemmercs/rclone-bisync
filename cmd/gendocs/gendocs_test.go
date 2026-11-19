package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMapGoTypeToTS(t *testing.T) {
	tests := []struct {
		goType string
		want   string
	}{
		{"string", "string"},
		{"int", "number"},
		{"int64", "number"},
		{"float64", "number"},
		{"bool", "boolean"},
		{"interface{}", "any"},
		{"any", "any"},
		{"*string", "string"},
		{"[]string", "string[]"},
		{"*[]int", "number[]"},
		{"map[string]string", "Record<string, string>"},
		{"map[string]interface{}", "Record<string, any>"},
		{"time.Time", "string"},
		{"*time.Time", "string"},
		{"MaxDeleteValue", "string | number"},
		{"ProviderOptionsMap", "Record<string, string | boolean | number>"},
	}

	for _, tt := range tests {
		got := mapGoTypeToTS(tt.goType)
		if got != tt.want {
			t.Errorf("mapGoTypeToTS(%q) = %q; want %q", tt.goType, got, tt.want)
		}
	}
}

func TestGenerateTypeScript(t *testing.T) {
	structs := []StructDoc{
		{
			Name:        "TestModel",
			Description: "TestModel is a sample model.",
			Fields: []FieldDoc{
				{
					Name:        "ID",
					JSONKey:     "id",
					Type:        "string",
					Description: "Unique ID",
					IsOptional:  false,
				},
				{
					Name:        "OptionalCount",
					JSONKey:     "count",
					Type:        "*int",
					Description: "Optional count",
					IsOptional:  true,
				},
			},
		},
	}

	out := generateTypeScript(structs)
	if !strings.Contains(out, "export interface TestModel {") {
		t.Fatalf("expected TestModel interface in output, got:\n%s", out)
	}
	if !strings.Contains(out, "id: string") {
		t.Errorf("expected 'id: string' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "count?: number") {
		t.Errorf("expected 'count?: number' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Unique ID") {
		t.Errorf("expected doc comment in output, got:\n%s", out)
	}
}

func TestRenderFunctions(t *testing.T) {
	var buf bytes.Buffer
	renderFrontMatter(&buf)
	if !strings.Contains(buf.String(), "Technical & Settings Reference") {
		t.Errorf("renderFrontMatter failed to include header")
	}

	buf.Reset()
	docs := []StructDoc{
		{
			Name:        "SampleConfig",
			Description: "Sample config struct",
			Fields: []FieldDoc{
				{Name: "Key", JSONKey: "key", Type: "string", Description: "Key desc"},
			},
		},
	}
	renderGlobalConfig(&buf, docs)
	if !strings.Contains(buf.String(), "### `SampleConfig`") {
		t.Errorf("renderGlobalConfig missing struct name")
	}

	buf.Reset()
	renderProfilePolicies(&buf, docs)
	if !strings.Contains(buf.String(), "### `SampleConfig`") {
		t.Errorf("renderProfilePolicies missing struct name")
	}

	buf.Reset()
	tools := []MCPToolDoc{
		{
			Name:         "bisync_test_tool",
			Description:  "A test tool",
			Confirmation: true,
			Parameters: []MCPParamDoc{
				{Name: "param1", Type: "string", Required: true, Description: "Param 1"},
			},
		},
	}
	renderMCPTools(&buf, tools)
	if !strings.Contains(buf.String(), "bisync_test_tool") {
		t.Errorf("renderMCPTools missing tool")
	}
	if !strings.Contains(buf.String(), "SHA-256 Ticket") {
		t.Errorf("renderMCPTools missing confirmation badge")
	}
}

func TestEndToEndDocGen(t *testing.T) {
	tmpDir := t.TempDir()
	outMd := filepath.Join(tmpDir, "test.md")
	outTs := filepath.Join(tmpDir, "test.ts")

	repoRoot := "../.."
	if _, err := os.Stat("internal/models/api.go"); err == nil {
		repoRoot = "."
	}

	if err := generateMarkdownDocs(repoRoot, outMd); err != nil {
		t.Fatalf("generateMarkdownDocs failed: %v", err)
	}
	if err := generateTypeScriptContracts(repoRoot, outTs); err != nil {
		t.Fatalf("generateTypeScriptContracts failed: %v", err)
	}

	mdContent, err := os.ReadFile(outMd)
	if err != nil || len(mdContent) == 0 {
		t.Fatalf("markdown file empty or unreadable: %v", err)
	}

	tsContent, err := os.ReadFile(outTs)
	if err != nil || len(tsContent) == 0 {
		t.Fatalf("typescript file empty or unreadable: %v", err)
	}
}
