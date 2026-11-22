package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/app"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

func TestVirtualRemotesHandlers(t *testing.T) {
	tmpDir := t.TempDir()
	profStore, err := profiles.NewStore(filepath.Join(tmpDir, "profiles.json"), filepath.Join(tmpDir, "history.json"))
	if err != nil {
		t.Fatalf("failed to create profStore: %v", err)
	}
	profSvc := profiles.NewService(profStore, nil, nil)

	rcloneServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/config/listremotes":
			_ = json.NewEncoder(w).Encode(map[string][]string{"remotes": {}})
		case "/core/obscure":
			var req map[string]string
			_ = json.NewDecoder(r.Body).Decode(&req)
			_ = json.NewEncoder(w).Encode(map[string]string{"obscured": "obs_" + req["clear"]})
		case "/config/create":
			w.WriteHeader(http.StatusOK)
		case "/backend/command":
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "hasher_command_ok"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer rcloneServer.Close()

	rcClient := client.NewRcloneClient(rcloneServer.URL, "", "")
	cfg := &config.Config{}
	hasherSvc := services.NewHasherService(cfg, rcClient, nil, profSvc, nil)
	cryptSvc := services.NewCryptService(cfg, rcClient, nil, profSvc, nil)

	ctr := &app.Container{
		Config:        cfg,
		Client:        rcClient,
		HasherService: hasherSvc,
		CryptService:  cryptSvc,
	}

	srv := NewServerWithContainer(ctr, nil)

	t.Run("POST /api/remotes/hasher/wrap - success", func(t *testing.T) {
		payload, _ := json.Marshal(models.HasherWrapRequest{
			Name:   "my_hasher",
			Remote: "my_sftp:",
			Hashes: "md5,sha1",
		})
		req, _ := http.NewRequest("POST", "/api/remotes/hasher/wrap", bytes.NewBuffer(payload))
		req.Header.Set("X-Dashboard-Request", "1")
		rec := httptest.NewRecorder()

		srv.Mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}
		var res models.CreateRemoteResult
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if res.RemoteName != "my_hasher" || res.RemoteType != "hasher" {
			t.Errorf("unexpected result: %+v", res)
		}
	})

	t.Run("POST /api/remotes/hasher/wrap - validation error", func(t *testing.T) {
		payload, _ := json.Marshal(models.HasherWrapRequest{
			Name:   "",
			Remote: "my_sftp:",
		})
		req, _ := http.NewRequest("POST", "/api/remotes/hasher/wrap", bytes.NewBuffer(payload))
		req.Header.Set("X-Dashboard-Request", "1")
		rec := httptest.NewRecorder()

		srv.Mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("POST /api/remotes/hasher/command - success", func(t *testing.T) {
		payload, _ := json.Marshal(models.HasherCommandRequest{
			Remote:  "my_hasher",
			Command: "dump",
			Args:    []string{"md5"},
		})
		req, _ := http.NewRequest("POST", "/api/remotes/hasher/command", bytes.NewBuffer(payload))
		req.Header.Set("X-Dashboard-Request", "1")
		rec := httptest.NewRecorder()

		srv.Mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
		var res models.HasherCommandResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if res.Status != "success" || res.Command != "dump" {
			t.Errorf("unexpected command response: %+v", res)
		}
	})

	t.Run("POST /api/remotes/hasher/command - unsupported command error", func(t *testing.T) {
		payload, _ := json.Marshal(models.HasherCommandRequest{
			Remote:  "my_hasher",
			Command: "unsupported_cmd",
		})
		req, _ := http.NewRequest("POST", "/api/remotes/hasher/command", bytes.NewBuffer(payload))
		req.Header.Set("X-Dashboard-Request", "1")
		rec := httptest.NewRecorder()

		srv.Mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("POST /api/remotes/crypt/wrap - success", func(t *testing.T) {
		payload, _ := json.Marshal(models.CryptWrapRequest{
			Name:               "my_crypt",
			Remote:             "my_drive:data",
			Password:           "secretpass",
			FilenameEncryption: "standard",
		})
		req, _ := http.NewRequest("POST", "/api/remotes/crypt/wrap", bytes.NewBuffer(payload))
		req.Header.Set("X-Dashboard-Request", "1")
		rec := httptest.NewRecorder()

		srv.Mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}
		var res models.CryptWrapResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if res.RemoteName != "my_crypt" || res.BaseRemote != "my_drive:data" {
			t.Errorf("unexpected response: %+v", res)
		}
	})

	t.Run("POST /api/remotes/crypt/wrap - validation error", func(t *testing.T) {
		payload, _ := json.Marshal(models.CryptWrapRequest{
			Name:     "my_crypt",
			Remote:   "my_drive:data",
			Password: "",
		})
		req, _ := http.NewRequest("POST", "/api/remotes/crypt/wrap", bytes.NewBuffer(payload))
		req.Header.Set("X-Dashboard-Request", "1")
		rec := httptest.NewRecorder()

		srv.Mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}
