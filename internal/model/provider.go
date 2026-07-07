package model

// ProviderKind identifies the local entry shape for a provider.
type ProviderKind string

const (
	// ProviderKindCLI identifies an executable command provider entry.
	ProviderKindCLI ProviderKind = "cli"
	// ProviderKindApp identifies an application entry, such as a macOS .app bundle.
	ProviderKindApp ProviderKind = "app"
)

// Provider is a discovered local provider entry.
type Provider struct {
	ID      string       `json:"id"`
	Name    string       `json:"name,omitempty"`
	Kind    ProviderKind `json:"kind"`
	Path    string       `json:"path"`
	Version string       `json:"version,omitempty"`
}
