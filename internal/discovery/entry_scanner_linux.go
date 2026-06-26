//go:build linux

package discovery

import "os"

func isAppBundle(string) bool {
	return false
}

func isExecutable(info os.FileInfo, _ string) bool {
	return !info.IsDir() && info.Mode()&0o111 != 0
}
