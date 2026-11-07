package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetConfigPaths_Success(t *testing.T) {
	// Arrange
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/config/paths" {
			t.Errorf("expected path /config/paths, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"config": "/home/user/.config/rclone/rclone.conf",
			"cache": "/home/user/.cache/rclone",
			"temp": "/tmp"
		}`))
	}))
	defer ts.Close()

	c := NewRcloneClient(ts.URL, "", "")

	// Act
	res, err := c.GetConfigPaths(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Config != "/home/user/.config/rclone/rclone.conf" {
		t.Errorf("expected config path, got %s", res.Config)
	}
	if res.Cache != "/home/user/.cache/rclone" {
		t.Errorf("expected cache path, got %s", res.Cache)
	}
}

func TestGetConfigPaths_Error(t *testing.T) {
	// Arrange
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal error"}`))
	}))
	defer ts.Close()

	c := NewRcloneClient(ts.URL, "", "")

	// Act
	_, err := c.GetConfigPaths(context.Background())

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
