package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
)

func TestCryptService_WrapRemoteInCrypt(t *testing.T) {
	tmpDir := t.TempDir()
	profStore, err := profiles.NewStore(filepath.Join(tmpDir, "profiles.json"), filepath.Join(tmpDir, "history.json"))
	if err != nil {
		t.Fatalf("failed to create profStore: %v", err)
	}
	profSvc := profiles.NewService(profStore, nil, nil)

	var lastCreatedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/config/listremotes":
			_ = json.NewEncoder(w).Encode(map[string][]string{"remotes": {"existing_rem:"}})
		case "/core/obscure":
			var req map[string]string
			_ = json.NewDecoder(r.Body).Decode(&req)
			_ = json.NewEncoder(w).Encode(map[string]string{"obscured": "obs_" + req["clear"]})
		case "/config/create":
			_ = json.NewDecoder(r.Body).Decode(&lastCreatedPayload)
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := client.NewRcloneClient(server.URL, "", "")
	telSvc := NewTelemetryService(&config.Config{}, c)
	svc := NewCryptService(&config.Config{}, c, nil, profSvc, telSvc)

	t.Run("validation errors", func(t *testing.T) {
		_, err := svc.WrapRemoteInCrypt(context.Background(), models.CryptWrapRequest{Name: ""}, "test")
		if err == nil {
			t.Error("expected error for empty name")
		}

		_, err = svc.WrapRemoteInCrypt(context.Background(), models.CryptWrapRequest{Name: "valid_crypt", Remote: ""}, "test")
		if err == nil {
			t.Error("expected error for empty remote")
		}

		_, err = svc.WrapRemoteInCrypt(context.Background(), models.CryptWrapRequest{Name: "valid_crypt", Remote: "drive:", Password: ""}, "test")
		if err == nil {
			t.Error("expected error for empty password")
		}

		_, err = svc.WrapRemoteInCrypt(context.Background(), models.CryptWrapRequest{Name: "existing_rem", Remote: "drive:", Password: "secret"}, "test")
		if err == nil {
			t.Error("expected error for remote collision")
		}
	})

	t.Run("successful crypt wrap with obscure and profile", func(t *testing.T) {
		res, err := svc.WrapRemoteInCrypt(context.Background(), models.CryptWrapRequest{
			Name:                    "vault_crypt",
			Remote:                  "mydrive:secret_folder",
			Password:                "my-plain-password",
			Password2:               "my-salt",
			FilenameEncryption:      "standard",
			DirectoryNameEncryption: true,
			CreateProfile:           true,
			LocalPath:               filepath.Join(tmpDir, "local_vault"),
			Schedule:                "daily",
		}, "admin")
		if err != nil {
			t.Fatalf("unexpected crypt wrap error: %v", err)
		}

		if res.RemoteName != "vault_crypt" || res.BaseRemote != "mydrive:secret_folder" || !res.Success {
			t.Errorf("unexpected response: %+v", res)
		}
		if res.ProfileID == "" {
			t.Error("expected profile to be created")
		}

		if lastCreatedPayload["type"] != "crypt" {
			t.Errorf("expected type crypt, got %v", lastCreatedPayload["type"])
		}
		params, ok := lastCreatedPayload["parameters"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected parameters map, got %T", lastCreatedPayload["parameters"])
		}
		if params["password"] != "obs_my-plain-password" {
			t.Errorf("expected password to be obscured, got %v", params["password"])
		}
		if params["password2"] != "obs_my-salt" {
			t.Errorf("expected salt password to be obscured, got %v", params["password2"])
		}
		if params["filename_encryption"] != "standard" || params["directory_name_encryption"] != "true" {
			t.Errorf("unexpected encryption parameters: %+v", params)
		}
	})
}
