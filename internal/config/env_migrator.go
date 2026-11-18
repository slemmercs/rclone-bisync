package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MigrationResult details the outcome of an environment file migration.
type MigrationResult struct {
	DeployedPath    string   `json:"deployed_path"`
	TemplatePath    string   `json:"template_path"`
	BackupPath      string   `json:"backup_path,omitempty"`
	MissingKeys     []string `json:"missing_keys"`
	AddedCount      int      `json:"added_count"`
	AlreadyUpToDate bool     `json:"already_up_to_date"`
	DryRun          bool     `json:"dry_run"`
}

// TemplateEntry represents a key-value entry in the template file, including any preceding comments.
type TemplateEntry struct {
	Comments []string
	Key      string
	RawLine  string
}

// ParseTemplateEntries parses a template .env file into an ordered list of TemplateEntry items.
func ParseTemplateEntries(templatePath string) ([]TemplateEntry, error) {
	cleanPath := filepath.Clean(templatePath)
	// #nosec G304 - reading template environment file
	f, err := os.Open(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open template env file: %w", err)
	}
	defer f.Close()

	var entries []TemplateEntry
	var currentComments []string
	seenKeys := make(map[string]bool)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		rawLine := scanner.Text()
		trimmed := strings.TrimSpace(rawLine)

		if trimmed == "" {
			if len(currentComments) > 0 {
				currentComments = append(currentComments, rawLine)
			}
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			currentComments = append(currentComments, rawLine)
			continue
		}

		if strings.Contains(trimmed, "=") {
			idx := strings.Index(trimmed, "=")
			key := strings.TrimSpace(trimmed[:idx])
			key = strings.TrimPrefix(key, "export ")
			key = strings.TrimSpace(key)

			if key != "" && !seenKeys[key] {
				seenKeys[key] = true
				entries = append(entries, TemplateEntry{
					Comments: currentComments,
					Key:      key,
					RawLine:  rawLine,
				})
				currentComments = nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan template env file: %w", err)
	}

	return entries, nil
}

// DetectMissingEntries compares existing keys in deployedPath against template entries.
func DetectMissingEntries(deployedPath string, templateEntries []TemplateEntry) ([]TemplateEntry, error) {
	cleanPath := filepath.Clean(deployedPath)
	existingKeys := make(map[string]bool)

	// #nosec G304 - reading deployed environment file
	f, err := os.Open(cleanPath)
	if err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			trimmed := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(trimmed, "#") && strings.Contains(trimmed, "=") {
				idx := strings.Index(trimmed, "=")
				key := strings.TrimSpace(trimmed[:idx])
				key = strings.TrimPrefix(key, "export ")
				key = strings.TrimSpace(key)
				if key != "" {
					existingKeys[key] = true
				}
			}
		}
		_ = f.Close()
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("failed to scan deployed env file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to open deployed env file: %w", err)
	}

	var missing []TemplateEntry
	for _, entry := range templateEntries {
		if !existingKeys[entry.Key] {
			missing = append(missing, entry)
		}
	}

	return missing, nil
}

// MigrateEnvFile non-destructively upgrades deployedPath against templatePath.
// If deployedPath does not exist, it initializes it from templatePath.
// If deployedPath exists and missing keys are detected:
//   - When dryRun is true, it reports missing keys without modifying disk.
//   - When dryRun is false, it creates an atomic backup (<deployedPath>.bak.<timestamp>)
//     and appends missing canonical keys and comment blocks via an atomic rename.
func MigrateEnvFile(deployedPath, templatePath string, dryRun bool) (*MigrationResult, error) {
	cleanDeployed := filepath.Clean(deployedPath)
	cleanTemplate := filepath.Clean(templatePath)

	templateEntries, err := ParseTemplateEntries(cleanTemplate)
	if err != nil {
		return nil, err
	}

	missingEntries, err := DetectMissingEntries(cleanDeployed, templateEntries)
	if err != nil {
		return nil, err
	}

	missingKeys := make([]string, 0, len(missingEntries))
	for _, entry := range missingEntries {
		missingKeys = append(missingKeys, entry.Key)
	}

	res := &MigrationResult{
		DeployedPath: cleanDeployed,
		TemplatePath: cleanTemplate,
		MissingKeys:  missingKeys,
		AddedCount:   len(missingKeys),
		DryRun:       dryRun,
	}

	if len(missingEntries) == 0 {
		res.AlreadyUpToDate = true
		return res, nil
	}

	if dryRun {
		return res, nil
	}

	destDir := filepath.Dir(cleanDeployed)
	if err := os.MkdirAll(destDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create directory for deployed env file: %w", err)
	}

	// #nosec G304 - reading template file
	templateBytes, err := os.ReadFile(cleanTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}

	if _, err := os.Stat(cleanDeployed); os.IsNotExist(err) {
		// #nosec G306,G703 - initializing deployed config with standard permissions
		if err := os.WriteFile(cleanDeployed, templateBytes, 0600); err != nil {
			return nil, fmt.Errorf("failed to initialize deployed env file from template: %w", err)
		}
		return res, nil
	}

	// File exists and needs missing keys appended
	// #nosec G304 - reading deployed file for backup
	deployedBytes, err := os.ReadFile(cleanDeployed)
	if err != nil {
		return nil, fmt.Errorf("failed to read deployed env file for backup: %w", err)
	}

	mode := os.FileMode(0600)
	if fi, err := os.Stat(cleanDeployed); err == nil {
		mode = fi.Mode().Perm()
	}

	timestamp := time.Now().UTC().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.bak.%s", cleanDeployed, timestamp)
	// #nosec G703 - creating timestamped backup of configured env file
	if err := os.WriteFile(backupPath, deployedBytes, mode); err != nil {
		return nil, fmt.Errorf("failed to create backup file %s: %w", backupPath, err)
	}
	res.BackupPath = backupPath

	var outLines []string
	scanner := bufio.NewScanner(strings.NewReader(string(deployedBytes)))
	for scanner.Scan() {
		outLines = append(outLines, scanner.Text())
	}

	if len(outLines) > 0 && strings.TrimSpace(outLines[len(outLines)-1]) != "" {
		outLines = append(outLines, "")
	}
	outLines = append(outLines,
		"# ==============================================================================",
		fmt.Sprintf("# Migrated Configuration Defaults (Appended on %s)", time.Now().UTC().Format(time.RFC3339)),
		"# ==============================================================================",
	)

	for _, entry := range missingEntries {
		outLines = append(outLines, "")
		outLines = append(outLines, entry.Comments...)
		outLines = append(outLines, entry.RawLine)
	}
	outLines = append(outLines, "")

	tmpFile, err := os.CreateTemp(destDir, ".bisync.env.mig-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp migration file: %w", err)
	}
	tmpName := tmpFile.Name()

	newContent := strings.Join(outLines, "\n")
	if _, err := tmpFile.WriteString(newContent); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
		return nil, fmt.Errorf("failed to write temp migration file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
		return nil, fmt.Errorf("failed to sync temp migration file: %w", err)
	}
	_ = tmpFile.Close()

	if err := os.Chmod(tmpName, mode); err != nil {
		_ = os.Remove(tmpName)
		return nil, fmt.Errorf("failed to chmod temp migration file: %w", err)
	}

	if err := os.Rename(tmpName, cleanDeployed); err != nil {
		_ = os.Remove(tmpName)
		return nil, fmt.Errorf("failed to atomically replace migrated env file: %w", err)
	}

	return res, nil
}
