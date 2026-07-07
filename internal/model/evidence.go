package model

// EvidenceRef points to evidence collected during discovery or run verification.
type EvidenceRef struct {
	ID      string         `json:"id"`
	Type    string         `json:"type,omitempty"`
	Content map[string]any `json:"content,omitempty"`
	Ref     string         `json:"ref,omitempty"`
}
