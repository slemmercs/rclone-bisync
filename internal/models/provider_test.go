package models

import (
	"encoding/json"
	"testing"
)

func TestProvidersResponseJSON(t *testing.T) {
	rawJSON := `{
		"providers": [
			{
				"Name": "drive",
				"Description": "Google Drive",
				"Prefix": "drive",
				"Hide": false,
				"Options": [
					{
						"Name": "client_id",
						"Help": "OAuth Client Id",
						"Type": "string",
						"Required": false,
						"IsPassword": false,
						"Sensitive": false,
						"Advanced": false,
						"Exclusive": false
					}
				]
			}
		]
	}`

	var res ProvidersResponse
	if err := json.Unmarshal([]byte(rawJSON), &res); err != nil {
		t.Fatalf("failed to unmarshal ProvidersResponse: %v", err)
	}

	if len(res.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(res.Providers))
	}

	p := res.Providers[0]
	if p.Name != "drive" || p.Description != "Google Drive" {
		t.Errorf("unexpected provider details: %+v", p)
	}

	if len(p.Options) != 1 || p.Options[0].Name != "client_id" {
		t.Errorf("unexpected options: %+v", p.Options)
	}
}

func TestRemoteRequestsJSON(t *testing.T) {
	req := CreateRemoteRequest{
		Name: "gdrive-test",
		Type: "drive",
		Parameters: map[string]interface{}{
			"scope": "drive",
		},
		Opt: map[string]interface{}{
			"nonInteractive": true,
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal CreateRemoteRequest: %v", err)
	}

	var parsed CreateRemoteRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal CreateRemoteRequest: %v", err)
	}

	if parsed.Name != "gdrive-test" || parsed.Type != "drive" {
		t.Errorf("unexpected parsed CreateRemoteRequest: %+v", parsed)
	}

	delReq := DeleteRemoteRequest{Name: "gdrive-test"}
	delData, err := json.Marshal(delReq)
	if err != nil {
		t.Fatalf("failed to marshal DeleteRemoteRequest: %v", err)
	}

	var parsedDel DeleteRemoteRequest
	if err := json.Unmarshal(delData, &parsedDel); err != nil {
		t.Fatalf("failed to unmarshal DeleteRemoteRequest: %v", err)
	}
	if parsedDel.Name != "gdrive-test" {
		t.Errorf("expected name gdrive-test, got %s", parsedDel.Name)
	}
}

func TestIsOAuthProvider(t *testing.T) {
	tests := []struct {
		provider string
		expected bool
	}{
		{"drive", true},
		{"DRIVE", true},
		{"  onedrive  ", true},
		{"dropbox", true},
		{"box", true},
		{"pcloud", true},
		{"google photos", true},
		{"s3", false},
		{"b2", false},
		{"sftp", false},
		{"local", false},
		{"", false},
	}

	for _, tt := range tests {
		got := IsOAuthProvider(tt.provider)
		if got != tt.expected {
			t.Errorf("IsOAuthProvider(%q) = %v; want %v", tt.provider, got, tt.expected)
		}
	}
}

func TestProviderOption_FlavorAndHide(t *testing.T) {
	rawJSON := `{
		"providers": [
			{
				"Name": "s3",
				"Description": "Amazon S3",
				"Options": [
					{
						"Name": "endpoint",
						"Help": "Endpoint for S3 API",
						"Provider": "!AWS",
						"Hide": 0,
						"Type": "string",
						"Required": false,
						"Examples": [
							{
								"Value": "s3.wasabisys.com",
								"Help": "Wasabi US East",
								"Provider": "Wasabi"
							}
						]
					},
					{
						"Name": "config_is_local",
						"Help": "Internal configurator flag",
						"Hide": 2,
						"Type": "string"
					},
					{
						"Name": "both_hidden",
						"Help": "Both hidden",
						"Hide": 3,
						"Type": "string"
					},
					{
						"Name": "cli_hidden_only",
						"Help": "CLI hidden only",
						"Hide": 1,
						"Type": "string"
					}
				]
			}
		]
	}`

	var res ProvidersResponse
	if err := json.Unmarshal([]byte(rawJSON), &res); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	opts := res.Providers[0].Options
	if len(opts) != 4 {
		t.Fatalf("expected 4 options, got %d", len(opts))
	}

	opt0 := opts[0]
	if opt0.Provider != "!AWS" {
		t.Errorf("expected Provider '!AWS', got %q", opt0.Provider)
	}
	if opt0.Hide != 0 || opt0.IsHiddenInConfig() {
		t.Errorf("expected Hide 0 and IsHiddenInConfig false, got %d, %v", opt0.Hide, opt0.IsHiddenInConfig())
	}
	if len(opt0.Examples) != 1 || opt0.Examples[0].Provider != "Wasabi" {
		t.Errorf("unexpected example provider: %+v", opt0.Examples)
	}

	opt1 := opts[1]
	if !opt1.IsHiddenInConfig() {
		t.Errorf("expected opt1 with Hide=2 to be hidden in config")
	}

	opt2 := opts[2]
	if !opt2.IsHiddenInConfig() {
		t.Errorf("expected opt2 with Hide=3 to be hidden in config")
	}

	opt3 := opts[3]
	if opt3.IsHiddenInConfig() {
		t.Errorf("expected opt3 with Hide=1 to not be hidden in config (only CLI hidden)")
	}
}
