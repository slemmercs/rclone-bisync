package models

import (
	"encoding/json"
	"testing"
)

func TestCryptWrapRequestJSON(t *testing.T) {
	req := CryptWrapRequest{
		Name:                    "secret_vault",
		Remote:                  "myremote:encrypted",
		Password:                "supersecret",
		Password2:               "secondsalt",
		FilenameEncryption:      "standard",
		DirectoryNameEncryption: true,
		CreateProfile:           true,
		LocalPath:               "/home/user/vault",
		Schedule:                "daily",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal CryptWrapRequest: %v", err)
	}

	var parsed CryptWrapRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal CryptWrapRequest: %v", err)
	}

	if parsed.Name != req.Name || parsed.Remote != req.Remote || parsed.Password != req.Password {
		t.Errorf("unmarshaled mismatch: got %+v, want %+v", parsed, req)
	}
	if parsed.FilenameEncryption != "standard" || !parsed.DirectoryNameEncryption {
		t.Errorf("encryption flags mismatch: got %+v", parsed)
	}
}

func TestCryptWrapResponseJSON(t *testing.T) {
	resp := CryptWrapResponse{
		RemoteName:  "secret_vault",
		BaseRemote:  "myremote:encrypted",
		ProfileID:   "prof_123",
		ProfileName: "Vault Sync",
		Success:     true,
		Message:     "crypt remote created successfully",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal CryptWrapResponse: %v", err)
	}

	var parsed CryptWrapResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal CryptWrapResponse: %v", err)
	}

	if parsed.RemoteName != resp.RemoteName || !parsed.Success {
		t.Errorf("unmarshaled mismatch: got %+v, want %+v", parsed, resp)
	}
}
