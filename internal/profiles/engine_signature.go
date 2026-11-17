package profiles

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/core/client"
)

// DefaultEngineSignature represents the baseline engine settings without custom rate/concurrency overrides.
const DefaultEngineSignature EngineSignature = "SIG_DEFAULT"

// EngineSignature represents a normalized, comparable signature of engine configuration.
// Profiles with identical signatures are 100% compatible and can run concurrently.
type EngineSignature string

// IsCompatible returns true if two signatures are identical, meaning their profiles
// can run concurrently under the same Rclone process-global configuration.
func (s EngineSignature) IsCompatible(other EngineSignature) bool {
	return s == other
}

// ProcessGlobalFlags represents extracted process-global Rclone options
// that affect fs.Config.Main in the Rclone daemon.
type ProcessGlobalFlags struct {
	Checkers      *int     // --checkers
	TPSLimit      *float64 // --tpslimit
	TPSLimitBurst *int     // --tpslimit-burst
}

// IsEmpty returns true if no process-global options are configured.
func (p ProcessGlobalFlags) IsEmpty() bool {
	return p.Checkers == nil && p.TPSLimit == nil && p.TPSLimitBurst == nil
}

// Merge returns a new ProcessGlobalFlags with values from override taking precedence over base.
func (base ProcessGlobalFlags) Merge(override ProcessGlobalFlags) ProcessGlobalFlags {
	res := base
	if override.Checkers != nil {
		res.Checkers = override.Checkers
	}
	if override.TPSLimit != nil {
		res.TPSLimit = override.TPSLimit
	}
	if override.TPSLimitBurst != nil {
		res.TPSLimitBurst = override.TPSLimitBurst
	}
	return res
}

// Signature computes a deterministic EngineSignature from ProcessGlobalFlags.
func (p ProcessGlobalFlags) Signature() EngineSignature {
	if p.IsEmpty() {
		return DefaultEngineSignature
	}

	var parts []string
	if p.Checkers != nil {
		parts = append(parts, fmt.Sprintf("CHK=%d", *p.Checkers))
	}
	if p.TPSLimit != nil {
		parts = append(parts, fmt.Sprintf("TPS=%s", strconv.FormatFloat(*p.TPSLimit, 'f', -1, 64)))
	}
	if p.TPSLimitBurst != nil {
		parts = append(parts, fmt.Sprintf("BURST=%d", *p.TPSLimitBurst))
	}

	return EngineSignature("SIG_" + strings.Join(parts, "_"))
}

// ToMainOptions converts ProcessGlobalFlags into a client.MainOptions struct.
func (p ProcessGlobalFlags) ToMainOptions() *client.MainOptions {
	return &client.MainOptions{
		Checkers:      p.Checkers,
		TPSLimit:      p.TPSLimit,
		TPSLimitBurst: p.TPSLimitBurst,
	}
}

// FromMainOptions converts a client.MainOptions struct into ProcessGlobalFlags.
func FromMainOptions(opts *client.MainOptions) ProcessGlobalFlags {
	if opts == nil {
		return ProcessGlobalFlags{}
	}
	return ProcessGlobalFlags{
		Checkers:      opts.Checkers,
		TPSLimit:      opts.TPSLimit,
		TPSLimitBurst: opts.TPSLimitBurst,
	}
}

// ParseProcessGlobalFlags parses a CLI flags string and extracts known process-global flags.
func ParseProcessGlobalFlags(rawFlags string) (ProcessGlobalFlags, error) {
	var res ProcessGlobalFlags
	raw := strings.TrimSpace(rawFlags)
	if raw == "" {
		return res, nil
	}

	tokens, err := SplitCommandLine(raw)
	if err != nil {
		return res, fmt.Errorf("failed to parse custom flags: %w", err)
	}

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		var flagName, flagVal string
		hasInlineVal := false

		if idx := strings.Index(token, "="); idx != -1 {
			flagName = token[:idx]
			flagVal = token[idx+1:]
			hasInlineVal = true
		} else {
			flagName = token
		}

		switch flagName {
		case "--checkers":
			if !hasInlineVal {
				if i+1 >= len(tokens) {
					return res, fmt.Errorf("flag %s requires a value", flagName)
				}
				i++
				flagVal = tokens[i]
			}
			val, err := strconv.Atoi(flagVal)
			if err != nil || val < 1 {
				return res, fmt.Errorf("invalid value for %s: %q (must be positive integer)", flagName, flagVal)
			}
			res.Checkers = &val

		case "--tpslimit":
			if !hasInlineVal {
				if i+1 >= len(tokens) {
					return res, fmt.Errorf("flag %s requires a value", flagName)
				}
				i++
				flagVal = tokens[i]
			}
			val, err := strconv.ParseFloat(flagVal, 64)
			if err != nil || val < 0 {
				return res, fmt.Errorf("invalid value for %s: %q (must be non-negative float)", flagName, flagVal)
			}
			res.TPSLimit = &val

		case "--tpslimit-burst":
			if !hasInlineVal {
				if i+1 >= len(tokens) {
					return res, fmt.Errorf("flag %s requires a value", flagName)
				}
				i++
				flagVal = tokens[i]
			}
			val, err := strconv.Atoi(flagVal)
			if err != nil || val < 0 {
				return res, fmt.Errorf("invalid value for %s: %q (must be non-negative integer)", flagName, flagVal)
			}
			res.TPSLimitBurst = &val
		}
	}

	return res, nil
}

// StripProcessGlobalFlags removes process-global flags (--checkers, --tpslimit, --tpslimit-burst)
// from the custom flags string so they do not leak into per-job bisync arguments.
func StripProcessGlobalFlags(rawFlags string) (string, error) {
	raw := strings.TrimSpace(rawFlags)
	if raw == "" {
		return "", nil
	}

	tokens, err := SplitCommandLine(raw)
	if err != nil {
		return "", fmt.Errorf("failed to parse custom flags: %w", err)
	}

	var remaining []string
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		flagName := token
		hasInlineVal := false
		if idx := strings.Index(token, "="); idx != -1 {
			flagName = token[:idx]
			hasInlineVal = true
		}

		switch flagName {
		case "--checkers", "--tpslimit", "--tpslimit-burst":
			if !hasInlineVal {
				if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "-") {
					i++
				}
			}
			continue
		default:
			remaining = append(remaining, token)
		}
	}

	return strings.Join(remaining, " "), nil
}

// FilterProcessGlobalTokens removes process-global flags from an already-split token slice.
func FilterProcessGlobalTokens(tokens []string) []string {
	if len(tokens) == 0 {
		return nil
	}
	var filtered []string
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		flagName := token
		hasInlineVal := false
		if idx := strings.Index(token, "="); idx != -1 {
			flagName = token[:idx]
			hasInlineVal = true
		}

		switch flagName {
		case "--checkers", "--tpslimit", "--tpslimit-burst":
			if !hasInlineVal {
				if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "-") {
					i++
				}
			}
			continue
		default:
			filtered = append(filtered, token)
		}
	}
	return filtered
}

// ResolveEngineSignature resolves the effective ProcessGlobalFlags and EngineSignature for a profile
// given the profile's flags and the fleet baseline (globals).
func ResolveEngineSignature(profileCustomFlags string, fleetCustomFlags string) (ProcessGlobalFlags, EngineSignature, error) {
	fleetGlobals, err := ParseProcessGlobalFlags(fleetCustomFlags)
	if err != nil {
		return ProcessGlobalFlags{}, DefaultEngineSignature, fmt.Errorf("failed to parse fleet custom flags: %w", err)
	}

	profileGlobals, err := ParseProcessGlobalFlags(profileCustomFlags)
	if err != nil {
		return ProcessGlobalFlags{}, DefaultEngineSignature, fmt.Errorf("failed to parse profile custom flags: %w", err)
	}

	effective := fleetGlobals.Merge(profileGlobals)
	return effective, effective.Signature(), nil
}

// CompatibilityCluster groups profile IDs by their resolved EngineSignature.
type CompatibilityCluster map[EngineSignature][]string

// ClusterProfiles groups a list of profiles by their resolved EngineSignature.
func ClusterProfiles(profiles []*Profile, fleetCustomFlags string) (CompatibilityCluster, error) {
	cluster := make(CompatibilityCluster)
	for _, p := range profiles {
		if p == nil {
			continue
		}
		_, sig, err := ResolveEngineSignature(p.Flags.CustomFlags, fleetCustomFlags)
		if err != nil {
			return nil, fmt.Errorf("profile %q has invalid flags: %w", p.ID, err)
		}
		cluster[sig] = append(cluster[sig], p.ID)
	}
	return cluster, nil
}
