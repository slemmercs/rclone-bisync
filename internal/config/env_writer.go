package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// FormatEnvValue formats a configuration value for inclusion in a .env file.
// Booleans and numbers are written unquoted; strings with spaces or quotes are escaped and quoted.
func FormatEnvValue(val string) string {
	trimmed := strings.TrimSpace(val)
	// Check for booleans
	if trimmed == "true" || trimmed == "false" {
		return trimmed
	}
	// Check for numeric integer or float
	if _, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return trimmed
	}
	if _, err := strconv.ParseFloat(trimmed, 64); err == nil && !strings.ContainsAny(trimmed, " \t\n\r") {
		return trimmed
	}
	// If already safely double-quoted
	if len(trimmed) >= 2 && strings.HasPrefix(trimmed, `"`) && strings.HasSuffix(trimmed, `"`) {
		return trimmed
	}
	escaped := strings.ReplaceAll(trimmed, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}

// UpdateEnvKeys atomically updates multiple key-value pairs in an environment file.
// Existing comments, ordering, and untouched keys are strictly preserved.
// Any keys not present in the file are appended to the end.
func UpdateEnvKeys(path string, updates map[string]string) error {
	if len(updates) == 0 {
		return nil
	}

	cleanPath := filepath.Clean(path)
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0750); err != nil {
		return fmt.Errorf("failed to create directory for env file: %w", err)
	}

	mode := os.FileMode(0640)
	var lines []string
	applied := make(map[string]bool)

	// #nosec G304 - reading configured environment file
	f, err := os.Open(cleanPath)
	if err == nil {
		if fi, serr := f.Stat(); serr == nil {
			mode = fi.Mode().Perm()
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			rawLine := scanner.Text()
			trimmed := strings.TrimSpace(rawLine)
			if !strings.HasPrefix(trimmed, "#") && strings.Contains(trimmed, "=") {
				idx := strings.Index(trimmed, "=")
				lineKey := strings.TrimSpace(trimmed[:idx])
				lineKey = strings.TrimPrefix(lineKey, "export ")
				lineKey = strings.TrimSpace(lineKey)

				if newVal, exists := updates[lineKey]; exists {
					lines = append(lines, fmt.Sprintf("%s=%s", lineKey, FormatEnvValue(newVal)))
					applied[lineKey] = true
					continue
				}
			}
			lines = append(lines, rawLine)
		}
		_ = f.Close()
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to scan env file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to open env file: %w", err)
	}

	// Append any new keys not found in existing lines
	var unapplied []string
	for k := range updates {
		if !applied[k] {
			unapplied = append(unapplied, k)
		}
	}

	if len(unapplied) > 0 {
		// Ensure a separating newline before appended section if not empty
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			lines = append(lines, "")
		}
		for _, k := range unapplied {
			lines = append(lines, fmt.Sprintf("%s=%s", k, FormatEnvValue(updates[k])))
		}
	}

	// Atomic write via temp file and rename
	dir := filepath.Dir(cleanPath)
	tmpFile, err := os.CreateTemp(dir, ".bisync.env.tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp env file: %w", err)
	}
	tmpName := tmpFile.Name()

	content := strings.Join(lines, "\n") + "\n"
	if _, err := tmpFile.WriteString(content); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("failed to write temp env file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("failed to sync temp env file: %w", err)
	}
	_ = tmpFile.Close()

	if err := os.Chmod(tmpName, mode); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("failed to chmod temp env file: %w", err)
	}

	if err := os.Rename(tmpName, cleanPath); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("failed to atomically replace env file: %w", err)
	}

	return nil
}
