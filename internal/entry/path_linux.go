//go:build linux

package entry

import (
	"os"

	"github.com/spacehz-lab/cal/internal/model"
)

func providerKind(_ string, info os.FileInfo) (model.ProviderKind, bool) {
	if !info.IsDir() && info.Mode()&0o111 != 0 {
		return model.ProviderKindCLI, true
	}
	return "", false
}
