//go:build darwin

package entry

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spacehz-lab/cal/internal/model"
)

func providerKind(path string, info os.FileInfo) (model.ProviderKind, bool) {
	if info.IsDir() && strings.EqualFold(filepath.Ext(filepath.Base(path)), ".app") {
		return model.ProviderKindApp, true
	}
	if !info.IsDir() && info.Mode()&0o111 != 0 {
		return model.ProviderKindCLI, true
	}
	return "", false
}
