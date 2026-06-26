package observe

// Result contains observations captured for one provider.
type Result struct {
	ProviderID   string        `json:"provider_id"`
	Observations []Observation `json:"observations"`
}

// Observation is one provider observation fact.
type Observation struct {
	Type    string         `json:"type"`
	Source  string         `json:"source,omitempty"`
	Content map[string]any `json:"content,omitempty"`
}
