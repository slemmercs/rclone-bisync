package config

// GlobalDefaults represents baseline operational settings applied across all profiles
// unless explicitly overridden at the profile or execution level.
type GlobalDefaults struct {
	MaxDelete        int    `json:"max_delete"`
	CanaryValidation bool   `json:"canary_validation"`
	ConflictResolve  string `json:"conflict_resolve"`
	CompareMode      string `json:"compare_mode"`
	RemoveEmptyDirs  bool   `json:"remove_empty_dirs"`
	CreateEmptyDirs  bool   `json:"create_empty_dirs"`
}

// DefaultGlobalDefaults returns the canonical default operational configuration.
func DefaultGlobalDefaults() GlobalDefaults {
	return GlobalDefaults{
		MaxDelete:        50,
		CanaryValidation: true,
		ConflictResolve:  "newer",
		CompareMode:      "modtime",
		RemoveEmptyDirs:  true,
		CreateEmptyDirs:  true,
	}
}
