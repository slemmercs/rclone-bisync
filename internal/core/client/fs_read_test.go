package client

import (
	"context"
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
