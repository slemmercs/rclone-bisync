package server

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
)

func (s *Server) routes() {
	s.Mux.HandleFunc("GET /api/health", s.handleDaemonHealth)
	s.Mux.HandleFunc("GET /api/diagnostics", s.handleDiagnostics)
	s.Mux.HandleFunc("GET /api/mcp/status", s.handleMCPStatus)
	s.Mux.HandleFunc("GET /api/logs/bisync/raw", s.handleRawBisyncLog)
	s.Mux.HandleFunc("GET /api/logs/bisync", s.handleBisyncLogs)
	s.Mux.HandleFunc("GET /api/logs/daemon/raw", s.handleRawDaemonLog)
	s.Mux.HandleFunc("GET /api/logs/daemon", s.handleDaemonLogs)
	s.Mux.HandleFunc("GET /api/telemetry", s.handleTelemetry)
	s.Mux.HandleFunc("POST /api/action", s.handleAction)
	s.Mux.HandleFunc("GET /api/deletions/pending", s.handlePendingDeletions)
	s.Mux.HandleFunc("GET /api/remotes", s.handleRemotes)
	s.Mux.HandleFunc("GET /api/remotes/providers", s.handleRemoteProviders)
	s.Mux.HandleFunc("POST /api/remotes/create", s.handleCreateRemote)
	s.Mux.HandleFunc("DELETE /api/remotes/{name}", s.handleDeleteRemote)
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
	s.registerOperationsRoutes()

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
	s.Mux.HandleFunc("GET /api/config/encryption", s.handleGetConfigEncryption)
	s.Mux.HandleFunc("POST /api/config/encryption/set", s.handleSetConfigEncryption)
	s.Mux.HandleFunc("POST /api/config/encryption/remove", s.handleRemoveConfigEncryption)
	s.Mux.HandleFunc("POST /api/config/encryption/save-passphrase", s.handleSaveDaemonPassphrase)

	// Core Daemon Controls Endpoints
	s.Mux.HandleFunc("GET /api/core/bwlimit", s.handleGetBandwidthLimit)
	s.Mux.HandleFunc("POST /api/core/bwlimit", s.handleSetBandwidthLimit)
	s.Mux.HandleFunc("GET /api/core/stats", s.handleGetCoreStats)

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
