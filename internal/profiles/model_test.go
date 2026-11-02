package profiles

import (
	"encoding/json"
	"testing"
)

func TestProfileFlags_JSONSerialization(t *testing.T) {
	maxDel := MaxDeleteValue("15")
	cmpMode := "checksum"
	canary := false

	p := Profile{
		ID:         "prof_test1",
		Name:       "Test Profile",
		Remote:     "drive:",
		LocalPath:  "/tmp/local",
		Schedule:   "manual",
		ReviewMode: ReviewDisabled,
		Flags: ProfileFlags{
			MaxDelete:        &maxDel,
			CompareMode:      &cmpMode,
			CanaryValidation: &canary,
			CustomFlags:      "--bwlimit 10M",
		},
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("failed to marshal profile: %v", err)
	}

	var decoded Profile
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal profile: %v", err)
	}

	if decoded.Flags.MaxDelete == nil || *decoded.Flags.MaxDelete != "15" {
		t.Errorf("expected MaxDelete 15, got %v", decoded.Flags.MaxDelete)
	}
	if decoded.Flags.CompareMode == nil || *decoded.Flags.CompareMode != "checksum" {
		t.Errorf("expected CompareMode 'checksum', got %v", decoded.Flags.CompareMode)
	}
	if decoded.Flags.CanaryValidation == nil || *decoded.Flags.CanaryValidation != false {
		t.Errorf("expected CanaryValidation false, got %v", decoded.Flags.CanaryValidation)
	}
	if decoded.Flags.ConflictResolve != nil {
		t.Errorf("expected ConflictResolve nil (inherited), got %v", decoded.Flags.ConflictResolve)
	}
	if decoded.Flags.CustomFlags != "--bwlimit 10M" {
		t.Errorf("expected CustomFlags '--bwlimit 10M', got %s", decoded.Flags.CustomFlags)
	}
}

func TestMaxDeleteValue_FlexibleTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected MaxDeleteValue
	}{
		{"numeric integer", `{"max_delete": 20}`, "20"},
		{"string percentage", `{"max_delete": "25%"}`, "25%"},
		{"string integer", `{"max_delete": "10"}`, "10"},
		{"null", `{"max_delete": null}`, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var pf ProfileFlags
			if err := json.Unmarshal([]byte(tc.input), &pf); err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}
			if tc.expected == "" {
				if pf.MaxDelete != nil && *pf.MaxDelete != "" {
					t.Errorf("expected nil or empty, got %v", pf.MaxDelete)
				}
			} else {
				if pf.MaxDelete == nil || *pf.MaxDelete != tc.expected {
					t.Errorf("expected %s, got %v", tc.expected, pf.MaxDelete)
				}
			}
		})
	}
}
