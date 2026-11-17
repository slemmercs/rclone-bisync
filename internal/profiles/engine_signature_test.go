package profiles

import (
	"reflect"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/core/client"
)

func TestParseProcessGlobalFlags(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    ProcessGlobalFlags
		expectError bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: ProcessGlobalFlags{},
		},
		{
			name:  "space separated flags",
			input: "--checkers 8 --tpslimit 12.5 --tpslimit-burst 25",
			expected: ProcessGlobalFlags{
				Checkers:      client.IntPtr(8),
				TPSLimit:      client.Float64Ptr(12.5),
				TPSLimitBurst: client.IntPtr(25),
			},
		},
		{
			name:  "equals separated flags",
			input: "--checkers=16 --tpslimit=4.0 --tpslimit-burst=10",
			expected: ProcessGlobalFlags{
				Checkers:      client.IntPtr(16),
				TPSLimit:      client.Float64Ptr(4.0),
				TPSLimitBurst: client.IntPtr(10),
			},
		},
		{
			name:  "explicit zero tpslimit (unlimited)",
			input: "--tpslimit 0 --checkers 4",
			expected: ProcessGlobalFlags{
				Checkers: client.IntPtr(4),
				TPSLimit: client.Float64Ptr(0.0),
			},
		},
		{
			name:  "mixed with job-scoped and boolean flags",
			input: "--transfers 4 --dry-run --checkers 12 --buffer-size 32M --tpslimit 5",
			expected: ProcessGlobalFlags{
				Checkers: client.IntPtr(12),
				TPSLimit: client.Float64Ptr(5.0),
			},
		},
		{
			name:        "missing value for checkers",
			input:       "--checkers",
			expectError: true,
		},
		{
			name:        "invalid checkers value",
			input:       "--checkers invalid",
			expectError: true,
		},
		{
			name:        "negative checkers value",
			input:       "--checkers -1",
			expectError: true,
		},
		{
			name:        "negative tpslimit value",
			input:       "--tpslimit -5",
			expectError: true,
		},
		{
			name:        "missing value for tpslimit-burst",
			input:       "--tpslimit-burst",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := ParseProcessGlobalFlags(tt.input)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error for input %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if (res.Checkers == nil) != (tt.expected.Checkers == nil) ||
				(res.Checkers != nil && *res.Checkers != *tt.expected.Checkers) {
				t.Errorf("Checkers mismatch: got %v, expected %v", res.Checkers, tt.expected.Checkers)
			}
			if (res.TPSLimit == nil) != (tt.expected.TPSLimit == nil) ||
				(res.TPSLimit != nil && *res.TPSLimit != *tt.expected.TPSLimit) {
				t.Errorf("TPSLimit mismatch: got %v, expected %v", res.TPSLimit, tt.expected.TPSLimit)
			}
			if (res.TPSLimitBurst == nil) != (tt.expected.TPSLimitBurst == nil) ||
				(res.TPSLimitBurst != nil && *res.TPSLimitBurst != *tt.expected.TPSLimitBurst) {
				t.Errorf("TPSLimitBurst mismatch: got %v, expected %v", res.TPSLimitBurst, tt.expected.TPSLimitBurst)
			}
		})
	}
}
func TestEngineSignature_EquivalenceAndFormatting(t *testing.T) {
	// 1. Empty flags yield DefaultEngineSignature
	empty := ProcessGlobalFlags{}
	if empty.Signature() != DefaultEngineSignature {
		t.Fatalf("expected empty flags to produce %s, got %s", DefaultEngineSignature, empty.Signature())
	}

	// 2. Different ordering / formatting produces equivalent signatures
	flagsA, errA := ParseProcessGlobalFlags("--checkers 8 --tpslimit 4")
	flagsB, errB := ParseProcessGlobalFlags("--tpslimit 4.0 --checkers 8")
	if errA != nil || errB != nil {
		t.Fatalf("unexpected parse errors: %v, %v", errA, errB)
	}

	sigA := flagsA.Signature()
	sigB := flagsB.Signature()
	if sigA != sigB {
		t.Fatalf("expected equivalent signatures, got sigA=%s and sigB=%s", sigA, sigB)
	}
	if sigA != "SIG_CHK=8_TPS=4" {
		t.Fatalf("expected SIG_CHK=8_TPS=4, got %s", sigA)
	}
	if !sigA.IsCompatible(sigB) {
		t.Fatal("expected sigA and sigB to be compatible")
	}

	// 3. Signature with burst
	flagsC, _ := ParseProcessGlobalFlags("--checkers 4 --tpslimit 10 --tpslimit-burst 20")
	if flagsC.Signature() != "SIG_CHK=4_TPS=10_BURST=20" {
		t.Fatalf("expected SIG_CHK=4_TPS=10_BURST=20, got %s", flagsC.Signature())
	}

	// 4. Incompatible signatures
	if sigA.IsCompatible(flagsC.Signature()) {
		t.Fatal("expected sigA and flagsC to be incompatible")
	}
}

func TestProcessGlobalFlags_Merge(t *testing.T) {
	fleet := ProcessGlobalFlags{
		Checkers:      client.IntPtr(8),
		TPSLimit:      client.Float64Ptr(20.0),
		TPSLimitBurst: client.IntPtr(40),
	}

	profileOverride := ProcessGlobalFlags{
		TPSLimit: client.Float64Ptr(5.0), // Overrides only TPSLimit
	}

	merged := fleet.Merge(profileOverride)

	if *merged.Checkers != 8 {
		t.Errorf("expected Checkers: 8, got %d", *merged.Checkers)
	}
	if *merged.TPSLimit != 5.0 {
		t.Errorf("expected TPSLimit: 5.0, got %f", *merged.TPSLimit)
	}
	if *merged.TPSLimitBurst != 40 {
		t.Errorf("expected TPSLimitBurst: 40, got %d", *merged.TPSLimitBurst)
	}
	if merged.Signature() != "SIG_CHK=8_TPS=5_BURST=40" {
		t.Errorf("expected SIG_CHK=8_TPS=5_BURST=40, got %s", merged.Signature())
	}
}

func TestProcessGlobalFlags_ToAndFromMainOptions(t *testing.T) {
	flags := ProcessGlobalFlags{
		Checkers:      client.IntPtr(16),
		TPSLimit:      client.Float64Ptr(10.5),
		TPSLimitBurst: client.IntPtr(30),
	}

	mainOpts := flags.ToMainOptions()
	if *mainOpts.Checkers != 16 || *mainOpts.TPSLimit != 10.5 || *mainOpts.TPSLimitBurst != 30 {
		t.Fatalf("ToMainOptions conversion failed: %+v", mainOpts)
	}

	roundTrip := FromMainOptions(mainOpts)
	if *roundTrip.Checkers != 16 || *roundTrip.TPSLimit != 10.5 || *roundTrip.TPSLimitBurst != 30 {
		t.Fatalf("FromMainOptions round-trip failed: %+v", roundTrip)
	}

	// Nil round trip
	nilFlags := FromMainOptions(nil)
	if !nilFlags.IsEmpty() {
		t.Fatal("expected nil mainOpts to convert to empty ProcessGlobalFlags")
	}
}

func TestStripProcessGlobalFlags(t *testing.T) {
	input := "--transfers 4 --checkers 8 --dry-run --tpslimit=10 --buffer-size 16M --tpslimit-burst 20"
	stripped, err := StripProcessGlobalFlags(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "--transfers 4 --dry-run --buffer-size 16M"
	if stripped != expected {
		t.Fatalf("expected stripped string %q, got %q", expected, stripped)
	}

	// Empty input
	empty, err := StripProcessGlobalFlags("")
	if err != nil || empty != "" {
		t.Fatalf("expected empty result, got %q, err=%v", empty, err)
	}
}

func TestFilterProcessGlobalTokens(t *testing.T) {
	tokens := []string{"--transfers", "4", "--checkers", "8", "--dry-run", "--tpslimit=10", "--fast-list"}
	filtered := FilterProcessGlobalTokens(tokens)
	expected := []string{"--transfers", "4", "--dry-run", "--fast-list"}

	if !reflect.DeepEqual(filtered, expected) {
		t.Fatalf("expected filtered tokens %v, got %v", expected, filtered)
	}

	if FilterProcessGlobalTokens(nil) != nil {
		t.Fatal("expected nil for nil tokens")
	}
}

func TestResolveEngineSignature(t *testing.T) {
	fleet := "--checkers 8 --tpslimit 20"
	profile := "--tpslimit 5"

	flags, sig, err := ResolveEngineSignature(profile, fleet)
	if err != nil {
		t.Fatalf("unexpected error from ResolveEngineSignature: %v", err)
	}

	if *flags.Checkers != 8 || *flags.TPSLimit != 5.0 {
		t.Fatalf("unexpected resolved flags: %+v", flags)
	}
	if sig != "SIG_CHK=8_TPS=5" {
		t.Fatalf("expected signature SIG_CHK=8_TPS=5, got %s", sig)
	}

	// Malformed inputs
	if _, _, err := ResolveEngineSignature("invalid", "--checkers"); err == nil {
		t.Fatal("expected error on invalid fleet flags")
	}
	if _, _, err := ResolveEngineSignature("--checkers", ""); err == nil {
		t.Fatal("expected error on invalid profile flags")
	}
}

func TestClusterProfiles(t *testing.T) {
	profiles := []*Profile{
		{
			ID:    "p1",
			Flags: ProfileFlags{CustomFlags: "--checkers 8 --tpslimit 5"},
		},
		{
			ID:    "p2",
			Flags: ProfileFlags{CustomFlags: "--tpslimit 5.0 --checkers 8"},
		},
		{
			ID:    "p3",
			Flags: ProfileFlags{CustomFlags: "--tpslimit 10"},
		},
		{
			ID:    "p4",
			Flags: ProfileFlags{}, // Baseline
		},
	}

	fleetFlags := "--checkers 8"

	clusters, err := ClusterProfiles(profiles, fleetFlags)
	if err != nil {
		t.Fatalf("unexpected error from ClusterProfiles: %v", err)
	}

	// p1 and p2 must be in the same cluster: SIG_CHK=8_TPS=5
	cluster5 := clusters["SIG_CHK=8_TPS=5"]
	if len(cluster5) != 2 || cluster5[0] != "p1" || cluster5[1] != "p2" {
		t.Fatalf("expected p1 and p2 in cluster SIG_CHK=8_TPS=5, got %v", cluster5)
	}

	// p3 has fleet checkers 8 + tpslimit 10 -> SIG_CHK=8_TPS=10
	cluster10 := clusters["SIG_CHK=8_TPS=10"]
	if len(cluster10) != 1 || cluster10[0] != "p3" {
		t.Fatalf("expected p3 in cluster SIG_CHK=8_TPS=10, got %v", cluster10)
	}

	// p4 inherits fleet checkers 8 -> SIG_CHK=8
	clusterDefault := clusters["SIG_CHK=8"]
	if len(clusterDefault) != 1 || clusterDefault[0] != "p4" {
		t.Fatalf("expected p4 in cluster SIG_CHK=8, got %v", clusterDefault)
	}

	// Malformed profile custom flag error
	malformed := []*Profile{
		{ID: "bad", Flags: ProfileFlags{CustomFlags: "--checkers"}},
	}
	if _, err := ClusterProfiles(malformed, fleetFlags); err == nil {
		t.Fatal("expected error for malformed custom flag in ClusterProfiles")
	}
}
