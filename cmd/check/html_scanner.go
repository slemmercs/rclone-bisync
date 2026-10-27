package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// BooleanAttributeViolation records an unsafe Alpine boolean binding found in an HTML file.
type BooleanAttributeViolation struct {
	File      string
	Line      int
	Attribute string
	Value     string
}

// targetBooleanAttributes lists attributes prone to Alpine v3 boolean coercion bugs.
var targetBooleanAttributes = map[string]bool{
	"disabled": true,
	"checked":  true,
	"readonly": true,
	"hidden":   true,
	"required": true,
	"selected": true,
}

// isTargetBooleanAttr returns true if attr is e.g. ":disabled" or "x-bind:disabled".
func isTargetBooleanAttr(attr string) (string, bool) {
	name := strings.ToLower(attr)
	if strings.HasPrefix(name, ":") {
		base := strings.TrimPrefix(name, ":")
		if targetBooleanAttributes[base] {
			return base, true
		}
	}
	if strings.HasPrefix(name, "x-bind:") {
		base := strings.TrimPrefix(name, "x-bind:")
		if targetBooleanAttributes[base] {
			return base, true
		}
	}
	return "", false
}

// ScanHTMLForAlpineViolations tokenizes HTML content and flags unsafe dotted boolean expressions.
// It ignores HTML comments and correctly handles multi-line tags and attributes.
func ScanHTMLForAlpineViolations(filePath string, content []byte) []BooleanAttributeViolation {
	var violations []BooleanAttributeViolation
	n := len(content)
	i := 0
	line := 1

	for i < n {
		b := content[i]
		if b == '\n' {
			line++
			i++
			continue
		}

		// Check for HTML comment: <!-- ... -->
		if b == '<' && i+3 < n && content[i+1] == '!' && content[i+2] == '-' && content[i+3] == '-' {
			i += 4
			for i+2 < n && !(content[i] == '-' && content[i+1] == '-' && content[i+2] == '>') {
				if content[i] == '\n' {
					line++
				}
				i++
			}
			if i+2 < n {
				i += 3 // Skip "-->"
			} else {
				i = n
			}
			continue
		}

		// Check for Tag open: <tag ... > (excluding closing tags </tag>)
		if b == '<' {
			i++
			if i < n && (content[i] == '/' || content[i] == '!' || content[i] == '?') {
				// Skip closing tags, doctypes, XML processing instructions
				for i < n && content[i] != '>' {
					if content[i] == '\n' {
						line++
					}
					i++
				}
				if i < n && content[i] == '>' {
					i++
				}
				continue
			}

			// Parse tag name
			for i < n && !unicode.IsSpace(rune(content[i])) && content[i] != '>' && content[i] != '/' {
				if content[i] == '\n' {
					line++
				}
				i++
			}

			// Parse attributes inside tag until '>'
			for i < n && content[i] != '>' {
				// Skip whitespace
				for i < n && unicode.IsSpace(rune(content[i])) {
					if content[i] == '\n' {
						line++
					}
					i++
				}
				if i >= n || content[i] == '>' {
					break
				}
				if content[i] == '/' {
					i++
					continue
				}

				attrLine := line
				attrNameStart := i
				for i < n && !unicode.IsSpace(rune(content[i])) && content[i] != '=' && content[i] != '>' && content[i] != '/' {
					if content[i] == '\n' {
						line++
					}
					i++
				}
				attrName := string(content[attrNameStart:i])

				// Skip whitespace before '='
				for i < n && unicode.IsSpace(rune(content[i])) {
					if content[i] == '\n' {
						line++
					}
					i++
				}

				attrVal := ""
				if i < n && content[i] == '=' {
					i++ // Skip '='
					// Skip whitespace after '='
					for i < n && unicode.IsSpace(rune(content[i])) {
						if content[i] == '\n' {
							line++
						}
						i++
					}

					if i < n {
						quote := content[i]
						if quote == '"' || quote == '\'' {
							i++ // Skip opening quote
							valStart := i
							for i < n && content[i] != quote {
								if content[i] == '\n' {
									line++
								}
								i++
							}
							attrVal = string(content[valStart:i])
							if i < n && content[i] == quote {
								i++ // Skip closing quote
							}
						} else {
							// Unquoted attribute value
							valStart := i
							for i < n && !unicode.IsSpace(rune(content[i])) && content[i] != '>' {
								i++
							}
							attrVal = string(content[valStart:i])
						}
					}
				}

				if _, isTarget := isTargetBooleanAttr(attrName); isTarget {
					trimmedVal := strings.TrimSpace(attrVal)
					if strings.Contains(trimmedVal, ".") {
						violations = append(violations, BooleanAttributeViolation{
							File:      filePath,
							Line:      attrLine,
							Attribute: attrName,
							Value:     trimmedVal,
						})
					}
				}
			}

			if i < n && content[i] == '>' {
				i++
			}
			continue
		}

		i++
	}

	return violations
}

// checkAlpineTemplates scans all HTML templates under root for unsafe dotted boolean bindings.
func checkAlpineTemplates(root string, verbose bool) error {
	var allViolations []BooleanAttributeViolation
	var htmlCount int

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || shouldSkip(path) || !strings.HasSuffix(path, ".html") {
			return nil
		}
		htmlCount++
		// #nosec G122,G304 - check tool reading html templates
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		violations := ScanHTMLForAlpineViolations(rel, content)
		allViolations = append(allViolations, violations...)
		return nil
	})

	if err != nil {
		return err
	}

	if verbose {
		fmt.Printf("    Audited %d HTML templates\n", htmlCount)
	}

	if len(allViolations) > 0 {
		var msgs []string
		for _, v := range allViolations {
			msgs = append(msgs, fmt.Sprintf("%s:%d -> %s=\"%s\"", v.File, v.Line, v.Attribute, v.Value))
		}
		return fmt.Errorf("found unsafe Alpine dotted boolean attribute bindings:\n  %s", strings.Join(msgs, "\n  "))
	}
	return nil
}
