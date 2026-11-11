package profiles

import (
	"encoding/json"
	"errors"
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

func TestProfile_Validate_EnumsAndNormalization(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	t.Run("valid schedule presets and case insensitivity", func(t *testing.T) {
		validSchedules := []string{"manual", "15m", "30m", "1h", "4h", "daily", "1H", "Daily", "15M"}
		for _, s := range validSchedules {
			p := Profile{
				ID:        "prof_valid_sched",
				Name:      "Valid Schedule Profile",
				Remote:    "myremote",
				LocalPath: "/tmp/data",
				Schedule:  s,
			}
			if err := p.Validate(); err != nil {
				t.Errorf("expected schedule %q to be valid, got: %v", s, err)
			}
		}
	})

	t.Run("invalid schedules rejected", func(t *testing.T) {
		invalidSchedules := []string{"0 * * * *", "45m", "weekly", "hourly", "2h", "invalid"}
		for _, s := range invalidSchedules {
			p := Profile{
				ID:        "prof_invalid_sched",
				Name:      "Invalid Schedule Profile",
				Remote:    "myremote",
				LocalPath: "/tmp/data",
				Schedule:  s,
			}
			err := p.Validate()
			if err == nil {
				t.Fatalf("expected error for schedule %q, got nil", s)
			}
			if !errors.Is(err, ErrInvalidSchedule) {
				t.Errorf("expected ErrInvalidSchedule for %q, got: %v", s, err)
			}
		}
	})

	t.Run("remote trailing colon normalization", func(t *testing.T) {
		p := Profile{
			ID:        "prof_colon",
			Name:      "Colon Profile",
			Remote:    "remote_storage::",
			LocalPath: "/tmp/data",
		}
		if err := p.Validate(); err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}
		if p.Remote != "remote_storage" {
			t.Errorf("expected remote 'remote_storage', got %q", p.Remote)
		}

		pEmpty := Profile{
			ID:        "prof_colon_empty",
			Name:      "Colon Empty Remote",
			Remote:    ":::",
			LocalPath: "/tmp/data",
		}
		if err := pEmpty.Validate(); !errors.Is(err, ErrInvalidRemote) {
			t.Errorf("expected ErrInvalidRemote for ':::', got %v", err)
		}
	})

	t.Run("review mode validation", func(t *testing.T) {
		for _, rm := range ValidReviewModes {
			p := Profile{
				ID:         "prof_rm",
				Name:       "Review Mode Profile",
				Remote:     "r",
				LocalPath:  "/tmp",
				ReviewMode: rm,
			}
			if err := p.Validate(); err != nil {
				t.Errorf("expected review mode %q to be valid, got %v", rm, err)
			}
		}

		pBad := Profile{
			ID:         "prof_rm_bad",
			Name:       "Bad Review Mode",
			Remote:     "r",
			LocalPath:  "/tmp",
			ReviewMode: "automatic",
		}
		if err := pBad.Validate(); !errors.Is(err, ErrInvalidReviewMode) {
			t.Errorf("expected ErrInvalidReviewMode, got: %v", err)
		}
	})

	t.Run("notification preference validation", func(t *testing.T) {
		for _, np := range ValidNotificationPrefs {
			p := Profile{
				ID:               "prof_np",
				Name:             "Notify Profile",
				Remote:           "r",
				LocalPath:        "/tmp",
				NotificationPref: np,
			}
			if err := p.Validate(); err != nil {
				t.Errorf("expected notification preference %q to be valid, got %v", np, err)
			}
		}

		pBad := Profile{
			ID:               "prof_np_bad",
			Name:             "Bad Notify Profile",
			Remote:           "r",
			LocalPath:        "/tmp",
			NotificationPref: "always",
		}
		if err := pBad.Validate(); !errors.Is(err, ErrInvalidNotificationPref) {
			t.Errorf("expected ErrInvalidNotificationPref, got: %v", err)
		}
	})

	t.Run("conflict resolve mode validation", func(t *testing.T) {
		for _, cm := range ValidConflictModes {
			p := Profile{
				ID:        "prof_cm",
				Name:      "Conflict Mode Profile",
				Remote:    "r",
				LocalPath: "/tmp",
				Flags: ProfileFlags{
					ConflictResolve: strPtr(cm),
				},
			}
			if err := p.Validate(); err != nil {
				t.Errorf("expected conflict mode %q to be valid, got %v", cm, err)
			}
		}

		pBad := Profile{
			ID:        "prof_cm_bad",
			Name:      "Bad Conflict Mode Profile",
			Remote:    "r",
			LocalPath: "/tmp",
			Flags: ProfileFlags{
				ConflictResolve: strPtr("ask-user"),
			},
		}
		if err := pBad.Validate(); !errors.Is(err, ErrInvalidConflictMode) {
			t.Errorf("expected ErrInvalidConflictMode, got: %v", err)
		}
	})

	t.Run("compare mode validation", func(t *testing.T) {
		for _, cmp := range ValidCompareModes {
			p := Profile{
				ID:        "prof_cmp",
				Name:      "Compare Mode Profile",
				Remote:    "r",
				LocalPath: "/tmp",
				Flags: ProfileFlags{
					CompareMode: strPtr(cmp),
				},
			}
			if err := p.Validate(); err != nil {
				t.Errorf("expected compare mode %q to be valid, got %v", cmp, err)
			}
		}

		pBad := Profile{
			ID:        "prof_cmp_bad",
			Name:      "Bad Compare Mode Profile",
			Remote:    "r",
			LocalPath: "/tmp",
			Flags: ProfileFlags{
				CompareMode: strPtr("hash-only"),
			},
		}
		if err := pBad.Validate(); !errors.Is(err, ErrInvalidCompareMode) {
			t.Errorf("expected ErrInvalidCompareMode, got: %v", err)
		}
	})
}

