package services

import (
"context"
"net/http"
"net/http/httptest"
"testing"

"github.com/slemmercs/rclone-bisync/internal/config"
"github.com/slemmercs/rclone-bisync/internal/core/client"
)

func TestCoreService_ClientNil(t *testing.T) {
	// Arrange
	svc := NewCoreService(&config.Config{}, nil)

	// Act & Assert
	_, err := svc.GetBandwidthLimit(context.Background())
	if err == nil || err.Error() != "rclone client is not configured" {
		t.Fatalf("expected client not configured error, got: %v", err)
	}

	_, err = svc.SetBandwidthLimit(context.Background(), "10M")
	if err == nil || err.Error() != "rclone client is not configured" {
		t.Fatalf("expected client not configured error, got: %v", err)
	}

	_, err = svc.GetCoreStats(context.Background())
	if err == nil || err.Error() != "rclone client is not configured" {
		t.Fatalf("expected client not configured error, got: %v", err)
	}
}

func TestCoreService_SetBandwidthLimit_EmptyRate(t *testing.T) {
	// Arrange
	cl := client.NewRcloneClient("http://localhost:5572", "", "")
	svc := NewCoreService(&config.Config{}, cl)

	// Act
	_, err := svc.SetBandwidthLimit(context.Background(), "   ")

	// Assert
	if err == nil || err.Error() != "rate cannot be empty" {
		t.Fatalf("expected rate cannot be empty, got: %v", err)
	}
}

func TestCoreService_BandwidthOperations(t *testing.T) {
	// Arrange
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/core/bwlimit":
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"bytesPerSecond": 10485760, "rate": "10M"}`))
			}
		case "/core/stats":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"bytes": 2048,
				"speed": 1024.0,
				"transfers": 1,
				"checks": 0,
				"errors": 0,
				"fatalErrors": false,
				"transferring": []
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	cl := client.NewRcloneClient(ts.URL, "", "")
	svc := NewCoreService(&config.Config{}, cl)

	// Act: GetBandwidthLimit
	getRes, err := svc.GetBandwidthLimit(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on GetBandwidthLimit: %v", err)
	}
	if getRes.RateFormatted != "10M" {
		t.Errorf("expected 10M, got %s", getRes.RateFormatted)
	}

	// Act: SetBandwidthLimit
	setRes, err := svc.SetBandwidthLimit(context.Background(), "10M")
	if err != nil {
		t.Fatalf("unexpected error on SetBandwidthLimit: %v", err)
	}
	if setRes.BytesPerSecond != 10485760 {
		t.Errorf("expected 10485760 B/s, got %d", setRes.BytesPerSecond)
	}

	// Act: GetCoreStats
	stats, err := svc.GetCoreStats(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on GetCoreStats: %v", err)
	}
	if stats.Bytes != 2048 || stats.Speed != 1024.0 {
		t.Errorf("unexpected stats: %+v", stats)
	}
}
