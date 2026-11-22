package models

import (
	"encoding/json"
	"strings"
)

// HasherWrapRequest defines the payload for wrapping an existing storage target in a hasher overlay remote.
type HasherWrapRequest struct {
	Name          string `json:"name"`                     // Unique name for the hasher remote (e.g., "sftp_hasher")
	Remote        string `json:"remote"`                   // Base remote or directory path to wrap (e.g., "mysftp:data")
	Hashes        string `json:"hashes,omitempty"`         // Comma-separated list of checksums (e.g., "md5,sha1", "auto")
	MaxAge        string `json:"max_age,omitempty"`        // Cache retention period (e.g., "off", "1w", "30d")
	AutoSize      int64  `json:"auto_size,omitempty"`      // Auto hash threshold size (bytes)
	CreateProfile bool   `json:"create_profile,omitempty"` // Automatically provision a linked sync profile
	LocalPath     string `json:"local_path,omitempty"`     // Local jail path if creating a linked profile
	Schedule      string `json:"schedule,omitempty"`       // Sync schedule preset ("manual", "15m", "30m", "1h", "4h", "daily")
}

// UnmarshalJSON provides flexible deserialization for HasherWrapRequest, accepting
// hashes as either a comma-separated string or a JSON array of strings, normalizing
// hash algorithm identifiers to standard rclone lowercase forms.
func (r *HasherWrapRequest) UnmarshalJSON(data []byte) error {
	type Alias HasherWrapRequest
	aux := struct {
		Hashes any `json:"hashes,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	switch v := aux.Hashes.(type) {
	case string:
		r.Hashes = normalizeHashString(v)
	case []any:
		var items []string
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				norm := normalizeSingleHash(s)
				if norm != "" {
					items = append(items, norm)
				}
			}
		}
		r.Hashes = strings.Join(items, ",")
	}
	return nil
}

func normalizeSingleHash(h string) string {
	cleaned := strings.ToLower(strings.TrimSpace(h))
	switch cleaned {
	case "sha-1", "sha_1":
		return "sha1"
	case "dropboxhash":
		return "dropbox"
	case "quickxorhash":
		return "quickxor"
	case "crc-32", "crc_32":
		return "crc32"
	default:
		return cleaned
	}
}

func normalizeHashString(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "auto") {
		return s
	}
	parts := strings.Split(s, ",")
	var normalized []string
	for _, p := range parts {
		if norm := normalizeSingleHash(p); norm != "" {
			normalized = append(normalized, norm)
		}
	}
	if len(normalized) == 0 {
		return s
	}
	return strings.Join(normalized, ",")
}

// HasherCommandRequest defines the payload for executing commands against a hasher remote via /backend/command.
type HasherCommandRequest struct {
	Remote  string            `json:"remote"`         // Target remote name (e.g., "sftp_hasher")
	Command string            `json:"command"`        // Command name: "dump", "fulldump", "drop", "import", "stickyimport"
	Args    []string          `json:"args,omitempty"` // Positional arguments (e.g., ["md5", "/path/to/sumfile"])
	Opt     map[string]string `json:"opt,omitempty"`  // Key-value options for the command
}

// HasherCommandResponse defines the result returned by a hasher backend command.
type HasherCommandResponse struct {
	Status  string      `json:"status"`           // "success" or "error"
	Command string      `json:"command"`          // Executed command
	Remote  string      `json:"remote"`           // Target remote
	Result  interface{} `json:"result,omitempty"` // Response payload from rclone backend/command
	Error   string      `json:"error,omitempty"`  // Error description if status == "error"
}
