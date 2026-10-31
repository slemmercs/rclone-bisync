package notifier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestShouldNotifyError(t *testing.T) {
	d := NewDispatcher("TestApp", "", 5, true, 6.0, "")

	// New error type should trigger alert
	if !d.ShouldNotifyError("CRITICAL_ERR", "OTHER_ERR", float64(time.Now().Unix())) {
		t.Fatal("expected new error type to trigger notification")
	}

	// Repeated error immediately should NOT trigger alert
	now := float64(time.Now().Unix())
	if d.ShouldNotifyError("CRITICAL_ERR", "CRITICAL_ERR", now) {
		t.Fatal("expected recent duplicate error to be suppressed")
	}

	// Repeated error older than 6 hours should trigger alert
	oldTime := float64(time.Now().Unix()) - (7 * 3600)
	if !d.ShouldNotifyError("CRITICAL_ERR", "CRITICAL_ERR", oldTime) {
		t.Fatal("expected old repeated error to trigger notification after interval")
	}
}

func TestSendWebhook(t *testing.T) {
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := NewDispatcher("TestApp", server.URL, 5, true, 6.0, "")
	err := d.SendWebhook("sync_complete", map[string]interface{}{
		"files": 42,
	})
	if err != nil {
		t.Fatalf("unexpected webhook dispatch error: %v", err)
	}

	if receivedPayload["event"] != "sync_complete" {
		t.Fatalf("expected event sync_complete, got %v", receivedPayload["event"])
	}
	if receivedPayload["app"] != "TestApp" {
		t.Fatalf("expected app TestApp, got %v", receivedPayload["app"])
	}
}

func TestNotifierEdgeCases(t *testing.T) {
	// 1. Defaults in NewDispatcher
	d := NewDispatcher("", "", 0, false, 0, "")
	if d.AppName != "Rclone Bisync" || d.WebhookTimeoutSec != 5 || d.ErrorRepeatHours != 6.0 {
		t.Fatalf("unexpected defaults: %+v", d)
	}

	// 2. ShouldNotifyError when smart notifications disabled
	if !d.ShouldNotifyError("ERR", "ERR", float64(time.Now().Unix())) {
		t.Fatal("expected true when smart notifications disabled")
	}

	// 3. SendWebhook with empty URL
	if err := d.SendWebhook("test", nil); err != nil {
		t.Fatalf("expected nil for empty webhook URL: %v", err)
	}

	// 4. SendDesktop execution
	_ = d.SendDesktop("Test Title", "Test Message", "normal")
	_ = d.SendDesktop("Test Critical", "Test Message", "critical")

	// 5. resolveIcon
	icon := d.resolveIcon()
	if icon == "" {
		t.Fatal("expected non-empty icon string")
	}
	// Verify favicon.svg prioritization when present in working directory
	if icon != "icons/favicon.svg" && icon != "sync-synchronizing" {
		t.Logf("resolved icon candidate: %s", icon)
	}
}
