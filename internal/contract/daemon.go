package contract

// DaemonStatus reports local daemon state.
type DaemonStatus struct {
	Running bool   `json:"running"`
	BaseURL string `json:"base_url,omitempty"`
	PID     int    `json:"pid,omitempty"`
}

// DaemonStopResponse reports whether a stop was requested.
type DaemonStopResponse struct {
	Stopping bool `json:"stopping"`
}
