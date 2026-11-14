package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

func TestExplorerService_CompareItems_Local(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{BaseDir: tmpDir, LocalPath: tmpDir}
	svc := NewExplorerService(cfg, nil, nil)
	ctx := context.Background()

	contentA := []byte("identical content 12345")
	contentB := []byte("different content 67890")

	f1 := filepath.Join(tmpDir, "file1.txt")
	f2 := filepath.Join(tmpDir, "file2.txt")
	f3 := filepath.Join(tmpDir, "file3.txt")

	_ = os.WriteFile(f1, contentA, 0600)
	_ = os.WriteFile(f2, contentA, 0600)
	_ = os.WriteFile(f3, contentB, 0600)

	// Test 2 identical files
	res, err := svc.CompareItems(ctx, []models.CompareItemRequest{
		{Remote: "local", Path: "file1.txt"},
		{Remote: "local", Path: "file2.txt"},
	}, tmpDir)
	if err != nil || !res.Matches || res.Algorithm != "sha256" {
		t.Fatalf("unexpected res: matches=%v algo=%s err=%v", res.Matches, res.Algorithm, err)
	}

	// Test 3 files (2 identical, 1 different) -> Equivalence clustering
	res3, err := svc.CompareItems(ctx, []models.CompareItemRequest{
		{Remote: "local", Path: "file1.txt"},
		{Remote: "local", Path: "file2.txt"},
		{Remote: "local", Path: "file3.txt"},
	}, tmpDir)
	if err != nil || res3.Matches || len(res3.Clusters) != 2 {
		t.Fatalf("expected 2 clusters, got matches=%v clusters=%+v err=%v", res3.Matches, res3.Clusters, err)
	}
}

func TestExplorerService_CompareItems_CrossBackend_And_Negotiation(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{BaseDir: tmpDir, LocalPath: tmpDir}

	localContent := []byte("cross backend test")
	localFile := filepath.Join(tmpDir, "local_test.txt")
	_ = os.WriteFile(localFile, localContent, 0600)

	remoteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/operations/stat" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"item": map[string]interface{}{
					"Name":    "remote_test.txt",
					"Size":    len(localContent),
					"IsDir":   false,
					"ModTime": "2026-11-14T10:00:00Z",
					"Hashes": map[string]interface{}{
						"MD5": "a831ca73f2b07a48e1bbfe9759d89735",
					},
				},
			})
		}
	}))
	defer remoteServer.Close()

	rcClient := client.NewRcloneClient(remoteServer.URL, "", "")
	svc := NewExplorerService(cfg, rcClient, nil)
	ctx := context.Background()

	res, err := svc.CompareItems(ctx, []models.CompareItemRequest{
		{Remote: "local", Path: "local_test.txt"},
		{Remote: "onedrive", Path: "remote_test.txt"},
	}, tmpDir)
	if err != nil || !res.Matches || res.Algorithm != "md5" {
		t.Fatalf("expected md5 match across backends, got algo=%s matches=%v", res.Algorithm, res.Matches)
	}
}

func TestExplorerService_CompareItems_CryptFallback(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{BaseDir: tmpDir, LocalPath: tmpDir}

	localContent := []byte("encrypted file simulation")
	localFile := filepath.Join(tmpDir, "local.bin")
	_ = os.WriteFile(localFile, localContent, 0600)

	remoteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/operations/stat" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"item": map[string]interface{}{
					"Name": "encrypted.bin", "Size": len(localContent), "IsDir": false,
					"ModTime": "2026-11-14T10:00:00Z", "Hashes": map[string]interface{}{},
				},
			})
		}
	}))
	defer remoteServer.Close()

	rcClient := client.NewRcloneClient(remoteServer.URL, "", "")
	svc := NewExplorerService(cfg, rcClient, nil)
	ctx := context.Background()

	res, err := svc.CompareItems(ctx, []models.CompareItemRequest{
		{Remote: "local", Path: "local.bin"},
		{Remote: "secret_crypt", Path: "encrypted.bin"},
	}, tmpDir)
	if err != nil || res.Algorithm != "none" || !res.Matches {
		t.Fatalf("expected size match with algo none, got algo=%s matches=%v err=%v", res.Algorithm, res.Matches, err)
	}
}

func TestExplorerService_CompareItems_ValidationAndJail(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{BaseDir: tmpDir, LocalPath: tmpDir}
	svc := NewExplorerService(cfg, nil, nil)
	ctx := context.Background()

	res, _ := svc.CompareItems(ctx, []models.CompareItemRequest{
		{Remote: "local", Path: "one.txt"},
	}, tmpDir)
	if res.Status != "error" {
		t.Errorf("expected error for < 2 items, got: %+v", res)
	}

	resTrav, _ := svc.CompareItems(ctx, []models.CompareItemRequest{
		{Remote: "local", Path: "../outside.txt"},
		{Remote: "local", Path: "inside.txt"},
	}, tmpDir)
	if resTrav.Matches || resTrav.Items[0].Error != "Path traversal forbidden" {
		t.Errorf("expected path traversal rejection, got: %+v", resTrav)
	}
}
