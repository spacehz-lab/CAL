package llm

// Prompt is the bounded prompt shape for one LLM completion call.
type Prompt struct {
	System string `json:"system"`
	User   string `json:"user"`
}
