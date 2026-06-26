package logging

import (
	"os"
	"path/filepath"
	"strings"
)

func defaultLogDir() (string, error) {
	if localAppData := strings.TrimSpace(os.Getenv("LocalAppData")); localAppData != "" {
		return filepath.Join(localAppData, "cal", "logs"), nil
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "cal", "logs"), nil
}
