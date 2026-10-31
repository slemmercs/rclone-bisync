package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Dispatcher handles desktop notifications and optional webhooks.
type Dispatcher struct {
	AppName                   string
	WebhookURL                string
	WebhookTimeoutSec         int
	EnabledSmartNotifications bool
	ErrorRepeatHours          float64
	IconPath                  string
	httpClient                *http.Client
}

// NewDispatcher creates a new notification Dispatcher.
func NewDispatcher(appName, webhookURL string, webhookTimeoutSec int, smartNotifications bool, errorRepeatHours float64, iconPath string) *Dispatcher {
	if appName == "" {
		appName = "Rclone Bisync"
	}
	if webhookTimeoutSec <= 0 {
		webhookTimeoutSec = 5
	}
	if errorRepeatHours <= 0 {
		errorRepeatHours = 6.0
	}

	d := &Dispatcher{
		AppName:                   appName,
		WebhookURL:                webhookURL,
		WebhookTimeoutSec:         webhookTimeoutSec,
		EnabledSmartNotifications: smartNotifications,
		ErrorRepeatHours:          errorRepeatHours,
		IconPath:                  iconPath,
		httpClient: &http.Client{
			Timeout: time.Duration(webhookTimeoutSec) * time.Second,
		},
	}

	if d.IconPath == "" {
		d.IconPath = d.resolveIcon()
	}

	return d
}

func (d *Dispatcher) resolveIcon() string {
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, "rclone-bisync", "icons", "favicon.svg"),
		filepath.Join(home, "rclone-bisync", "dashboard", "frontend", "favicon.svg"),
		"icons/favicon.svg",
		"dashboard/frontend/favicon.svg",
		"favicon.svg",
		filepath.Join(home, "rclone-bisync", "icons", "cloud-sync-light.svg"),
		filepath.Join(home, "rclone-bisync", "dashboard", "frontend", "cloud-sync-light.svg"),
		"icons/cloud-sync-light.svg",
		"dashboard/frontend/cloud-sync-light.svg",
		"cloud-sync-light.svg",
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return "sync-synchronizing"
}

// SendDesktop triggers a desktop notification via notify-send.
func (d *Dispatcher) SendDesktop(title, message, urgency string) error {
	notifyBin, err := exec.LookPath("notify-send")
	if err != nil {
		// notify-send not available in PATH
		return nil
	}

	u := "normal"
	if urgency == "critical" {
		u = "critical"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// #nosec G204 - desktop notifications via resolved notify-send binary
	cmd := exec.CommandContext(ctx, notifyBin, "-a", d.AppName, "-u", u, "-i", d.IconPath, title, message)
	return cmd.Run()
}

// ShouldNotifyError determines if an error notification should be dispatched based on state.
func (d *Dispatcher) ShouldNotifyError(errType string, lastErrorType string, lastErrorTimestamp float64) bool {
	if !d.EnabledSmartNotifications {
		return true
	}

	if errType != lastErrorType {
		return true
	}

	hoursSince := (float64(time.Now().Unix()) - lastErrorTimestamp) / 3600.0
	return hoursSince >= d.ErrorRepeatHours
}

// SendWebhook delivers a JSON payload to the configured webhook endpoint.
func (d *Dispatcher) SendWebhook(eventType string, data map[string]interface{}) error {
	if d.WebhookURL == "" {
		return nil
	}

	payload := map[string]interface{}{
		"event":     eventType,
		"timestamp": time.Now().Unix(),
		"app":       d.AppName,
		"data":      data,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, d.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
