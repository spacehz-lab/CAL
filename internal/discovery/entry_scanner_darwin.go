//go:build darwin

package discovery

import (
	"os"
	"path/filepath"
	"strings"
)

func isAppBundle(path string) bool {
	return strings.EqualFold(filepath.Ext(filepath.Base(path)), ".app")
}

func isExecutable(info os.FileInfo, _ string) bool {
	return !info.IsDir() && info.Mode()&0o111 != 0
}
