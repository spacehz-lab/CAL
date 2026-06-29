package proposalflow

import (
	"encoding/json"

	sharedllm "github.com/spacehz-lab/cal/internal/llm"
)

func cliSurfacePrompt(req Request, prof profile) sharedllm.Prompt {
	return jsonPrompt(cliSurfaceSystemPrompt, map[string]any{
		"provider":          req.Provider,
		"observations":      req.Observations,
		"debug_filter":      req.DebugFilter,
		"max_surface_items": prof.maxSurfaceItems,
	})
}

func cliCapabilityPrompt(req Request, prof profile, policy CapabilityPolicy, surfaces []surface) sharedllm.Prompt {
	return jsonPrompt(cliCapabilitySystemPrompt, map[string]any{
		"provider":              req.Provider,
		"surface_items":         capabilityPromptSurfaces(surfaces),
		"existing_capabilities": existingCapabilities(req, prof.maxCapabilities*3),
		"capability_policy":     policy,
		"debug_filter":          req.DebugFilter,
		"max_capabilities":      prof.maxCapabilities,
	})
}

func capabilityPromptSurfaces(surfaces []surface) []capabilitySurface {
	items := make([]capabilitySurface, 0, len(surfaces))
	for _, surface := range surfaces {
		items = append(items, capabilitySurface{
			ID:          surface.ID,
			Kind:        surface.Kind,
			Name:        surface.Name,
			Description: surface.Description,
		})
	}
	return items
}

func cliBindingPrompt(req Request, prof profile, capability capabilityPlan, surfaces []surface) sharedllm.Prompt {
	return jsonPrompt(cliBindingSystemPrompt, map[string]any{
		"provider":                      req.Provider,
		"observations":                  req.Observations,
		"surface_items":                 relevantSurfaces(surfaces, capability.SourceSurfaceIDs),
		"capability_plan":               capability,
		"max_candidates_per_capability": prof.maxCandidatesPerCapability,
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

func relevantSurfaces(surfaces []surface, ids []string) []surface {
	if len(ids) == 0 {
		return surfaces
	}
	wanted := map[string]struct{}{}
	for _, id := range ids {
		wanted[id] = struct{}{}
	}
	filtered := make([]surface, 0, len(surfaces))
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
Surface is only an entry-point inventory. Do not choose capability_id, execution args, probe inputs, or verifiers. Use kind only from command, subcommand, mode, option. Use command for primary commands, subcommand for documented nested commands, mode for documented operating modes, and option when a CLI exposes useful behavior primarily through flags. For command-list CLIs, include documented primary commands broadly up to max_surface_items. For flag-driven CLIs, emit each documented operation flag as kind="option" instead of subcommand. Descriptions must be grounded in observations and must not infer broader reusable capability semantics. Prefer primary commands and command families over enumerating every algorithm, format, cipher, digest, flag variant, alias, or metadata-only entry. Use keep when the current observation, command name, or option name gives a stable reusable operation meaning that is suitable for initial Capability planning. Do not defer solely because a surface is state-changing, network-dependent, configuration-changing, destructive, or may require confirmation; those risks belong to later Binding, Verification, or Use policy. For complex command-list CLIs, do not keep every primary command. Keep commands with clear common operation names, clear data/object names, or explicit descriptions that identify a reusable operation, including core state-changing operations such as install, update, upgrade, uninstall, link, unlink, pin, unpin, tap, untap, and cleanup when documented. Use defer only when the current observation is too shallow or ambiguous to infer a stable semantic operation, or when the surface is interactive, server/listener, protocol-specific, low-level, or clearly needs command-specific help, man output, or safer inspection before reliable Capability planning. Use skip for metadata-only, self-documentation, or alias-only entries.`

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
Only produce provider-specific candidate executions and probe material. Do not verify, produce verifier material, or claim success. For CLI providers, execution.kind must be "cli". CLI args must be argument array only and must not include the provider executable path or executable name. Every {{placeholder}} in args or stdout_path_input must have a probe input or fixture. input_constraints may only describe inputs referenced by execution. Prefer one candidate. Return more than one candidate only when observations clearly show different execution families or input modes, and never exceed max_candidates_per_capability. If the primary output is stdout and later verification needs a file artifact, set stdout_path_input to the output path input. If an output artifact is checked later, execution must produce it through an arg placeholder or stdout_path_input. Description must be provider-independent and no broader than the execution.`

const cliEvidenceSystemPrompt = `Return only JSON. For one candidate and probe material, choose how CAL should verify it and propose deterministic built-in checks when execution is used.
Response shape: {"verify":{"level":"L0|L1|L2|L3","method":"execute|contract","checks":[{"subject":"exit_code|stdout|stderr|output|source|target|artifact","predicate":"equals|not_equals|exists|non_empty|format|contains|contains_any|regex|bytes_equal_transform|hash_line_matches","params":{}}]}}.
Do not write code, scripts, verifier packages, or pass/fail claims. Use method="execute" only for safe, short, local, read-only probes; CAL executes the candidate and evaluates checks locally. Even execute+L1 must have checks. Use method="contract" when a real probe would install, remove, update, upgrade, clean, link, unlink, tap, untap, edit, start services, require network, require interaction, or change external state. Dry-run flags do not by themselves make package-manager update or upgrade commands safe. Contract verification cannot exceed L1, must use checks:[], and must not include pass/fail claims from execution. Use L3 when execute checks verify the semantic result itself, L2 for output structure or key properties, L1 for execution shape, parameter binding, operation path, safe failure-path evidence, or documented contract evidence, and L0 when reliable verification evidence is unavailable. Checks may reference exit_code, stdout, stderr, output, or path inputs such as source and target. Use params.value for equals, not_equals, and contains. Use artifact with params.input when the artifact path input is not target.`
