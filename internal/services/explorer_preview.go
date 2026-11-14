package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

const (
	// MaxTextPreviewSize defines maximum preview size for text/code (2 MB).
	MaxTextPreviewSize = 2 * 1024 * 1024
	// MaxImagePreviewSize defines maximum preview size for image data URIs (5 MB).
	MaxImagePreviewSize = 5 * 1024 * 1024
)

// PreviewItem retrieves safe preview payload (UTF-8 text or base64 image data URL) for a file.
func (e *ExplorerService) PreviewItem(ctx context.Context, remote, path string, localBase ...string) (*models.FilePreviewResponse, error) {
	cleanPath := strings.Trim(strings.TrimSpace(path), "/")
	remoteClean := strings.TrimSpace(remote)

	if strings.ToLower(remoteClean) == "local" || remoteClean == "" {
		return e.previewLocalFile(cleanPath, localBase...)
	}

	return e.previewRemoteFile(ctx, remoteClean, cleanPath)
}

func (e *ExplorerService) previewLocalFile(cleanPath string, localBase ...string) (*models.FilePreviewResponse, error) {
	rawBase := ""
	if len(localBase) > 0 && localBase[0] != "" {
		rawBase = localBase[0]
	} else if e.Config != nil {
		rawBase = e.Config.LocalPath
	}
	if rawBase == "" {
		rawBase = "~/gdrive"
	}

	_, targetAbs, safe := IsLocalPathJailed(rawBase, cleanPath)
	if !safe {
		return nil, fmt.Errorf("access denied: path traversal detected")
	}

	fi, err := os.Stat(targetAbs)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("cannot preview directory")
	}

	// #nosec G304 - targetAbs is strictly verified by IsLocalPathJailed
	f, err := os.Open(targetAbs)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	fileName := filepath.Base(targetAbs)
	size := fi.Size()

	// Sniff initial bytes for MIME detection and binary classification
	sniffBuf := make([]byte, 512)
	nSniff, _ := io.ReadFull(f, sniffBuf)
	mimeType := DetectMimeType(fileName, sniffBuf[:nSniff])

	resp := &models.FilePreviewResponse{
		Remote:   "local",
		Path:     cleanPath,
		Name:     fileName,
		Size:     size,
		MimeType: mimeType,
	}

	if mimeType == "application/pdf" {
		resp.IsBinary = true
		resp.IsPDF = true
		return resp, nil
	}

	if strings.HasPrefix(mimeType, "image/") && mimeType != "image/svg+xml" {
		if size > MaxImagePreviewSize {
			resp.Truncated = true
			resp.Error = "image exceeds maximum preview size of 5MB"
			return resp, nil
		}
		// Reset file offset to beginning for full image read
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("failed to read image file: %w", err)
		}
		imgData, err := io.ReadAll(io.LimitReader(f, MaxImagePreviewSize))
		if err != nil {
			return nil, fmt.Errorf("failed to read image: %w", err)
		}
		resp.DataURL = fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(imgData))
		return resp, nil
	}

	// For text/documents, check binary
	if isBinaryData(mimeType, sniffBuf[:nSniff]) {
		resp.IsBinary = true
		return resp, nil
	}

	// Read up to MaxTextPreviewSize
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	limitReader := io.LimitReader(f, MaxTextPreviewSize+1)
	textBuf, err := io.ReadAll(limitReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	if int64(len(textBuf)) > MaxTextPreviewSize {
		resp.Truncated = true
		resp.Content = string(textBuf[:MaxTextPreviewSize])
	} else {
		resp.Content = string(textBuf)
	}

	return resp, nil
}

func (e *ExplorerService) previewRemoteFile(ctx context.Context, remote, cleanPath string) (*models.FilePreviewResponse, error) {
	if e.Client == nil {
		return nil, fmt.Errorf("rclone client is not configured")
	}

	item, err := e.Client.StatItem(ctx, remote, cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat remote item: %w", err)
	}

	if isDir, _ := item["IsDir"].(bool); isDir {
		return nil, fmt.Errorf("cannot preview directory")
	}

	fileName, _ := item["Name"].(string)
	if fileName == "" {
		fileName = filepath.Base(cleanPath)
	}

	var size int64
	if s, ok := item["Size"].(float64); ok {
		size = int64(s)
	}

	mimeType, _ := item["MimeType"].(string)
	if mimeType == "" {
		mimeType = DetectMimeType(fileName, nil)
	}

	resp := &models.FilePreviewResponse{
		Remote:   remote,
		Path:     cleanPath,
		Name:     fileName,
		Size:     size,
		MimeType: mimeType,
	}

	if mimeType == "application/pdf" {
		resp.IsBinary = true
		resp.IsPDF = true
		return resp, nil
	}

	if strings.HasPrefix(mimeType, "image/") && mimeType != "image/svg+xml" {
		if size > MaxImagePreviewSize {
			resp.Truncated = true
			resp.Error = "image exceeds maximum preview size of 5MB"
			return resp, nil
		}
		imgData, err := e.Client.CatFile(ctx, remote, cleanPath, 0, MaxImagePreviewSize)
		if err != nil {
			return nil, fmt.Errorf("failed to read remote image: %w", err)
		}
		resp.DataURL = fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(imgData))
		return resp, nil
	}

	readCount := int64(MaxTextPreviewSize + 1)
	if size > 0 && size < readCount {
		readCount = size
	}

	buf, err := e.Client.CatFile(ctx, remote, cleanPath, 0, readCount)
	if err != nil {
		return nil, fmt.Errorf("failed to read remote file: %w", err)
	}

	sniffLen := len(buf)
	if sniffLen > 512 {
		sniffLen = 512
	}
	if isBinaryData(mimeType, buf[:sniffLen]) {
		resp.IsBinary = true
		return resp, nil
	}

	if int64(len(buf)) > MaxTextPreviewSize {
		resp.Truncated = true
		resp.Content = string(buf[:MaxTextPreviewSize])
	} else {
		resp.Content = string(buf)
	}

	return resp, nil
}

