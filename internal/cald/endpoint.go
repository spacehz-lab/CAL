package cald

import (
	"path/filepath"
)

const (
	connectionDirName = "cald"
	endpointFileName  = "endpoint.json"
)

// EndpointFile describes the cald HTTP endpoint written under the CAL home.
type EndpointFile struct {
	PID       int    `json:"pid"`
	BaseURL   string `json:"base_url"`
	StartedAt string `json:"started_at"`
}

// EndpointFilePath returns the path to cald endpoint metadata for one CAL home.
func EndpointFilePath(home string) string {
	return filepath.Join(home, connectionDirName, endpointFileName)
}
