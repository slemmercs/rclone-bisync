package models

// CryptWrapRequest defines the payload for wrapping an existing storage target in an encrypted crypt overlay remote.
type CryptWrapRequest struct {
	Name                    string `json:"name"`                                // Unique name for the crypt remote (e.g., "drive_crypt")
	Remote                  string `json:"remote"`                              // Base remote and subpath to wrap (e.g., "mydrive:encrypted")
	Password                string `json:"password"`                            // Primary encryption passphrase (plaintext, will be obscured)
	Password2               string `json:"password2,omitempty"`                 // Optional salt passphrase (plaintext, will be obscured)
	FilenameEncryption      string `json:"filename_encryption,omitempty"`       // "standard" (default), "obfuscate", or "off"
	DirectoryNameEncryption bool   `json:"directory_name_encryption,omitempty"` // Encrypt directory names (default true)
	CreateProfile           bool   `json:"create_profile,omitempty"`            // Automatically provision a linked sync profile
	LocalPath               string `json:"local_path,omitempty"`                // Local jail path if creating a linked profile
	Schedule                string `json:"schedule,omitempty"`                  // Sync schedule preset ("manual", "15m", "30m", "1h", "4h", "daily")
}

// CryptWrapResponse defines the result returned after creating a crypt remote.
type CryptWrapResponse struct {
	RemoteName  string `json:"remote_name"`
	BaseRemote  string `json:"base_remote"`
	ProfileID   string `json:"profile_id,omitempty"`
	ProfileName string `json:"profile_name,omitempty"`
	Success     bool   `json:"success"`
	Message     string `json:"message"`
}
