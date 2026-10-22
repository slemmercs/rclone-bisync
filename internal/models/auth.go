package models

// Auth status constants defining the authentication lifecycle states.
const (
	AuthStatusConnected             = "connected"
	AuthStatusMigrationRequired     = "migration_required"
	AuthStatusAuthorizationRequired = "authorization_required"
	AuthStatusConfigurationRequired = "configuration_required"
	AuthStatusError                 = "error"
)

// RemoteAuthInfo represents the authentication state and metadata for a single remote.
type RemoteAuthInfo struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	TypeLabel       string `json:"type_label"`
	IsOAuth         bool   `json:"is_oauth"`
	Status          string `json:"status"`
	StatusMessage   string `json:"status_message"`
	ClientID        string `json:"client_id,omitempty"`
	HasClientSecret bool   `json:"has_client_secret"`
	HasToken        bool   `json:"has_token"`
	TokenExpiry     string `json:"token_expiry,omitempty"`
	IsActiveTarget  bool   `json:"is_active_target"`
	LastChecked     string `json:"last_checked"`
}

// AuthHealthSummary provides aggregate statistics on remote authentication health.
type AuthHealthSummary struct {
	TotalRemotes          int `json:"total_remotes"`
	OAuthRemotes          int `json:"oauth_remotes"`
	Connected             int `json:"connected"`
	MigrationRequired     int `json:"migration_required"`
	AuthorizationRequired int `json:"authorization_required"`
	ConfigurationRequired int `json:"configuration_required"`
	Errors                int `json:"errors"`
}

// AuthStatusResponse is the API payload returned by GET /api/remotes/auth.
type AuthStatusResponse struct {
	Summary AuthHealthSummary `json:"summary"`
	Remotes []RemoteAuthInfo  `json:"remotes"`
}

// UpdateCredentialsRequest is the API payload for updating OAuth credentials.
type UpdateCredentialsRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// ReconnectResponse is the API payload returned when initiating a remote reconnect.
type ReconnectResponse struct {
	Status  string `json:"status"`
	Remote  string `json:"remote"`
	AuthURL string `json:"auth_url,omitempty"`
	Msg     string `json:"msg,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ValidateResponse is the API payload returned after validating remote connectivity.
type ValidateResponse struct {
	Status string `json:"status"`
	Remote string `json:"remote"`
	Valid  bool   `json:"valid"`
	Msg    string `json:"msg,omitempty"`
	Error  string `json:"error,omitempty"`
}

// OAuthAuditEvent records lifecycle events in the OAuth manager.
type OAuthAuditEvent struct {
	Event     string `json:"event"`
	Remote    string `json:"remote"`
	Status    string `json:"status"`
	Details   string `json:"details,omitempty"`
	Timestamp string `json:"timestamp"`
}

// AuditEventsResponse is the API payload for GET /api/remotes/auth/audit.
type AuditEventsResponse struct {
	Events []OAuthAuditEvent `json:"events"`
	Count  int               `json:"count"`
}
