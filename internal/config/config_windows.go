//go:build windows

package config

func defaultProviderPaths() []string {
	return []string{"PATH", "%ProgramFiles%", "%ProgramFiles(x86)%", `%LocalAppData%\Programs`}
}
