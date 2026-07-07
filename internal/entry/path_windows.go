//go:build windows

package entry

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spacehz-lab/cal/internal/model"
)

func providerKind(path string, info os.FileInfo) (model.ProviderKind, bool) {
	if info.IsDir() {
		return "", false
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return "", false
	}
	for _, allowed := range executableExtensions() {
		if ext == allowed {
			return model.ProviderKindCLI, true
		}
	}
	return "", false
}

func executableExtensions() []string {
	pathext := os.Getenv("PATHEXT")
	if strings.TrimSpace(pathext) == "" {
		pathext = ".com;.exe;.bat;.cmd;.ps1"
	}
	parts := strings.Split(pathext, ";")
	extensions := make([]string, 0, len(parts))
	for _, part := range parts {
		extension := strings.ToLower(strings.TrimSpace(part))
		if extension != "" {
			extensions = append(extensions, extension)
		}
	}
	return extensions
}
