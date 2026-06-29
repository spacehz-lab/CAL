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

func cliCapabilityPrompt(req Request, prof profile, policy CapabilityPolicy, surfaces []surfaceItem) sharedllm.Prompt {
	return jsonPrompt(cliCapabilitySystemPrompt, map[string]any{
		"provider":              req.Provider,
		"surface_items":         capabilityPromptSurfaces(surfaces),
		"existing_capabilities": existingCapabilities(req, prof.maxCapabilities*3),
		"capability_policy":     policy,
		"debug_filter":          req.DebugFilter,
		"max_capabilities":      prof.maxCapabilities,
	})
}

func capabilityPromptSurfaces(surfaces []surfaceItem) []capabilitySurfaceItem {
	items := make([]capabilitySurfaceItem, 0, len(surfaces))
	for _, surface := range surfaces {
		items = append(items, capabilitySurfaceItem{
			ID:          surface.ID,
			Kind:        surface.Kind,
			Name:        surface.Name,
			Description: surface.Description,
		})
	}
	return items
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
Response shape: {"surface_items":[{"id":"s1","kind":"command|subcommand|mode|option","name":"...","description":"...","evidence_source":"help|man|stdout","decision":"keep|defer|skip"}]}.
Surface is only an entry-point inventory. Do not choose capability_id, execution args, probe inputs, or verifiers. Use kind only from command, subcommand, mode, option. Use command for primary commands, subcommand for documented nested commands, mode for documented operating modes, and option when a CLI exposes useful behavior primarily through flags. For command-list CLIs, include documented primary commands broadly up to max_surface_items; do not omit a primary command merely because it is risky, interactive, server/listener, network-dependent, destructive, or needs deeper observation. Mark those entries decision="defer" instead. For flag-driven CLIs, emit each documented operation flag as kind="option" instead of subcommand. Descriptions must be grounded in observations and must not infer broader reusable capability semantics. Prefer primary commands and command families over enumerating every algorithm, format, cipher, digest, flag variant, alias, or metadata-only entry. Use keep when the current observation, command name, or option name gives a stable reusable operation meaning that is suitable for initial Capability planning. For complex command-list CLIs, do not keep every primary command. Keep commands with clear common operation names, clear data/object names, or explicit descriptions that identify a reusable operation. Defer commands whose names are ambiguous, protocol-specific, interactive, network-dependent, server/listener, configuration-changing, destructive, or low-level utility surfaces. Use defer for documented commands that are real entry points but need command-specific help, man output, or safer inspection before reliable Capability planning. Use skip for metadata-only, self-documentation, or alias-only entries.`

const cliCapabilitySystemPrompt = `Return only JSON. Choose provider-independent capability ids for kept CLI surfaces.
Response shape: {"capabilities":[{"capability_id":"subject.operation","description":"...","source_surface_ids":["s1"],"confidence":"high|medium|low"}]}.
Capability is a semantic planning stage only. Do not produce execution args, probes, verifiers, input schemas, or binding constraints.
capability_id must be exactly two lowercase dotted parts: <subject>.<operation>. Choose subject from capability_policy.preferred_subjects and operation from capability_policy.preferred_operations whenever a semantically correct combination exists. Do not create a protocol-specific, format-specific, algorithm-specific, or implementation-specific subject when a supplied generic subject preserves the reusable operation meaning. Create a new lowercase subject or operation only when observations cannot be expressed by supplied generic terms. New terms must be minimal, generic, reusable, and provider-independent.
capability_id must describe the reusable semantic operation, not the provider surface. Do not include provider names, executable names, command names, flags, paths, versions, formats, encodings, algorithms, modes, variants, target artifact kinds, or input/output media in capability_id.
Any discriminator that changes format, encoding, algorithm, mode, variant, target artifact kind, or concrete input/output shape belongs to later Binding inputs and constraints, not to capability_id.
Merge surfaces into the same capability only when they share the same semantic subject and operation and can plausibly share compatible Binding inputs, execution shape, and output semantics. Do not merge surfaces that require clearly different inputs, execution shapes, output semantics, command families, or opposite operations. Prefer the most direct source surface for each capability. Do not add extra source_surface_ids merely because another command can also participate in the broad operation. If the same semantic operation appears across different command families, prefer separate narrower capabilities or keep only the clearest directly executable source surface. source_surface_ids must only reference ids from the supplied surface_items.
Reuse an existing_capabilities id only when its id and description follow these rules and are semantically equivalent to the observed subject and operation. Create a new capability when no existing capability clearly describes the operation. Do not reuse invalid, provider-specific, command-specific, or discriminator-specific existing ids.
description must be provider-independent and must describe exactly the selected semantic capability. It must not add unsupported operations, hide opposite operations, or narrow capability_id with a format, encoding, algorithm, mode, or target artifact discriminator.
If debug_filter is set, only return that exact capability_id when the supplied surfaces support it and it satisfies these rules. Otherwise return no capabilities.`

const cliBindingSystemPrompt = `Return only JSON. For one planned capability, produce provider-specific CLI candidate executions and safe probe material.
Response shape: {"candidates":[{"provider_id":"optional","capability_id":"same as plan","description":"...","input_constraints":{},"execution":{"kind":"cli","spec":{"args":["subcommand","{{source}}","{{target}}"],"stdout_path_input":"optional"}}}],"probe_material":[{"candidate_index":0,"inputs":{"source":"{{workdir}}/input.txt","target":"{{workdir}}/output.artifact"},"fixtures":[{"input":"source","filename":"input.txt","content":"hello"}]}]}.
CLI args must not include the executable path. Every {{placeholder}} in args must have a probe input or fixture. If an output artifact is checked later, execution must produce it through an arg placeholder or stdout_path_input. Description must be provider-independent and no broader than the execution.`

const cliEvidenceSystemPrompt = `Return only JSON. For one candidate and probe material, propose a deterministic verifier.
Response shape: {"verifier_packages":[{"id":"local_snake_case_id","description":"...","verify_py":"python3 source"}],"verifier":{"id":"local_snake_case_id"}}.
Use an existing verifier id only when it is already appropriate from observations and inputs. Otherwise include one generated verifier package. Verifier ids are lowercase snake_case and must not start with verifier_. verify_py reads JSON from stdin with keys verifier and inputs, inspects only declared inputs and produced artifacts, and prints {"passed":true,"evidence":[{"id":request["verifier"]["id"],"type":request["verifier"]["id"],"content":{...}}],"outputs":{...}} or {"passed":false,"error":{"code":"snake_case","message":"..."}}. Do not claim pass/fail; CAL executes it.`
