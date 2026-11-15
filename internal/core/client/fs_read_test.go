package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRcloneClient_StatItem(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/operations/stat" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"item": {
					"Path": "notes.txt",
					"Name": "notes.txt",
					"Size": 128,
					"MimeType": "text/plain",
					"IsDir": false
				}
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cl := NewRcloneClient(ts.URL, "", "")
	item, err := cl.StatItem(context.Background(), "myremote:", "notes.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name, ok := item["Name"].(string); !ok || name != "notes.txt" {
		t.Errorf("expected notes.txt, got %v", item["Name"])
	}
}

func TestRcloneClient_CatStreamAndFile(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/operations/getfile" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":"SGVsbG8gd29ybGQh"}`)) // base64 for "Hello world!"
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cl := NewRcloneClient(ts.URL, "", "")

	// CatStream
	stream, size, err := cl.CatStream(context.Background(), "myremote:", "notes.txt", 0, 0)
	if err != nil {
		t.Fatalf("CatStream error: %v", err)
	}
	content, err := io.ReadAll(stream)
	_ = stream.Close()
	if err != nil {
		t.Fatalf("ReadAll error: %v", err)
	}
	if string(content) != "Hello world!" {
		t.Errorf("expected 'Hello world!', got '%s'", string(content))
	}
	if size != 12 {
		t.Errorf("expected size 12, got %d", size)
	}

	// CatFile
	data, err := cl.CatFile(context.Background(), "myremote:", "notes.txt", 0, 5)
	if err != nil {
		t.Fatalf("CatFile error: %v", err)
	}
	if string(data) != "Hello" {
		t.Errorf("expected 'Hello', got '%s'", string(data))
	}
}

func TestRcloneClient_CatStreamMaxSize(t *testing.T) {
	var capturedMaxSize float64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/operations/getfile" {
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if m, ok := body["maxSize"].(float64); ok {
				capturedMaxSize = m
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":"ZGF0YQ=="}`)) // "data"
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cl := NewRcloneClient(ts.URL, "", "")

	// Explicit maxSize
	stream, _, err := cl.CatStream(context.Background(), "remote:", "large.bin", 0, 0, 10*1024*1024)
	if err != nil {
		t.Fatalf("CatStream error: %v", err)
	}
	_ = stream.Close()
	if capturedMaxSize != 10*1024*1024 {
		t.Errorf("expected maxSize %d, got %v", 10*1024*1024, capturedMaxSize)
	}

	// Inferred from count
	capturedMaxSize = 0
	stream, _, err = cl.CatStream(context.Background(), "remote:", "large.bin", 0, 5*1024*1024)
	if err != nil {
		t.Fatalf("CatStream error: %v", err)
	}
	_ = stream.Close()
	if capturedMaxSize != 5*1024*1024 {
		t.Errorf("expected maxSize %d from count, got %v", 5*1024*1024, capturedMaxSize)
	}
}

func TestRcloneClient_DownloadStream(t *testing.T) {
	directServeCalled := false
	getfileCalled := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/[myremote:]/served.txt" {
			directServeCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("direct served stream content"))
			return
		}
		if r.URL.Path == "/operations/getfile" {
			getfileCalled = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":"ZmFsbGJhY2sgY29udGVudA=="}`)) // "fallback content"
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cl := NewRcloneClient(ts.URL, "", "")

	// 1. Direct serve success
	stream, _, err := cl.DownloadStream(context.Background(), "myremote:", "served.txt", 1024)
	if err != nil {
		t.Fatalf("DownloadStream direct error: %v", err)
	}
	body, _ := io.ReadAll(stream)
	_ = stream.Close()
	if !directServeCalled || string(body) != "direct served stream content" {
		t.Errorf("expected direct serve, got %s (called=%v)", string(body), directServeCalled)
	}

	// 2. Direct serve 404 falls back to getfile
	streamFallback, _, err := cl.DownloadStream(context.Background(), "myremote:", "other.txt", 1024)
	if err != nil {
		t.Fatalf("DownloadStream fallback error: %v", err)
	}
	bodyFallback, _ := io.ReadAll(streamFallback)
	_ = streamFallback.Close()
	if !getfileCalled || string(bodyFallback) != "fallback content" {
		t.Errorf("expected getfile fallback, got %s (called=%v)", string(bodyFallback), getfileCalled)
	}
}
