package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ModuleInfo mirrors the JSON schema emitted by `go list -m -json all`.
type ModuleInfo struct {
	Path     string      `json:"Path"`
	Version  string      `json:"Version"`
	Dir      string      `json:"Dir"`
	Main     bool        `json:"Main"`
	Indirect bool        `json:"Indirect"`
	Replace  *ModuleInfo `json:"Replace"`
}

var copyleftMatchers = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{"AGPL", regexp.MustCompile(`(?i)\b(affero\s+general\s+public\s+license|agpl-([123]|v[123]))\b`)},
	{"GPL", regexp.MustCompile(`(?i)\b(gnu\s+general\s+public\s+license|gpl-([123]|v[123]))\b`)},
	{"LGPL", regexp.MustCompile(`(?i)\b(gnu\s+lesser\s+general\s+public\s+license|gnu\s+library\s+general\s+public\s+license|lgpl-([123]|v[123]))\b`)},
	{"SSPL", regexp.MustCompile(`(?i)\b(server\s+side\s+public\s+license|sspl)\b`)},
	{"EUPL", regexp.MustCompile(`(?i)\b(european\s+union\s+public\s+licen[sc]e|eupl)\b`)},
	{"CC-BY-SA", regexp.MustCompile(`(?i)\b(creative\s+commons\s+attribution-sharealike|cc-by-sa)\b`)},
}

var licenseFileNames = []string{
	"LICENSE",
	"LICENSE.txt",
	"LICENSE.md",
	"LICENCE",
	"LICENCE.txt",
	"LICENCE.md",
	"COPYING",
	"COPYING.txt",
	"UNLICENSE",
	"NOTICE",
}

// checkLicenses audits Go dependencies and frontend assets for license compliance and copyleft avoidance.
func checkLicenses(root string, verbose bool) error {
	// 1. Audit THIRD_PARTY_LICENSES.md completeness
	if err := verifyThirdPartyLicensesDoc(root); err != nil {
		return err
	}

	// 2. Query all Go dependencies via go list
	modules, err := queryGoModules(root)
	if err != nil {
		return fmt.Errorf("failed to list Go modules: %w", err)
	}

	var copyleftViolations []string
	var missingLicenses []string
	auditedCount := 0

	for _, mod := range modules {
		if mod.Main {
			continue
		}
		modDir := mod.Dir
		if mod.Replace != nil && mod.Replace.Dir != "" {
			modDir = mod.Replace.Dir
		}
		if modDir == "" {
			continue
		}

		licText, licFile := findLicenseContent(modDir)
		if licText == "" {
			missingLicenses = append(missingLicenses, fmt.Sprintf("%s (%s)", mod.Path, mod.Version))
			continue
		}

		auditedCount++

		// Check for copyleft triggers
		for _, cm := range copyleftMatchers {
			if cm.pattern.MatchString(licText) {
				// Guard: Ensure it's not a mere disclaimer or dual-license exception
				if isCopyleftViolation(licText, cm.name) {
					copyleftViolations = append(copyleftViolations, fmt.Sprintf("%s [%s in %s]", mod.Path, cm.name, licFile))
					break
				}
			}
		}
	}

	if len(copyleftViolations) > 0 {
		return fmt.Errorf("copyleft license violations detected:\n  %s", strings.Join(copyleftViolations, "\n  "))
	}

	if len(missingLicenses) > 0 {
		return fmt.Errorf("missing license files for dependencies:\n  %s", strings.Join(missingLicenses, "\n  "))
	}

	if verbose {
		fmt.Printf("    Audited %d Go modules: 0 copyleft violations, all licenses verified\n", auditedCount)
		fmt.Printf("    Audited THIRD_PARTY_LICENSES.md: all direct runtime dependencies documented\n")
	}

	return nil
}

func queryGoModules(root string) ([]ModuleInfo, error) {
	cmd := exec.Command("go", "list", "-m", "-json", "all")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		// Attempt go mod download and retry
		_ = exec.Command("go", "mod", "download").Run()
		cmd = exec.Command("go", "list", "-m", "-json", "all")
		cmd.Dir = root
		out, err = cmd.Output()
		if err != nil {
			return nil, err
		}
	}

	var modules []ModuleInfo
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var m ModuleInfo
		if err := dec.Decode(&m); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		modules = append(modules, m)
	}
	return modules, nil
}

func findLicenseContent(dir string) (string, string) {
	for _, name := range licenseFileNames {
		target := filepath.Join(dir, name)
		// #nosec G304 - quality gate checks license content in module directories
		if content, err := os.ReadFile(target); err == nil && len(content) > 0 {
			return string(content), name
		}
	}

	// Secondary check: search directory entries for any file starting with LICENSE or LICENCE
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			lower := strings.ToLower(e.Name())
			if strings.HasPrefix(lower, "license") || strings.HasPrefix(lower, "licence") {
				// #nosec G304 - quality gate checks license files in module cache
				if content, err := os.ReadFile(filepath.Join(dir, e.Name())); err == nil && len(content) > 0 {
					return string(content), e.Name()
				}
			}
		}
	}
	return "", ""
}

func isCopyleftViolation(content, copyleftType string) bool {
	lower := strings.ToLower(content)

	// Mozilla Public License 2.0 defines Secondary Licenses (GPL/LGPL/AGPL) in sec 1.12
	if strings.Contains(lower, "mozilla public license") {
		return false
	}

	if strings.Contains(lower, "mit license") ||
		strings.Contains(lower, "permission is hereby granted, free of charge") ||
		strings.Contains(lower, "apache license, version 2.0") ||
		strings.Contains(lower, "apache license\nversion 2.0") ||
		strings.Contains(lower, "redistribution and use in source and binary forms") {
		// If it explicitly grants GPL/LGPL/AGPL as sole or primary license:
		if strings.Contains(lower, "everyone is permitted to copy and distribute verbatim copies") ||
			strings.Contains(lower, "general public license\nversion") ||
			strings.Contains(lower, "general public license version") {
			return true
		}
		return false
	}
	return true
}

func verifyThirdPartyLicensesDoc(root string) error {
	docPath := filepath.Join(root, "THIRD_PARTY_LICENSES.md")
	// #nosec G304 - quality gate verifies third party license doc in repo root
	data, err := os.ReadFile(docPath)
	if err != nil {
		return fmt.Errorf("THIRD_PARTY_LICENSES.md missing: %w", err)
	}
	content := string(data)

	// Required frontend and icon components
	requiredComponents := []string{
		"Lucide",
		"Alpine.js",
		"Chart.js",
		"Tailwind CSS",
	}
	for _, comp := range requiredComponents {
		if !strings.Contains(content, comp) {
			return fmt.Errorf("THIRD_PARTY_LICENSES.md is missing documentation for: %s", comp)
		}
	}

	// Required direct Go runtime dependencies from go.mod
	// #nosec G304 - quality gate verifies direct dependencies in repo go.mod
	goModData, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err == nil {
		lines := strings.Split(string(goModData), "\n")
		inRequire := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "require (" {
				inRequire = true
				continue
			}
			if inRequire && trimmed == ")" {
				break
			}
			if inRequire && trimmed != "" && !strings.HasPrefix(trimmed, "//") {
				parts := strings.Fields(trimmed)
				if len(parts) >= 2 && !strings.Contains(trimmed, "// indirect") {
					modPath := parts[0]
					if !strings.Contains(content, modPath) {
						return fmt.Errorf("THIRD_PARTY_LICENSES.md is missing direct dependency: %s", modPath)
					}
				}
			}
		}
	}

	return nil
}
