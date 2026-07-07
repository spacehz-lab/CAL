package probe

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spacehz-lab/cal/internal/proposal"
)

func writeFixture(workDir string, fixture proposal.Fixture) (string, error) {
	if strings.TrimSpace(fixture.Input) == "" {
		return "", fmt.Errorf("probe fixture input is required")
	}
	if strings.TrimSpace(fixture.Filename) == "" {
		return "", fmt.Errorf("probe fixture filename is required")
	}
	if filepath.IsAbs(fixture.Filename) {
		return "", fmt.Errorf("probe fixture filename must be relative")
	}
	clean := filepath.Clean(fixture.Filename)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("probe fixture filename escapes probe workdir")
	}
	path := filepath.Join(workDir, clean)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create probe fixture directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(fixture.Content), 0o644); err != nil {
		return "", fmt.Errorf("write probe fixture: %w", err)
	}
	return path, nil
}
