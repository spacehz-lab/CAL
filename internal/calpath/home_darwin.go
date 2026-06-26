package calpath

import (
	"os"
	"path/filepath"
)

func defaultHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Application Support", "cal"), nil
}
