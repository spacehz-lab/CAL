package logging

import (
	"os"
	"path/filepath"
)

func defaultLogDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Logs", "cal"), nil
}
