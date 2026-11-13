package services

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
)

func TestExplorerPreview_LocalFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Text file
	textFile := filepath.Join(tmpDir, "readme.md")
	if err := os.WriteFile(textFile, []byte("# Hello Bisync\nTesting preview."), 0600); err != nil {
		t.Fatal(err)
	}

	// Binary file
	binFile := filepath.Join(tmpDir, "data.bin")
	if err := os.WriteFile(binFile, []byte{0x00, 0x01, 0x02, 0x03, 0xFF}, 0600); err != nil {
		t.Fatal(err)
	}

	// Tiny PNG (1x1 pixel)
	pngBytes, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==")
	imgFile := filepath.Join(tmpDir, "test.png")
	if err := os.WriteFile(imgFile, pngBytes, 0600); err != nil {
		t.Fatal(err)
	}

	// Subdir
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0750); err != nil {
		t.Fatal(err)
	}

	svc := NewExplorerService(&config.Config{LocalPath: tmpDir}, nil, nil)

	// 1. Markdown preview
	prev, err := svc.PreviewItem(context.Background(), "local", "readme.md")
	if err != nil {
		t.Fatalf("failed to preview markdown: %v", err)
	}
	if prev.IsBinary || !strings.Contains(prev.Content, "# Hello Bisync") {
		t.Errorf("unexpected preview content: %+v", prev)
	}
	if prev.MimeType != "text/markdown; charset=utf-8" {
		t.Errorf("unexpected mime: %s", prev.MimeType)
	}

	// 2. Binary preview
	prevBin, err := svc.PreviewItem(context.Background(), "local", "data.bin")
	if err != nil {
		t.Fatalf("failed to preview binary: %v", err)
	}
	if !prevBin.IsBinary {
		t.Errorf("expected IsBinary true, got %+v", prevBin)
	}
	if prevBin.Content != "" {
		t.Errorf("expected empty content for binary file")
	}

	// 3. Image preview
	prevImg, err := svc.PreviewItem(context.Background(), "local", "test.png")
	if err != nil {
		t.Fatalf("failed to preview image: %v", err)
	}
	if !strings.HasPrefix(prevImg.DataURL, "data:image/png;base64,") {
		t.Errorf("unexpected DataURL: %s", prevImg.DataURL)
	}

	// 4. Directory rejection
	_, err = svc.PreviewItem(context.Background(), "local", "subdir")
	if err == nil || !strings.Contains(err.Error(), "cannot preview directory") {
		t.Errorf("expected cannot preview directory error, got: %v", err)
	}

	// 5. Jail traversal rejection
	_, err = svc.PreviewItem(context.Background(), "local", "../etc/passwd")
	if err == nil || !strings.Contains(err.Error(), "path traversal detected") {
		t.Errorf("expected traversal rejection, got: %v", err)
	}

	// 6. DownloadStream local
	stream, name, size, err := svc.DownloadStream(context.Background(), "local", "readme.md")
	if err != nil {
		t.Fatalf("download stream error: %v", err)
	}
	defer stream.Close()
	content, _ := io.ReadAll(stream)
	if name != "readme.md" || size != int64(len(content)) || !strings.Contains(string(content), "# Hello Bisync") {
		t.Errorf("unexpected stream content: %s, name: %s, size: %d", string(content), name, size)
	}

	// 7. PDF preview
	pdfFile := filepath.Join(tmpDir, "document.pdf")
	if err := os.WriteFile(pdfFile, []byte("%PDF-1.4 dummy binary pdf content"), 0600); err != nil {
		t.Fatal(err)
	}
	prevPDF, err := svc.PreviewItem(context.Background(), "local", "document.pdf")
	if err != nil {
		t.Fatalf("failed to preview pdf: %v", err)
	}
	if !prevPDF.IsPDF || !prevPDF.IsBinary || prevPDF.Content != "" {
		t.Errorf("expected IsPDF=true, IsBinary=true, empty Content, got: %+v", prevPDF)
	}

	// 8. DOCX / OpenXML preview
	docxFile := filepath.Join(tmpDir, "report.docx")
	if err := os.WriteFile(docxFile, []byte("PK\x03\x04[Content_Types].xml dummy zip content"), 0600); err != nil {
		t.Fatal(err)
	}
	prevDocx, err := svc.PreviewItem(context.Background(), "local", "report.docx")
	if err != nil {
		t.Fatalf("failed to preview docx: %v", err)
	}
	if !prevDocx.IsBinary || prevDocx.Content != "" {
		t.Errorf("expected IsBinary=true and empty Content for docx, got: %+v", prevDocx)
	}

}

func TestExplorerPreview_RemoteFiles(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/operations/stat":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"item": {
					"Path": "remote.txt",
					"Name": "remote.txt",
					"Size": 18,
					"MimeType": "text/plain",
					"IsDir": false
				}
			}`))
		case "/operations/getfile":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":"UmVtb3RlIGZpbGUgYm9keSEh"}`)) // base64 for "Remote file body!!"
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	cl := client.NewRcloneClient(ts.URL, "", "")
	svc := NewExplorerService(&config.Config{}, cl, nil)

	// Remote preview
	prev, err := svc.PreviewItem(context.Background(), "myremote:", "remote.txt")
	if err != nil {
		t.Fatalf("remote preview error: %v", err)
	}
	if prev.Content != "Remote file body!!" || prev.Size != 18 {
		t.Errorf("unexpected remote preview: %+v", prev)
	}

	// Remote download stream
	stream, name, size, err := svc.DownloadStream(context.Background(), "myremote:", "remote.txt")
	if err != nil {
		t.Fatalf("remote download stream error: %v", err)
	}
	defer stream.Close()
	body, _ := io.ReadAll(stream)
	if string(body) != "Remote file body!!" || name != "remote.txt" || size != 18 {
		t.Errorf("unexpected download output: %s, name: %s, size: %d", string(body), name, size)
	}
}

func TestExplorerPreview_RemoteClientNil(t *testing.T) {
	svc := NewExplorerService(&config.Config{}, nil, nil)

	_, err := svc.PreviewItem(context.Background(), "myremote:", "remote.txt")
	if err == nil || !strings.Contains(err.Error(), "rclone client is not configured") {
		t.Errorf("expected client not configured error, got: %v", err)
	}

	_, _, _, err = svc.DownloadStream(context.Background(), "myremote:", "remote.txt")
	if err == nil || !strings.Contains(err.Error(), "rclone client is not configured") {
		t.Errorf("expected client not configured error, got: %v", err)
	}
}

func TestDetectMimeType(t *testing.T) {
	cases := []struct {
		filename string
		sample   []byte
		expected string
	}{
		{"test.pdf", nil, "application/pdf"},
		{"test.docx", nil, "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"test.xlsx", nil, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"test.pptx", nil, "application/vnd.openxmlformats-officedocument.presentationml.presentation"},
		{"test.zip", nil, "application/zip"},
		{"test.tar", nil, "application/octet-stream"},
		{"test.md", nil, "text/markdown; charset=utf-8"},
		{"test.json", nil, "application/json"},
		{"test.yaml", nil, "text/yaml; charset=utf-8"},
		{"test.txt", nil, "text/plain; charset=utf-8"},
		{"test.sh", nil, "text/x-shellscript; charset=utf-8"},
		{"test.go", nil, "text/x-go; charset=utf-8"},
		{"test.js", nil, "application/javascript"},
		{"test.ts", nil, "application/typescript"},
		{"test.css", nil, "text/css; charset=utf-8"},
		{"test.html", nil, "text/html; charset=utf-8"},
		{"test.csv", nil, "text/csv; charset=utf-8"},
		{"test.xml", nil, "text/xml; charset=utf-8"},
		{"test.svg", nil, "image/svg+xml"},
		{"test.py", nil, "text/x-python; charset=utf-8"},
		{"test.rs", nil, "text/x-rust; charset=utf-8"},
		{"test.sql", nil, "text/x-sql; charset=utf-8"},
		{"unknown.custom", []byte("<!DOCTYPE html><html></html>"), "text/html; charset=utf-8"},
		{"unknown.custom", nil, "application/octet-stream"},
	}

	for _, tc := range cases {
		got := DetectMimeType(tc.filename, tc.sample)
		if got != tc.expected {
			t.Errorf("DetectMimeType(%q, sample=%v) = %q, expected %q", tc.filename, tc.sample != nil, got, tc.expected)
		}
	}
}

func TestIsBinaryData(t *testing.T) {
	tests := []struct {
		mime     string
		data     []byte
		expected bool
	}{
		{"application/pdf", nil, true},
		{"application/zip", nil, true},
		{"application/octet-stream", nil, true},
		{"text/plain", []byte("hello"), false},
		{"application/json", []byte("{}"), false},
		{"application/xml", []byte("<a/>"), false},
		{"application/javascript", []byte("console.log(1)"), false},
		{"application/unknown", []byte("text data"), false},
		{"application/unknown", []byte{0x00, 0x01}, true},
		{"application/unknown", []byte{0xff, 0xfe, 0xfd}, true},
	}

	for _, tt := range tests {
		got := isBinaryData(tt.mime, tt.data)
		if got != tt.expected {
			t.Errorf("isBinaryData(%q, %v) = %v; want %v", tt.mime, tt.data, got, tt.expected)
		}
	}
}
