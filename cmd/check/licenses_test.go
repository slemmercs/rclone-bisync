package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsCopyleftViolation(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		copyleftType string
		want         bool
	}{
		{
			name:         "Pure GPLv3",
			content:      "GNU GENERAL PUBLIC LICENSE\nVersion 3, 29 June 2007\nEveryone is permitted to copy and distribute verbatim copies...",
			copyleftType: "GPL",
			want:         true,
		},
		{
			name:         "Pure AGPLv3",
			content:      "GNU AFFERO GENERAL PUBLIC LICENSE\nVersion 3, 19 November 2007\nEveryone is permitted to copy...",
			copyleftType: "AGPL",
			want:         true,
		},
		{
			name:         "Pure LGPLv3",
			content:      "GNU LESSER GENERAL PUBLIC LICENSE\nVersion 3, 29 June 2007\nEveryone is permitted to copy...",
			copyleftType: "LGPL",
			want:         true,
		},
		{
			name:         "Permissive MIT License",
			content:      "MIT License\n\nCopyright (c) 2026 Developer\nPermission is hereby granted, free of charge...",
			copyleftType: "GPL",
			want:         false,
		},
		{
			name:         "Permissive Apache 2.0",
			content:      "Apache License\nVersion 2.0, January 2004\nhttp://www.apache.org/licenses/",
			copyleftType: "GPL",
			want:         false,
		},
		{
			name:         "Permissive BSD 3-Clause",
			content:      "Copyright (c) 2026. All rights reserved.\nRedistribution and use in source and binary forms...",
			copyleftType: "GPL",
			want:         false,
		},
		{
			name:         "MPL-2.0 Mentioning Secondary GPL/LGPL/AGPL",
			content:      "Mozilla Public License, version 2.0\n1.12. Secondary License means either the GNU General Public License...",
			copyleftType: "AGPL",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCopyleftViolation(tt.content, tt.copyleftType)
			if got != tt.want {
				t.Errorf("isCopyleftViolation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerifyThirdPartyLicensesDoc(t *testing.T) {
	tempDir := t.TempDir()

	// Missing THIRD_PARTY_LICENSES.md
	if err := verifyThirdPartyLicensesDoc(tempDir); err == nil {
		t.Errorf("expected error when THIRD_PARTY_LICENSES.md is missing, got nil")
	}

	// Incomplete doc
	incompleteDoc := filepath.Join(tempDir, "THIRD_PARTY_LICENSES.md")
	_ = os.WriteFile(incompleteDoc, []byte("# Third Party Licenses\nLucide\n"), 0600)
	if err := verifyThirdPartyLicensesDoc(tempDir); err == nil {
		t.Errorf("expected error when THIRD_PARTY_LICENSES.md lacks components, got nil")
	}

	// Full doc with go.mod
	fullDoc := `# Third Party Licenses
- Lucide
- Alpine.js
- Chart.js
- Tailwind CSS
- github.com/test/pkg
`
	_ = os.WriteFile(incompleteDoc, []byte(fullDoc), 0600)
	goMod := `module myapp

go 1.26

require (
	github.com/test/pkg v1.0.0
	github.com/indirect/pkg v1.0.0 // indirect
)
`
	_ = os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0600)
	if err := verifyThirdPartyLicensesDoc(tempDir); err != nil {
		t.Errorf("expected valid doc to pass, got: %v", err)
	}
}

func TestFindLicenseContent(t *testing.T) {
	tempDir := t.TempDir()
	content, filename := findLicenseContent(tempDir)
	if content != "" || filename != "" {
		t.Errorf("expected empty content in empty dir, got %s, %s", content, filename)
	}

	licPath := filepath.Join(tempDir, "LICENSE")
	_ = os.WriteFile(licPath, []byte("MIT License Test"), 0600)
	content, filename = findLicenseContent(tempDir)
	if content != "MIT License Test" || filename != "LICENSE" {
		t.Errorf("unexpected license read: content=%q, filename=%q", content, filename)
	}
}
