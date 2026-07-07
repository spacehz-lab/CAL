package e2e

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestV1SeededCapabilityRunUseAndEval(t *testing.T) {
	repo, calctlBin, caldBin := functionalBinaries(t)
	temp := t.TempDir()
	home := filepath.Join(temp, "home")
	providerPath := filepath.Join(temp, "fake-exporter")
	writeFakeExecutable(t, providerPath)

	env := withHomeEnv(os.Environ(), home)
	startCald(t, repo, env, caldBin)

	provider := addV1Provider(t, repo, env, calctlBin, providerPath)
	if provider.ID == "" || provider.Kind != model.ProviderKindCLI || provider.Path != providerPath {
		t.Fatalf("provider = %#v, want registered CLI provider", provider)
	}
	capability := seedDocumentConvertCapability(t, home, provider)

	var providers providerListResponse
	runJSON(t, repo, env, &providers, calctlBin, "providers", "list", "--json")
	if len(providers.Providers) != 1 || providers.Providers[0].ID != provider.ID {
		t.Fatalf("providers = %#v, want seeded provider", providers)
	}

	var capabilities capabilityListResponse
	runJSON(t, repo, env, &capabilities, calctlBin, "capabilities", "list", "--json")
	if capabilities.Count != 1 || len(capabilities.Capabilities) != 1 || capabilities.Capabilities[0].ID != capability.ID || capabilities.Capabilities[0].Bindings.Available != 1 {
		t.Fatalf("capabilities = %#v, want one promoted capability", capabilities)
	}

	source := filepath.Join(temp, "source.txt")
	target := filepath.Join(temp, "target.pdf")
	if err := os.WriteFile(source, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	var run runResponse
	runJSON(t, repo, env, &run, calctlBin, "runs", "create", "--capability-id", capability.ID, "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(target)+`}`, "--verify", "--json")
	if run.Run == nil || run.Run.Status != model.RunStatusSucceeded || !run.Run.Verified || run.Run.ProviderID != provider.ID {
		t.Fatalf("run = %#v, want verified successful provider run", run)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("run target missing: %v", err)
	}

	var use useResponse
	runJSON(t, repo, env, &use, calctlBin, "use", "export this document as pdf", "--inputs-json", `{"source":`+strconv.Quote(source)+`}`, "--verify", "--json")
	if use.Status != model.RunStatusSucceeded || use.Selection == nil || use.Selection.CapabilityID != capability.ID || use.Selection.ProviderID != provider.ID {
		t.Fatalf("use = %#v, want selected seeded capability", use)
	}
	if use.Run == nil || use.Run.Status != model.RunStatusSucceeded || !use.Run.Verified || use.Run.BindingID != use.Selection.BindingID {
		t.Fatalf("use run = %#v, want verified selected binding run", use.Run)
	}
	useTarget, ok := use.Run.Inputs["target"].(string)
	if !ok || useTarget == "" {
		t.Fatalf("use inputs = %#v, want generated target", use.Run.Inputs)
	}
	if _, err := os.Stat(useTarget); err != nil {
		t.Fatalf("use target missing: %v", err)
	}

	var metrics evalResponse
	runJSON(t, repo, env, &metrics, calctlBin, "eval", "--json")
	if metrics.Capability.Capabilities != 1 || metrics.Capability.Bindings != 1 || metrics.Capability.PromotedBindings != 1 || metrics.Capability.BindingsWithVerify != 1 {
		t.Fatalf("eval capability = %#v, want seeded capability coverage", metrics.Capability)
	}
	if metrics.Reuse.Runs.Total != 2 || metrics.Reuse.Runs.ByName[string(model.RunStatusSucceeded)] != 2 || metrics.Reuse.Verified != 2 {
		t.Fatalf("eval reuse = %#v, want two verified successful runs", metrics.Reuse)
	}
	if metrics.Acquisition.Traces.Total != 0 {
		t.Fatalf("eval acquisition = %#v, want no acquisition traces for seeded fixture", metrics.Acquisition)
	}
}
