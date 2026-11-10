package models

// ProviderExample represents an example option value for a provider configuration field.
type ProviderExample struct {
	Value    string `json:"Value"`
	Help     string `json:"Help,omitempty"`
	Provider string `json:"Provider,omitempty"`
}

// ProviderOption represents a configurable field on an Rclone storage provider.
type ProviderOption struct {
	Name       string            `json:"Name"`
	Help       string            `json:"Help"`
	Type       string            `json:"Type"`
	Default    interface{}       `json:"Default,omitempty"`
	DefaultStr string            `json:"DefaultStr,omitempty"`
	Required   bool              `json:"Required"`
	IsPassword bool              `json:"IsPassword"`
	Sensitive  bool              `json:"Sensitive"`
	Advanced   bool              `json:"Advanced"`
	Exclusive  bool              `json:"Exclusive"`
	Examples   []ProviderExample `json:"Examples,omitempty"`
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

// ProviderCategory represents a grouping of storage providers.
type ProviderCategory string

const (
	CategoryCloud    ProviderCategory = "cloud"
	CategoryObject   ProviderCategory = "object"
	CategoryProtocol ProviderCategory = "protocol"
	CategoryVirtual  ProviderCategory = "virtual"
	CategoryOther    ProviderCategory = "other"
)

// ProviderCatalogItem represents an enriched provider metadata entry.
type ProviderCatalogItem struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Prefix      string           `json:"prefix,omitempty"`
	Category    ProviderCategory `json:"category"`
	Popular     bool             `json:"popular"`
	Options     []ProviderOption `json:"options,omitempty"`
}

// ProviderCatalogResponse represents categorized providers for the UI wizard and MCP tools.
type ProviderCatalogResponse struct {
	Providers  []ProviderCatalogItem `json:"providers"`
	Categories []ProviderCategory    `json:"categories"`
}

// CreateRemoteBridgeRequest defines the payload for creating a remote with an optional profile bridge.
type CreateRemoteBridgeRequest struct {
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Parameters    map[string]interface{} `json:"parameters"`
	Opt           map[string]interface{} `json:"opt,omitempty"`
	CreateProfile bool                   `json:"create_profile,omitempty"`
	LocalPath     string                 `json:"local_path,omitempty"`
	RemoteSubpath string                 `json:"remote_subpath,omitempty"`
	ProfileName   string                 `json:"profile_name,omitempty"`
	Schedule      string                 `json:"schedule,omitempty"`
}

// CreateRemoteResult represents the response after creating a remote (and optional profile).
type CreateRemoteResult struct {
	RemoteName  string `json:"remote_name"`
	RemoteType  string `json:"remote_type"`
	ProfileID   string `json:"profile_id,omitempty"`
	ProfileName string `json:"profile_name,omitempty"`
	NeedsAuth   bool   `json:"needs_auth,omitempty"`
	AuthURL     string `json:"auth_url,omitempty"`
	Status      string `json:"status,omitempty"`
	Success     bool   `json:"success"`
	Message     string `json:"message"`
}
