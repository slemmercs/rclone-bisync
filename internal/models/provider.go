package models

// ProviderOption represents a configurable field on an Rclone storage provider.
type ProviderOption struct {
	Name       string      `json:"Name"`
	Help       string      `json:"Help"`
	Type       string      `json:"Type"`
	Default    interface{} `json:"Default,omitempty"`
	DefaultStr string      `json:"DefaultStr,omitempty"`
	Required   bool        `json:"Required"`
	IsPassword bool        `json:"IsPassword"`
	Sensitive  bool        `json:"Sensitive"`
	Advanced   bool        `json:"Advanced"`
	Exclusive  bool        `json:"Exclusive"`
	Examples   []string    `json:"Examples,omitempty"`
}

// Provider represents an Rclone backend storage provider definition.
type Provider struct {
	Name        string           `json:"Name"`
	Description string           `json:"Description"`
	Prefix      string           `json:"Prefix,omitempty"`
	Hide        bool             `json:"Hide"`
	Options     []ProviderOption `json:"Options"`
}

// ProvidersResponse represents the response payload from Rclone RC /config/providers.
type ProvidersResponse struct {
	Providers []Provider `json:"providers"`
}

// CreateRemoteRequest defines the payload for creating a remote via /config/create.
type CreateRemoteRequest struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
	Opt        map[string]interface{} `json:"opt,omitempty"`
}

// DeleteRemoteRequest defines the payload for removing a remote via /config/delete.
type DeleteRemoteRequest struct {
	Name string `json:"name"`
}
