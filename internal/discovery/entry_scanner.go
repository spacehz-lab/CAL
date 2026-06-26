package discovery

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/spacehz-lab/cal/internal/core"
)

// EntryOptions configures Discovery Entry scanning.
type EntryOptions struct {
	Paths   []string
	Entries []string
}

// ScanEntries runs Discovery Entry and returns discovered provider records.
func ScanEntries(ctx context.Context, opts EntryOptions) ([]core.Provider, error) {
	return newEntryScanner().Scan(ctx, opts)
}

type entryScanner struct {
	found map[string]core.Provider
}

func newEntryScanner() *entryScanner {
	return &entryScanner{
		found: map[string]core.Provider{},
	}
}

func (scanner *entryScanner) Scan(ctx context.Context, opts EntryOptions) ([]core.Provider, error) {
	scanner.logStarted(opts)
	for _, discoveryPath := range opts.Paths {
		if err := scanner.inspectDiscoveryPath(ctx, discoveryPath); err != nil {
			scanner.logFailed("paths")
			return nil, err
		}
	}
	for _, entryPath := range opts.Entries {
		if err := scanner.inspectEntryPath(ctx, entryPath); err != nil {
			scanner.logFailed("entries")
			return nil, err
		}
	}

	providers := make([]core.Provider, 0, len(scanner.found))
	for _, provider := range scanner.found {
		providers = append(providers, provider)
	}
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].ID < providers[j].ID
	})
	scanner.logCompleted(providers)
	return providers, nil
}

func (scanner *entryScanner) inspectDiscoveryPath(ctx context.Context, discoveryPath string) error {
	if strings.TrimSpace(discoveryPath) == "" {
		return nil
	}
	if discoveryPath == "PATH" {
		for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
			if err := scanner.inspectDirectory(ctx, dir); err != nil {
				return err
			}
		}
		return nil
	}
	return scanner.inspectDirectory(ctx, scanner.expandPath(discoveryPath))
}

func (scanner *entryScanner) inspectEntryPath(ctx context.Context, entryPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(entryPath) == "" {
		return nil
	}

	path := scanner.expandPath(entryPath)
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	if info.IsDir() {
		if !isAppBundle(path) {
			return nil
		}
		provider, err := scanner.newProvider(core.ProviderKindApp, scanner.providerName(path), path)
		if err != nil {
			return err
		}
		scanner.found[provider.ID] = provider
		return nil
	}
	if !isExecutable(info, path) {
		return nil
	}
	provider, err := scanner.newProvider(core.ProviderKindCLI, scanner.providerName(path), path)
	if err != nil {
		return err
	}
	scanner.found[provider.ID] = provider
	return nil
}

func (scanner *entryScanner) inspectDirectory(ctx context.Context, dir string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(dir) == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		kind := core.ProviderKindCLI
		switch {
		case info.IsDir() && isAppBundle(path):
			kind = core.ProviderKindApp
		case isExecutable(info, path):
		default:
			continue
		}
		provider, err := scanner.newProvider(kind, scanner.providerName(path), path)
		if err != nil {
			return err
		}
		scanner.found[provider.ID] = provider
	}
	return nil
}

func (scanner *entryScanner) providerName(path string) string {
	name := filepath.Base(path)
	if isAppBundle(path) {
		return strings.TrimSuffix(name, filepath.Ext(name))
	}
	return name
}

func (scanner *entryScanner) expandPath(path string) string {
	if strings.HasPrefix(path, "$HOME/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "$HOME/"))
		}
	}
	if path == "$HOME" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	return os.ExpandEnv(path)
}

func (scanner *entryScanner) newProvider(kind core.ProviderKind, name, path string) (core.Provider, error) {
	cleanPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return core.Provider{}, err
	}
	return core.Provider{
		ID:   core.ProviderID(runtime.GOOS, kind, cleanPath),
		Name: name,
		Kind: kind,
		Path: cleanPath,
	}, nil
}

func (scanner *entryScanner) logStarted(opts EntryOptions) {
	slog.Info("discovery entry started",
		"paths_count", len(opts.Paths),
		"entries_count", len(opts.Entries),
	)
}

func (scanner *entryScanner) logCompleted(providers []core.Provider) {
	cliCount := 0
	appCount := 0
	for _, provider := range providers {
		switch provider.Kind {
		case core.ProviderKindCLI:
			cliCount++
		case core.ProviderKindApp:
			appCount++
		}
	}
	slog.Info("discovery entry completed",
		"providers_count", len(providers),
		"cli_count", cliCount,
		"app_count", appCount,
	)
}

func (scanner *entryScanner) logFailed(stage string) {
	slog.Warn("discovery entry failed",
		"stage", stage,
		"code", "entry_scan_failed",
	)
}
