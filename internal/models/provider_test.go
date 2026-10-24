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
