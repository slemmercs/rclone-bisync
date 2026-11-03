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
	TypeParamMaxDelete
)

type OptionMapping struct {
	Type      TargetType
	TargetKey string
}

var CLIToRCMap = map[string]OptionMapping{
	// Logging & console formatting
	"--use-json-log": {TypeConfigBool, "LogJSON"},
	"--verbose":      {TypeConfigBool, "Verbose"},
	"-v":             {TypeConfigBool, "Verbose"},
	"--color":        {TypeIgnore, "color"},

	// Drive-specific options (passed as connection string options on path1)
	"--drive-export-formats":    {TypeBackendStr, "export_formats"},
	"--drive-import-formats":    {TypeBackendStr, "import_formats"},
	"--drive-skip-gdocs":        {TypeBackendBool, "skip_gdocs"},
	"--drive-use-trash":         {TypeBackendBool, "use_trash"},
	"--drive-shared-with-me":    {TypeBackendBool, "shared_with_me"},
	"--drive-acknowledge-abuse": {TypeBackendBool, "acknowledge_abuse"},
	"--drive-skip-shortcuts":    {TypeBackendBool, "skip_shortcuts"},

	// OneDrive-specific options
	"--onedrive-no-versions": {TypeBackendBool, "no_versions"},
	"--onedrive-chunk-size":  {TypeBackendStr, "chunk_size"},

	// S3-specific options
	"--s3-chunk-size":    {TypeBackendStr, "chunk_size"},
	"--s3-storage-class": {TypeBackendStr, "storage_class"},

	// Dropbox-specific options
	"--dropbox-chunk-size": {TypeBackendStr, "chunk_size"},

	// Bisync parameters
	"--check-access":      {TypeParamBool, "checkAccess"},
	"--check-first":       {TypeParamBool, "checkFirst"},
	"--max-delete":        {TypeParamMaxDelete, "maxDelete"},
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
	"--track-renames":          {TypeConfigBool, "TrackRenames"},
	"--track-renames-strategy": {TypeConfigStr, "TrackRenamesStrategy"},
	"--fast-list":              {TypeConfigBool, "UseListR"},
	"--buffer-size":            {TypeConfigStr, "BufferSize"},
	"--retries":                {TypeConfigInt, "Retries"},
	"--retries-sleep":          {TypeConfigStr, "RetriesSleep"},
	"--low-level-retries":      {TypeConfigInt, "LowLevelRetries"},
	"--tpslimit":               {TypeConfigFloat, "TPSLimit"},
	"--tpslimit-burst":         {TypeConfigInt, "TPSLimitBurst"},

	// Execution Controls & Safety
	"--dry-run":               {TypeParamBool, "dryRun"},
	"--resync":                {TypeParamBool, "resync"},
	"--force":                 {TypeParamBool, "force"},
	"--check-filename":        {TypeIgnore, "checkFilename"},
	"--create-empty-src-dirs": {TypeParamBool, "createEmptySrcDirs"},
}

// FlagCompilationResult captures the parsed parameters, engine config, filter rules,
// backend connection options, unrecognized flags, and errors from CLI argument validation.
type FlagCompilationResult struct {
	Valid             bool                   `json:"valid"`
	Params            map[string]interface{} `json:"params"`
	Config            map[string]interface{} `json:"config,omitempty"`
	Filter            map[string]interface{} `json:"filter,omitempty"`
	BackendOptions    []string               `json:"backend_options,omitempty"`
	UnrecognizedFlags []string               `json:"unrecognized_flags,omitempty"`
	Errors            []string               `json:"errors,omitempty"`
}

// CompileFlagsToParams parses and compiles CLI flags into bisync parameters, engine config,
// and backend options without requiring local filesystem presence or canary validation.
func CompileFlagsToParams(argsList []string) (FlagCompilationResult, error) {
	bisyncParams := make(map[string]interface{})
	engineConfig := make(map[string]interface{})
	var backendOptions []string
	var excludeRules []string
	var unrecognized []string
	var errMsgs []string

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
		flagName := arg
		flagVal := ""
		hasEqual := false
		if cutFlag, cutVal, ok := strings.Cut(arg, "="); ok {
			flagName = cutFlag
			flagVal = cutVal
			hasEqual = true
		}

		if mapping, found := CLIToRCMap[flagName]; found {
			switch mapping.Type {
			case TypeIgnore:
				if !hasEqual && i+1 < len(argsList) && !strings.HasPrefix(argsList[i+1], "-") {
					i++
				}
			case TypeParamBool:
				if hasEqual {
					bisyncParams[mapping.TargetKey] = (flagVal == "true" || flagVal == "1")
				} else {
					bisyncParams[mapping.TargetKey] = true
				}
			case TypeConfigBool:
				valBool := true
				if hasEqual {
					valBool = (flagVal == "true" || flagVal == "1")
				}
				if mapping.TargetKey == "Verbose" {
					if valBool {
						engineConfig["Verbose"] = 1
					} else {
						engineConfig["Verbose"] = 0
					}
				} else {
					engineConfig[mapping.TargetKey] = valBool
				}
			case TypeBackendBool:
				valStr := "true"
				if hasEqual {
					valStr = strings.ToLower(strings.TrimSpace(flagVal))
				}
				backendOptions = append(backendOptions, fmt.Sprintf("%s=%s", mapping.TargetKey, valStr))
			case TypeBackendStr:
				val := flagVal
				if !hasEqual {
					i++
					if i < len(argsList) {
						val = argsList[i]
					}
				}
				if strings.Contains(val, ",") || strings.Contains(val, " ") {
					backendOptions = append(backendOptions, fmt.Sprintf("%s=\"%s\"", mapping.TargetKey, val))
				} else {
					backendOptions = append(backendOptions, fmt.Sprintf("%s=%s", mapping.TargetKey, val))
				}
			case TypeParam:
				val := flagVal
				if !hasEqual {
					i++
					if i < len(argsList) {
						val = argsList[i]
					}
				}
				bisyncParams[mapping.TargetKey] = val
			case TypeParamMaxDelete:
				valStr := flagVal
				if !hasEqual {
					i++
					if i < len(argsList) {
						valStr = argsList[i]
					}
				}
				valStr = strings.TrimSuffix(strings.TrimSpace(valStr), "%")
				if intVal, err := strconv.Atoi(valStr); err == nil {
					bisyncParams[mapping.TargetKey] = intVal
					engineConfig["MaxDelete"] = intVal
				} else {
					errMsgs = append(errMsgs, fmt.Sprintf("invalid integer or percentage for %s: %q", flagName, valStr))
				}
			case TypeConfigStr:
				val := flagVal
				if !hasEqual {
					i++
					if i < len(argsList) {
						val = argsList[i]
					}
				}
				engineConfig[mapping.TargetKey] = val
			case TypeConfigInt:
				valStr := flagVal
				if !hasEqual {
					i++
					if i < len(argsList) {
						valStr = argsList[i]
					}
				}
				if intVal, err := strconv.Atoi(valStr); err == nil {
					engineConfig[mapping.TargetKey] = intVal
				} else {
					errMsgs = append(errMsgs, fmt.Sprintf("invalid integer for %s: %q", flagName, valStr))
				}
			case TypeConfigFloat:
				valStr := flagVal
				if !hasEqual {
					i++
					if i < len(argsList) {
						valStr = argsList[i]
					}
				}
				if fVal, err := strconv.ParseFloat(valStr, 64); err == nil {
					engineConfig[mapping.TargetKey] = fVal
				} else {
					errMsgs = append(errMsgs, fmt.Sprintf("invalid float for %s: %q", flagName, valStr))
				}
			}
		} else {
			if strings.HasPrefix(arg, "-") {
				unrecognized = append(unrecognized, flagName)
			}
			if !hasEqual && strings.HasPrefix(arg, "-") && i+1 < len(argsList) && !strings.HasPrefix(argsList[i+1], "-") {
				i++
			}
		}
		i++
	}

	var filterMap map[string]interface{}
	if len(excludeRules) > 0 {
		filterMap = map[string]interface{}{
			"ExcludeRule": excludeRules,
		}
	}

	result := FlagCompilationResult{
		Valid:             len(errMsgs) == 0,
		Params:            bisyncParams,
		Config:            engineConfig,
		Filter:            filterMap,
		BackendOptions:    backendOptions,
		UnrecognizedFlags: unrecognized,
		Errors:            errMsgs,
	}

	var err error
	if len(errMsgs) > 0 {
		err = fmt.Errorf("flag validation failed: %s", strings.Join(errMsgs, "; "))
	}
	return result, err
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
	compiled, err := CompileFlagsToParams(argsList)
	if err != nil {
		return nil, err
	}

	bisyncParams := compiled.Params
	if bisyncParams == nil {
		bisyncParams = make(map[string]interface{})
	}
	bisyncParams["path1"] = remotePath
	bisyncParams["path2"] = localPath
	if resyncAttempt {
		bisyncParams["resync"] = true
	} else if _, ok := bisyncParams["resync"]; !ok {
		bisyncParams["resync"] = false
	}
	if forceAttempt {
		bisyncParams["force"] = true
	} else if _, ok := bisyncParams["force"]; !ok {
		bisyncParams["force"] = false
	}
	if dryRunAttempt {
		bisyncParams["dryRun"] = true
	} else if _, ok := bisyncParams["dryRun"]; !ok {
		bisyncParams["dryRun"] = false
	}

	// Attach backend connection string options to path1
	if len(compiled.BackendOptions) > 0 {
		currRemote := bisyncParams["path1"].(string)
		if colonIdx := strings.Index(currRemote, ":"); colonIdx != -1 {
			baseRemote := currRemote[:colonIdx]
			subPath := currRemote[colonIdx+1:]
			optsStr := strings.Join(compiled.BackendOptions, ",")
			bisyncParams["path1"] = fmt.Sprintf("%s,%s:%s", baseRemote, optsStr, subPath)
		}
	}

	// Strict Canary Guard
	if checkAccess, ok := bisyncParams["checkAccess"].(bool); ok && checkAccess {
		testFile := filepath.Join(localPath, "RCLONE_TEST")
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("safety guard triggered: --check-access is enabled, but '%s' was not found - aborting sync to prevent unintended deletions or sync of an unmounted filesystem", testFile)
		}
	}

	// Ensure filtersFile is an absolute path if explicitly provided
	if ff, ok := bisyncParams["filtersFile"].(string); ok && ff != "" {
		if abs, err := filepath.Abs(ff); err == nil {
			bisyncParams["filtersFile"] = abs
		}
	}

	if len(compiled.Filter) > 0 {
		bisyncParams["_filter"] = compiled.Filter
	}

	if len(compiled.Config) > 0 {
		bisyncParams["_config"] = compiled.Config
	}

	return bisyncParams, nil
}
