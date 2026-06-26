package control

// Status describes local cald availability.
type Status struct {
	Running   bool   `json:"running"`
	Mode      string `json:"mode"`
	PID       int    `json:"pid,omitempty"`
	Endpoint  string `json:"endpoint,omitempty"`
	StartedAt string `json:"started_at,omitempty"`
	Message   string `json:"message,omitempty"`
}
