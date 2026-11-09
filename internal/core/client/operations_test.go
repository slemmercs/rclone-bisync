package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMoveFile_Success(t *testing.T) {
	// Arrange
	var receivedPayload map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/operations/movefile" {
			t.Errorf("expected path /operations/movefile, got %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	c := NewRcloneClient(ts.URL, "", "")

	// Act
	err := c.MoveFile(context.Background(), "gdrive:", "source/file.txt", "gdrive:", "dest/file.txt")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedPayload["srcRemote"] != "source/file.txt" {
		t.Errorf("expected srcRemote 'source/file.txt', got %v", receivedPayload["srcRemote"])
	}
	if receivedPayload["dstRemote"] != "dest/file.txt" {
		t.Errorf("expected dstRemote 'dest/file.txt', got %v", receivedPayload["dstRemote"])
	}
}

func TestMoveFile_Error(t *testing.T) {
	// Arrange
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "file not found"}`))
	}))
	defer ts.Close()

	c := NewRcloneClient(ts.URL, "", "")

	// Act
	err := c.MoveFile(context.Background(), "gdrive:", "missing.txt", "gdrive:", "dest.txt")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestMoveDir_Success(t *testing.T) {
	// Arrange
	var receivedPayload map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/operations/movedir" {
			t.Errorf("expected path /operations/movedir, got %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	c := NewRcloneClient(ts.URL, "", "")

	// Act
	err := c.MoveDir(context.Background(), "gdrive:folderA", "gdrive:folderB")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedPayload["srcFs"] != "gdrive:folderA" {
		t.Errorf("expected srcFs 'gdrive:folderA', got %v", receivedPayload["srcFs"])
	}
	if receivedPayload["dstFs"] != "gdrive:folderB" {
		t.Errorf("expected dstFs 'gdrive:folderB', got %v", receivedPayload["dstFs"])
	}
}

func TestPurge_Success(t *testing.T) {
	// Arrange
	var receivedPayload map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/operations/purge" {
			t.Errorf("expected path /operations/purge, got %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	c := NewRcloneClient(ts.URL, "", "")

	// Act
	err := c.Purge(context.Background(), "gdrive:", "folder/to/purge")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedPayload["remote"] != "folder/to/purge" {
		t.Errorf("expected remote 'folder/to/purge', got %v", receivedPayload["remote"])
	}
}

func TestPurge_Error(t *testing.T) {
	// Arrange
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "directory not found"}`))
	}))
	defer ts.Close()

	c := NewRcloneClient(ts.URL, "", "")

	// Act
	err := c.Purge(context.Background(), "gdrive:", "missing")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPublicLink_Success(t *testing.T) {
	// Arrange
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/operations/publiclink" {
			t.Errorf("expected path /operations/publiclink, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"url": "https://drive.google.com/file/d/12345/view"}`))
	}))
	defer ts.Close()

	c := NewRcloneClient(ts.URL, "", "")

	// Act
	resp, err := c.PublicLink(context.Background(), "gdrive:", "docs/report.pdf", false)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Supported {
		t.Errorf("expected supported to be true")
	}
	if resp.URL != "https://drive.google.com/file/d/12345/view" {
		t.Errorf("expected URL 'https://drive.google.com/file/d/12345/view', got %s", resp.URL)
	}
	if resp.Status != "success" {
		t.Errorf("expected status 'success', got %s", resp.Status)
	}
}

func TestPublicLink_Unsupported(t *testing.T) {
	// Arrange
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "doesn't support public links"}`))
	}))
	defer ts.Close()

	c := NewRcloneClient(ts.URL, "", "")

	// Act
	resp, err := c.PublicLink(context.Background(), "local:", "file.txt", false)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Supported {
		t.Errorf("expected supported to be false")
	}
	if resp.Status != "error" {
		t.Errorf("expected status 'error', got %s", resp.Status)
	}
	if resp.Error == "" {
		t.Errorf("expected non-empty error message")
	}
}
