package calpath

import (
	"os"
	"strings"
)

func stringsTrimEnv(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}
