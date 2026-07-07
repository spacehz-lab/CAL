package logging

import (
	"os"
	"path/filepath"
	"strings"
)

const envLocalAppData = "LocalAppData"

func defaultDir() (string, error) {
	if localAppData := strings.TrimSpace(os.Getenv(envLocalAppData)); localAppData != "" {
		return filepath.Join(localAppData, defaultName, "logs"), nil
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, defaultName, "logs"), nil
}
