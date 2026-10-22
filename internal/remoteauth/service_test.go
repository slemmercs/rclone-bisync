package remoteauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

func TestAuditLogger(t *testing.T) {
	// Arrange
	logger := NewAuditLogger(3)

	// Act 1: Log events within capacity
	logger.LogEvent("oauth_reconnect_started", "gdrive", "started", "Beginning re-authentication")
	logger.LogEvent("oauth_credentials_updated", "gdrive", "success", "Updated custom OAuth app credentials")
	logger.LogEvent("oauth_reconnect_success", "gdrive", "success", "Connected")

	// Assert 1: Items retrieved newest-first
	events := logger.GetEvents(10)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].Event != "oauth_reconnect_success" {
		t.Fatalf("expected newest event first, got %s", events[0].Event)
	}

	// Act 2: Exceed capacity to trigger ring-buffer eviction
	logger.LogEvent("oauth_validation_success", "onedrive", "success", "Valid")
	eventsAfterEviction := logger.GetEvents(10)
	if len(eventsAfterEviction) != 3 {
		t.Fatalf("expected capacity capped at 3, got %d", len(eventsAfterEviction))
	}
	if eventsAfterEviction[0].Event != "oauth_validation_success" {
		t.Fatalf("expected newest event to be oauth_validation_success, got %s", eventsAfterEviction[0].Event)
	}

	// Act 3: Verify sensitive string redaction in details
	logger.LogEvent("oauth_secret_leak_test", "gdrive", "info", "ya29.sensitive_token_value_here_123456789")
	latest := logger.GetEvents(1)
	if latest[0].Details != "********" {
		t.Fatalf("expected sensitive details to be masked with ********, got: %s", latest[0].Details)
	}

	// Act 4: Clear
	logger.Clear()
	if len(logger.GetEvents(10)) != 0 {
		t.Fatal("expected empty events after Clear")
	}
}

func TestValidator(t *testing.T) {
	// Arrange
	var aboutHandler func(w http.ResponseWriter, r *http.Request)
	var listHandler func(w http.ResponseWriter, r *http.Request)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/operations/about":
			if aboutHandler != nil {
				aboutHandler(w, r)
			} else {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"total": 1000}`))
			}
		case "/operations/list":
			if listHandler != nil {
				listHandler(w, r)
			} else {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"list": []}`))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rClient := client.NewRcloneClient(server.URL, "user", "pass")
	svc := NewRemoteAuthService(&config.Config{}, rClient, nil)
	ctx := context.Background()

	// Scenario 1: operations/about succeeds
	resp1, err := svc.ValidateRemote(ctx, "gdrive")
	if err != nil || !resp1.Valid || resp1.Status != models.AuthStatusConnected {
		t.Fatalf("expected connected for about success: %+v, err=%v", resp1, err)
	}

	// Scenario 2: operations/about fails, operations/list succeeds
	aboutHandler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}
	resp2, err := svc.ValidateRemote(ctx, "gdrive")
	if err != nil || !resp2.Valid || resp2.Status != models.AuthStatusConnected {
		t.Fatalf("expected connected for list fallback success: %+v, err=%v", resp2, err)
	}

	// Scenario 3: operations/about and operations/list fail with OAuth expired error
	listHandler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "oauth2: cannot fetch token: 400 Bad Request, invalid_grant"}`))
	}
	resp3, err := svc.ValidateRemote(ctx, "gdrive")
	if err != nil || resp3.Valid || resp3.Status != models.AuthStatusAuthorizationRequired {
		t.Fatalf("expected authorization_required for invalid_grant error: %+v, err=%v", resp3, err)
	}

	// Scenario 4: operations/list fails with generic connection error
	listHandler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`dial tcp 1.2.3.4: connection refused`))
	}
	resp4, err := svc.ValidateRemote(ctx, "gdrive")
	if err != nil || resp4.Valid || resp4.Status != models.AuthStatusError {
		t.Fatalf("expected error status for connection refused: %+v, err=%v", resp4, err)
	}

	// Scenario 5: User-initiated ValidateRemote records audit events
	auditEvents := svc.GetAuditEvents(10)
	if len(auditEvents) == 0 {
		t.Fatal("expected audit events to be recorded for ValidateRemote calls")
	}
}

func TestHealthAssessment(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/config/listremotes":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"remotes": ["gdrive_shared", "gdrive_custom", "s3_backup", "dropbox_no_token", "broken_remote"]}`))
		case "/config/get":
			var req map[string]string
			_ = json.NewDecoder(r.Body).Decode(&req)
			switch req["name"] {
			case "gdrive_shared":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"type":"drive","client_id":"","token":"{\"access_token\":\"ya29.sample\",\"expiry\":\"2026-10-22T20:00:00Z\"}"}`))
			case "gdrive_custom":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"type":"drive","client_id":"custom-app-id.apps.googleusercontent.com","client_secret":"sec123","token":"{\"access_token\":\"ya29.sample\"}"}`))
			case "s3_backup":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"type":"s3","endpoint":"s3.amazonaws.com"}`))
			case "dropbox_no_token":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"type":"dropbox"}`))
			case "broken_remote":
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`internal rclone error`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		case "/operations/about":
			var req map[string]string
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req["fs"] == "gdrive_shared:" || req["fs"] == "gdrive_custom:" || req["fs"] == "s3_backup:" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"total": 5000000}`))
			} else {
				w.WriteHeader(http.StatusBadRequest)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		RemotePath: "gdrive_shared:",
	}
	rClient := client.NewRcloneClient(server.URL, "user", "pass")
	svc := NewRemoteAuthService(cfg, rClient, nil)
	ctx := context.Background()

	// Act & Assert 1: Google Drive with shared client_id -> Migration Required
	h1, err := svc.GetRemoteAuthHealth(ctx, "gdrive_shared")
	if err != nil {
		t.Fatalf("unexpected error for gdrive_shared: %v", err)
	}
	if h1.Status != models.AuthStatusMigrationRequired {
		t.Fatalf("expected status %s, got %s", models.AuthStatusMigrationRequired, h1.Status)
	}
	if !h1.IsActiveTarget {
		t.Fatal("expected gdrive_shared to be marked as active target")
	}

	// Act & Assert 2: Google Drive with custom credentials -> Connected
	h2, err := svc.GetRemoteAuthHealth(ctx, "gdrive_custom")
	if err != nil {
		t.Fatalf("unexpected error for gdrive_custom: %v", err)
	}
	if h2.Status != models.AuthStatusConnected {
		t.Fatalf("expected status %s, got %s", models.AuthStatusConnected, h2.Status)
	}
	if !h2.HasClientSecret {
		t.Fatal("expected has_client_secret to be true")
	}

	// Act & Assert 3: Non-OAuth provider (s3) -> Connected, is_oauth=false
	h3, err := svc.GetRemoteAuthHealth(ctx, "s3_backup")
	if err != nil {
		t.Fatalf("unexpected error for s3_backup: %v", err)
	}
	if h3.IsOAuth || h3.Status != models.AuthStatusConnected {
		t.Fatalf("expected non-oauth connected for s3, got isOAuth=%v, status=%s", h3.IsOAuth, h3.Status)
	}

	// Act & Assert 4: OAuth provider without token -> Authorization Required
	h4, err := svc.GetRemoteAuthHealth(ctx, "dropbox_no_token")
	if err != nil {
		t.Fatalf("unexpected error for dropbox_no_token: %v", err)
	}
	if h4.Status != models.AuthStatusAuthorizationRequired {
		t.Fatalf("expected status %s for missing token, got %s", models.AuthStatusAuthorizationRequired, h4.Status)
	}

	// Act & Assert 5: Remote with broken config/get -> Error
	h5, err := svc.GetRemoteAuthHealth(ctx, "broken_remote")
	if err != nil {
		t.Fatalf("unexpected error for broken_remote: %v", err)
	}
	if h5.Status != models.AuthStatusError {
		t.Fatalf("expected status %s for broken config, got %s", models.AuthStatusError, h5.Status)
	}

	// Act & Assert 6: Query aggregate auth status
	allResp, err := svc.GetAllRemotesAuthStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected GetAllRemotesAuthStatus error: %v", err)
	}
	if allResp.Summary.TotalRemotes != 5 {
		t.Fatalf("expected 5 total remotes, got %d", allResp.Summary.TotalRemotes)
	}
	if allResp.Summary.OAuthRemotes != 3 {
		t.Fatalf("expected 3 OAuth remotes, got %d", allResp.Summary.OAuthRemotes)
	}
	if allResp.Summary.MigrationRequired != 1 {
		t.Fatalf("expected 1 migration required, got %d", allResp.Summary.MigrationRequired)
	}
	if allResp.Summary.AuthorizationRequired != 1 {
		t.Fatalf("expected 1 authorization required, got %d", allResp.Summary.AuthorizationRequired)
	}
	if allResp.Summary.Connected != 2 {
		t.Fatalf("expected 2 connected, got %d", allResp.Summary.Connected)
	}
	if allResp.Summary.Errors != 1 {
		t.Fatalf("expected 1 error, got %d", allResp.Summary.Errors)
	}

	// Act & Assert 7: Passive health assessment must NEVER pollute the audit log
	if len(svc.GetAuditEvents(10)) != 0 {
		t.Fatalf("expected 0 audit events during passive health assessment, got %d", len(svc.GetAuditEvents(10)))
	}
}

func TestMigrationUpdateCredentials(t *testing.T) {
	// Arrange
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/config/update" {
			_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	rClient := client.NewRcloneClient(server.URL, "user", "pass")
	svc := NewRemoteAuthService(&config.Config{}, rClient, nil)
	ctx := context.Background()

	// Act 1: Validation checks on inputs
	if err := svc.UpdateCredentials(ctx, "", "id", "sec"); err == nil {
		t.Fatal("expected error on empty remote name")
	}
	if err := svc.UpdateCredentials(ctx, "gdrive", "", "sec"); err == nil {
		t.Fatal("expected error on empty client_id")
	}

	// Act 2: Valid update
	err := svc.UpdateCredentials(ctx, "gdrive:", "my-client-id", "my-client-secret")
	if err != nil {
		t.Fatalf("expected UpdateCredentials to succeed: %v", err)
	}

	// Assert 2: Verify received parameters
	if receivedPayload["name"] != "gdrive" {
		t.Fatalf("expected name gdrive, got %v", receivedPayload["name"])
	}
	params, ok := receivedPayload["parameters"].(map[string]interface{})
	if !ok || params["client_id"] != "my-client-id" || params["client_secret"] != "my-client-secret" {
		t.Fatalf("unexpected parameters payload: %v", params)
	}

	// Assert 3: Audit event recorded without leaking client_secret
	events := svc.GetAuditEvents(5)
	if len(events) == 0 {
		t.Fatal("expected audit event for credentials update")
	}
	if strings.Contains(events[0].Details, "my-client-secret") {
		t.Fatal("security violation: client_secret leaked into audit details")
	}
}

func TestReconnectRemote(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/config/oauthstatus":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "running", "authUrl": "http://127.0.0.1:53682/auth?state=xyz"}`))
		case "/config/oauthstop":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	mockRunnerCalled := false
	mockRunner := func(ctx context.Context, name string, env []string, args ...string) (string, error) {
		mockRunnerCalled = true
		if len(args) != 4 || args[0] != "config" || args[1] != "reconnect" || args[2] != "gdrive:" || args[3] != "--auto-confirm" {
			return "", fmt.Errorf("unexpected args: %v", args)
		}
		return "Log in to Google Drive in your browser", nil
	}

	rClient := client.NewRcloneClient(server.URL, "user", "pass")
	svc := NewRemoteAuthService(&config.Config{}, rClient, mockRunner)
	ctx := context.Background()

	// Act 1: Empty remote
	if _, err := svc.ReconnectRemote(ctx, ""); err == nil {
		t.Fatal("expected error on empty remote")
	}

	// Act 2: Successful reconnect
	resp, err := svc.ReconnectRemote(ctx, "gdrive")
	if err != nil {
		t.Fatalf("expected ReconnectRemote to succeed: %v", err)
	}
	if !mockRunnerCalled {
		t.Fatal("expected mockRunner to be invoked")
	}
	if resp.Status != "success" || resp.Remote != "gdrive" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.AuthURL != "http://127.0.0.1:53682/auth?state=xyz" {
		t.Fatalf("expected authUrl to be populated from OAuthStatus, got: %s", resp.AuthURL)
	}

	// Act 3: Concurrency protection (overlapping reconnects on the same remote)
	var wg sync.WaitGroup
	started := make(chan struct{})
	blockRunner := func(ctx context.Context, name string, env []string, args ...string) (string, error) {
		close(started)
		<-ctx.Done()
		return "", ctx.Err()
	}

	blockingSvc := NewRemoteAuthService(&config.Config{}, rClient, blockRunner)
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = blockingSvc.ReconnectRemote(ctx, "gdrive")
	}()

	<-started // Wait for first reconnect to begin
	// Attempt concurrent second reconnect on the same remote
	respOverlap, errOverlap := blockingSvc.ReconnectRemote(ctx, "gdrive")
	if errOverlap == nil || respOverlap.Status != "error" {
		t.Fatalf("expected overlapping reconnect to be rejected: %+v, err=%v", respOverlap, errOverlap)
	}

	// Stop OAuth to release the blocked goroutine
	if err := blockingSvc.StopOAuth(ctx); err != nil {
		t.Fatalf("expected StopOAuth to succeed: %v", err)
	}
	wg.Wait()

	// Test CancelReconnect
	var wg2 sync.WaitGroup
	started2 := make(chan struct{})
	blockRunner2 := func(ctx context.Context, name string, env []string, args ...string) (string, error) {
		close(started2)
		<-ctx.Done()
		return "", ctx.Err()
	}
	cancellingSvc := NewRemoteAuthService(&config.Config{}, rClient, blockRunner2)
	wg2.Add(1)
	go func() {
		defer wg2.Done()
		_, _ = cancellingSvc.ReconnectRemote(ctx, "gdrive")
	}()
	<-started2
	if !cancellingSvc.CancelReconnect("gdrive") {
		t.Fatal("expected CancelReconnect to return true for active reconnect")
	}
	if cancellingSvc.CancelReconnect("nonexistent") {
		t.Fatal("expected CancelReconnect to return false for nonexistent reconnect")
	}
	wg2.Wait()


	// Act 4: Runner error handling
	failRunner := func(ctx context.Context, name string, env []string, args ...string) (string, error) {
		return "", errors.New("browser execution failed")
	}
	failSvc := NewRemoteAuthService(&config.Config{}, rClient, failRunner)
	failResp, failErr := failSvc.ReconnectRemote(ctx, "onedrive")
	if failErr == nil || failResp.Status != "error" {
		t.Fatalf("expected error on failed runner: %+v, err=%v", failResp, failErr)
	}
}

func TestServiceConstructorAndLabels(t *testing.T) {
	// Arrange
	cfg := &config.Config{
		RCAPIURL: "http://127.0.0.1:5572",
		RCUser:   "admin",
		RCPass:   "secret",
	}

	// Act 1: Constructor with nil client & runner
	svc := NewRemoteAuthService(cfg, nil, nil)
	if svc == nil || svc.Client == nil || svc.Runner == nil {
		t.Fatal("expected NewRemoteAuthService to initialize default client and runner")
	}

	// Act 2: Constructor with invalid capacity for logger
	logger := NewAuditLogger(0)
	if logger.maxSize != DefaultMaxAuditEvents {
		t.Fatalf("expected default max size %d, got %d", DefaultMaxAuditEvents, logger.maxSize)
	}

	// Act 3: GetProviderLabel fallbacks
	if lbl := GetProviderLabel("drive"); lbl != "Google Drive" {
		t.Fatalf("expected Google Drive, got %s", lbl)
	}
	if lbl := GetProviderLabel("customprovider"); lbl != "Customprovider" {
		t.Fatalf("expected Customprovider, got %s", lbl)
	}
	if lbl := GetProviderLabel(""); lbl != "Unknown" {
		t.Fatalf("expected Unknown, got %s", lbl)
	}

	// Act 4: IsOAuthProvider
	if !IsOAuthProvider("onedrive") || !IsOAuthProvider("DROPBOX") {
		t.Fatal("expected onedrive and dropbox to be oauth providers")
	}
	if IsOAuthProvider("s3") || IsOAuthProvider("local") {
		t.Fatal("expected s3 and local not to be oauth providers")
	}
}

func TestCollectSessionEnvironment(t *testing.T) {
	// Act
	env := collectSessionEnvironment()
	// Assert: Should contain standard env entries
	if len(env) == 0 {
		t.Fatal("expected non-empty environment")
	}
}

