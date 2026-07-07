package plan

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	InputTarget          = "target"
	artifactRootDir      = "cal"
	artifactDir          = "artifacts"
	artifactFileSuffix   = ".out"
	artifactDateLayout   = "2006-01-02"
	defaultUseIDFallback = "use"
)

var ErrArtifactPathFailed = errors.New("artifact path failed")

// TargetPath returns the local temporary target artifact path for one use.
func TargetPath(useID string, now time.Time) (string, error) {
	useID = strings.TrimSpace(useID)
	if useID == "" {
		useID = defaultUseIDFallback
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	day := now.UTC().Format(artifactDateLayout)
	dir := filepath.Join(os.TempDir(), artifactRootDir, artifactDir, day)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("%w: %w", ErrArtifactPathFailed, err)
	}
	return filepath.Join(dir, useID+artifactFileSuffix), nil
}
