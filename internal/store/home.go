package store

import (
	"github.com/spacehz-lab/cal/internal/calpath"
)

// OpenFromEnv opens the store using the configured CAL home or the platform default.
func OpenFromEnv() (*Store, error) {
	home, err := calpath.HomeDir()
	if err != nil {
		return nil, err
	}
	return Open(home)
}
