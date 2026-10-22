package services

import (
"bytes"
"context"
"encoding/json"
"fmt"
"io"
"os"
"os/exec"
"path/filepath"
"sort"
"strings"
"sync"
"time"

"github.com/slemmercs/rclone-bisync/internal/config"
"github.com/slemmercs/rclone-bisync/internal/core/client"
"github.com/slemmercs/rclone-bisync/internal/core/lock"
"github.com/slemmercs/rclone-bisync/internal/core/state"
)

var ProviderLabels = map[string]string{
	"drive":               "Google Drive",
	"onedrive":            "Microsoft OneDrive",
	"dropbox":             "Dropbox",
	"s3":                  "Amazon S3",
	"b2":                  "Backblaze B2",
	"box":                 "Box",
	"pcloud":              "pCloud",
	"mega":                "Mega",
	"sftp":                "SFTP",
	"smb":                 "SMB / CIFS",
	"webdav":              "WebDAV",
	"azureblob":           "Azure Blob Storage",
	"google cloud storage": "Google Cloud Storage",
}

// FileCacheItem stores cached file content with mtime/size headers.
type FileCacheItem struct {
	MTime int64
	Size  int64
	Data  interface{}
}

// LiveHealthCache stores daemon health check with timestamp.
type LiveHealthCache struct {
	Time float64
	Data map[string]interface{}
}

// SyncService orchestrates bisync operations, exploration, and telemetry.
type SyncService struct {
	Config          *config.Config
	Client          *client.RcloneClient
	Canary          *CanaryService
	lockMu          sync.Mutex
	fileCache       map[string]FileCacheItem
	liveHealthCache LiveHealthCache
}

// NewSyncService initializes a new SyncService.
func NewSyncService(cfg *config.Config, rcClient *client.RcloneClient) *SyncService {
	if rcClient == nil {
		user := ""
		pass := ""
		if cfg.RCAuthEnabled {
			user = cfg.RCUser
			pass = cfg.RCPass
		}
		rcClient = client.NewRcloneClient(cfg.RCAPIURL, user, pass)
	}
	canarySvc := NewCanaryService(cfg, rcClient)
	return &SyncService{
		Config:    cfg,
		Client:    rcClient,
		Canary:    canarySvc,
		fileCache: make(map[string]FileCacheItem),
	}
}

// IsSyncing checks if a sync is running via flock lockfile.
func (s *SyncService) IsSyncing() bool {
	return lock.IsLocked(s.Config.LockFile)
}

// HandleAction executes system actions (start_sync, stop_sync, force_resync, etc.).
func (s *SyncService) HandleAction(ctx context.Context, action string) (map[string]interface{}, error) {
	switch action {
	case "start_sync":
		if s.IsSyncing() {
			return map[string]interface{}{"status": "error", "msg": "Sync already in progress."}, nil
		}
		if err := s.runBisyncSystemd(nil, "bisync-gdrive-manual", "dashboard"); err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": "success", "msg": "Sync started via systemd (origin: dashboard)"}, nil

	case "stop_sync":
		stopped := 0
		for _, jobID := range s.Client.ListJobs(ctx) {
			if s.Client.StopJob(ctx, jobID) {
				stopped++
			}
		}
		_ = exec.Command("systemctl", "--user", "stop",
"bisync-gdrive.service",
"bisync-gdrive-manual.service",
"bisync-gdrive-resync.service",
"bisync-gdrive-force.service",
"bisync-gdrive-dryrun.service",
).Run()
		return map[string]interface{}{
			"status": "success",
			"msg":    fmt.Sprintf("Sync stopped (%d daemon jobs halted)", stopped),
		}, nil

	case "force_resync":
		if s.IsSyncing() {
			return map[string]interface{}{"status": "error", "msg": "Cannot resync: another sync is actively running."}, nil
		}
		if err := s.runBisyncSystemd([]string{"--resync"}, "bisync-gdrive-resync", "dashboard"); err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": "success", "msg": "Resync initiated via systemd (origin: dashboard)"}, nil

	case "force_sync":
		if s.IsSyncing() {
			return map[string]interface{}{"status": "error", "msg": "Cannot force sync: another sync is actively running."}, nil
		}
		if err := s.runBisyncSystemd([]string{"--force"}, "bisync-gdrive-force", "dashboard"); err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": "success", "msg": "Force sync initiated via systemd (deletions authorized)"}, nil

	case "dry_run":
		if s.IsSyncing() {
			return map[string]interface{}{"status": "error", "msg": "Cannot start dry run: another sync is actively running."}, nil
		}
		if err := s.runBisyncSystemd([]string{"--dry-run"}, "bisync-gdrive-dryrun", "dashboard"); err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": "success", "msg": "Dry-run simulation started via systemd (origin: dashboard)"}, nil

	case "remove_lock":
		if s.IsSyncing() {
			return map[string]interface{}{"status": "error", "msg": "Cannot remove lock: a synchronization run is actively in progress."}, nil
		}
		if _, err := os.Stat(s.Config.LockFile); err == nil {
			_ = os.Remove(s.Config.LockFile)
			return map[string]interface{}{"status": "success", "msg": "Lock file removed"}, nil
		}
		return map[string]interface{}{"status": "info", "msg": "No lock file found"}, nil

	case "clear_rate_limit":
		if state.ClearRateLimitBackoff(s.Config.RateLimitBackoffFile) {
			return map[string]interface{}{"status": "success", "msg": "Rate limit backoff cleared"}, nil
		}
		return map[string]interface{}{"status": "info", "msg": "No active rate limit flag found"}, nil

	default:
		return map[string]interface{}{"status": "error", "msg": fmt.Sprintf("Unknown action: '%s'", action)}, nil
	}
}

func (s *SyncService) runBisyncSystemd(extraArgs []string, unitName, origin string) error {
	_ = exec.Command("systemctl", "--user", "reset-failed", unitName).Run()

	homeDir, _ := os.UserHomeDir()
	binPath := filepath.Join(s.Config.BaseDir, "bin", "rclone-bisync")
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		// Fallback to rclone-bisync in BaseDir or PATH
		if _, err := os.Stat(filepath.Join(s.Config.BaseDir, "rclone-bisync")); err == nil {
			binPath = filepath.Join(s.Config.BaseDir, "rclone-bisync")
		} else if p, err := exec.LookPath("rclone-bisync"); err == nil {
			binPath = p
		}
	}

	execArgs := []string{binPath, "sync", fmt.Sprintf("--origin=%s", origin)}
	if len(extraArgs) > 0 {
		execArgs = append(execArgs, extraArgs...)
	}

	cmdArgs := []string{
		"systemd-run", "--user",
		fmt.Sprintf("--unit=%s", unitName),
		"--service-type=oneshot",
		fmt.Sprintf("--working-directory=%s", s.Config.BaseDir),
		fmt.Sprintf("--property=EnvironmentFile=-%s/.config/dbus-vars", homeDir),
		fmt.Sprintf("--property=EnvironmentFile=%s", filepath.Join(s.Config.BaseDir, "rc-secrets.env")),
		fmt.Sprintf("--property=EnvironmentFile=%s", filepath.Join(s.Config.BaseDir, "bisync.env")),
		"--property=MemoryMax=1G",
		"--property=CPUQuota=80%",
		"--property=NoNewPrivileges=true",
		"--property=PrivateTmp=true",
		"--property=StandardOutput=journal",
		"--property=StandardError=journal",
		"--property=SyslogIdentifier=bisync-gdrive",
	}
	cmdArgs = append(cmdArgs, execArgs...)

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	return cmd.Run()
}

// GetDaemonHealth retrieves daemon health with caching.
func (s *SyncService) GetDaemonHealth(ctx context.Context) map[string]interface{} {
	now := float64(time.Now().Unix())

	s.lockMu.Lock()
	if s.liveHealthCache.Data != nil && (now-s.liveHealthCache.Time) <= 5 {
		data := s.liveHealthCache.Data
		s.lockMu.Unlock()
		return data
	}
	s.lockMu.Unlock()

	healthy, latency, _ := s.Client.CheckHealth(ctx, 1, 0)
	statusStr := "offline"
	if healthy {
		statusStr = "healthy"
	} else {
		latency = 0.0
	}

	health := map[string]interface{}{
		"status":           statusStr,
		"response_time_ms": latency,
		"last_check":       now,
	}

	s.lockMu.Lock()
	s.liveHealthCache.Time = now
	s.liveHealthCache.Data = health
	s.lockMu.Unlock()

	return health
}

// TailFile reads the last N whole lines from a file by seeking from EOF.
func TailFile(filepathStr string, linesCount int) []string {
	f, err := os.Open(filepathStr)
	if err != nil {
		return []string{fmt.Sprintf("Log file target not found at: %s", filepathStr)}
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil || stat.Size() == 0 {
		return []string{}
	}

	position := stat.Size()
	var bufferSize int64 = 8192
	var lines []string
	remainder := []byte{}

	for position > 0 && len(lines) < linesCount {
		toRead := bufferSize
		if position < toRead {
			toRead = position
		}
		position -= toRead

		_, _ = f.Seek(position, io.SeekStart)
		chunk := make([]byte, toRead)
		_, _ = io.ReadFull(f, chunk)
		chunk = append(chunk, remainder...)

		chunkLines := bytes.Split(chunk, []byte("\n"))
		remainder = chunkLines[0]

		for i := len(chunkLines) - 1; i >= 1; i-- {
			lineBytes := bytes.TrimRight(chunkLines[i], "\r")
			if len(lineBytes) > 0 {
				lines = append(lines, string(lineBytes))
				if len(lines) >= linesCount {
					break
				}
			}
		}
	}

	if len(remainder) > 0 && len(lines) < linesCount {
		trimmed := bytes.TrimRight(remainder, "\r")
		if len(trimmed) > 0 {
			lines = append(lines, string(trimmed))
		}
	}

	// Reverse to restore original chronological order
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}

	return lines
}

// TailJSONLog reads log file, unwrapping nested JSON strings if present.
func TailJSONLog(filepathStr string, entriesCount int) []string {
	rawLines := TailFile(filepathStr, entriesCount*2)
	entries := make([]string, 0)

	for _, line := range rawLines {
		lineStr := strings.TrimSpace(line)
		if lineStr == "" {
			continue
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(lineStr), &parsed); err == nil {
			if msg, ok := parsed["msg"].(string); ok && (strings.Contains(msg, `{"`) || strings.Contains(msg, `{'`)) {
				sublines := strings.Split(msg, "\n")
				unwrappedAny := false
				for _, sub := range sublines {
					subTrim := strings.TrimSpace(sub)
					if strings.HasPrefix(subTrim, "{") && strings.HasSuffix(subTrim, "}") {
						var inner map[string]interface{}
						if err := json.Unmarshal([]byte(subTrim), &inner); err == nil {
							b, _ := json.Marshal(inner)
							entries = append(entries, string(b))
							unwrappedAny = true
						}
					}
				}
				if !unwrappedAny {
					b, _ := json.Marshal(parsed)
					entries = append(entries, string(b))
				}
			} else {
				b, _ := json.Marshal(parsed)
				entries = append(entries, string(b))
			}
		} else {
			entries = append(entries, lineStr)
		}
	}

	if len(entries) > entriesCount {
		return entries[len(entries)-entriesCount:]
	}
	return entries
}

// ParsePendingDeletionsFromLog scans daemon log for aborted deletions.
func (s *SyncService) ParsePendingDeletionsFromLog() state.PendingDeletionsData {
	daemonLog := filepath.Join(s.Config.BaseDir, "rclone-daemon.log")
	if _, err := os.Stat(daemonLog); os.IsNotExist(err) {
		return state.PendingDeletionsData{HasDeletions: false, TotalCount: 0, Files: []string{}, Source: "unknown"}
	}

	var deletedFiles []string
	abortMsg := ""
	source := "Google Drive (remote)"
	targetAction := "Will be deleted from local disk (~/gdrive)"

	lines := TailFile(daemonLog, 2000)
	for i := len(lines) - 1; i >= 0; i-- {
		lineStr := strings.TrimSpace(lines[i])
		if lineStr == "" {
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(lineStr), &data); err != nil {
			continue
		}

		rawMsg, _ := data["msg"].(string)
		sublines := strings.Split(rawMsg, "\n")
		for j := len(sublines) - 1; j >= 0; j-- {
			imsg := strings.TrimSpace(sublines[j])
			if strings.HasPrefix(imsg, "{") && strings.HasSuffix(imsg, "}") {
				var inner map[string]interface{}
				if err := json.Unmarshal([]byte(imsg), &inner); err == nil {
					if innerMsg, ok := inner["msg"].(string); ok {
						imsg = innerMsg
					}
				}
			}

			lower := strings.ToLower(imsg)
			if strings.Contains(lower, "too many deletes") {
				abortMsg = imsg
				if strings.Contains(lower, "path2") {
					source = "Local (~/gdrive)"
					targetAction = "Will be deleted from Google Drive"
				} else {
					source = "Google Drive (remote)"
					targetAction = "Will be deleted from local disk (~/gdrive)"
				}
			}

			if strings.Contains(imsg, "File was deleted") {
				parts := strings.SplitN(imsg, "File was deleted", 2)
				if len(parts) == 2 {
					fp := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(parts[1]), "-"))
					if fp != "" {
						found := false
						for _, f := range deletedFiles {
							if f == fp {
								found = true
								break
							}
						}
						if !found {
							deletedFiles = append(deletedFiles, fp)
						}
					}
				}
			}
		}

		if abortMsg != "" && strings.Contains(lineStr, "Synching Path1") {
			break
		}
	}

	if len(deletedFiles) > 0 || abortMsg != "" {
		// Reverse deletedFiles to preserve chronological order
		for i, j := 0, len(deletedFiles)-1; i < j; i, j = i+1, j-1 {
			deletedFiles[i], deletedFiles[j] = deletedFiles[j], deletedFiles[i]
		}
		msg := abortMsg
		if msg == "" {
			msg = fmt.Sprintf("%d files deleted", len(deletedFiles))
		}
		return state.PendingDeletionsData{
			HasDeletions: true,
			TotalCount:   len(deletedFiles),
			Source:       source,
			TargetAction: targetAction,
			Message:      msg,
			Files:        deletedFiles,
		}
	}

	return state.PendingDeletionsData{HasDeletions: false, TotalCount: 0, Files: []string{}}
}

// GetPendingDeletions returns structured pending deletions or falls back to log parsing.
func (s *SyncService) GetPendingDeletions() state.PendingDeletionsData {
	structured, err := state.ReadPendingDeletions(s.Config.PendingDeletionsFile)
	if err == nil && structured != nil {
		return *structured
	}
	return s.ParsePendingDeletionsFromLog()
}

// GetRemotesSummary queries remotes and storage metrics.
func (s *SyncService) GetRemotesSummary(ctx context.Context) (map[string]interface{}, error) {
	remotes, err := s.Client.ListRemotes(ctx)
	if err != nil {
		return nil, err
	}

	remoteData := make([]map[string]interface{}, 0, len(remotes))
	var totalCap, totalUsed, totalFree int64

	activeRemote := strings.TrimRight(s.Config.RemotePath, ":")

	for _, r := range remotes {
		cleanName := strings.TrimRight(r, ":")
		cfg := s.Client.GetRemoteConfig(ctx, cleanName)
		about := s.Client.GetRemoteAbout(ctx, cleanName)

		rtype := "unknown"
		if cfg != nil {
			if t, ok := cfg["type"].(string); ok {
				rtype = t
			}
		}

		typeLabel := strings.ToUpper(rtype)
		if lbl, ok := ProviderLabels[rtype]; ok {
			typeLabel = lbl
		}

		isActive := (cleanName == activeRemote)

		if about != nil {
			if total, ok := about["total"].(float64); ok {
				totalCap += int64(total)
			}
			if used, ok := about["used"].(float64); ok {
				totalUsed += int64(used)
			}
			if free, ok := about["free"].(float64); ok {
				totalFree += int64(free)
			}
		}

		if about == nil {
			about = make(map[string]interface{})
		}

		remoteData = append(remoteData, map[string]interface{}{
"name":             cleanName,
"fs":               fmt.Sprintf("%s:", cleanName),
"type":             rtype,
"type_label":       typeLabel,
"is_active_target": isActive,
"about":            about,
})
	}

	summary := map[string]interface{}{
		"total_capacity_bytes": totalCap,
		"total_used_bytes":     totalUsed,
		"total_free_bytes":     totalFree,
		"provider_count":       len(remoteData),
	}

	var activeRemoteFormatted interface{}
	if activeRemote != "" {
		activeRemoteFormatted = fmt.Sprintf("%s:", activeRemote)
	}

	return map[string]interface{}{
		"remotes":       remoteData,
		"summary":       summary,
		"active_remote": activeRemoteFormatted,
		"count":         len(remoteData),
	}, nil
}

// BrowsePath lists directory items and verifies canary presence, enforcing path jails for local FS.
func (s *SyncService) BrowsePath(ctx context.Context, remote, path string) client.DirectoryListing {
	cleanPath := strings.Trim(strings.TrimSpace(path), "/")
	remoteClean := strings.TrimSpace(remote)

	if strings.ToLower(remoteClean) == "local" || remoteClean == "" {
		rawBase := s.Config.LocalPath
		if rawBase == "" {
			rawBase = "~/gdrive"
		}
		if strings.HasPrefix(rawBase, "~") {
			home, _ := os.UserHomeDir()
			rawBase = filepath.Join(home, strings.TrimPrefix(rawBase, "~"))
		}
		baseDir, err := filepath.Abs(rawBase)
		if err != nil {
			return client.DirectoryListing{Remote: "local", Path: cleanPath, Error: "Invalid base directory"}
		}

		targetDir, err := filepath.Abs(filepath.Join(baseDir, cleanPath))
		if err != nil {
			return client.DirectoryListing{Remote: "local", Path: cleanPath, Error: "Invalid target path"}
		}

		// Security Check: enforce path traversal jail
		rel, err := filepath.Rel(baseDir, targetDir)
		if err != nil || strings.HasPrefix(rel, "..") {
			return client.DirectoryListing{
				Remote: "local",
				Path:   cleanPath,
				Error:  "Directory traversal forbidden",
			}
		}

		fi, err := os.Stat(targetDir)
		if os.IsNotExist(err) {
			return client.DirectoryListing{
				Remote: "local",
				Path:   cleanPath,
				Error:  fmt.Sprintf("Directory does not exist: %s", cleanPath),
			}
		}
		if err != nil || !fi.IsDir() {
			return client.DirectoryListing{
				Remote: "local",
				Path:   cleanPath,
				Error:  "Target is not a directory",
			}
		}

		entries, err := os.ReadDir(targetDir)
		if err != nil {
			return client.DirectoryListing{
				Remote: "local",
				Path:   cleanPath,
				Error:  err.Error(),
			}
		}

		items := make([]map[string]interface{}, 0, len(entries))
		hasCanary := false

		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			isDir := entry.IsDir()
			var size int64 = -1
			if !isDir {
				size = info.Size()
			}

			itemRelPath := entry.Name()
			if cleanPath != "" {
				itemRelPath = fmt.Sprintf("%s/%s", cleanPath, entry.Name())
			}

			mimeType := "application/octet-stream"
			if isDir {
				mimeType = "inode/directory"
			}

			items = append(items, map[string]interface{}{
"Path":     itemRelPath,
"Name":     entry.Name(),
				"Size":     size,
				"MimeType": mimeType,
				"ModTime":  info.ModTime().UTC().Format(time.RFC3339),
				"IsDir":    isDir,
			})

			if entry.Name() == "RCLONE_TEST" {
				hasCanary = true
			}
		}

		// Sort: directories first, then alphabetical
		sort.Slice(items, func(i, j int) bool {
dirI := items[i]["IsDir"].(bool)
dirJ := items[j]["IsDir"].(bool)
if dirI != dirJ {
return dirI
}
return strings.ToLower(items[i]["Name"].(string)) < strings.ToLower(items[j]["Name"].(string))
		})

		return client.DirectoryListing{
			Remote:    "local",
			Path:      cleanPath,
			HasCanary: hasCanary,
			Items:     items,
		}
	}

	return s.Client.ListDirectory(ctx, remoteClean, cleanPath)
}

// CreateCanary provisions an RCLONE_TEST canary file on target local or remote filesystem.
func (s *SyncService) CreateCanary(ctx context.Context, remote, path string) (map[string]interface{}, error) {
	return s.Canary.CreateCanary(ctx, remote, path)
}

// CopyItem transfers a file or directory with path jail enforcement.
func (s *SyncService) CopyItem(ctx context.Context, srcRemote, srcPath, dstRemote, dstPath string, isDir bool) (map[string]interface{}, error) {
	cleanSrcPath := strings.Trim(strings.TrimSpace(srcPath), "/")
	cleanDstPath := strings.Trim(strings.TrimSpace(dstPath), "/")
	srcIsLocal := strings.ToLower(strings.TrimSpace(srcRemote)) == "local" || strings.TrimSpace(srcRemote) == ""
	dstIsLocal := strings.ToLower(strings.TrimSpace(dstRemote)) == "local" || strings.TrimSpace(dstRemote) == ""

	rawBase := s.Config.LocalPath
	if strings.HasPrefix(rawBase, "~") {
		home, _ := os.UserHomeDir()
		rawBase = filepath.Join(home, strings.TrimPrefix(rawBase, "~"))
	}
	baseDir, _ := filepath.Abs(rawBase)

	var srcFs, srcFile, dstFs, dstFile string

	if srcIsLocal {
		targetSrc, _ := filepath.Abs(filepath.Join(baseDir, cleanSrcPath))
		rel, err := filepath.Rel(baseDir, targetSrc)
		if err != nil || strings.HasPrefix(rel, "..") {
			return map[string]interface{}{"status": "error", "error": "Source path traversal forbidden"}, nil
		}

		if _, err := os.Stat(targetSrc); os.IsNotExist(err) {
			return map[string]interface{}{"status": "error", "error": fmt.Sprintf("Source does not exist: %s", cleanSrcPath)}, nil
		}

		if isDir {
			srcFs = targetSrc
			srcFile = ""
		} else {
			srcFs = filepath.Dir(targetSrc)
			srcFile = filepath.Base(targetSrc)
		}
	} else {
		remoteName := strings.TrimRight(srcRemote, ":") + ":"
		if isDir {
			if cleanSrcPath != "" {
				srcFs = fmt.Sprintf("%s%s", remoteName, cleanSrcPath)
			} else {
				srcFs = remoteName
			}
			srcFile = ""
		} else {
			parentDir := filepath.Dir(cleanSrcPath)
			if parentDir == "." || parentDir == "/" {
				srcFs = remoteName
			} else {
				srcFs = fmt.Sprintf("%s%s", remoteName, parentDir)
			}
			srcFile = filepath.Base(cleanSrcPath)
		}
	}

	if dstIsLocal {
		targetDst, _ := filepath.Abs(filepath.Join(baseDir, cleanDstPath))
		rel, err := filepath.Rel(baseDir, targetDst)
		if err != nil || strings.HasPrefix(rel, "..") {
			return map[string]interface{}{"status": "error", "error": "Destination path traversal forbidden"}, nil
		}

		if isDir {
			dstFs = targetDst
			dstFile = ""
		} else {
			fi, err := os.Stat(targetDst)
			if (err == nil && fi.IsDir()) || strings.HasSuffix(dstPath, "/") {
				dstFs = targetDst
				dstFile = srcFile
			} else {
				dstFs = filepath.Dir(targetDst)
				dstFile = filepath.Base(targetDst)
				if dstFile == "" || dstFile == "." {
					dstFile = srcFile
				}
			}
		}
	} else {
		remoteName := strings.TrimRight(dstRemote, ":") + ":"
		if isDir {
			if cleanDstPath != "" {
				dstFs = fmt.Sprintf("%s%s", remoteName, cleanDstPath)
			} else {
				dstFs = remoteName
			}
			dstFile = ""
		} else {
			dstFs = remoteName
			if cleanDstPath != "" {
				dstFs = fmt.Sprintf("%s%s", remoteName, cleanDstPath)
			}
			dstFile = srcFile
		}
	}

	if isDir {
		ok, err := s.Client.CopyDir(ctx, srcFs, dstFs)
		if err != nil || !ok {
			return map[string]interface{}{"status": "error", "error": fmt.Sprintf("copy dir failed: %v", err)}, nil
		}
		return map[string]interface{}{"status": "success", "src": srcFs, "dst": dstFs}, nil
	}

	ok, err := s.Client.CopyFile(ctx, srcFs, srcFile, dstFs, dstFile)
	if err != nil || !ok {
		return map[string]interface{}{"status": "error", "error": fmt.Sprintf("copy file failed: %v", err)}, nil
	}
	return map[string]interface{}{"status": "success", "src": fmt.Sprintf("%s/%s", srcFs, srcFile), "dst": fmt.Sprintf("%s/%s", dstFs, dstFile)}, nil
}

// DeleteItem removes a file or directory, safeguarding RCLONE_TEST canary and enforcing path jail.
func (s *SyncService) DeleteItem(ctx context.Context, remote, path string, isDir bool) (map[string]interface{}, error) {
	cleanPath := strings.Trim(strings.TrimSpace(path), "/")
	if cleanPath == "" {
		return map[string]interface{}{"status": "error", "error": "Root directory cannot be deleted"}, nil
	}

	// Canary protection
	if filepath.Base(cleanPath) == "RCLONE_TEST" {
		return map[string]interface{}{"status": "error", "error": "Canary test file (RCLONE_TEST) cannot be deleted"}, nil
	}

	remoteClean := strings.TrimSpace(remote)
	isLocal := strings.ToLower(remoteClean) == "local" || remoteClean == ""

	if isLocal {
		rawBase := s.Config.LocalPath
		if strings.HasPrefix(rawBase, "~") {
			home, _ := os.UserHomeDir()
			rawBase = filepath.Join(home, strings.TrimPrefix(rawBase, "~"))
		}
		baseDir, _ := filepath.Abs(rawBase)
		target, _ := filepath.Abs(filepath.Join(baseDir, cleanPath))

		rel, err := filepath.Rel(baseDir, target)
		if err != nil || strings.HasPrefix(rel, "..") || target == baseDir {
			return map[string]interface{}{"status": "error", "error": "Path traversal or root deletion forbidden"}, nil
		}

		if _, err := os.Stat(target); os.IsNotExist(err) {
			return map[string]interface{}{"status": "error", "error": fmt.Sprintf("Target does not exist: %s", cleanPath)}, nil
		}

		if isDir {
			if err := os.RemoveAll(target); err != nil {
				return map[string]interface{}{"status": "error", "error": fmt.Sprintf("Failed to delete local directory: %v", err)}, nil
			}
		} else {
			if err := os.Remove(target); err != nil {
				return map[string]interface{}{"status": "error", "error": fmt.Sprintf("Failed to delete local file: %v", err)}, nil
			}
		}

		return map[string]interface{}{
			"status": "success",
			"msg":    fmt.Sprintf("Deleted %s", cleanPath),
			"remote": "local",
			"path":   cleanPath,
		}, nil
	}

	remoteName := strings.TrimRight(remoteClean, ":") + ":"
	if isDir {
		ok, err := s.Client.PurgeDir(ctx, remoteName, cleanPath)
		if err != nil || !ok {
			return map[string]interface{}{"status": "error", "error": fmt.Sprintf("Purge failed: %v", err)}, nil
		}
		return map[string]interface{}{"status": "success", "fs": remoteName, "remote": cleanPath}, nil
	}

	ok, err := s.Client.DeleteFile(ctx, remoteName, cleanPath)
	if err != nil || !ok {
		return map[string]interface{}{"status": "error", "error": fmt.Sprintf("Delete file failed: %v", err)}, nil
	}
	return map[string]interface{}{"status": "success", "fs": remoteName, "remote": cleanPath}, nil
}
