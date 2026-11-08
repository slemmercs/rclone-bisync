package models

// MoveRequest specifies parameters for moving a file or directory.
type MoveRequest struct {
	ProfileID    string `json:"profile_id,omitempty"`
	SrcProfileID string `json:"src_profile_id,omitempty"`
	DstProfileID string `json:"dst_profile_id,omitempty"`
	Remote       string `json:"remote,omitempty"`
	SrcRemote    string `json:"src_remote"`
	SrcPath      string `json:"src_path"`
	DstRemote    string `json:"dst_remote"`
	DstPath      string `json:"dst_path"`
	IsDir        bool   `json:"is_dir"`
}

// MoveResponse describes the outcome of a move operation.
type MoveResponse struct {
	Status string `json:"status"` // "success" or "error"
	Msg    string `json:"msg,omitempty"`
	Src    string `json:"src,omitempty"`
	Dst    string `json:"dst,omitempty"`
	Error  string `json:"error,omitempty"`
}

// PurgeRequest specifies parameters for purging a directory.
type PurgeRequest struct {
	ProfileID string `json:"profile_id,omitempty"`
	Remote    string `json:"remote"`
	Path      string `json:"path"`
}

// PurgeResponse describes the outcome of a directory purge.
type PurgeResponse struct {
	Status string `json:"status"` // "success" or "error"
	Msg    string `json:"msg,omitempty"`
	Path   string `json:"path,omitempty"`
	Error  string `json:"error,omitempty"`
}

// PublicLinkRequest specifies parameters for resolving a public cloud URL.
type PublicLinkRequest struct {
	ProfileID string `json:"profile_id,omitempty"`
	Remote    string `json:"remote"`
	Path      string `json:"path"`
	Unlink    bool   `json:"unlink,omitempty"`
}

// PublicLinkResponse contains the public cloud link with support detection.
type PublicLinkResponse struct {
	Status    string `json:"status"` // "success" or "error"
	Msg       string `json:"msg,omitempty"`
	URL       string `json:"url,omitempty"`
	Supported bool   `json:"supported"`
	Error     string `json:"error,omitempty"`
}
