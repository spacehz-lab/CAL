package entry

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spacehz-lab/cal/internal/model"
)

func resolveProviderPath(providerPath string) (model.Provider, error) {
	path, err := normalizeProviderPath(providerPath)
	if err != nil {
		return model.Provider{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return model.Provider{}, newError(CodeTargetProviderNotFound, "provider path did not resolve to a supported provider")
	}

	kind, ok := providerKind(path, info)
	if !ok {
		return model.Provider{}, newError(CodeTargetProviderNotFound, "provider path did not resolve to a supported provider")
	}
	return model.Provider{
		ID:   model.ProviderID(runtime.GOOS, kind, path),
		Name: providerName(path, kind),
		Kind: kind,
		Path: path,
	}, nil
}

func normalizeProviderPath(providerPath string) (string, error) {
	providerPath = strings.TrimSpace(providerPath)
	if providerPath == "" {
		return "", newError(CodeInvalidProviderPath, "provider_path is required")
	}

	expanded := expandProviderPath(providerPath)
	cleanPath, err := filepath.Abs(filepath.Clean(expanded))
	if err != nil {
		return "", wrapError(CodeInvalidProviderPath, "normalize provider_path", err)
	}
	return cleanPath, nil
}

func providerName(path string, kind model.ProviderKind) string {
	name := filepath.Base(path)
	if kind == model.ProviderKindApp {
		return strings.TrimSuffix(name, filepath.Ext(name))
	}
	return name
}

func expandProviderPath(path string) string {
	if path == "$HOME" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "$HOME/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "$HOME/"))
		}
	}
	return os.ExpandEnv(path)
}
