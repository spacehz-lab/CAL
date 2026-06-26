//go:build linux

package config

func defaultProviderPaths() []string {
	return []string{"PATH", "$HOME/.local/bin", "/usr/local/bin", "/usr/bin", "/bin", "/snap/bin"}
}
