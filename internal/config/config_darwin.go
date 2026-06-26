//go:build darwin

package config

func defaultProviderPaths() []string {
	return []string{"PATH", "/Applications", "/System/Applications", "$HOME/Applications"}
}
