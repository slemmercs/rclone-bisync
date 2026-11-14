package models

// BrowseRequest represents payload for browsing remote or local paths.
type BrowseRequest struct {
	Remote    string `json:"remote"`
	Path      string `json:"path"`
	ProfileID string `json:"profile_id,omitempty"`
}

// BrowseItem represents a directory entry in the explorer.
type BrowseItem struct {
	Path     string `json:"Path,omitempty"`
	Name     string `json:"Name"`
	Size     int64  `json:"Size"`
	Mtime    string `json:"Mtime,omitempty"`
	ModTime  string `json:"ModTime,omitempty"`
	MimeType string `json:"MimeType,omitempty"`
	IsDir    bool   `json:"IsDir"`
}

// BrowseResponse represents directory listing results.
type BrowseResponse struct {
	Remote    string                   `json:"remote"`
	Path      string                   `json:"path"`
	Items     []map[string]interface{} `json:"items"`
	HasCanary bool                     `json:"has_canary"`
	Error     string                   `json:"error,omitempty"`
}

// CanaryResponse represents safety canary status.
type CanaryResponse struct {
	Status string `json:"status"`
	Remote string `json:"remote,omitempty"`
	Path   string `json:"path,omitempty"`
	Msg    string `json:"msg,omitempty"`
	Error  string `json:"error,omitempty"`
}

// CopyRequest represents payload for cross-target copy operations.
type CopyRequest struct {
	SrcRemote string `json:"src_remote"`
	SrcPath   string `json:"src_path"`
	DstRemote string `json:"dst_remote"`
	DstPath   string `json:"dst_path"`
	IsDir     bool   `json:"is_dir"`
	ProfileID string `json:"profile_id,omitempty"`
}

// CopyResponse represents the result of a copy operation.
type CopyResponse struct {
	Status string `json:"status"`
	Msg    string `json:"msg,omitempty"`
	Src    string `json:"src,omitempty"`
	Dst    string `json:"dst,omitempty"`
	Error  string `json:"error,omitempty"`
}

// UploadResponse represents the result of a file upload operation.
type UploadResponse struct {
	Status string `json:"status"`
	Msg    string `json:"msg,omitempty"`
	Remote string `json:"remote,omitempty"`
	Path   string `json:"path,omitempty"`
	Size   int64  `json:"size,omitempty"`
	Error  string `json:"error,omitempty"`
}

// DeleteRequest represents payload for deleting an item or directory.
type DeleteRequest struct {
	Remote    string `json:"remote"`
	Path      string `json:"path"`
	IsDir     bool   `json:"is_dir"`
	ProfileID string `json:"profile_id,omitempty"`
}

// DeleteResponse represents the result of a delete operation.
type DeleteResponse struct {
	Status string `json:"status"`
	Msg    string `json:"msg,omitempty"`
	Remote string `json:"remote,omitempty"`
	Path   string `json:"path,omitempty"`
	Fs     string `json:"fs,omitempty"`
	Error  string `json:"error,omitempty"`
}

// PendingDeletionsResponse summarizes pending unconfirmed deletions.
type PendingDeletionsResponse struct {
	HasDeletions bool          `json:"has_deletions"`
	TotalCount   int           `json:"total_count"`
	Source       string        `json:"source,omitempty"`
	TargetAction string        `json:"target_action,omitempty"`
	Message      string        `json:"message,omitempty"`
	Files        []interface{} `json:"files"`
	Error        string        `json:"error,omitempty"`
	AbortedAt    *float64      `json:"aborted_at,omitempty"`
}

// CompareItemRequest specifies a single item target to hash and compare.
type CompareItemRequest struct {
	Remote    string `json:"remote"` // "local" or configured remote name (e.g. "gdrive")
	Path      string `json:"path"`   // Relative file path
	ProfileID string `json:"profile_id,omitempty"`
}

// CompareItemResult holds the hash and size metadata for a compared file.
type CompareItemResult struct {
	Remote      string            `json:"remote"`
	Path        string            `json:"path"`
	Name        string            `json:"name"`
	Size        int64             `json:"size"`
	ModTime     string            `json:"mod_time"`
	Hashes      map[string]string `json:"hashes"` // e.g. {"md5": "...", "sha1": "..."}
	PrimaryHash string            `json:"primary_hash"`
	Error       string            `json:"error,omitempty"`
}

// FileCompareResponse represents the comparison outcome between two or more files.
type FileCompareResponse struct {
	Status    string              `json:"status"` // "success" or "error"
	Matches   bool                `json:"matches"`
	Algorithm string              `json:"algorithm"` // e.g. "sha256", "sha1", "md5", "quickxorhash", "none"
	Clusters  [][]int             `json:"clusters"`  // Indices of items grouped by matching hash
	Items     []CompareItemResult `json:"items"`
	Error     string              `json:"error,omitempty"`
}
