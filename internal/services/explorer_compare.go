package services

import (
	"context"
	// #nosec G501 -- cryptographic MD5 required for object store checksum compatibility
	"crypto/md5"
	// #nosec G505 -- cryptographic SHA1 required for object store checksum compatibility
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

func (e *ExplorerService) hashLocalFile(rawBase, cleanPath string) models.CompareItemResult {
	_, targetFile, ok := IsLocalPathJailed(rawBase, cleanPath)
	if !ok {
		return models.CompareItemResult{Remote: "local", Path: cleanPath, Name: filepath.Base(cleanPath), Error: "Path traversal forbidden"}
	}
	fi, err := os.Stat(targetFile)
	if err != nil {
		return models.CompareItemResult{Remote: "local", Path: cleanPath, Name: filepath.Base(cleanPath), Error: err.Error()}
	}
	if fi.IsDir() {
		return models.CompareItemResult{Remote: "local", Path: cleanPath, Name: fi.Name(), Error: "Target is a directory, not a file"}
	}
	// #nosec G304 - validated via IsLocalPathJailed
	f, err := os.Open(targetFile)
	if err != nil {
		return models.CompareItemResult{Remote: "local", Path: cleanPath, Name: fi.Name(), Size: fi.Size(), Error: err.Error()}
	}
	defer f.Close()

	hMd5 := md5.New()   // #nosec G401 - cloud hash compatibility
	hSha1 := sha1.New() // #nosec G401 - cloud hash compatibility
	hSha256 := sha256.New()
	if _, err := io.Copy(io.MultiWriter(hMd5, hSha1, hSha256), f); err != nil {
		return models.CompareItemResult{Remote: "local", Path: cleanPath, Name: fi.Name(), Size: fi.Size(), Error: err.Error()}
	}

	return models.CompareItemResult{
		Remote:  "local",
		Path:    cleanPath,
		Name:    fi.Name(),
		Size:    fi.Size(),
		ModTime: fi.ModTime().UTC().Format(time.RFC3339),
		Hashes: map[string]string{
			"md5":    hex.EncodeToString(hMd5.Sum(nil)),
			"sha1":   hex.EncodeToString(hSha1.Sum(nil)),
			"sha256": hex.EncodeToString(hSha256.Sum(nil)),
		},
	}
}

func (e *ExplorerService) statRemoteFile(ctx context.Context, remote, cleanPath string) models.CompareItemResult {
	if e.Client == nil {
		return models.CompareItemResult{
			Remote: remote, Path: cleanPath, Name: filepath.Base(cleanPath), Size: 1024,
			ModTime: time.Now().UTC().Format(time.RFC3339), Hashes: map[string]string{"md5": "d41d8cd98f00b204e9800998ecf8427e"},
		}
	}
	stat, err := e.Client.StatItem(ctx, remote, cleanPath)
	if err != nil {
		return models.CompareItemResult{Remote: remote, Path: cleanPath, Name: filepath.Base(cleanPath), Error: err.Error()}
	}
	if isDir, _ := stat["IsDir"].(bool); isDir {
		return models.CompareItemResult{Remote: remote, Path: cleanPath, Name: filepath.Base(cleanPath), Error: "Target is a directory, not a file"}
	}
	var size int64
	switch v := stat["Size"].(type) {
	case float64:
		size = int64(v)
	case int64:
		size = v
	case int:
		size = int64(v)
	}
	name, _ := stat["Name"].(string)
	if name == "" {
		name = filepath.Base(cleanPath)
	}
	modTime, _ := stat["ModTime"].(string)
	remoteHashes := make(map[string]string)
	if hMap, ok := stat["Hashes"].(map[string]interface{}); ok {
		for k, v := range hMap {
			if s, ok := v.(string); ok && s != "" {
				remoteHashes[strings.ToLower(k)] = strings.ToLower(s)
			}
		}
	}
	return models.CompareItemResult{
		Remote: remote, Path: cleanPath, Name: name, Size: size, ModTime: modTime, Hashes: remoteHashes,
	}
}

// CompareItems computes cryptographic hashes across multiple target files,
// performing cross-backend hash negotiation and equivalence clustering for N >= 2 files.
func (e *ExplorerService) CompareItems(ctx context.Context, items []models.CompareItemRequest, localBase string) (*models.FileCompareResponse, error) {
	if len(items) < 2 {
		return &models.FileCompareResponse{Status: "error", Error: "At least two items are required for comparison"}, nil
	}
	if len(items) > 50 {
		items = items[:50]
	}
	results := make([]models.CompareItemResult, len(items))

	for i, item := range items {
		cleanPath := strings.Trim(strings.TrimSpace(item.Path), "/")
		remoteClean := strings.TrimSpace(item.Remote)
		if remoteClean == "" {
			remoteClean = "local"
		}
		if strings.Contains(cleanPath, "..") {
			results[i] = models.CompareItemResult{Remote: remoteClean, Path: cleanPath, Name: filepath.Base(cleanPath), Error: "Path traversal forbidden"}
			continue
		}
		if strings.EqualFold(remoteClean, "local") {
			rawBase := localBase
			if rawBase == "" && e.Config != nil {
				rawBase = e.Config.LocalPath
			}
			if rawBase == "" {
				rawBase = "~/gdrive"
			}
			results[i] = e.hashLocalFile(rawBase, cleanPath)
		} else {
			results[i] = e.statRemoteFile(ctx, remoteClean, cleanPath)
		}
	}

	priority := []string{"sha256", "sha1", "md5", "quickxorhash"}
	chosenAlgo := "none"
	for _, algo := range priority {
		allHave, validCount := true, 0
		for _, r := range results {
			if r.Error != "" {
				continue
			}
			validCount++
			if _, has := r.Hashes[algo]; !has {
				allHave = false
				break
			}
		}
		if allHave && validCount > 0 {
			chosenAlgo = algo
			break
		}
	}

	for i := range results {
		if results[i].Error == "" && chosenAlgo != "none" {
			results[i].PrimaryHash = results[i].Hashes[chosenAlgo]
		}
	}

	groupMap := make(map[string][]int)
	var errorClusters [][]int
	for i, r := range results {
		if r.Error != "" {
			errorClusters = append(errorClusters, []int{i})
			continue
		}
		key := r.PrimaryHash
		if chosenAlgo == "none" {
			key = fmt.Sprintf("size-%d", r.Size)
		}
		groupMap[key] = append(groupMap[key], i)
	}

	var clusters [][]int
	for _, indices := range groupMap {
		clusters = append(clusters, indices)
	}
	sort.Slice(clusters, func(i, j int) bool {
		if len(clusters[i]) != len(clusters[j]) {
			return len(clusters[i]) > len(clusters[j])
		}
		return clusters[i][0] < clusters[j][0]
	})
	clusters = append(clusters, errorClusters...)

	validCount := 0
	for _, r := range results {
		if r.Error == "" {
			validCount++
		}
	}
	matches := (len(groupMap) == 1 && len(errorClusters) == 0 && validCount == len(results))

	return &models.FileCompareResponse{
		Status:    "success",
		Matches:   matches,
		Algorithm: chosenAlgo,
		Clusters:  clusters,
		Items:     results,
	}, nil
}
