package proposal

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/runtime"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestMaterializerPlansProposalProbe(t *testing.T) {
	installProposalTestVerifiers(t, "file_parse_pdf")
	materializer := mustParse(t, proposalJSON("make-pdf", "proposal:test"))
	workDir := t.TempDir()

	plan, err := materializer.Plan(context.Background(), ProbePlanRequest{
		Candidate: caltrace.Candidate{
			ProviderID:   "provider_cli",
			CapabilityID: "document.export_pdf",
			Execution: core.Execution{
				Kind: core.ExecutionKindCLI,
				Spec: map[string]any{
					core.ExecutionSpecArgs: []any{"make-pdf", "--in", "{{source}}", "--out", "{{target}}"},
				},
			},
		},
		WorkDir: workDir,
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.Inputs["target"] != filepath.Join(workDir, "output.pdf") {
		t.Fatalf("inputs = %#v, want target rendered inside workdir", plan.Inputs)
	}
	source, ok := plan.Inputs["source"].(string)
	if !ok || source == "" {
		t.Fatalf("source input = %#v, want materialized fixture path", plan.Inputs["source"])
	}
	if content, err := os.ReadFile(source); err != nil || string(content) != "hello\n" {
		t.Fatalf("fixture = %q, %v, want hello fixture", content, err)
	}
	if plan.Verifier.ID != "file_parse_pdf" {
		t.Fatalf("verifier = %#v, want parse verifier", plan.Verifier)
	}
}

func TestMaterializerAnchorsRelativePathInputsToWorkDir(t *testing.T) {
	installProposalTestVerifiers(t, "file_exists")
	materializer := mustParse(t, `{
		"candidates": [{
			"provider_id": "provider_cli",
			"capability_id": "document.convert_html",
			"description": "Convert a document into a requested format artifact.",
			"execution": {"kind": "cli", "spec": {"args": ["convert", "--format", "{{format}}", "--out", "{{target}}"]}}
		}],
		"probe_plans": [{
			"candidate_index": 0,
			"inputs": {"target": "nested/output.html", "format": "html"},
			"verifier": {"id": "file_exists"}
		}]
	}`)
	workDir := t.TempDir()

	plan, err := materializer.Plan(context.Background(), ProbePlanRequest{
		Candidate: caltrace.Candidate{
			ProviderID:   "provider_cli",
			CapabilityID: "document.convert_html",
			Execution: core.Execution{
				Kind: core.ExecutionKindCLI,
				Spec: map[string]any{core.ExecutionSpecArgs: []any{"convert", "--format", "{{format}}", "--out", "{{target}}"}},
			},
		},
		WorkDir: workDir,
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.Inputs["target"] != filepath.Join(workDir, "nested", "output.html") {
		t.Fatalf("target input = %#v, want path anchored to workdir", plan.Inputs["target"])
	}
	if plan.Inputs["format"] != "html" {
		t.Fatalf("format input = %#v, want scalar preserved", plan.Inputs["format"])
	}
}

func TestMaterializerKeepsDotPrefixedScalarInputs(t *testing.T) {
	installProposalTestVerifiers(t, "file_parse_json")
	materializer := mustParse(t, `{
		"candidates": [{
			"provider_id": "provider_cli",
			"capability_id": "json.apply_filter",
			"description": "Apply a filter expression to a JSON input file.",
			"execution": {"kind": "cli", "spec": {"args": ["{{filter}}", "{{source}}"], "stdout_path_input": "target"}}
		}],
		"probe_plans": [{
			"candidate_index": 0,
			"inputs": {"filter": ".name", "target": "{{workdir}}/output.json"},
			"fixtures": [{"input": "source", "filename": "input.json", "content": "{\"name\":\"cal\"}\n"}],
			"verifier": {"id": "file_parse_json"}
		}]
	}`)

	plan, err := materializer.Plan(context.Background(), ProbePlanRequest{
		Candidate: caltrace.Candidate{
			ProviderID:   "provider_cli",
			CapabilityID: "json.apply_filter",
			Execution: core.Execution{
				Kind: core.ExecutionKindCLI,
				Spec: map[string]any{
					core.ExecutionSpecArgs:            []any{"{{filter}}", "{{source}}"},
					core.ExecutionSpecStdoutPathInput: "target",
				},
			},
		},
		WorkDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.Inputs["filter"] != ".name" {
		t.Fatalf("filter input = %#v, want scalar jq filter", plan.Inputs["filter"])
	}
}

func TestMaterializerAnchorsExplicitRelativePathInputs(t *testing.T) {
	installProposalTestVerifiers(t, "file_exists")
	materializer := mustParse(t, `{
		"candidates": [{
			"provider_id": "provider_cli",
			"capability_id": "file.write",
			"description": "Write a file artifact.",
			"execution": {"kind": "cli", "spec": {"args": ["write-file", "{{target}}"]}}
		}],
		"probe_plans": [{
			"candidate_index": 0,
			"inputs": {"target": "./nested/output.txt"},
			"verifier": {"id": "file_exists"}
		}]
	}`)
	workDir := t.TempDir()

	plan, err := materializer.Plan(context.Background(), ProbePlanRequest{
		Candidate: caltrace.Candidate{
			ProviderID:   "provider_cli",
			CapabilityID: "file.write",
			Execution: core.Execution{
				Kind: core.ExecutionKindCLI,
				Spec: map[string]any{core.ExecutionSpecArgs: []any{"write-file", "{{target}}"}},
			},
		},
		WorkDir: workDir,
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.Inputs["target"] != filepath.Join(workDir, "nested", "output.txt") {
		t.Fatalf("target input = %#v, want explicit relative path anchored", plan.Inputs["target"])
	}
}

func TestMaterializerRejectsProbePlanMissingExecutionTemplateInput(t *testing.T) {
	installProposalTestVerifiers(t, "file_parse_json")
	materializer := mustParse(t, `{
		"candidates": [{
			"provider_id": "provider_cli",
			"capability_id": "json.apply_filter",
			"description": "Apply a filter expression to a JSON input file.",
			"execution": {"kind": "cli", "spec": {"args": ["{{filter}}", "{{source}}"], "stdout_path_input": "target"}}
		}],
		"probe_plans": [{
			"candidate_index": 0,
			"inputs": {"target": "{{workdir}}/output.json"},
			"fixtures": [{"input": "source", "filename": "input.json", "content": "{\"name\":\"cal\"}\n"}],
			"verifier": {"id": "file_parse_json"}
		}]
	}`)

	_, err := materializer.Plan(context.Background(), ProbePlanRequest{
		Candidate: caltrace.Candidate{
			ProviderID:   "provider_cli",
			CapabilityID: "json.apply_filter",
			Execution: core.Execution{
				Kind: core.ExecutionKindCLI,
				Spec: map[string]any{
					core.ExecutionSpecArgs:            []any{"{{filter}}", "{{source}}"},
					core.ExecutionSpecStdoutPathInput: "target",
				},
			},
		},
		WorkDir: t.TempDir(),
	})
	if err == nil || !strings.Contains(err.Error(), `proposal probe inputs missing execution template input "filter"`) {
		t.Fatalf("Plan() error = %v, want missing filter input", err)
	}
}

func TestMaterializerRejectsUnproducedTargetInput(t *testing.T) {
	installProposalTestVerifiers(t, "file_parse_json")
	materializer := mustParse(t, `{
		"candidates": [{
			"provider_id": "provider_cli",
			"capability_id": "json.apply_filter",
			"description": "Apply a filter expression to a JSON input file.",
			"execution": {"kind": "cli", "spec": {"args": ["{{filter}}", "{{source}}"]}}
		}],
		"probe_plans": [{
			"candidate_index": 0,
			"inputs": {"filter": ".", "target": "{{workdir}}/output.json"},
			"fixtures": [{"input": "source", "filename": "input.json", "content": "{\"name\":\"cal\"}\n"}],
			"verifier": {"id": "file_parse_json"}
		}]
	}`)

	_, err := materializer.Plan(context.Background(), ProbePlanRequest{
		Candidate: caltrace.Candidate{
			ProviderID:   "provider_cli",
			CapabilityID: "json.apply_filter",
			Execution: core.Execution{
				Kind: core.ExecutionKindCLI,
				Spec: map[string]any{core.ExecutionSpecArgs: []any{"{{filter}}", "{{source}}"}},
			},
		},
		WorkDir: t.TempDir(),
	})
	if err == nil || !strings.Contains(err.Error(), "proposal probe target input is not produced by execution args or stdout_path_input") {
		t.Fatalf("Plan() error = %v, want unproduced target input", err)
	}
}

func TestMaterializerRejectsMissingStdoutPathInput(t *testing.T) {
	installProposalTestVerifiers(t, "file_parse_json")
	materializer := mustParse(t, `{
		"candidates": [{
			"provider_id": "provider_cli",
			"capability_id": "json.apply_filter",
			"description": "Apply a filter expression to a JSON input file.",
			"execution": {"kind": "cli", "spec": {"args": ["{{filter}}", "{{source}}"], "stdout_path_input": "target"}}
		}],
		"probe_plans": [{
			"candidate_index": 0,
			"inputs": {"filter": "."},
			"fixtures": [{"input": "source", "filename": "input.json", "content": "{\"name\":\"cal\"}\n"}],
			"verifier": {"id": "file_parse_json"}
		}]
	}`)

	_, err := materializer.Plan(context.Background(), ProbePlanRequest{
		Candidate: caltrace.Candidate{
			ProviderID:   "provider_cli",
			CapabilityID: "json.apply_filter",
			Execution: core.Execution{
				Kind: core.ExecutionKindCLI,
				Spec: map[string]any{
					core.ExecutionSpecArgs:            []any{"{{filter}}", "{{source}}"},
					core.ExecutionSpecStdoutPathInput: "target",
				},
			},
		},
		WorkDir: t.TempDir(),
	})
	if err == nil || !strings.Contains(err.Error(), `proposal probe inputs missing stdout path input "target"`) {
		t.Fatalf("Plan() error = %v, want missing stdout path input", err)
	}
}

func TestMaterializerRejectsUnsafeFixturePath(t *testing.T) {
	materializer := mustParse(t, `{
		"candidates": [{
			"provider_id": "provider_cli",
			"capability_id": "document.export_pdf",
			"description": "Export a document to a PDF artifact.",
			"execution": {"kind": "cli", "spec": {"args": ["make-pdf"]}}
		}],
		"probe_plans": [{
			"candidate_index": 0,
			"fixtures": [{"input": "source", "filename": "../escape.txt", "content": "x"}],
			"verifier": {"id": "file_exists"}
		}]
	}`)

	_, err := materializer.Plan(context.Background(), ProbePlanRequest{
		Candidate: caltrace.Candidate{
			ProviderID:   "provider_cli",
			CapabilityID: "document.export_pdf",
			Execution: core.Execution{
				Kind: core.ExecutionKindCLI,
				Spec: map[string]any{core.ExecutionSpecArgs: []any{"make-pdf"}},
			},
		},
		WorkDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("Plan() error = nil, want unsafe fixture path error")
	}
}

func TestMaterializerRejectsRelativeInputPathEscape(t *testing.T) {
	materializer := mustParse(t, `{
		"candidates": [{
			"provider_id": "provider_cli",
			"capability_id": "document.export_pdf",
			"description": "Export a document to a PDF artifact.",
			"execution": {"kind": "cli", "spec": {"args": ["make-pdf"]}}
		}],
		"probe_plans": [{
			"candidate_index": 0,
			"inputs": {"target": "../outside.pdf"},
			"verifier": {"id": "file_parse_pdf"}
		}]
	}`)

	_, err := materializer.Plan(context.Background(), ProbePlanRequest{
		Candidate: caltrace.Candidate{
			ProviderID:   "provider_cli",
			CapabilityID: "document.export_pdf",
			Execution: core.Execution{
				Kind: core.ExecutionKindCLI,
				Spec: map[string]any{core.ExecutionSpecArgs: []any{"make-pdf"}},
			},
		},
		WorkDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("Plan() error = nil, want relative input escape error")
	}
}

func TestMaterializerRejectsInputPathOutsideWorkDir(t *testing.T) {
	materializer := mustParse(t, `{
		"candidates": [{
			"provider_id": "provider_cli",
			"capability_id": "document.export_pdf",
			"description": "Export a document to a PDF artifact.",
			"execution": {"kind": "cli", "spec": {"args": ["make-pdf"]}}
		}],
		"probe_plans": [{
			"candidate_index": 0,
			"inputs": {"target": "/tmp/outside.pdf"},
			"verifier": {"id": "file_parse_pdf"}
		}]
	}`)

	_, err := materializer.Plan(context.Background(), ProbePlanRequest{
		Candidate: caltrace.Candidate{
			ProviderID:   "provider_cli",
			CapabilityID: "document.export_pdf",
			Execution: core.Execution{
				Kind: core.ExecutionKindCLI,
				Spec: map[string]any{core.ExecutionSpecArgs: []any{"make-pdf"}},
			},
		},
		WorkDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("Plan() error = nil, want unsafe input path error")
	}
}

func TestMaterializerRejectsUnsupportedVerifier(t *testing.T) {
	materializer := mustParse(t, `{
		"candidates": [{
			"provider_id": "provider_cli",
			"capability_id": "document.export_pdf",
			"description": "Export a document to a PDF artifact.",
			"execution": {"kind": "cli", "spec": {"args": ["make-pdf"]}}
		}],
		"probe_plans": [{
			"candidate_index": 0,
			"inputs": {"target": "{{workdir}}/output.pdf"},
			"verifier": {"id": "not_registered"}
		}]
	}`)

	_, err := materializer.Plan(context.Background(), ProbePlanRequest{
		Candidate: caltrace.Candidate{
			ProviderID:   "provider_cli",
			CapabilityID: "document.export_pdf",
			Execution: core.Execution{
				Kind: core.ExecutionKindCLI,
				Spec: map[string]any{core.ExecutionSpecArgs: []any{"make-pdf"}},
			},
		},
		WorkDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("Plan() error = nil, want unsupported verifier error")
	}
}

func TestMaterializerInstallsGeneratedVerifierPackage(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CAL_HOME", home)
	materializer := mustParse(t, `{
		"verifier_packages": [{
			"id": "contains_probe_text",
			"description": "Passes when the target file contains the probe text.",
			"verify_py": "import json\nimport sys\nrequest = json.load(sys.stdin)\nverifier_id = request['verifier']['id']\ntarget = request['inputs']['target']\nwith open(target, 'r', encoding='utf-8') as handle:\n    content = handle.read()\nprint(json.dumps({'passed': 'probe ok' in content, 'evidence': [{'id': verifier_id, 'type': verifier_id}], 'outputs': {'target': target}}))\n"
		}],
		"candidates": [{
			"provider_id": "provider_cli",
			"capability_id": "text.write_file",
			"description": "Write a text file artifact.",
			"execution": {"kind": "cli", "spec": {"args": ["write-note", "--out", "{{target}}"]}}
		}],
		"probe_plans": [{
			"candidate_index": 0,
			"inputs": {"target": "{{workdir}}/note.txt"},
			"verifier": {"id": "contains_probe_text"}
		}]
	}`)

	plan, err := materializer.Plan(context.Background(), ProbePlanRequest{
		Candidate: caltrace.Candidate{
			ProviderID:   "provider_cli",
			CapabilityID: "text.write_file",
			Execution: core.Execution{
				Kind: core.ExecutionKindCLI,
				Spec: map[string]any{core.ExecutionSpecArgs: []any{"write-note", "--out", "{{target}}"}},
			},
		},
		WorkDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if !strings.HasPrefix(plan.Verifier.ID, "verifier_contains_probe_text_") || plan.Verifier.ID == "contains_probe_text" {
		t.Fatalf("verifier = %#v, want rewritten verifier package id", plan.Verifier)
	}
	secondPlan, err := materializer.Plan(context.Background(), ProbePlanRequest{
		Candidate: caltrace.Candidate{
			ProviderID:   "provider_cli",
			CapabilityID: "text.write_file",
			Execution: core.Execution{
				Kind: core.ExecutionKindCLI,
				Spec: map[string]any{core.ExecutionSpecArgs: []any{"write-note", "--out", "{{target}}"}},
			},
		},
		WorkDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("second Plan() error = %v", err)
	}
	if secondPlan.Verifier.ID != plan.Verifier.ID {
		t.Fatalf("second verifier = %#v, want stable rewritten id %q", secondPlan.Verifier, plan.Verifier.ID)
	}
	target := plan.Inputs["target"].(string)
	if err := os.WriteFile(target, []byte("probe ok"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	evidence, _, err := runtime.NewRegistry().Verify(context.Background(), plan.Verifier, plan.Inputs)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if len(evidence) != 1 || evidence[0].ID != plan.Verifier.ID {
		t.Fatalf("evidence = %#v, want generated verifier evidence", evidence)
	}
}

func TestMaterializerGeneratedVerifierRequiresAvailableHome(t *testing.T) {
	t.Setenv("CAL_HOME", filepath.Join(t.TempDir(), "missing"))
	materializer := mustParse(t, `{
		"verifier_packages": [{
			"id": "requires_home",
			"verify_py": "print('{}')\n"
		}],
		"candidates": [{
			"provider_id": "provider_cli",
			"capability_id": "text.write_file",
			"description": "Write a text file artifact.",
			"execution": {"kind": "cli", "spec": {"args": ["write-note", "--out", "{{target}}"]}}
		}],
		"probe_plans": [{
			"candidate_index": 0,
			"inputs": {"target": "{{workdir}}/note.txt"},
			"verifier": {"id": "requires_home"}
		}]
	}`)

	_, err := materializer.Plan(context.Background(), ProbePlanRequest{
		Candidate: caltrace.Candidate{
			ProviderID:   "provider_cli",
			CapabilityID: "text.write_file",
			Execution: core.Execution{
				Kind: core.ExecutionKindCLI,
				Spec: map[string]any{core.ExecutionSpecArgs: []any{"write-note", "--out", "{{target}}"}},
			},
		},
		WorkDir: t.TempDir(),
	})
	if err == nil || !strings.Contains(err.Error(), "not available") {
		t.Fatalf("Plan() error = %v, want unavailable home error", err)
	}
}

func TestMaterializerPlansFileExistsProbe(t *testing.T) {
	installProposalTestVerifiers(t, "file_exists")
	materializer := mustParse(t, `{
		"candidates": [{
			"provider_id": "provider_cli",
			"capability_id": "text.write_file",
			"description": "Write a text file artifact.",
			"execution": {"kind": "cli", "spec": {"args": ["write-note", "--out", "{{target}}"]}}
		}],
		"probe_plans": [{
			"candidate_index": 0,
			"inputs": {"target": "{{workdir}}/note.txt"},
			"verifier": {"id": "file_exists"}
		}]
	}`)

	plan, err := materializer.Plan(context.Background(), ProbePlanRequest{
		Candidate: caltrace.Candidate{
			ProviderID:   "provider_cli",
			CapabilityID: "text.write_file",
			Execution: core.Execution{
				Kind: core.ExecutionKindCLI,
				Spec: map[string]any{core.ExecutionSpecArgs: []any{"write-note", "--out", "{{target}}"}},
			},
		},
		WorkDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.Verifier.ID != "file_exists" {
		t.Fatalf("verifier = %#v, want file_exists", plan.Verifier)
	}
}

func TestMaterializerAcceptsVerifierPackageInputsAtVerificationTime(t *testing.T) {
	installProposalTestVerifiers(t, "file_exists")
	materializer := mustParse(t, `{
		"candidates": [{
			"provider_id": "provider_cli",
			"capability_id": "text.write_file",
			"description": "Write a text file artifact.",
			"execution": {"kind": "cli", "spec": {"args": ["write-note", "--out", "{{target}}"]}}
		}],
		"probe_plans": [{
			"candidate_index": 0,
			"inputs": {"target": "{{workdir}}/note.txt"},
			"verifier": {"id": "file_exists"}
		}]
		}`)

	plan, err := materializer.Plan(context.Background(), ProbePlanRequest{
		Candidate: caltrace.Candidate{
			ProviderID:   "provider_cli",
			CapabilityID: "text.write_file",
			Execution: core.Execution{
				Kind: core.ExecutionKindCLI,
				Spec: map[string]any{core.ExecutionSpecArgs: []any{"write-note", "--out", "{{target}}"}},
			},
		},
		WorkDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.Verifier.ID != "file_exists" {
		t.Fatalf("verifier = %#v, want file_exists", plan.Verifier)
	}
}

func installProposalTestVerifiers(t *testing.T, ids ...string) {
	t.Helper()
	t.Setenv("CAL_HOME", t.TempDir())
	for _, id := range ids {
		err := runtime.InstallVerifier(runtime.GeneratedVerifierPackage{
			ID:       id,
			VerifyPY: "import json\nimport sys\nrequest = json.load(sys.stdin)\nverifier_id = request['verifier']['id']\nprint(json.dumps({'passed': True, 'evidence': [{'id': verifier_id, 'type': verifier_id}], 'outputs': request.get('inputs') or {}}))\n",
		})
		if err != nil {
			t.Fatalf("InstallVerifier(%q) error = %v", id, err)
		}
	}
}
