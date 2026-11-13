package services

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

func TestConfigEncryptionService_GetStatus(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Missing file
	missingPath := filepath.Join(tmpDir, "missing.conf")
	svc := NewConfigEncryptionService(&config.Config{RcloneConfigFile: missingPath}, nil, nil)
	status, err := svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error for missing config: %v", err)
	}
	if status.IsEncrypted {
		t.Errorf("expected missing file to not be encrypted")
	}

	// 2. Unencrypted file
	plainPath := filepath.Join(tmpDir, "plain.conf")
	if err := os.WriteFile(plainPath, []byte("[myremote]\ntype = drive\n"), 0600); err != nil {
		t.Fatal(err)
	}
	svc = NewConfigEncryptionService(&config.Config{RcloneConfigFile: plainPath}, nil, nil)
	status, err = svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error for plain config: %v", err)
	}
	if status.IsEncrypted {
		t.Errorf("expected plain file to not be encrypted")
	}

	// 3. Encrypted file
	encPath := filepath.Join(tmpDir, "encrypted.conf")
	encContent := "# Encrypted rclone configuration File\n\nRCLONE_ENCRYPT_V0:\nSGVsbG8gV29ybGQ="
	if err := os.WriteFile(encPath, []byte(encContent), 0600); err != nil {
		t.Fatal(err)
	}
	svc = NewConfigEncryptionService(&config.Config{RcloneConfigFile: encPath}, nil, nil)
	status, err = svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error for encrypted config: %v", err)
	}
	if !status.IsEncrypted {
		t.Errorf("expected encrypted file to be detected as encrypted")
	}
}

func TestConfigEncryptionService_SetPassword(t *testing.T) {
	tmpDir := t.TempDir()
	confPath := filepath.Join(tmpDir, "rclone.conf")

	type cmdCall struct {
		Name string
		Args []string
		Env  []string
	}
	var calls []cmdCall

	mockRunner := func(ctx context.Context, name string, env []string, args ...string) (string, error) {
		calls = append(calls, cmdCall{Name: name, Args: args, Env: env})
		return "success", nil
	}

	svc := NewConfigEncryptionService(&config.Config{RcloneConfigFile: confPath}, nil, mockRunner)

	// Act 1: Empty password
	_, err := svc.SetPassword(context.Background(), models.SetEncryptionRequest{Password: ""})
	if err == nil {
		t.Fatal("expected error for empty password, got nil")
	}

	// Act 2: Valid password
	resp, err := svc.SetPassword(context.Background(), models.SetEncryptionRequest{
		Password:    "newSecret123",
		OldPassword: "oldSecret123",
	})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "success" {
		t.Errorf("expected status success, got %s", resp.Status)
	}
	if len(calls) == 0 {
		t.Fatalf("expected command runner to be called")
	}
	firstCall := calls[0]
	if firstCall.Name != "rclone" {
		t.Errorf("expected first cmd rclone, got %s", firstCall.Name)
	}

	argString := strings.Join(firstCall.Args, " ")
	if !strings.Contains(argString, "config encryption set") || !strings.Contains(argString, "--config") {
		t.Errorf("unexpected args: %v", firstCall.Args)
	}

	envString := strings.Join(firstCall.Env, " ")
	if !strings.Contains(envString, "NEW_PASS=newSecret123") || !strings.Contains(envString, "OLD_PASS=oldSecret123") {
		t.Errorf("expected env variables to contain passwords, got: %s", envString)
	}
}

func TestConfigEncryptionService_RemovePassword(t *testing.T) {
	tmpDir := t.TempDir()
	encPath := filepath.Join(tmpDir, "encrypted.conf")
	encContent := "# Encrypted rclone configuration File\n\nRCLONE_ENCRYPT_V0:\nSGVsbG8="
	if err := os.WriteFile(encPath, []byte(encContent), 0600); err != nil {
		t.Fatal(err)
	}

	var capturedArgs [][]string
	mockRunner := func(ctx context.Context, name string, env []string, args ...string) (string, error) {
		capturedArgs = append(capturedArgs, args)
		return "success", nil
	}

	svc := NewConfigEncryptionService(&config.Config{RcloneConfigFile: encPath}, nil, mockRunner)

	resp, err := svc.RemovePassword(context.Background(), models.RemoveEncryptionRequest{
		Password: "oldSecret123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "success" {
		t.Errorf("expected status success, got %s", resp.Status)
	}
	if len(capturedArgs) == 0 {
		t.Fatalf("expected command runner to be called")
	}
	firstArgs := strings.Join(capturedArgs[0], " ")
	if !strings.Contains(firstArgs, "config encryption remove") {
		t.Errorf("expected config encryption remove args, got: %v", capturedArgs[0])
	}
}

func TestConfigEncryptionService_RemovePassword_AlreadyUnencrypted(t *testing.T) {
	tmpDir := t.TempDir()
	plainPath := filepath.Join(tmpDir, "plain.conf")
	if err := os.WriteFile(plainPath, []byte("[remote]\ntype=drive\n"), 0600); err != nil {
		t.Fatal(err)
	}

	runnerCalled := false
	mockRunner := func(ctx context.Context, name string, env []string, args ...string) (string, error) {
		runnerCalled = true
		return "", fmt.Errorf("should not be called")
	}

	svc := NewConfigEncryptionService(&config.Config{RcloneConfigFile: plainPath}, nil, mockRunner)

	resp, err := svc.RemovePassword(context.Background(), models.RemoveEncryptionRequest{Password: "any"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runnerCalled {
		t.Errorf("expected runner not to be called for unencrypted file")
	}
	if !strings.Contains(resp.Message, "not encrypted") {
		t.Errorf("unexpected message: %s", resp.Message)
	}
}

func TestConfigEncryptionService_CreatePasswordHelperScript(t *testing.T) {
	scriptPath, err := createPasswordHelperScript()
	if err != nil {
		t.Fatalf("failed to create password helper script: %v", err)
	}
	defer func() {
		_ = os.Remove(scriptPath)
	}()

	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("failed to stat script: %v", err)
	}

	// Verify permissions are executable by owner (0700)
	if info.Mode().Perm()&0100 == 0 {
		t.Errorf("expected script to be executable by owner, perm: %o", info.Mode().Perm())
	}

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read script: %v", err)
	}
	if !strings.Contains(string(content), "RCLONE_PASSWORD_CHANGE") {
		t.Errorf("script content missing RCLONE_PASSWORD_CHANGE logic: %s", string(content))
	}
}

func TestConfigEncryptionService_PassphrasePersistenceAndWarning(t *testing.T) {
	tmpDir := t.TempDir()
	confPath := filepath.Join(tmpDir, "encrypted.conf")
	secretsPath := filepath.Join(tmpDir, "rc-secrets.env")

	// 1. Write encrypted config
	encContent := "# Encrypted rclone configuration File\n\nRCLONE_ENCRYPT_V0:\nSGVsbG8="
	if err := os.WriteFile(confPath, []byte(encContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Ensure RCLONE_CONFIG_PASS env var is cleared for test isolation
	oldEnv := os.Getenv("RCLONE_CONFIG_PASS")
	_ = os.Unsetenv("RCLONE_CONFIG_PASS")
	defer func() {
		if oldEnv != "" {
			_ = os.Setenv("RCLONE_CONFIG_PASS", oldEnv)
		}
	}()

	mockRunner := func(ctx context.Context, name string, env []string, args ...string) (string, error) {
		return "drive:\n", nil
	}

	cfg := &config.Config{
		RcloneConfigFile: confPath,
		SecretsFile:      secretsPath,
	}
	svc := NewConfigEncryptionService(cfg, nil, mockRunner)

	// Status before secrets file: Encrypted=true, HasDaemonPassphrase=false, Warning non-empty
	status, err := svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.IsEncrypted {
		t.Errorf("expected IsEncrypted=true")
	}
	if status.HasDaemonPassphrase {
		t.Errorf("expected HasDaemonPassphrase=false")
	}
	if status.Warning == "" {
		t.Errorf("expected warning when daemon passphrase is missing")
	}

	// Act 1: SaveDaemonPassphrase with valid password
	saveResp, err := svc.SaveDaemonPassphrase(context.Background(), models.SaveDaemonPassphraseRequest{Password: "mypassword123"})
	if err != nil {
		t.Fatalf("unexpected error saving passphrase: %v", err)
	}
	if saveResp.Status != "success" {
		t.Errorf("expected status success, got %s", saveResp.Status)
	}

	// Verify secrets file contains RCLONE_CONFIG_PASS
	parsed, err := config.ParseEnvFile(secretsPath)
	if err != nil {
		t.Fatalf("failed to parse secrets file: %v", err)
	}
	if parsed["RCLONE_CONFIG_PASS"] != "mypassword123" {
		t.Errorf("expected 'mypassword123', got '%s'", parsed["RCLONE_CONFIG_PASS"])
	}

	// Status after saving: HasDaemonPassphrase=true, Warning=""
	status, err = svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.HasDaemonPassphrase {
		t.Errorf("expected HasDaemonPassphrase=true after saving")
	}
	if status.Warning != "" {
		t.Errorf("expected empty warning after saving daemon passphrase, got %s", status.Warning)
	}

	// Act 2: SetPassword updates secrets file
	setResp, err := svc.SetPassword(context.Background(), models.SetEncryptionRequest{
		Password:    "newpass456",
		OldPassword: "mypassword123",
	})
	if err != nil {
		t.Fatalf("unexpected error setting password: %v", err)
	}
	if setResp.Status != "success" {
		t.Errorf("expected status success, got %s", setResp.Status)
	}
	parsed, _ = config.ParseEnvFile(secretsPath)
	if parsed["RCLONE_CONFIG_PASS"] != "newpass456" {
		t.Errorf("expected 'newpass456' in secrets file, got '%s'", parsed["RCLONE_CONFIG_PASS"])
	}

	// Act 3: RemovePassword purges secrets file
	remResp, err := svc.RemovePassword(context.Background(), models.RemoveEncryptionRequest{
		Password: "newpass456",
	})
	if err != nil {
		t.Fatalf("unexpected error removing password: %v", err)
	}
	if remResp.Status != "success" {
		t.Errorf("expected status success, got %s", remResp.Status)
	}
	parsed, _ = config.ParseEnvFile(secretsPath)
	if _, exists := parsed["RCLONE_CONFIG_PASS"]; exists {
		t.Errorf("expected RCLONE_CONFIG_PASS to be removed from secrets file")
	}

	// Act 4: SaveDaemonPassphrase with failing runner (invalid password)
	failRunner := func(ctx context.Context, name string, env []string, args ...string) (string, error) {
		return "", fmt.Errorf("bad password")
	}
	failSvc := NewConfigEncryptionService(cfg, nil, failRunner)
	_, err = failSvc.SaveDaemonPassphrase(context.Background(), models.SaveDaemonPassphraseRequest{Password: "wrongpass"})
	if err == nil {
		t.Errorf("expected error when runner fails password validation, got nil")
	}
}

func TestConfigEncryptionService_RestartDaemon(t *testing.T) {
	// 1. Systemctl runner success
	runnerCalled := false
	mockRunner := func(ctx context.Context, name string, env []string, args ...string) (string, error) {
		runnerCalled = true
		if name == "systemctl" && len(args) >= 3 && args[2] == "rclone-daemon.service" {
			return "", nil
		}
		return "", fmt.Errorf("unexpected command")
	}

	svc := NewConfigEncryptionService(&config.Config{}, nil, mockRunner)
	if err := svc.RestartDaemon(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !runnerCalled {
		t.Errorf("expected systemctl runner to be invoked")
	}

	// 2. Runner fails, falls back to Client.Quit
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/core/quit" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	rcClient := client.NewRcloneClient(ts.URL, "", "")
	failingRunner := func(ctx context.Context, name string, env []string, args ...string) (string, error) {
		return "", fmt.Errorf("systemctl not available")
	}

	fallbackSvc := NewConfigEncryptionService(&config.Config{}, rcClient, failingRunner)
	if err := fallbackSvc.RestartDaemon(context.Background()); err != nil {
		t.Fatalf("unexpected error during fallback restart: %v", err)
	}

	// 3. Client nil and runner fails
	nilClientSvc := NewConfigEncryptionService(&config.Config{}, nil, failingRunner)
	if err := nilClientSvc.RestartDaemon(context.Background()); err != nil {
		t.Fatalf("unexpected error when client is nil: %v", err)
	}
}
