package profiles

import (
	"fmt"
	"strings"
	"unicode"
)

// ProhibitedFlags defines Rclone CLI flags that cannot be overridden via CustomFlags
// to preserve security, runtime isolation, and internal telemetry integrity.
var ProhibitedFlags = map[string]string{
	"--config":   "Rclone configuration is managed exclusively by Rclone RC Framework",
	"-c":         "Rclone configuration is managed exclusively by Rclone RC Framework",
	"--use-mmap": "System-level memory overrides are prohibited",
	"--rc":       "Internal Remote Control server configuration cannot be overridden",
	"--rc-addr":  "Internal Remote Control port cannot be overridden",
	"--rc-user":  "RC authentication is managed internally",
	"--rc-pass":  "RC authentication is managed internally",
	"--log-file": "Logging is intercepted and handled by Bisync Hub Telemetry Engine",
	"--json":     "Log formatting is managed internally for structured parsing",
}

// SplitCommandLine parses a command string into discrete arguments following POSIX shell quoting rules.
// It supports single quotes, double quotes, and backslash escapes without launching any shell subprocess.
func SplitCommandLine(cmd string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	var inSingle bool
	var inDouble bool
	var escaped bool
	hasToken := false

	runes := []rune(cmd)
	for i := 0; i < len(runes); i++ {
		r := runes[i]

		if escaped {
			current.WriteRune(r)
			escaped = false
			hasToken = true
			continue
		}

		if inSingle {
			if r == '\'' {
				inSingle = false
			} else {
				current.WriteRune(r)
			}
			hasToken = true
			continue
		}

		if inDouble {
			switch r {
			case '"':
				inDouble = false
			case '\\':
				if i+1 < len(runes) && (runes[i+1] == '"' || runes[i+1] == '\\' || runes[i+1] == '$' || runes[i+1] == '`' || runes[i+1] == '\n') {
					escaped = true
				} else {
					current.WriteRune(r)
				}
			default:
				current.WriteRune(r)
			}
			hasToken = true
			continue
		}

		switch {
		case r == '\\':
			escaped = true
			hasToken = true
		case r == '\'':
			inSingle = true
			hasToken = true
		case r == '"':
			inDouble = true
			hasToken = true
		case unicode.IsSpace(r):
			if hasToken {
				tokens = append(tokens, current.String())
				current.Reset()
				hasToken = false
			}
		default:
			current.WriteRune(r)
			hasToken = true
		}
	}

	if inSingle || inDouble {
		return nil, fmt.Errorf("unclosed quote in custom flags")
	}
	if escaped {
		return nil, fmt.Errorf("unexpected trailing escape character")
	}
	if hasToken {
		tokens = append(tokens, current.String())
	}

	return tokens, nil
}

// ValidateCustomFlags parses and inspects custom CLI flags against safety policies and prohibited flags.
func ValidateCustomFlags(raw string) ([]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	tokens, err := SplitCommandLine(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid custom_flags syntax: %w", err)
	}

	for _, token := range tokens {
		flagName := strings.Split(token, "=")[0]
		if reason, blocked := ProhibitedFlags[flagName]; blocked {
			return nil, fmt.Errorf("prohibited custom flag '%s': %s", flagName, reason)
		}
	}

	return tokens, nil
}
