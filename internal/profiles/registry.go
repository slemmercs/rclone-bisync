package profiles

import (
	_ "embed"
	"encoding/json"
	"strings"
	"sync"
)

//go:embed provider_options.json
var providerOptionsJSON []byte

// ProviderOptionDef represents declarative metadata for a target provider option.
type ProviderOptionDef struct {
	Provider    string   `json:"provider"`
	Key         string   `json:"key"`
	Flag        string   `json:"flag"`
	Type        string   `json:"type"` // "bool", "text", "select"
	Label       string   `json:"label"`
	Default     string   `json:"default,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Options     []string `json:"options,omitempty"`
	Desc        string   `json:"desc,omitempty"`
}

var (
	registryOnce    sync.Once
	providerOptions []ProviderOptionDef
	providerMap     map[string][]ProviderOptionDef
	knownProviders  map[string]bool
)

func initRegistry() {
	registryOnce.Do(func() {
		if err := json.Unmarshal(providerOptionsJSON, &providerOptions); err != nil {
			panic("failed to unmarshal embedded provider_options.json: " + err.Error())
		}
		providerMap = make(map[string][]ProviderOptionDef)
		knownProviders = make(map[string]bool)
		for _, opt := range providerOptions {
			prov := strings.ToLower(strings.TrimSpace(opt.Provider))
			providerMap[prov] = append(providerMap[prov], opt)
			knownProviders[prov] = true
		}
	})
}

// GetProviderOptions returns all registered provider options.
func GetProviderOptions() []ProviderOptionDef {
	initRegistry()
	res := make([]ProviderOptionDef, len(providerOptions))
	copy(res, providerOptions)
	return res
}

// GetProviderOptionsFor returns options registered for a specific provider.
func GetProviderOptionsFor(provider string) []ProviderOptionDef {
	initRegistry()
	prov := strings.ToLower(strings.TrimSpace(provider))
	opts := providerMap[prov]
	res := make([]ProviderOptionDef, len(opts))
	copy(res, opts)
	return res
}

// IsTier1Provider returns true if the provider has curated registry options.
func IsTier1Provider(provider string) bool {
	initRegistry()
	prov := strings.ToLower(strings.TrimSpace(provider))
	return knownProviders[prov]
}
