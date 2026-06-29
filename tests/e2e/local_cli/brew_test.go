package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	e2etest "github.com/spacehz-lab/cal/internal/testsupport/e2e"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestBrewReplayPromotesContractAndExecuteBindings(t *testing.T) {
	if os.Getenv("CAL_LOCAL_CLI_E2E") != "1" {
		t.Skip("set CAL_LOCAL_CLI_E2E=1 to run local real-CLI e2e")
	}
	providerPath, err := exec.LookPath("brew")
	if err != nil {
		t.Skipf("brew not available: %v", err)
	}

	repo := e2etest.RepoRoot(t)
	temp := t.TempDir()
	calctlBin := filepath.Join(temp, "calctl")
	caldBin := filepath.Join(temp, "cald")
	e2etest.Build(t, repo, calctlBin, "./cmd/calctl")
	e2etest.Build(t, repo, caldBin, "./cmd/cald")

	home := filepath.Join(temp, "home")
	env := e2etest.WithHomeEnv(os.Environ(), home)
	e2etest.StartCald(t, repo, env, caldBin)

	proposalPath := writeBrewMixedReplayProposal(t, filepath.Join(temp, "brew-proposal.json"))
	var acquisition struct {
		State                string                    `json:"state"`
		TraceID              string                    `json:"trace_id"`
		CapabilitiesPromoted int                       `json:"capabilities_promoted"`
		BindingsPromoted     int                       `json:"bindings_promoted"`
		Providers            []e2etest.ProviderSummary `json:"providers"`
	}
	provider := runDiscoveryForProviderPath(t, repo, env, calctlBin, providerPath, &acquisition, "--proposal-path", proposalPath, "--json")
	if acquisition.State != "succeeded" || acquisition.CapabilitiesPromoted != 2 || acquisition.BindingsPromoted != 2 || acquisition.TraceID == "" || len(acquisition.Providers) != 1 {
		t.Fatalf("acquisition = %#v, want brew contract and execute bindings promoted", acquisition)
	}
	if provider.ID != acquisition.Providers[0].ID {
		t.Fatalf("provider = %#v acquisition providers = %#v, want matching provider", provider, acquisition.Providers)
	}

	trace := e2etest.ReadJSONFile[caltrace.Trace](t, filepath.Join(home, "discovery", acquisition.TraceID, "trace.json"))
	assertBrewProbe(t, trace, "package.install", core.VerifyLevelL1, core.VerifyMethodContract)
	assertBrewProbe(t, trace, "system.inspect", core.VerifyLevelL2, core.VerifyMethodExecute)

	var packageRun struct {
		Status string `json:"status"`
		Error  struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	e2etest.RunFailJSON(t, repo, env, &packageRun, calctlBin, "runs", "create", "--capability-id", "package.install", "--inputs-json", `{"package":"cal-contract-only-example"}`, "--json")
	if packageRun.Status != "failed" || packageRun.Error.Code != "binding_not_found" {
		t.Fatalf("package.install default run = %#v, want binding_not_found below default L2 threshold", packageRun)
	}

	var inspectRun struct {
		Status   string             `json:"status"`
		Verified bool               `json:"verified"`
		Evidence []core.EvidenceRef `json:"evidence"`
		Outputs  map[string]any     `json:"outputs"`
	}
	e2etest.RunJSON(t, repo, env, &inspectRun, calctlBin, "runs", "create", "--capability-id", "system.inspect", "--inputs-json", `{}`, "--verify", "--json")
	if inspectRun.Status != "succeeded" || !inspectRun.Verified || len(inspectRun.Evidence) != 1 {
		t.Fatalf("system.inspect run = %#v, want verified L2 brew config execution", inspectRun)
	}
	if output, _ := inspectRun.Outputs["output"].(string); output == "" {
		t.Fatalf("system.inspect outputs = %#v, want captured brew config output", inspectRun.Outputs)
	}
}

func assertBrewProbe(t *testing.T, trace caltrace.Trace, capabilityID string, level core.VerifyLevel, method core.VerifyMethod) {
	t.Helper()
	for _, probe := range trace.Probes {
		if probe.CandidateIndex < 0 || probe.CandidateIndex >= len(trace.Candidates) {
			continue
		}
		if trace.Candidates[probe.CandidateIndex].CapabilityID != capabilityID {
			continue
		}
		if !probe.Passed || probe.Verify.Level != level || probe.Verify.Method != method {
			t.Fatalf("probe for %s = %#v, want passed %s %s", capabilityID, probe, level, method)
		}
		if method == core.VerifyMethodContract {
			if len(probe.Evidence) != 1 || probe.Evidence[0].Type != "contract" {
				t.Fatalf("probe for %s evidence = %#v, want contract evidence", capabilityID, probe.Evidence)
			}
		}
		return
	}
	t.Fatalf("trace probes = %#v, missing probe for %s", trace.Probes, capabilityID)
}

func writeBrewMixedReplayProposal(t *testing.T, path string) string {
	t.Helper()
	content := `{
  "metadata": {"source": "replay", "prompt_version": "test-v1", "model": "fixture", "schema_version": "proposal.v1"},
  "candidates": [
    {
      "capability_id": "package.install",
      "description": "Install a package through Homebrew.",
      "execution": {"kind": "cli", "spec": {"args": ["install", "{{package}}"]}}
    },
    {
      "capability_id": "system.inspect",
      "description": "Inspect local Homebrew configuration.",
      "execution": {"kind": "cli", "spec": {"args": ["config"]}}
    }
  ],
  "probe_plans": [
    {
      "candidate_index": 0,
      "inputs": {"package": "cal-contract-only-example"},
      "verify": {"level":"L1","method":"contract","checks":[]}
    },
    {
      "candidate_index": 1,
      "inputs": {},
      "verify": {"level":"L2","method":"execute","checks":[{"subject":"output","predicate":"contains","params":{"value":"HOMEBREW_VERSION"}}]}
    }
  ]
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write brew mixed replay proposal: %v", err)
	}
	return path
}
