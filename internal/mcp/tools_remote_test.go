package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slemmercs/rclone-bisync/internal/app"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

func TestRemoteTools_ReadOnlyGuard(t *testing.T) {
	srv, err := NewServer(nil, ServerConfig{
		Version:  "v1.0.0-test",
		ReadOnly: true,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// 1. bisync_create_remote blocked in read-only
	res, err := callMCP(srv.handleCreateRemote, map[string]any{
		"name": "my_s3",
		"type": "s3",
	})
	if err != nil || !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "read-only") {
		t.Errorf("expected read-only rejection for create_remote, got: %v, err: %v", res, err)
	}

	// 2. bisync_delete_remote blocked in read-only
	res, err = callMCP(srv.handleDeleteRemote, map[string]any{
		"name": "my_s3",
	})
	if err != nil || !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "read-only") {
		t.Errorf("expected read-only rejection for delete_remote, got: %v, err: %v", res, err)
	}

	// 3. bisync_create_crypt_remote blocked in read-only
	res, err = callMCP(srv.handleCreateCryptRemoteTool, map[string]any{
		"name":     "my_crypt",
		"remote":   "drive:",
		"password": "pass",
	})
	if err != nil || !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "read-only") {
		t.Errorf("expected read-only rejection for create_crypt_remote, got: %v, err: %v", res, err)
	}

	// 4. bisync_hasher_command mutative commands blocked in read-only
	res, err = callMCP(srv.handleHasherCommandTool, map[string]any{
		"remote":  "my_hasher",
		"command": "drop",
	})
	if err != nil || !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "read-only") {
		t.Errorf("expected read-only rejection for hasher drop command, got: %v, err: %v", res, err)
	}
}

func TestRemoteTools_ListProviders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/config/providers" {
			_ = json.NewEncoder(w).Encode(models.ProvidersResponse{
				Providers: []models.Provider{
					{Name: "drive", Description: "Google Drive"},
					{Name: "s3", Description: "Amazon S3 Compliant"},
					{Name: "sftp", Description: "SSH/SFTP"},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := client.NewRcloneClient(server.URL, "", "")
	wizSvc := services.NewRemoteWizardService(&config.Config{}, c, nil, nil, nil)
	ctr := &app.Container{
		RemoteWizardService: wizSvc,
	}

	srv, err := NewServer(ctr, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// 1. Query all
	res, err := callMCP(srv.handleListProviders, map[string]any{})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error listing providers: %v", res)
	}
	text := res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "drive") || !strings.Contains(text, "s3") {
		t.Fatalf("expected drive and s3 in output, got: %s", text)
	}

	// 2. Query filtered by category
	res, err = callMCP(srv.handleListProviders, map[string]any{"category": "cloud"})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error filtering providers: %v", res)
	}
	text = res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "drive") || strings.Contains(text, "sftp") {
		t.Fatalf("expected only cloud providers, got: %s", text)
	}
}

func TestRemoteTools_CreateRemote(t *testing.T) {
	tmpDir := t.TempDir()
	profStore, err := profiles.NewStore(filepath.Join(tmpDir, "profiles.json"), filepath.Join(tmpDir, "history.json"))
	if err != nil {
		t.Fatalf("failed to create profStore: %v", err)
	}
	profSvc := profiles.NewService(profStore, nil, nil)

	oauthRunning := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/config/listremotes":
			_ = json.NewEncoder(w).Encode(map[string][]string{"remotes": {}})
		case "/config/create":
			w.WriteHeader(http.StatusOK)
		case "/config/oauthstatus":
			if oauthRunning {
				_ = json.NewEncoder(w).Encode(map[string]any{"status": "running", "authUrl": "http://127.0.0.1:53682/auth?state=xyz"})
			} else {
				_ = json.NewEncoder(w).Encode(map[string]any{"status": "idle"})
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := client.NewRcloneClient(server.URL, "", "")
	telSvc := services.NewTelemetryService(&config.Config{}, c)
	wizSvc := services.NewRemoteWizardService(&config.Config{}, c, nil, profSvc, telSvc)
	ctr := &app.Container{
		RemoteWizardService: wizSvc,
		ProfileService:      profSvc,
	}

	srv, err := NewServer(ctr, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// 1. Create S3 remote with profile bridge
	res, err := callMCP(srv.handleCreateRemote, map[string]any{
		"name":           "prod_s3",
		"type":           "s3",
		"parameters":     map[string]any{"provider": "AWS"},
		"create_profile": true,
		"local_path":     filepath.Join(tmpDir, "s3_local"),
		"profile_name":   "Prod S3 Profile",
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error creating remote: %v", res)
	}
	text := res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "prod_s3") || !strings.Contains(text, "Prod S3 Profile") {
		t.Fatalf("unexpected create remote result: %s", text)
	}

	// 2. Create interactive OAuth remote
	oauthRunning = true
	res, err = callMCP(srv.handleCreateRemote, map[string]any{
		"name": "gdrive_personal",
		"type": "drive",
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error creating oauth remote: %v", res)
	}
	text = res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "oauth_pending") || !strings.Contains(text, "127.0.0.1:53682") {
		t.Fatalf("expected oauth_pending status with authURL, got: %s", text)
	}

	// 3. Invalid schedule rejected on linked profile creation
	res, err = callMCP(srv.handleCreateRemote, map[string]any{
		"name":           "invalid_sched_remote",
		"type":           "s3",
		"create_profile": true,
		"local_path":     filepath.Join(tmpDir, "local_test"),
		"schedule":       "0 * * * *",
	})
	if err != nil || !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "invalid schedule") {
		t.Fatalf("expected invalid schedule rejection, got: %v", res)
	}

	// 4. Remote name with trailing colon is normalized
	res, err = callMCP(srv.handleCreateRemote, map[string]any{
		"name": "remote_with_colon:",
		"type": "s3",
	})
	if err != nil || res.IsError {
		t.Fatalf("expected colon to be stripped and succeed, got: %v", res)
	}
	text = res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "remote_with_colon") {
		t.Fatalf("expected normalized remote name in result, got: %s", text)
	}

}

func TestRemoteTools_DeleteRemoteConfirmation(t *testing.T) {
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/config/delete" {
			deleted = true
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := client.NewRcloneClient(server.URL, "", "")
	wizSvc := services.NewRemoteWizardService(&config.Config{}, c, nil, nil, nil)
	ctr := &app.Container{
		RemoteWizardService: wizSvc,
	}

	srv, err := NewServer(ctr, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// 1. Call without confirmation token -> returns confirmation ticket
	res, err := callMCP(srv.handleDeleteRemote, map[string]any{
		"name": "target_rem",
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error requesting delete ticket: %v", res)
	}
	text := res.Content[0].(mcp.TextContent).Text
	var ticketPayload struct {
		Status string `json:"status"`
		Token  string `json:"token"`
	}
	if err := json.Unmarshal([]byte(text), &ticketPayload); err != nil {
		t.Fatalf("failed to parse ticket payload: %v", err)
	}
	if ticketPayload.Status != "confirmation_required" || ticketPayload.Token == "" {
		t.Fatalf("expected confirmation_required with token, got: %+v", ticketPayload)
	}
	if deleted {
		t.Fatalf("remote should not be deleted before confirmation")
	}

	// 2. Call with valid token -> deletes remote
	res, err = callMCP(srv.handleDeleteRemote, map[string]any{
		"name":               "target_rem",
		"confirmation_token": ticketPayload.Token,
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error confirming delete: %v", res)
	}
	if !deleted {
		t.Fatalf("expected remote to be deleted after valid confirmation")
	}
}

func TestExtractMapArg(t *testing.T) {
	if extractMapArg(nil, "key") != nil {
		t.Error("expected nil for nil args")
	}
	if extractMapArg("not a map", "key") != nil {
		t.Error("expected nil for string args")
	}

	m := map[string]any{"params": map[string]any{"a": "b"}}
	if res := extractMapArg(m, "params"); res == nil || res["a"] != "b" {
		t.Errorf("expected parsed map, got: %v", res)
	}

	mJSON := map[string]any{"params": `{"foo":"bar"}`}
	if res := extractMapArg(mJSON, "params"); res == nil || res["foo"] != "bar" {
		t.Errorf("expected unmarshaled map, got: %v", res)
	}

	mBad := map[string]any{"params": `{bad json`}
	if extractMapArg(mBad, "params") != nil {
		t.Error("expected nil for invalid JSON string")
	}

	mEmpty := map[string]any{"params": "   "}
	if extractMapArg(mEmpty, "params") != nil {
		t.Error("expected nil for empty string")
	}

	mWrong := map[string]any{"params": 123}
	if extractMapArg(mWrong, "params") != nil {
		t.Error("expected nil for integer value")
	}
}

func TestNormalizeDecision(t *testing.T) {
	decisions := []struct {
		input    string
		expected models.ActionDecision
		valid    bool
	}{
		{"keep-local", models.DecisionKeepLocal, true},
		{"keep_remote", models.DecisionKeepRemote, true},
		{"KEEP-BOTH", models.DecisionKeepBoth, true},
		{"accept_delete", models.DecisionAcceptDelete, true},
		{"restore-item", models.DecisionRestoreItem, true},
		{"skip", models.DecisionSkip, true},
		{"dismiss", "dismiss", true},
		{"unknown-action", "", false},
	}

	for _, d := range decisions {
		dec, ok := normalizeDecision(d.input)
		if ok != d.valid || dec != d.expected {
			t.Errorf("normalizeDecision(%q) = (%q, %v); expected (%q, %v)", d.input, dec, ok, d.expected, d.valid)
		}
	}
}

func TestRemoteTools_HasherCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/backend/command" {
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "dump_completed"})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := client.NewRcloneClient(server.URL, "", "")
	hasherSvc := services.NewHasherService(&config.Config{}, c, nil, nil, nil)
	ctr := &app.Container{
		HasherService: hasherSvc,
	}

	srv, err := NewServer(ctr, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// 1. Missing remote
	res, err := callMCP(srv.handleHasherCommandTool, map[string]any{
		"command": "dump",
	})
	if err != nil || !res.IsError {
		t.Fatalf("expected error for missing remote, got: %v", res)
	}

	// 2. Successful dump
	res, err = callMCP(srv.handleHasherCommandTool, map[string]any{
		"remote":  "myhasher",
		"command": "dump",
		"args":    "md5 sha1",
		"opt":     map[string]any{"format": "json"},
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error executing hasher command: %v", res)
	}
	text := res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "dump_completed") {
		t.Fatalf("expected dump_completed in output, got: %s", text)
	}
}

func TestRemoteTools_CreateCryptRemote(t *testing.T) {
	tmpDir := t.TempDir()
	profStore, err := profiles.NewStore(filepath.Join(tmpDir, "profiles.json"), filepath.Join(tmpDir, "history.json"))
	if err != nil {
		t.Fatalf("failed to create profStore: %v", err)
	}
	profSvc := profiles.NewService(profStore, nil, nil)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/config/listremotes":
			_ = json.NewEncoder(w).Encode(map[string][]string{"remotes": {}})
		case "/core/obscure":
			var req map[string]string
			_ = json.NewDecoder(r.Body).Decode(&req)
			_ = json.NewEncoder(w).Encode(map[string]string{"obscured": "obs_" + req["clear"]})
		case "/config/create":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := client.NewRcloneClient(server.URL, "", "")
	cryptSvc := services.NewCryptService(&config.Config{}, c, nil, profSvc, nil)
	ctr := &app.Container{
		CryptService:   cryptSvc,
		ProfileService: profSvc,
	}

	srv, err := NewServer(ctr, ServerConfig{Version: "v1.0.0-test"})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// 1. Missing password
	res, err := callMCP(srv.handleCreateCryptRemoteTool, map[string]any{
		"name":   "mycrypt",
		"remote": "drive:secret",
	})
	if err != nil || !res.IsError {
		t.Fatalf("expected error for missing password, got: %v", res)
	}

	// 2. Successful creation with profile
	res, err = callMCP(srv.handleCreateCryptRemoteTool, map[string]any{
		"name":                "mycrypt",
		"remote":              "drive:secret",
		"password":            "pass123",
		"filename_encryption": "standard",
		"create_profile":      true,
		"local_path":          filepath.Join(tmpDir, "local_crypt"),
		"schedule":            "daily",
	})
	if err != nil || res.IsError {
		t.Fatalf("unexpected error creating crypt remote: %v", res)
	}
	text := res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "mycrypt") {
		t.Fatalf("expected mycrypt in output, got: %s", text)
	}
}

func TestExtractStringHelpers(t *testing.T) {
	// extractStringSliceArg
	if extractStringSliceArg(nil, "key") != nil {
		t.Error("expected nil for nil args")
	}
	mStr := map[string]any{"args": "one two three"}
	slice := extractStringSliceArg(mStr, "args")
	if len(slice) != 3 || slice[0] != "one" || slice[2] != "three" {
		t.Errorf("expected 3 items from string, got: %v", slice)
	}
	mSlice := map[string]any{"args": []any{"a", "b"}}
	slice2 := extractStringSliceArg(mSlice, "args")
	if len(slice2) != 2 || slice2[0] != "a" {
		t.Errorf("expected 2 items from slice, got: %v", slice2)
	}

	// extractStringMapArg
	if extractStringMapArg(nil, "opt") != nil {
		t.Error("expected nil for nil args")
	}
	mOpt := map[string]any{"opt": map[string]any{"k": "v", "num": 42}}
	resMap := extractStringMapArg(mOpt, "opt")
	if resMap["k"] != "v" || resMap["num"] != "42" {
		t.Errorf("expected parsed string map, got: %v", resMap)
	}
}

