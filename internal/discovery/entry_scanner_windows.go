//go:build windows

package discovery

import (
	"os"
	"path/filepath"
	"strings"
)

func isAppBundle(string) bool {
	return false
}

func isExecutable(info os.FileInfo, path string) bool {
	if info.IsDir() {
		return false
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return false
	}
	pathext := os.Getenv("PATHEXT")
	if strings.TrimSpace(pathext) == "" {
		pathext = ".com;.exe;.bat;.cmd"
	}
	for _, allowed := range strings.Split(pathext, ";") {
		if strings.EqualFold(ext, strings.TrimSpace(allowed)) {
			return true
		}
	}
	return false
}
