package e2e

import (
	"testing"

	e2etest "github.com/spacehz-lab/cal/internal/testsupport/e2e"
)

func runDiscoveryForProviderPath(t *testing.T, repo string, env []string, calctlBin string, providerPath string, output any, args ...string) e2etest.ProviderSummary {
	t.Helper()
	var provider e2etest.ProviderSummary
	e2etest.RunJSON(t, repo, env, &provider, calctlBin, "providers", "add", "--provider-path", providerPath, "--json")
	command := append([]string{"discovery", "run", "--provider-id", provider.ID}, args...)
	e2etest.RunJSON(t, repo, env, output, calctlBin, command...)
	return provider
}
