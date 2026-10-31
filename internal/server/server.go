package server

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/app"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/security"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
	"github.com/slemmercs/rclone-bisync/internal/remoteauth"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

type Server struct {
	Container        *app.Container
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

// NewServerWithContainer creates a Server instance bound to an initialized Container.
func NewServerWithContainer(container *app.Container, frontendFS fs.FS) *Server {
	if container == nil {
		container, _ = app.NewContainer(&config.Config{})
	}
	s := &Server{
		Container:        container,
		Config:           container.Config,
		SyncService:      container.SyncService,
		TelemetryService: container.TelemetryService,
		CanaryService:    container.CanaryService,
		DeletionService:  container.DeletionService,
		ExplorerService:  container.ExplorerService,
		ReviewService:    container.ReviewService,
		AuditService:     container.AuditService,
		LogService:       container.LogService,
		AuthService:      container.AuthService,
		ProfileService:   container.ProfileService,
		FrontendFS:       frontendFS,
		Mux:              http.NewServeMux(),
	}
	s.routes()
	return s
}

// NewServer initializes a new Server using the central DI container.
func NewServer(cfg *config.Config, syncService *services.SyncService, frontendFS fs.FS) *Server {
	var opts []app.Option
	if syncService != nil {
		opts = append(opts, app.WithSyncService(syncService))
		if syncService.Client != nil {
			opts = append(opts, app.WithClient(syncService.Client))
		}
	}
	container, _ := app.NewContainer(cfg, opts...)
	if container != nil && container.ProfileScheduler != nil {
		_ = container.ProfileScheduler.Start(context.Background())
	}
	return NewServerWithContainer(container, frontendFS)
}

type statusRecordingWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusRecordingWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusRecordingWriter) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

func (w *statusRecordingWriter) Status() int {
	if w.statusCode == 0 {
		return http.StatusOK
	}
	return w.statusCode
}

func (w *statusRecordingWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (s *Server) resolveActor(r *http.Request) string {
	if user := strings.TrimSpace(r.Header.Get("X-Forwarded-User")); user != "" {
		return user
	}
	if user := strings.TrimSpace(r.Header.Get("Remote-User")); user != "" {
		return user
	}
	if email := strings.TrimSpace(r.Header.Get("X-Forwarded-Email")); email != "" {
		return email
	}
	return "USER"
}

func (s *Server) validateAPIKey(r *http.Request) bool {
	if s.Config == nil || s.Config.DashboardAPIKey == "" {
		return false
	}
	expectedKey := s.Config.DashboardAPIKey
	if apiKey := strings.TrimSpace(r.Header.Get("X-API-Key")); apiKey != "" {
		return apiKey == expectedKey
	}
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		token := strings.TrimSpace(authHeader[7:])
		return token == expectedKey
	}
	return false
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	actor := s.resolveActor(r)
	r = r.WithContext(security.ContextWithActor(r.Context(), actor))

	sw := &statusRecordingWriter{ResponseWriter: w, statusCode: http.StatusOK}

	defer func() {
		durMs := float64(time.Since(start).Microseconds()) / 1000.0
		isPolling := r.Method == http.MethodGet && (r.URL.Path == "/api/telemetry" ||
			r.URL.Path == "/api/health" ||
			r.URL.Path == "/api/profiles/summary" ||
			strings.HasPrefix(r.URL.Path, "/api/logs/"))

		if isPolling && sw.Status() < 400 {
			slog.Debug("http_access",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.Status(),
				"duration_ms", durMs,
				"remote_addr", r.RemoteAddr,
				"actor", actor,
			)
		} else {
			slog.Info("http_access",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.Status(),
				"duration_ms", durMs,
				"remote_addr", r.RemoteAddr,
				"actor", actor,
			)
		}
	}()

	isMutating := r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete || r.Method == http.MethodPatch

	// Optional API key authorization for mutating endpoints when DASHBOARD_API_KEY is configured
	hasValidAPIKey := s.validateAPIKey(r)
	if isMutating && strings.HasPrefix(r.URL.Path, "/api/") {
		if s.Config != nil && s.Config.DashboardAPIKey != "" && !hasValidAPIKey {
			writeJSON(sw, http.StatusUnauthorized, models.ErrorResponse{
				Status: "error",
				Msg:    "Unauthorized: Invalid or missing API key.",
			})
			return
		}

		// CSRF & Security middleware (required for non-API-key requests)
		if !hasValidAPIKey {
			if r.Header.Get("X-Dashboard-Request") != "1" {
				writeJSON(sw, http.StatusForbidden, models.ErrorResponse{
					Status: "error",
					Msg:    "Security policy violation: Missing required X-Dashboard-Request header.",
				})
				return
			}

			origin := r.Header.Get("Origin")
			if origin != "" && !isAllowedOrigin(origin) {
				writeJSON(sw, http.StatusForbidden, models.ErrorResponse{
					Status: "error",
					Msg:    fmt.Sprintf("Security policy violation: Untrusted request origin '%s'.", origin),
				})
				return
			}

			referer := r.Header.Get("Referer")
			if referer != "" && !isAllowedOrigin(referer) {
				writeJSON(sw, http.StatusForbidden, models.ErrorResponse{
					Status: "error",
					Msg:    "Security policy violation: Untrusted request referer.",
				})
				return
			}
		}
	}

	s.Mux.ServeHTTP(sw, r)
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
	s.Mux.HandleFunc("GET /api/logs/bisync", s.handleBisyncLogs)
	s.Mux.HandleFunc("GET /api/logs/daemon/raw", s.handleRawDaemonLog)
	s.Mux.HandleFunc("GET /api/logs/daemon", s.handleDaemonLogs)
	s.Mux.HandleFunc("GET /api/telemetry", s.handleTelemetry)
	s.Mux.HandleFunc("POST /api/action", s.handleAction)
	s.Mux.HandleFunc("GET /api/deletions/pending", s.handlePendingDeletions)
	s.Mux.HandleFunc("GET /api/remotes", s.handleRemotes)
	s.Mux.HandleFunc("GET /api/remotes/auth", s.handleRemotesAuth)
	s.Mux.HandleFunc("POST /api/remotes/auth/check", s.handleRemotesAuthCheck)
	s.Mux.HandleFunc("POST /api/remotes/dedupe", s.handleRemoteDedupe)
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
	s.Mux.HandleFunc("POST /api/audit/reset", s.handleAuditReset)

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
	// Global Fleet Settings Endpoints
	s.Mux.HandleFunc("GET /api/config", s.handleGetConfig)
	s.Mux.HandleFunc("PUT /api/config", s.handleUpdateConfig)

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

			patterns := []string{
				"index.html",
				"partials/*.html",
				"partials/modals/*.html",
				"partials/components/*.html",
				"partials/patterns/*.html",
				"partials/layouts/*.html",
			}
			var tmplFiles []string
			for _, pat := range patterns {
				if matches, err := fs.Glob(staticSubFS, pat); err == nil && len(matches) > 0 {
					tmplFiles = append(tmplFiles, matches...)
				}
			}

			var indexTmpl *template.Template
			var tmplErr error
			if len(tmplFiles) > 0 {
				indexTmpl, tmplErr = template.ParseFS(staticSubFS, tmplFiles...)
			} else {
				tmplErr = fmt.Errorf("no template files found")
			}

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
