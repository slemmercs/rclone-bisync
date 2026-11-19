package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	outPath := flag.String("out", "", "Output markdown file path")
	tsOutPath := flag.String("ts-out", "", "Output TypeScript file path")
	flag.Parse()

	// Default to markdown doc generation if no flags provided
	if *outPath == "" && *tsOutPath == "" {
		*outPath = "site/technical-reference.md"
	}

	repoRoot := "."
	if _, err := os.Stat("internal/models/api.go"); os.IsNotExist(err) {
		repoRoot = ".."
	}

	if *outPath != "" {
		if err := generateMarkdownDocs(repoRoot, *outPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating markdown docs: %v\n", err)
			os.Exit(1)
		}
	}

	if *tsOutPath != "" {
		if err := generateTypeScriptContracts(repoRoot, *tsOutPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating TypeScript contracts: %v\n", err)
			os.Exit(1)
		}
	}
}

func generateMarkdownDocs(repoRoot, outPath string) error {
	apiModelsPath := filepath.Join(repoRoot, "internal/models/api.go")
	profileModelsPath := filepath.Join(repoRoot, "internal/profiles/model.go")
	mcpDir := filepath.Join(repoRoot, "internal/mcp")

	globalStructs := []string{
		"GlobalDefaults",
		"ConfigDaemonSettings",
		"ConfigLoggingSettings",
		"ConfigCleanupSettings",
		"ConfigTelemetrySettings",
		"ConfigFeatureSettings",
		"ConfigNotificationSettings",
	}

	profileStructs := []string{
		"Profile",
		"ProfileFlags",
		"SafetySettings",
		"ResolvedFlags",
	}

	apiDocs, err := parseStructsFromFile(apiModelsPath, globalStructs)
	if err != nil {
		return fmt.Errorf("parsing api models: %w", err)
	}

	profileDocs, err := parseStructsFromFile(profileModelsPath, profileStructs)
	if err != nil {
		return fmt.Errorf("parsing profile models: %w", err)
	}

	mcpTools, err := parseMCPTools(mcpDir)
	if err != nil {
		return fmt.Errorf("parsing MCP tools: %w", err)
	}

	var buf bytes.Buffer
	renderFrontMatter(&buf)
	renderGlobalConfig(&buf, apiDocs)
	renderProfilePolicies(&buf, profileDocs)
	renderMCPTools(&buf, mcpTools)

	// #nosec G301 - creating documentation directory
	if err := os.MkdirAll(filepath.Dir(outPath), 0750); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// #nosec G306 - writing public documentation file
	if err := os.WriteFile(outPath, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}

	fmt.Printf("✓ Successfully generated markdown docs %s (%d bytes)\n", outPath, buf.Len())
	return nil
}

func generateTypeScriptContracts(repoRoot, tsOutPath string) error {
	modelsDir := filepath.Join(repoRoot, "internal/models")
	profileModelsPath := filepath.Join(repoRoot, "internal/profiles/model.go")

	modelStructs, err := parseAllStructsFromDir(modelsDir)
	if err != nil {
		return fmt.Errorf("parsing models dir: %w", err)
	}

	profileStructs, err := parseStructsFromFile(profileModelsPath, []string{
		"SafetySettings",
		"ProfileFlags",
		"ResolvedFlags",
		"ExecutionOverrides",
		"OperationalState",
		"Profile",
		"ProfileVersion",
	})
	if err != nil {
		return fmt.Errorf("parsing profile structs: %w", err)
	}

	allStructs := append(modelStructs, profileStructs...)
	tsContent := generateTypeScript(allStructs)

	// #nosec G301 - creating types directory
	if err := os.MkdirAll(filepath.Dir(tsOutPath), 0750); err != nil {
		return fmt.Errorf("creating types directory: %w", err)
	}

	// #nosec G306 - writing generated TypeScript models
	if err := os.WriteFile(tsOutPath, []byte(tsContent), 0600); err != nil {
		return fmt.Errorf("writing TypeScript file: %w", err)
	}

	fmt.Printf("✓ Successfully generated TypeScript contracts %s (%d bytes)\n", tsOutPath, len(tsContent))
	return nil
}

