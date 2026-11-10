package server

import (
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/slemmercs/rclone-bisync/internal/core/logger"
	"github.com/slemmercs/rclone-bisync/internal/core/security"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
	"github.com/slemmercs/rclone-bisync/internal/remoteauth"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

type Server struct {
	Container               *app.Container
	Config                  *config.Config
	SyncService             *services.SyncService
	TelemetryService        *services.TelemetryService
	CanaryService           *services.CanaryService
	DeletionService         *services.DeletionService
	ExplorerService         *services.ExplorerService
	ReviewService           *services.ReviewService
	AuditService            *services.AuditService
	LogService              *services.LogService
	CoreService             *services.CoreService
	ConfigEncryptionService *services.ConfigEncryptionService
	AuthService             *remoteauth.RemoteAuthService
	ProfileService          *profiles.Service
	UploadDock              *services.UploadDock
	RemoteWizardService     *services.RemoteWizardService
	FrontendFS              fs.FS
	Mux                     *http.ServeMux
	Version                 string
	StartTime               time.Time
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
		Container:               container,
		Config:                  container.Config,
		SyncService:             container.SyncService,
		TelemetryService:        container.TelemetryService,
		CanaryService:           container.CanaryService,
		DeletionService:         container.DeletionService,
		ExplorerService:         container.ExplorerService,
		ReviewService:           container.ReviewService,
		AuditService:            container.AuditService,
		LogService:              container.LogService,
		CoreService:             container.CoreService,
		ConfigEncryptionService: container.ConfigEncryptionService,
		AuthService:             container.AuthService,
		ProfileService:          container.ProfileService,
		UploadDock:              container.UploadDock,
		RemoteWizardService:     container.RemoteWizardService,
		FrontendFS:              frontendFS,
		Mux:                     http.NewServeMux(),
		StartTime:               time.Now(),
	}
	if s.UploadDock == nil {
		s.UploadDock = services.NewUploadDock(s.ProfileService)
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

func (s *Server) getLogger() *logger.Logger {
	if s.AuditService != nil && s.AuditService.Logger != nil {
		return s.AuditService.Logger
	}
	if s.Container != nil && s.Container.Logger != nil {
		return s.Container.Logger
	}
	return nil
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
		lvl := slog.LevelDebug
		if sw.Status() >= 400 {
			lvl = slog.LevelWarn
		}
		slog.Log(r.Context(), lvl, "http_access",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.Status(),
			"duration_ms", durMs,
			"remote_addr", r.RemoteAddr,
			"actor", actor,
		)
	}()

	isMutating := r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete || r.Method == http.MethodPatch

	// Optional API key authorization for mutating endpoints when DASHBOARD_API_KEY is configured
	hasValidAPIKey := s.validateAPIKey(r)
	if isMutating && strings.HasPrefix(r.URL.Path, "/api/") {
		if s.Config != nil && s.Config.DashboardAPIKey != "" && !hasValidAPIKey {
			writeJSONError(sw, http.StatusUnauthorized, "Unauthorized: Invalid or missing API key.")
			return
		}

		// CSRF & Security middleware (required for non-API-key requests)
		if !hasValidAPIKey {
			if r.Header.Get("X-Dashboard-Request") != "1" {
				writeJSONError(sw, http.StatusForbidden, "Security policy violation: Missing required X-Dashboard-Request header.")
				return
			}

			origin := r.Header.Get("Origin")
			if origin != "" && !isAllowedOrigin(origin) {
				writeJSONError(sw, http.StatusForbidden, fmt.Sprintf("Security policy violation: Untrusted request origin '%s'.", origin))
				return
			}

			referer := r.Header.Get("Referer")
			if referer != "" && !isAllowedOrigin(referer) {
				writeJSONError(sw, http.StatusForbidden, "Security policy violation: Untrusted request referer.")
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

func writeJSONError(w http.ResponseWriter, statusCode int, msg string) {
	writeJSON(w, statusCode, models.NewErrorResponse(msg))
}
