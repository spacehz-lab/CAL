package proposalflow

import (
	"encoding/json"

	sharedllm "github.com/spacehz-lab/cal/internal/llm"
)

const (
	cliPromptVersion     = "proposalflow-cli-v1"
	cliProposalSource    = "llm"
	cliProposalSchema    = "proposalflow.v1"
	defaultMaxSurface    = 40
	defaultMaxCapability = 12
	defaultConcurrency   = 2
)

func cliProfile() profile {
	return profile{
		id:              "cli",
		maxSurfaceItems: defaultMaxSurface,
		maxCapabilities: defaultMaxCapability,
		concurrency:     defaultConcurrency,
	}
}

func cliSurfacePrompt(req Request, prof profile) sharedllm.Prompt {
	return jsonPrompt(cliSurfaceSystemPrompt, map[string]any{
		"provider":          req.Provider,
		"observations":      req.Observations,
		"debug_filter":      req.DebugFilter,
		"max_surface_items": prof.maxSurfaceItems,
	})
}

func cliCapabilityPrompt(req Request, prof profile, surfaces []surfaceItem) sharedllm.Prompt {
	return jsonPrompt(cliCapabilitySystemPrompt, map[string]any{
		"provider":                req.Provider,
		"surface_items":           surfaces,
		"existing_capability_ids": existingCapabilityIDs(req, prof.maxCapabilities*3),
		"debug_filter":            req.DebugFilter,
		"max_capabilities":        prof.maxCapabilities,
	})
}

func cliBindingPrompt(req Request, capability capabilityPlanItem, surfaces []surfaceItem) sharedllm.Prompt {
	return jsonPrompt(cliBindingSystemPrompt, map[string]any{
		"provider":        req.Provider,
		"observations":    req.Observations,
		"surface_items":   relevantSurfaces(surfaces, capability.SourceSurfaceIDs),
		"capability_plan": capability,
	})
}

func cliEvidencePrompt(req Request, candidateIndex int, candidate any, material probeMaterial) sharedllm.Prompt {
	return jsonPrompt(cliEvidenceSystemPrompt, map[string]any{
		"provider":        req.Provider,
		"candidate_index": candidateIndex,
		"candidate":       candidate,
		"probe_material":  material,
		"observations":    req.Observations,
	})
}

func jsonPrompt(system string, value any) sharedllm.Prompt {
	content, _ := json.Marshal(value)
	return sharedllm.Prompt{System: system, User: string(content)}
}

func relevantSurfaces(surfaces []surfaceItem, ids []string) []surfaceItem {
	if len(ids) == 0 {
		return surfaces
	}
	wanted := map[string]struct{}{}
	for _, id := range ids {
		wanted[id] = struct{}{}
	}
	filtered := make([]surfaceItem, 0, len(surfaces))
	for _, surface := range surfaces {
		if _, ok := wanted[surface.ID]; ok {
			filtered = append(filtered, surface)
		}
	}
	if len(filtered) == 0 {
		return surfaces
	}
	return filtered
}

const cliSurfaceSystemPrompt = `Return only JSON. Extract documented CLI operation surfaces from observations.
Response shape: {"surface_items":[{"id":"s1","kind":"command|subcommand|mode|option","name":"...","description":"...","evidence_source":"help|man|stdout","decision":"keep|defer|skip","rationale":"..."}]}.
Surface is only an entry-point inventory. Do not choose capability_id, execution args, probe inputs, or verifiers. Use kind only from command, subcommand, mode, option. Use command for primary commands, subcommand for documented nested commands, mode for documented operating modes, and option when a CLI exposes useful behavior primarily through flags. For command-list CLIs, include documented primary commands broadly up to max_surface_items; do not omit a primary command merely because it is risky, interactive, server/listener, network-dependent, destructive, or needs deeper observation. Mark those entries decision="defer" instead. For flag-driven CLIs, emit each documented operation flag as kind="option" instead of subcommand. Descriptions must be grounded in observations and must not infer broader reusable capability semantics. Prefer primary commands and command families over enumerating every algorithm, format, cipher, digest, flag variant, alias, or metadata-only entry. Use keep when a documented command or option is enough to seed Capability planning, even if later stages need deeper command help. Use defer only when the surface should not be planned from the current observation because it is interactive, server/listener, network-dependent, destructive, or lacks a stable operation meaning. Use skip for metadata-only, self-documentation, or alias-only entries.`

const cliCapabilitySystemPrompt = `Return only JSON. Choose provider-independent capability ids for kept CLI surfaces.
Response shape: {"capabilities":[{"capability_id":"subject.operation","description":"...","source_surface_ids":["s1"],"confidence":"high|medium|low","rationale":"..."}]}.
Reuse an existing_capability_ids value only when semantically equivalent. New ids must match lowercase <subject>.<operation> and must not include provider names, executable names, command names, flags, paths, versions, random suffixes, formats, encodings, algorithms, modes, or target artifact kinds. Runtime discriminators belong in Binding inputs and constraints. If debug_filter is set, only return that matching capability when observations support it.`

const cliBindingSystemPrompt = `Return only JSON. For one planned capability, produce provider-specific CLI candidate executions and safe probe material.
Response shape: {"candidates":[{"provider_id":"optional","capability_id":"same as plan","description":"...","input_constraints":{},"execution":{"kind":"cli","spec":{"args":["subcommand","{{source}}","{{target}}"],"stdout_path_input":"optional"}},"rationale":"..."}],"probe_material":[{"candidate_index":0,"inputs":{"source":"{{workdir}}/input.txt","target":"{{workdir}}/output.artifact"},"fixtures":[{"input":"source","filename":"input.txt","content":"hello"}],"rationale":"..."}]}.
CLI args must not include the executable path. Every {{placeholder}} in args must have a probe input or fixture. If an output artifact is checked later, execution must produce it through an arg placeholder or stdout_path_input. Description must be provider-independent and no broader than the execution.`

const cliEvidenceSystemPrompt = `Return only JSON. For one candidate and probe material, propose a deterministic verifier.
Response shape: {"verifier_packages":[{"id":"local_snake_case_id","description":"...","verify_py":"python3 source"}],"verifier":{"id":"local_snake_case_id"},"rationale":"..."}.
Use an existing verifier id only when it is already appropriate from observations and inputs. Otherwise include one generated verifier package. Verifier ids are lowercase snake_case and must not start with verifier_. verify_py reads JSON from stdin with keys verifier and inputs, inspects only declared inputs and produced artifacts, and prints {"passed":true,"evidence":[{"id":request["verifier"]["id"],"type":request["verifier"]["id"],"content":{...}}],"outputs":{...}} or {"passed":false,"error":{"code":"snake_case","message":"..."}}. Do not claim pass/fail; CAL executes it.`
