package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FormatSecretValue formats a secret value for inclusion in a .env file,
// escaping internal quotes and slashes and enclosing in double quotes.
func FormatSecretValue(val string) string {
	escaped := strings.ReplaceAll(val, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}

// UpdateSecret atomically sets or updates a key-value pair in a .env file.
// The file is created with 0600 permissions if it does not exist.
func UpdateSecret(path, key, value string) error {
	cleanPath := filepath.Clean(path)
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	formattedLine := fmt.Sprintf("%s=%s", key, FormatSecretValue(value))

	var lines []string
	keyFound := false

	// #nosec G304 - reading configured secrets file
	f, err := os.Open(cleanPath)
	if err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			rawLine := scanner.Text()
			trimmed := strings.TrimSpace(rawLine)
			if !strings.HasPrefix(trimmed, "#") && strings.Contains(trimmed, "=") {
				idx := strings.Index(trimmed, "=")
				lineKey := strings.TrimSpace(trimmed[:idx])
				lineKey = strings.TrimPrefix(lineKey, "export ")
				lineKey = strings.TrimSpace(lineKey)
				if lineKey == key {
					lines = append(lines, formattedLine)
					keyFound = true
					continue
				}
			}
			lines = append(lines, rawLine)
		}
		_ = f.Close()
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to scan secrets file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to open secrets file: %w", err)
	}

	if !keyFound {
		lines = append(lines, formattedLine)
	}

	return writeSecretLinesAtomically(cleanPath, lines)
}

// RemoveSecret atomically removes a key from a .env file.
func RemoveSecret(path, key string) error {
	cleanPath := filepath.Clean(path)
	// #nosec G304 - reading configured secrets file
	f, err := os.Open(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open secrets file: %w", err)
	}

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		rawLine := scanner.Text()
		trimmed := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(trimmed, "#") && strings.Contains(trimmed, "=") {
			idx := strings.Index(trimmed, "=")
			lineKey := strings.TrimSpace(trimmed[:idx])
			lineKey = strings.TrimPrefix(lineKey, "export ")
			lineKey = strings.TrimSpace(lineKey)
			if lineKey == key {
				continue // remove key
			}
		}
		lines = append(lines, rawLine)
	}
	_ = f.Close()
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to scan secrets file: %w", err)
	}

	return writeSecretLinesAtomically(cleanPath, lines)
}

func writeSecretLinesAtomically(path string, lines []string) error {
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, ".tmp.secrets-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmpFile.Name()

	content := strings.Join(lines, "\n") + "\n"
	if _, err := tmpFile.WriteString(content); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("failed to write temp secrets file: %w", err)
	}

	if err := tmpFile.Chmod(0600); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("failed to chmod temp secrets file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("failed to sync temp secrets file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("failed to close temp secrets file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("failed to replace secrets file: %w", err)
	}

	return nil
}
