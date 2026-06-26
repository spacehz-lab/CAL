package calpath

import (
	"os"
	"path/filepath"
)

func defaultHomeDir() (string, error) {
	if localAppData := stringsTrimEnv("LocalAppData"); localAppData != "" {
		return filepath.Join(localAppData, "cal"), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "cal"), nil
}
