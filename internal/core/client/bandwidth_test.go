package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetBandwidthLimit_Success(t *testing.T) {
	// Arrange
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/core/bwlimit" {
			t.Errorf("expected path /core/bwlimit, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"bytesPerSecond": 10485760, "rate": "10M"}`))
	}))
	defer ts.Close()

	c := NewRcloneClient(ts.URL, "", "")

	// Act
	res, err := c.SetBandwidthLimit(context.Background(), "10M")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.BytesPerSecond != 10485760 {
		t.Errorf("expected 10485760 B/s, got %d", res.BytesPerSecond)
	}
	if res.RateFormatted != "10M" {
		t.Errorf("expected rate 10M, got %s", res.RateFormatted)
	}
	if res.Status != "ok" {
		t.Errorf("expected status ok, got %s", res.Status)
	}
}

func TestSetBandwidthLimit_Error(t *testing.T) {
	// Arrange
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "unknown bandwidth unit"}`))
	}))
	defer ts.Close()

	c := NewRcloneClient(ts.URL, "", "")

	// Act
	_, err := c.SetBandwidthLimit(context.Background(), "invalid-unit")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetBandwidthLimit_Success(t *testing.T) {
	// Arrange
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/core/bwlimit" {
			t.Errorf("expected path /core/bwlimit, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"bytesPerSecond": 5242880, "rate": "5M"}`))
	}))
	defer ts.Close()

	c := NewRcloneClient(ts.URL, "", "")

	// Act
	res, err := c.GetBandwidthLimit(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.BytesPerSecond != 5242880 {
		t.Errorf("expected 5242880 B/s, got %d", res.BytesPerSecond)
	}
	if res.RateFormatted != "5M" {
		t.Errorf("expected rate 5M, got %s", res.RateFormatted)
	}
}

func TestGetParsedCoreStats_Success(t *testing.T) {
	// Arrange
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/core/stats" {
			t.Errorf("expected path /core/stats, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"bytes": 1048576,
			"speed": 52428.8,
			"transfers": 1,
			"checks": 10,
			"errors": 0,
			"fatalErrors": false,
			"transferring": [
				{
					"name": "photo.jpg",
					"size": 2097152,
					"bytes": 1048576,
					"percentage": 50,
					"speed": 52428.8,
					"speedAvg": 50000.0,
					"eta": 20
				}
			]
		}`))
	}))
	defer ts.Close()

	c := NewRcloneClient(ts.URL, "", "")

	// Act
	stats, err := c.GetParsedCoreStats(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Bytes != 1048576 {
		t.Errorf("expected bytes 1048576, got %d", stats.Bytes)
	}
	if stats.Speed != 52428.8 {
		t.Errorf("expected speed 52428.8, got %f", stats.Speed)
	}
	if len(stats.Transferring) != 1 {
		t.Fatalf("expected 1 transferring item, got %d", len(stats.Transferring))
	}
	item := stats.Transferring[0]
	if item.Name != "photo.jpg" || item.Percentage != 50 || item.ETA != 20 {
		t.Errorf("unexpected transfer item: %+v", item)
	}
}

func TestGetParsedCoreStats_Error(t *testing.T) {
	// Arrange
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`Bad Gateway`))
	}))
	defer ts.Close()

	c := NewRcloneClient(ts.URL, "", "")

	// Act
	_, err := c.GetParsedCoreStats(context.Background())

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
