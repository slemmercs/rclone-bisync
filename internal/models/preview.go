package models

// FilePreviewResponse encapsulates file metadata, MIME type, and safe preview payload.
type FilePreviewResponse struct {
	Remote    string `json:"remote"`
	Path      string `json:"path"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	MimeType  string `json:"mime_type"`
	IsBinary  bool   `json:"is_binary"`
	Content   string `json:"content,omitempty"`  // UTF-8 decoded text content
	DataURL   string `json:"data_url,omitempty"` // Base64 data URI (data:<mime>;base64,<encoded>) for image preview
	Truncated bool   `json:"truncated"`
	Error     string `json:"error,omitempty"`
}
