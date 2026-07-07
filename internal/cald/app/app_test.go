package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
)

func TestNewRequiresHome(t *testing.T) {
	if _, err := New(Options{Home: " "}); !errors.Is(err, ErrHomeRequired) {
		t.Fatalf("New() error = %v, want ErrHomeRequired", err)
	}
}

func TestNewCreatesStoreAndDefaultsWorkRoot(t *testing.T) {
	home := t.TempDir()
	app := newTestApp(t, home)

	if app.Home() != home {
		t.Fatalf("Home() = %q, want %q", app.Home(), home)
	}
	if app.WorkRoot() != filepath.Join(home, defaultWorkDir) {
		t.Fatalf("WorkRoot() = %q, want default under home", app.WorkRoot())
	}
	for _, dir := range []string{"providers", "capabilities", "traces", "runs"} {
		if info, err := os.Stat(filepath.Join(home, dir)); err != nil || !info.IsDir() {
			t.Fatalf("store dir %s stat = %v, info = %#v", dir, err, info)
		}
	}
}

func TestAddProviderAndListProviders(t *testing.T) {
	skipWindowsShell(t)
	app := newTestApp(t, t.TempDir())
	path := writeProviderScript(t)

	response, err := app.AddProvider(context.Background(), &contract.AddProviderRequest{ProviderPath: path})
	if err != nil {
		t.Fatalf("AddProvider() error = %v", err)
	}
	if len(response.Providers) != 1 {
		t.Fatalf("providers len = %d, want 1", len(response.Providers))
	}
	if response.Providers[0].Kind != model.ProviderKindCLI || response.Providers[0].Path != path {
		t.Fatalf("provider = %#v, want CLI provider at %s", response.Providers[0], path)
	}

	list, err := app.ListProviders(context.Background())
	if err != nil {
		t.Fatalf("ListProviders() error = %v", err)
	}
	if len(list.Providers) != 1 || list.Providers[0].ID != response.Providers[0].ID {
		t.Fatalf("ListProviders() = %#v, want registered provider", list)
	}
}

func TestListCapabilitiesSummarizesPromotedBindings(t *testing.T) {
	app := newTestApp(t, t.TempDir())
	providerID := "provider_test"
	saveCapability(t, app, providerID)

	response, err := app.ListCapabilities(context.Background(), &contract.CapabilityListRequest{ProviderID: providerID})
	if err != nil {
		t.Fatalf("ListCapabilities() error = %v", err)
	}
	if response.Count != 1 {
		t.Fatalf("Count = %d, want 1", response.Count)
	}
	summary := response.Capabilities[0]
	if summary.ID != "document.echo" || summary.Bindings.Available != 1 {
		t.Fatalf("summary = %#v, want document.echo with one binding", summary)
	}
	if len(summary.Bindings.ProviderIDs) != 1 || summary.Bindings.ProviderIDs[0] != providerID {
		t.Fatalf("provider ids = %#v, want %s", summary.Bindings.ProviderIDs, providerID)
	}
	if len(summary.Bindings.VerifyLevels) != 1 || summary.Bindings.VerifyLevels[0] != string(model.VerifyLevelL2) {
		t.Fatalf("verify levels = %#v, want L2", summary.Bindings.VerifyLevels)
	}
}

func TestRunUseAndEval(t *testing.T) {
	skipWindowsShell(t)
	app := newTestApp(t, t.TempDir())
	provider := registerTestProvider(t, app)
	saveCapability(t, app, provider.ID)

	runResult, err := app.Run(context.Background(), &contract.RunRequest{
		CapabilityID: "document.echo",
		Inputs:       map[string]any{"text": "ok"},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if runResult.Run == nil || runResult.Run.Status != model.RunStatusSucceeded || runResult.Run.Outputs["stdout"] != "ok" {
		t.Fatalf("run = %#v, want succeeded stdout ok", runResult)
	}

	useResult, err := app.Use(context.Background(), &contract.UseRequest{
		Intent: "echo document",
		Inputs: map[string]any{"text": "again"},
	})
	if err != nil {
		t.Fatalf("Use() error = %v", err)
	}
	if useResult.Status != model.RunStatusSucceeded || useResult.Selection == nil || useResult.Selection.ProviderID != provider.ID {
		t.Fatalf("use = %#v, want succeeded selection for provider", useResult)
	}

	evalResult, err := app.Eval(context.Background(), &contract.EvalRequest{ProviderID: provider.ID})
	if err != nil {
		t.Fatalf("Eval() error = %v", err)
	}
	if evalResult.Reuse.Runs.Total != 2 || evalResult.Capability.Capabilities != 1 {
		t.Fatalf("eval = %#v, want two runs and one capability", evalResult)
	}
}

func TestAcquireModesWithoutLLM(t *testing.T) {
	app := newTestApp(t, t.TempDir())

	_, err := app.Acquire(context.Background(), &contract.AcquisitionRequest{ProviderID: "provider_test"})
	if !errors.Is(err, ErrLLMNotConfigured) {
		t.Fatalf("Acquire(live) error = %v, want ErrLLMNotConfigured", err)
	}
	_, err = app.Acquire(context.Background(), &contract.AcquisitionRequest{Mode: contract.AcquisitionModeReplay, ProviderID: "provider_test"})
	if !errors.Is(err, ErrProposalPathRequired) {
		t.Fatalf("Acquire(replay) error = %v, want ErrProposalPathRequired", err)
	}
}

func TestAcquireReplayWithoutLLMPromotesBinding(t *testing.T) {
	skipWindowsShell(t)
	app := newTestApp(t, t.TempDir())
	provider := registerAcquisitionProvider(t, app)
	proposalPath := writeReplayProposal(t)

	response, err := app.Acquire(context.Background(), &contract.AcquisitionRequest{
		ProviderID:   provider.ID,
		Mode:         contract.AcquisitionModeReplay,
		ProposalPath: proposalPath,
		Hint:         "convert a document",
	})
	if err != nil {
		t.Fatalf("Acquire(replay) error = %v", err)
	}
	if response.TraceID == "" || response.CapabilitiesPromoted != 1 || response.BindingsPromoted != 1 || response.Trace == nil {
		t.Fatalf("response = %#v, want promoted replay trace", response)
	}
	if response.Trace.Status != model.TraceStatusCompleted || len(response.Trace.Probes) != 1 || !response.Trace.Probes[0].Passed {
		t.Fatalf("trace = %#v, want completed passing probe", response.Trace)
	}
	if len(response.Trace.Candidates) != 1 || response.Trace.Candidates[0].ProviderID != provider.ID || response.Trace.Candidates[0].Source != "proposal:replay" {
		t.Fatalf("candidates = %#v, want replay candidate for active provider", response.Trace.Candidates)
	}
}

func TestAcquireRulesWithoutLLMPromotesBinding(t *testing.T) {
	skipWindowsShell(t)
	app := newTestApp(t, t.TempDir())
	provider := registerAcquisitionProvider(t, app)

	response, err := app.Acquire(context.Background(), &contract.AcquisitionRequest{
		ProviderID: provider.ID,
		Mode:       contract.AcquisitionModeRules,
		Hint:       "convert a document",
	})
	if err != nil {
		t.Fatalf("Acquire(rules) error = %v", err)
	}
	if response.TraceID == "" || response.CapabilitiesPromoted != 1 || response.BindingsPromoted != 1 || response.Trace == nil {
		t.Fatalf("response = %#v, want promoted rules trace", response)
	}
	if response.Trace.Status != model.TraceStatusCompleted || len(response.Trace.Probes) != 1 || !response.Trace.Probes[0].Passed {
		t.Fatalf("trace = %#v, want completed passing probe", response.Trace)
	}
	if len(response.Trace.Candidates) != 1 || response.Trace.Candidates[0].Source != "rules:cli_help_marker" {
		t.Fatalf("candidates = %#v, want rules marker candidate", response.Trace.Candidates)
	}
}

func newTestApp(t *testing.T, home string) *App {
	t.Helper()
	app, err := New(Options{Home: home})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return app
}

func registerTestProvider(t *testing.T, app *App) model.Provider {
	t.Helper()
	response, err := app.AddProvider(context.Background(), &contract.AddProviderRequest{ProviderPath: writeProviderScript(t)})
	if err != nil {
		t.Fatalf("AddProvider() error = %v", err)
	}
	if len(response.Providers) != 1 {
		t.Fatalf("providers len = %d, want 1", len(response.Providers))
	}
	return response.Providers[0]
}

func saveCapability(t *testing.T, app *App, providerID string) {
	t.Helper()
	capability := model.Capability{
		ID:          "document.echo",
		Description: "Echo document text.",
		Bindings: []model.Binding{{
			ID:           "binding_echo",
			CapabilityID: "document.echo",
			ProviderID:   providerID,
			Execution: model.Execution{
				Kind: model.ExecutionKindCLI,
				Spec: map[string]any{
					model.ExecutionSpecArgs: []string{"{{text}}"},
				},
			},
			Verify:   &model.VerifySpec{Level: model.VerifyLevelL2, Method: model.VerifyMethodContract},
			Evidence: []model.EvidenceRef{{ID: "evidence_1"}},
			State:    model.BindingStatePromoted,
		}},
	}
	if err := app.store.SaveCapability(&capability); err != nil {
		t.Fatalf("SaveCapability() error = %v", err)
	}
}

func writeProviderScript(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "echo-provider")
	content := "#!/bin/sh\nprintf \"%s\" \"$1\"\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func registerAcquisitionProvider(t *testing.T, app *App) model.Provider {
	t.Helper()
	response, err := app.AddProvider(context.Background(), &contract.AddProviderRequest{ProviderPath: writeAcquisitionProviderScript(t)})
	if err != nil {
		t.Fatalf("AddProvider() error = %v", err)
	}
	if len(response.Providers) != 1 {
		t.Fatalf("providers len = %d, want 1", len(response.Providers))
	}
	return response.Providers[0]
}

func writeAcquisitionProviderScript(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "acquisition-provider")
	content := `#!/bin/sh
if [ "$1" = "--help" ]; then
  echo "Usage: acquisition-provider export-pdf --source <path> --target <path>"
  echo "CAL_CAPABILITY document.convert"
  echo "CAL_COMMAND export-pdf --source {{source}} --target {{target}}"
  exit 0
fi
if [ "$1" = "export-pdf" ] || [ "$1" = "make-pdf" ]; then
  target=""
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --target|--out)
        target="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [ -z "$target" ]; then
    exit 2
  fi
  printf '%s\n' '%PDF-1.4' '%%EOF' > "$target"
  exit 0
fi
exit 64
`
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func writeReplayProposal(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "proposal.json")
	content := `{
  "metadata": {"source":"replay","prompt_version":"test-v1","model":"fixture","schema_version":"proposal.v1"},
  "candidates": [{
    "capability_id": "document.convert",
    "description": "Export a document.",
    "execution": {"kind":"cli","spec":{"args":["make-pdf","--out","{{target}}"]}}
  }],
  "probe_plans": [{
    "candidate_index": 0,
    "inputs": {"target": "{{workdir}}/output.pdf"},
    "verify": {"level":"L2","method":"execute","checks":[{"subject":{"type":"file","input":"target"},"predicate":"format","params":{"format":"pdf"}}]}
  }]
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func skipWindowsShell(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell provider fixture is unix-only")
	}
}
