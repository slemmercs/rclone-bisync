package server

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/core/security"
	"github.com/slemmercs/rclone-bisync/internal/engine/runner"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
	"github.com/slemmercs/rclone-bisync/internal/remoteauth"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

type Server struct {
	Config           *config.Config
	SyncService      *services.SyncService
	TelemetryService *services.TelemetryService
	CanaryService    *services.CanaryService
	DeletionService  *services.DeletionService
	ExplorerService  *services.ExplorerService
	ReviewService    *services.ReviewService
	AuditService     *services.AuditService
	LogService       *services.LogService
	AuthService      *remoteauth.RemoteAuthService
	ProfileService   *profiles.Service
	FrontendFS       fs.FS
	Mux              *http.ServeMux
	Version          string
}

var versionRegex = regexp.MustCompile(`^v?(\d+)\.(\d+)`)

// FormatMajorMinor normalizes a version string into vMajor.Minor format (e.g. "v1.0.0-beta.1" -> "v1.0", "dev" -> "v1.0").
func FormatMajorMinor(raw string) string {
	matches := versionRegex.FindStringSubmatch(strings.TrimSpace(raw))
	if len(matches) >= 3 {
		return "v" + matches[1] + "." + matches[2]
	}
	return "v1.0"
}

// FormatMajorMinorVersion returns the configured version in vMajor.Minor format.
func (s *Server) FormatMajorMinorVersion() string {
	return FormatMajorMinor(s.Version)
}

func NewServer(cfg *config.Config, syncService *services.SyncService, frontendFS fs.FS) *Server {
	var rcClient *client.RcloneClient
	if syncService != nil {
		rcClient = syncService.Client
	}
	var telemetrySvc *services.TelemetryService
	var canarySvc *services.CanaryService
	var deletionSvc *services.DeletionService
	var explorerSvc *services.ExplorerService
	var reviewSvc *services.ReviewService
	var auditSvc *services.AuditService

	if syncService != nil {
		telemetrySvc = syncService.Telemetry
		canarySvc = syncService.Canary
		deletionSvc = syncService.Deletion
		explorerSvc = syncService.Explorer
		reviewSvc = syncService.Review
		auditSvc = syncService.Audit
	}
	if telemetrySvc == nil {
		telemetrySvc = services.NewTelemetryService(cfg, rcClient)
	}
	if canarySvc == nil {
		canarySvc = services.NewCanaryService(cfg, rcClient)
	}
	if deletionSvc == nil {
		deletionSvc = services.NewDeletionService(cfg)
	}
	if explorerSvc == nil {
		explorerSvc = services.NewExplorerService(cfg, rcClient, canarySvc)
	}
	if reviewSvc == nil {
		reviewSvc = services.NewReviewService(cfg, rcClient)
	}

	var profSvc *profiles.Service
	if cfg != nil && cfg.ProfilesFile != "" {
		profStore, err := profiles.NewStore(cfg.ProfilesFile, cfg.ProfilesHistoryFile)
		if err == nil {
			_, _ = profStore.BootstrapDefault(cfg.LocalPath, cfg.RemotePath)
			executor := func(ctx context.Context, p *profiles.Profile, opts profiles.SyncOptions) error {
				if err := os.MkdirAll(p.LocalPath, 0755); err != nil {
					return fmt.Errorf("failed to create local path %s: %w", p.LocalPath, err)
				}
				profCfg := *cfg
				profCfg.ProfileID = p.ID
				profCfg.RemotePath = p.FullRemoteTarget()
				profCfg.LocalPath = p.LocalPath
				profCfg.LockFile = p.LockFilePath()
				syncRunner := runner.NewSyncRunner(&profCfg)
				origin := "scheduler"
				if opts.Force || opts.Resync || opts.DryRun {
					origin = "dashboard"
				}
				success := syncRunner.Run(ctx, opts.Resync, opts.Force, opts.DryRun, origin, "", false)
				if !success {
					return fmt.Errorf("sync failed for profile %s", p.ID)
				}
				return nil
			}
			sched := profiles.NewScheduler(profStore, executor)
			profSvc = profiles.NewService(profStore, rcClient, sched)
			_ = sched.Start(context.Background())
		}
	}

	s := &Server{
		Config:           cfg,
		SyncService:      syncService,
		TelemetryService: telemetrySvc,
		CanaryService:    canarySvc,
		DeletionService:  deletionSvc,
		ExplorerService:  explorerSvc,
		ReviewService:    reviewSvc,
		AuditService:     auditSvc,
		LogService:       services.NewLogService(),
		AuthService:      remoteauth.NewRemoteAuthService(cfg, rcClient, nil),
		ProfileService:   profSvc,
		FrontendFS:       frontendFS,
		Mux:              http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CSRF & Security middleware
	if (r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete || r.Method == http.MethodPatch) && strings.HasPrefix(r.URL.Path, "/api/") {
		if r.Header.Get("X-Dashboard-Request") != "1" {
			writeJSON(w, http.StatusForbidden, models.ErrorResponse{
				Status: "error",
				Msg:    "Security policy violation: Missing required X-Dashboard-Request header.",
			})
			return
		}

		origin := r.Header.Get("Origin")
		if origin != "" && !isAllowedOrigin(origin) {
			writeJSON(w, http.StatusForbidden, models.ErrorResponse{
				Status: "error",
				Msg:    fmt.Sprintf("Security policy violation: Untrusted request origin '%s'.", origin),
			})
			return
		}

		referer := r.Header.Get("Referer")
		if referer != "" && !isAllowedOrigin(referer) {
			writeJSON(w, http.StatusForbidden, models.ErrorResponse{
				Status: "error",
				Msg:    "Security policy violation: Untrusted request referer.",
			})
			return
		}
	}

	s.Mux.ServeHTTP(w, r)
}

func isAllowedOrigin(rawURL string) bool {
	if rawURL == "" {
		return true
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	hostname := u.Hostname()
	if hostname == "" {
		return false
	}
	if hostname == "localhost" || strings.HasSuffix(hostname, ".local") || strings.HasSuffix(hostname, ".lan") {
		return true
	}

	ip := net.ParseIP(hostname)
	if ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
			return true
		}
		// Carrier-grade NAT (100.64.0.0/10) for Tailscale homelab mesh networks
		if ip4 := ip.To4(); ip4 != nil {
			if ip4[0] == 100 && (ip4[1]&0xc0) == 64 {
				return true
			}
		}
	}

	return false
}

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	redacted := security.RedactSecrets(data)
	_ = json.NewEncoder(w).Encode(redacted)
}

func (s *Server) routes() {
	s.Mux.HandleFunc("GET /api/health", s.handleDaemonHealth)
	s.Mux.HandleFunc("GET /api/logs/bisync/raw", s.handleRawBisyncLog)
	s.Mux.HandleFunc("GET /api/logs/daemon/raw", s.handleRawDaemonLog)
	s.Mux.HandleFunc("GET /api/logs/daemon", s.handleDaemonLogs)
	s.Mux.HandleFunc("GET /api/telemetry", s.handleTelemetry)
	s.Mux.HandleFunc("POST /api/action", s.handleAction)
	s.Mux.HandleFunc("GET /api/deletions/pending", s.handlePendingDeletions)
	s.Mux.HandleFunc("GET /api/remotes", s.handleRemotes)
	s.Mux.HandleFunc("GET /api/remotes/auth", s.handleRemotesAuth)
	s.Mux.HandleFunc("POST /api/remotes/auth/check", s.handleRemotesAuthCheck)
	s.Mux.HandleFunc("GET /api/remotes/auth/audit", s.handleRemotesAuthAudit)
	s.Mux.HandleFunc("POST /api/remotes/auth/oauth-stop", s.handleOAuthStop)
	s.Mux.HandleFunc("POST /api/remotes/{name}/reconnect", s.handleRemoteReconnect)
	s.Mux.HandleFunc("POST /api/remotes/{name}/reconnect/cancel", s.handleRemoteReconnectCancel)
	s.Mux.HandleFunc("POST /api/remotes/{name}/credentials", s.handleRemoteCredentials)
	s.Mux.HandleFunc("POST /api/remotes/{name}/validate", s.handleRemoteValidate)
	s.Mux.HandleFunc("POST /api/browse", s.handleBrowse)
	s.Mux.HandleFunc("POST /api/browse/canary", s.handleProvisionCanary)
	s.Mux.HandleFunc("POST /api/browse/copy", s.handleCopyItem)
	s.Mux.HandleFunc("POST /api/browse/delete", s.handleDeleteItem)

	// Review & Conflict Resolution Endpoints
	s.Mux.HandleFunc("GET /api/review/session", s.handleReviewSession)
	s.Mux.HandleFunc("POST /api/review/action", s.handleReviewAction)
	s.Mux.HandleFunc("POST /api/review/action-batch", s.handleReviewActionBatch)
	s.Mux.HandleFunc("POST /api/review/execute", s.handleReviewExecute)
	s.Mux.HandleFunc("POST /api/review/rollback", s.handleReviewRollback)
	s.Mux.HandleFunc("POST /api/review/dismiss", s.handleReviewDismiss)

	// Immutable Audit Ledger Endpoints
	s.Mux.HandleFunc("GET /api/audit/records", s.handleAuditRecords)
	s.Mux.HandleFunc("GET /api/audit/verify", s.handleAuditVerify)
	s.Mux.HandleFunc("GET /api/audit/export", s.handleAuditExport)

	// Profile Manager Endpoints
	s.Mux.HandleFunc("GET /api/profiles", s.handleListProfiles)
	s.Mux.HandleFunc("POST /api/profiles", s.handleCreateProfile)
	s.Mux.HandleFunc("GET /api/profiles/{id}", s.handleGetProfile)
	s.Mux.HandleFunc("PUT /api/profiles/{id}", s.handleUpdateProfile)
	s.Mux.HandleFunc("DELETE /api/profiles/{id}", s.handleDeleteProfile)
	s.Mux.HandleFunc("POST /api/profiles/{id}/clone", s.handleCloneProfile)
	s.Mux.HandleFunc("POST /api/profiles/{id}/rollback", s.handleRollbackProfile)
	s.Mux.HandleFunc("POST /api/profiles/{id}/sync", s.handleProfileSync)
	s.Mux.HandleFunc("GET /api/profiles/{id}/history", s.handleProfileHistory)

	// Static assets & SPA
	if s.FrontendFS != nil {
		staticSubFS, err := fs.Sub(s.FrontendFS, "dashboard/frontend")
		if err == nil {
			fileServer := http.FileServer(http.FS(staticSubFS))
			s.Mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

			s.Mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
				favicon, err := fs.ReadFile(staticSubFS, "favicon.svg")
				if err != nil {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "image/svg+xml")
				_, _ = w.Write(favicon)
			})

			indexTmpl, tmplErr := template.ParseFS(staticSubFS,
				"index.html",
				"partials/*.html",
				"partials/modals/*.html",
			)

			s.Mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
				if tmplErr != nil {
					// Fallback if templates couldn't be parsed (e.g. empty test FS)
					indexHTML, err := fs.ReadFile(staticSubFS, "index.html")
					if err != nil {
						http.Error(w, "index.html not found", http.StatusNotFound)
						return
					}
					rendered := strings.ReplaceAll(string(indexHTML), "{{ .Version }}", s.FormatMajorMinorVersion())
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					_, _ = w.Write([]byte(rendered))
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				tmplData := map[string]any{
					"Version": s.FormatMajorMinorVersion(),
				}
				if err := indexTmpl.Execute(w, tmplData); err != nil {
					http.Error(w, fmt.Sprintf("failed to render template: %v", err), http.StatusInternalServerError)
				}
			})
		}
	}
}
