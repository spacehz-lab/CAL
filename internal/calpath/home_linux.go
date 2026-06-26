package calpath

import (
	"os"
	"path/filepath"
)

func defaultHomeDir() (string, error) {
	if data := stringsTrimEnv("XDG_DATA_HOME"); data != "" {
		return filepath.Join(data, "cal"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "cal"), nil
}
