package models

// ConfigPathsResponse represents filesystem paths returned by /config/paths.
type ConfigPathsResponse struct {
	Config string `json:"config"`
	Cache  string `json:"cache"`
	Temp   string `json:"temp"`
}

// EncryptionStatusResponse reports whether rclone.conf is password-encrypted.
type EncryptionStatusResponse struct {
	IsEncrypted         bool   `json:"is_encrypted"`
	ConfigFile          string `json:"config_file"`
	HasDaemonPassphrase bool   `json:"has_daemon_passphrase"`
	SecretsFile         string `json:"secrets_file,omitempty"`
	Warning             string `json:"warning,omitempty"`
	Error               string `json:"error,omitempty"`
}

// SaveDaemonPassphraseRequest contains a passphrase to persist in rc-secrets.env.
type SaveDaemonPassphraseRequest struct {
	Password string `json:"password"`
}

// SetEncryptionRequest contains the new password and optional old password for rclone.conf.
type SetEncryptionRequest struct {
	Password    string `json:"password"`
	OldPassword string `json:"old_password,omitempty"`
}

// RemoveEncryptionRequest contains the current password to clear encryption.
type RemoveEncryptionRequest struct {
	Password string `json:"password"`
}

// EncryptionActionResponse reports the result of setting or removing password encryption.
type EncryptionActionResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}
