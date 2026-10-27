package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type TargetType int

const (
	TypeIgnore TargetType = iota
	TypeParamBool
	TypeConfigBool
	TypeBackendBool
	TypeBackendStr
	TypeParam
	TypeConfigStr
	TypeConfigInt
	TypeConfigFloat
)

type OptionMapping struct {
	Type      TargetType
	TargetKey string
}

var CLIToRCMap = map[string]OptionMapping{
	// Logging & console formatting
	"--use-json-log": {TypeConfigBool, "LogJSON"},
	"--color":        {TypeIgnore, "color"},

	// Drive-specific options (passed as connection string options on path1)
	"--drive-export-formats":    {TypeBackendStr, "export_formats"},
	"--drive-import-formats":    {TypeBackendStr, "import_formats"},
	"--drive-skip-gdocs":        {TypeBackendBool, "skip_gdocs"},
	"--drive-use-trash":         {TypeBackendBool, "use_trash"},
	"--drive-acknowledge-abuse": {TypeBackendBool, "acknowledge_abuse"},
	"--drive-skip-shortcuts":    {TypeBackendBool, "skip_shortcuts"},

	// Bisync parameters
	"--check-access":      {TypeParamBool, "checkAccess"},
	"--check-first":       {TypeParamBool, "checkFirst"},
	"--track-renames":     {TypeParamBool, "trackRenames"},
	"--remove-empty-dirs": {TypeParamBool, "removeEmptyDirs"},
	"--resilient":         {TypeParamBool, "resilient"},
	"--recover":           {TypeParamBool, "recover"},
	"--compare":           {TypeParam, "compare"},

	// Conflict handling
	"--conflict-resolve": {TypeParam, "conflictResolve"},
	"--conflict-loser":   {TypeParam, "conflictLoser"},

	// Filters
	"--filters-file": {TypeParam, "filtersFile"},

	// Global Engine Config (under _config)
	"--fast-list":         {TypeConfigBool, "UseListR"},
	"--buffer-size":       {TypeConfigStr, "BufferSize"},
	"--retries":           {TypeConfigInt, "Retries"},
	"--retries-sleep":     {TypeConfigStr, "RetriesSleep"},
	"--low-level-retries": {TypeConfigInt, "LowLevelRetries"},
	"--tpslimit":          {TypeConfigFloat, "TPSLimit"},
	"--tpslimit-burst":    {TypeConfigInt, "TPSLimitBurst"},
}

// CompileBisyncPayload translates CLI arguments into /sync/bisync parameters.
func CompileBisyncPayload(
	argsList []string,
	localPath string,
	remotePath string,
	resyncAttempt bool,
	forceAttempt bool,
	dryRunAttempt bool,
) (map[string]interface{}, error) {
	bisyncParams := map[string]interface{}{
		"path1":  remotePath,
		"path2":  localPath,
		"resync": resyncAttempt,
		"force":  forceAttempt,
		"dryRun": dryRunAttempt,
	}
	engineConfig := make(map[string]interface{})
	var backendOptions []string
	var excludeRules []string

	i := 0
	for i < len(argsList) {
		arg := argsList[i]
		if arg == "--exclude" {
			i++
			if i < len(argsList) {
				pattern := strings.TrimSpace(argsList[i])
				if pattern != "" {
					excludeRules = append(excludeRules, pattern)
				}
			}
			i++
			continue
		}
		if mapping, found := CLIToRCMap[arg]; found {
			switch mapping.Type {
			case TypeIgnore:
				if i+1 < len(argsList) && !strings.HasPrefix(argsList[i+1], "-") {
					i++
				}
			case TypeParamBool:
				bisyncParams[mapping.TargetKey] = true
			case TypeConfigBool:
				engineConfig[mapping.TargetKey] = true
			case TypeBackendBool:
				backendOptions = append(backendOptions, fmt.Sprintf("%s=true", mapping.TargetKey))
			case TypeBackendStr:
				i++
				if i < len(argsList) {
					val := argsList[i]
					if strings.Contains(val, ",") || strings.Contains(val, " ") {
						backendOptions = append(backendOptions, fmt.Sprintf("%s=\"%s\"", mapping.TargetKey, val))
					} else {
						backendOptions = append(backendOptions, fmt.Sprintf("%s=%s", mapping.TargetKey, val))
					}
				}
			case TypeParam:
				i++
				if i < len(argsList) {
					bisyncParams[mapping.TargetKey] = argsList[i]
				}
			case TypeConfigStr:
				i++
				if i < len(argsList) {
					engineConfig[mapping.TargetKey] = argsList[i]
				}
			case TypeConfigInt:
				i++
				if i < len(argsList) {
					if intVal, err := strconv.Atoi(argsList[i]); err == nil {
						engineConfig[mapping.TargetKey] = intVal
					} else {
						engineConfig[mapping.TargetKey] = argsList[i]
					}
				}
			case TypeConfigFloat:
				i++
				if i < len(argsList) {
					if fVal, err := strconv.ParseFloat(argsList[i], 64); err == nil {
						engineConfig[mapping.TargetKey] = fVal
					} else {
						engineConfig[mapping.TargetKey] = argsList[i]
					}
				}
			}
		} else {
			if strings.HasPrefix(arg, "-") && i+1 < len(argsList) && !strings.HasPrefix(argsList[i+1], "-") {
				i++
			}
		}
		i++
	}

	// Attach backend connection string options to path1
	if len(backendOptions) > 0 {
		currRemote := bisyncParams["path1"].(string)
		if strings.HasSuffix(currRemote, ":") {
			baseRemote := strings.TrimSuffix(currRemote, ":")
			optsStr := strings.Join(backendOptions, ",")
			bisyncParams["path1"] = fmt.Sprintf("%s,%s:", baseRemote, optsStr)
		}
	}

	// Strict Canary Guard
	if checkAccess, ok := bisyncParams["checkAccess"].(bool); ok && checkAccess {
		testFile := filepath.Join(localPath, "RCLONE_TEST")
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("safety guard triggered: --check-access is enabled, but \x27%s\x27 was not found - aborting sync to prevent unintended deletions or sync of an unmounted filesystem", testFile)
		}
	}

	// Ensure filtersFile is an absolute path if explicitly provided
	if ff, ok := bisyncParams["filtersFile"].(string); ok && ff != "" {
		if abs, err := filepath.Abs(ff); err == nil {
			bisyncParams["filtersFile"] = abs
		}
	}

	if len(excludeRules) > 0 {
		bisyncParams["_filter"] = map[string]interface{}{
			"ExcludeRule": excludeRules,
		}
	}

	if len(engineConfig) > 0 {
		bisyncParams["_config"] = engineConfig
	}

	return bisyncParams, nil
}
