package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
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

	var capturedCmd string
	var capturedArgs []string
	var capturedEnv []string

	mockRunner := func(ctx context.Context, name string, env []string, args ...string) (string, error) {
		capturedCmd = name
		capturedArgs = args
		capturedEnv = env
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
	if capturedCmd != "rclone" {
		t.Errorf("expected cmd rclone, got %s", capturedCmd)
	}

	argString := strings.Join(capturedArgs, " ")
	if !strings.Contains(argString, "config encryption set") || !strings.Contains(argString, "--config") {
		t.Errorf("unexpected args: %v", capturedArgs)
	}

	envString := strings.Join(capturedEnv, " ")
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

	var capturedArgs []string
	mockRunner := func(ctx context.Context, name string, env []string, args ...string) (string, error) {
		capturedArgs = args
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
	argString := strings.Join(capturedArgs, " ")
	if !strings.Contains(argString, "config encryption remove") {
		t.Errorf("expected config encryption remove args, got: %v", capturedArgs)
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
