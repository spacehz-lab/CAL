package logging

import (
	"os"
	"path/filepath"
	"strings"
)

const envXDGStateHome = "XDG_STATE_HOME"

func defaultDir() (string, error) {
	if state := strings.TrimSpace(os.Getenv(envXDGStateHome)); state != "" {
		return filepath.Join(state, defaultName, "logs"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", defaultName, "logs"), nil
}
