package services

import (
	"bytes"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// DetectMimeType determines the appropriate MIME type from file extension or initial bytes.
func DetectMimeType(fileName string, sample []byte) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".md", ".markdown":
		return "text/markdown; charset=utf-8"
	case ".json":
		return "application/json"
	case ".yaml", ".yml":
		return "text/yaml; charset=utf-8"
	case ".log", ".txt", ".conf", ".ini", ".env":
		return "text/plain; charset=utf-8"
	case ".sh", ".bash":
		return "text/x-shellscript; charset=utf-8"
	case ".go":
		return "text/x-go; charset=utf-8"
	case ".js":
		return "application/javascript"
	case ".ts":
		return "application/typescript"
	case ".css":
		return "text/css; charset=utf-8"
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".csv":
		return "text/csv; charset=utf-8"
	case ".xml":
		return "text/xml; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".zip":
		return "application/zip"
	case ".tar", ".gz", ".tgz", ".bz2", ".xz", ".7z", ".rar":
		return "application/octet-stream"
	case ".py":
		return "text/x-python; charset=utf-8"
	case ".rs":
		return "text/x-rust; charset=utf-8"
	case ".sql":
		return "text/x-sql; charset=utf-8"
	}

	if detected := mime.TypeByExtension(ext); detected != "" {
		return detected
	}

	if len(sample) > 0 {
		return http.DetectContentType(sample)
	}

	return "application/octet-stream"
}

func isBinaryData(mimeType string, data []byte) bool {
	// Office documents, archives, PDFs, executables, and octet streams are strictly binary
	if strings.Contains(mimeType, "openxmlformats") ||
		strings.Contains(mimeType, "msword") ||
		strings.Contains(mimeType, "ms-excel") ||
		strings.Contains(mimeType, "ms-powerpoint") ||
		strings.Contains(mimeType, "zip") ||
		strings.Contains(mimeType, "pdf") ||
		strings.Contains(mimeType, "octet-stream") ||
		strings.Contains(mimeType, "tar") ||
		strings.Contains(mimeType, "compressed") ||
		strings.Contains(mimeType, "x-executable") {
		return true
	}

	if strings.HasPrefix(mimeType, "text/") ||
		strings.Contains(mimeType, "json") ||
		strings.Contains(mimeType, "xml") ||
		strings.Contains(mimeType, "javascript") ||
		strings.Contains(mimeType, "yaml") {
		return false
	}
	if bytes.IndexByte(data, 0) != -1 {
		return true
	}
	return !utf8.Valid(data)
}
