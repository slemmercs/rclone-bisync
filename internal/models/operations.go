package models

// MoveRequest specifies parameters for moving a file or directory.
type MoveRequest struct {
SrcRemote    string `json:"src_remote"`
SrcPath      string `json:"src_path"`
DstRemote    string `json:"dst_remote"`
DstPath      string `json:"dst_path"`
IsDir        bool   `json:"is_dir"`
SrcProfileID string `json:"src_profile_id,omitempty"`
DstProfileID string `json:"dst_profile_id,omitempty"`
}

// MoveResponse describes the outcome of a move operation.
type MoveResponse struct {
Status string `json:"status"` // "success" or "error"
Src    string `json:"src"`
Dst    string `json:"dst"`
Error  string `json:"error,omitempty"`
}

// PurgeRequest specifies parameters for purging a directory.
type PurgeRequest struct {
Remote    string `json:"remote"`
Path      string `json:"path"`
ProfileID string `json:"profile_id,omitempty"`
}

// PurgeResponse describes the outcome of a directory purge.
type PurgeResponse struct {
Status string `json:"status"` // "success" or "error"
Path   string `json:"path"`
Error  string `json:"error,omitempty"`
}

// PublicLinkRequest specifies parameters for resolving a public cloud URL.
type PublicLinkRequest struct {
Remote    string `json:"remote"`
Path      string `json:"path"`
ProfileID string `json:"profile_id,omitempty"`
Unlink    bool   `json:"unlink,omitempty"`
}

// PublicLinkResponse contains the public cloud link.
type PublicLinkResponse struct {
URL    string `json:"url,omitempty"`
Status string `json:"status"`
Error  string `json:"error,omitempty"`
}
