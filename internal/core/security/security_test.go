package security

import (
	"testing"
)

type CredentialContainer struct {
	Username string            `json:"username"`
	Password string            `json:"password"`
	Meta     map[string]string `json:"meta"`
}

func TestRedactSecretsBasic(t *testing.T) {
	input := map[string]interface{}{
		"normal_field": "visible",
		"rc_pass":      "my_secret_pass",
		"empty_pass":   "",
		"count":        42,
		"active":       true,
	}

	result := RedactSecrets(input).(map[string]interface{})

	if result["normal_field"] != "visible" {
		t.Errorf("expected visible, got %v", result["normal_field"])
	}
	if result["rc_pass"] != RedactedString {
		t.Errorf("expected %s, got %v", RedactedString, result["rc_pass"])
	}
	if result["empty_pass"] != "" {
		t.Errorf("expected empty string preserved, got %v", result["empty_pass"])
	}
	if result["count"] != 42 {
		t.Errorf("expected 42, got %v", result["count"])
	}
}

func TestRedactSecretsNested(t *testing.T) {
	input := map[string]interface{}{
		"services": []interface{}{
			map[string]interface{}{
				"name":    "google",
				"api_key": "AIzaSyD-secret",
				"user_id": 123,
			},
		},
		"nested": map[string]interface{}{
			"slack_webhook": "https://hooks.slack.com/services/T00/B00/X00",
			"safe_note":     "all good",
		},
	}

	result := RedactSecrets(input).(map[string]interface{})

	services := result["services"].([]interface{})
	firstService := services[0].(map[string]interface{})
	if firstService["api_key"] != RedactedString {
		t.Errorf("expected api_key redacted, got %v", firstService["api_key"])
	}
	if firstService["name"] != "google" {
		t.Errorf("expected google, got %v", firstService["name"])
	}

	nested := result["nested"].(map[string]interface{})
	if nested["slack_webhook"] != RedactedString {
		t.Errorf("expected webhook redacted, got %v", nested["slack_webhook"])
	}
	if nested["safe_note"] != "all good" {
		t.Errorf("expected safe_note preserved, got %v", nested["safe_note"])
	}
}

func TestRedactSensitiveValuesRegex(t *testing.T) {
	input := map[string]interface{}{
		"arbitrary_string": "bearer ya29.a0ARrdaM-12345abcdef",
		"git_ref":          "ghp_123456789012345678901234567890123456",
		"normal_url":       "https://rclone.org",
	}

	result := RedactSecrets(input).(map[string]interface{})

	if result["arbitrary_string"] != RedactedString {
		t.Errorf("expected ya29 token redacted, got %v", result["arbitrary_string"])
	}
	if result["git_ref"] != RedactedString {
		t.Errorf("expected ghp token redacted, got %v", result["git_ref"])
	}
	if result["normal_url"] != "https://rclone.org" {
		t.Errorf("expected URL preserved, got %v", result["normal_url"])
	}
}

func TestRedactStruct(t *testing.T) {
	cred := CredentialContainer{
		Username: "admin",
		Password: "verysecretpassword",
		Meta: map[string]string{
			"auth_token": "token123",
			"desc":       "testing",
		},
	}

	result := RedactSecrets(cred).(map[string]interface{})

	if result["username"] != "admin" {
		t.Errorf("expected username admin, got %v", result["username"])
	}
	if result["password"] != RedactedString {
		t.Errorf("expected password redacted, got %v", result["password"])
	}

	meta := result["meta"].(map[string]interface{})
	if meta["auth_token"] != RedactedString {
		t.Errorf("expected auth_token redacted, got %v", meta["auth_token"])
	}
	if meta["desc"] != "testing" {
		t.Errorf("expected desc preserved, got %v", meta["desc"])
	}
}
