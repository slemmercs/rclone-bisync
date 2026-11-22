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

func TestHasherService_WrapRemoteInHasher(t *testing.T) {
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
	svc := NewHasherService(&config.Config{}, c, nil, profSvc, telSvc)

	t.Run("validation errors", func(t *testing.T) {
		_, err := svc.WrapRemoteInHasher(context.Background(), models.HasherWrapRequest{Name: ""}, "test")
		if err == nil {
			t.Error("expected error for empty name")
		}

		_, err = svc.WrapRemoteInHasher(context.Background(), models.HasherWrapRequest{Name: "invalid/name"}, "test")
		if err == nil {
			t.Error("expected error for invalid name")
		}

		_, err = svc.WrapRemoteInHasher(context.Background(), models.HasherWrapRequest{Name: "valid_hasher", Remote: ""}, "test")
		if err == nil {
			t.Error("expected error for empty remote")
		}

		_, err = svc.WrapRemoteInHasher(context.Background(), models.HasherWrapRequest{Name: "existing_rem", Remote: "base:"}, "test")
		if err == nil {
			t.Error("expected error for existing remote collision")
		}
	})

	t.Run("successful wrap with defaults and profile", func(t *testing.T) {
		res, err := svc.WrapRemoteInHasher(context.Background(), models.HasherWrapRequest{
			Name:          "sftp_hasher",
			Remote:        "mysftp:data",
			Hashes:        "md5,sha1",
			MaxAge:        "1w",
			AutoSize:      1024,
			CreateProfile: true,
			LocalPath:     filepath.Join(tmpDir, "local_data"),
			Schedule:      "15m",
		}, "admin")
		if err != nil {
			t.Fatalf("unexpected wrap error: %v", err)
		}

		if res.RemoteName != "sftp_hasher" || res.RemoteType != "hasher" || !res.Success {
			t.Errorf("unexpected result: %+v", res)
		}
		if res.ProfileID == "" {
			t.Error("expected profile to be created")
		}

		if lastCreatedPayload["type"] != "hasher" {
			t.Errorf("expected type hasher, got %v", lastCreatedPayload["type"])
		}
		params, ok := lastCreatedPayload["parameters"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected parameters map, got %T", lastCreatedPayload["parameters"])
		}
		if params["remote"] != "mysftp:data" || params["hashes"] != "md5,sha1" || params["max_age"] != "1w" {
			t.Errorf("unexpected parameters: %+v", params)
		}
	})
}

func TestHasherService_ExecuteHasherCommand(t *testing.T) {
	var lastCmdPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/backend/command" {
			_ = json.NewDecoder(r.Body).Decode(&lastCmdPayload)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"result": "dump completed"})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := client.NewRcloneClient(server.URL, "", "")
	svc := NewHasherService(&config.Config{}, c, nil, nil, nil)

	t.Run("validation errors", func(t *testing.T) {
		_, err := svc.ExecuteHasherCommand(context.Background(), models.HasherCommandRequest{Remote: ""})
		if err == nil {
			t.Error("expected error for empty remote")
		}

		_, err = svc.ExecuteHasherCommand(context.Background(), models.HasherCommandRequest{Remote: "myhasher", Command: "rmdir"})
		if err == nil {
			t.Error("expected error for disallowed command")
		}
	})

	t.Run("successful command execution", func(t *testing.T) {
		resp, err := svc.ExecuteHasherCommand(context.Background(), models.HasherCommandRequest{
			Remote:  "myhasher",
			Command: "dump",
			Args:    []string{"md5"},
			Opt:     map[string]string{"format": "json"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Status != "success" || resp.Command != "dump" || resp.Remote != "myhasher" {
			t.Errorf("unexpected response: %+v", resp)
		}
		if lastCmdPayload["command"] != "dump" || lastCmdPayload["fs"] != "myhasher:" {
			t.Errorf("unexpected backend payload: %+v", lastCmdPayload)
		}
	})
}
