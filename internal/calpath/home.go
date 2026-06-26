package calpath

import (
	"os"
	"path/filepath"
	"strings"
)

const envHome = "CAL_HOME"

// HomeDir returns the configured or platform default CAL home directory.
func HomeDir() (string, error) {
	if value := strings.TrimSpace(os.Getenv(envHome)); value != "" {
		return filepath.Clean(value), nil
	}
	return defaultHomeDir()
}

// WithHomeEnv returns env with the CAL home set to home.
func WithHomeEnv(env []string, home string) []string {
	prefix := envHome + "="
	value := prefix + filepath.Clean(home)
	for index, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			env[index] = value
			return env
		}
	}
	return append(env, value)
}

// HomeDirFromEnv returns the CAL home from an explicit environment slice.
func HomeDirFromEnv(env []string) string {
	prefix := envHome + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return entry[len(prefix):]
		}
	}
	return ""
}
