package proposal

import (
	"encoding/json"
	"sort"

	"github.com/spacehz-lab/cal/internal/core"
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
		"provider":              req.Provider,
		"candidate_index":       candidateIndex,
		"candidate":             candidate,
		"probe_material":        material,
		"observations":          req.Observations,
		"verify_subject_rules":  core.VerifySubjectRules(),
		"available_file_inputs": evidenceFileInputs(material),
	})
}

func evidenceFileInputs(material probeMaterial) []string {
	inputs := map[string]struct{}{}
	for input := range material.Inputs {
		if input != "" {
			inputs[input] = struct{}{}
		}
	}
	for _, fixture := range material.Fixtures {
		if fixture.Input != "" {
			inputs[fixture.Input] = struct{}{}
		}
	}
	names := make([]string, 0, len(inputs))
	for input := range inputs {
		names = append(names, input)
	}
	sort.Strings(names)
	return names
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

Goal:
Stage 1 builds a bounded entry-point inventory for later Capability planning. It does not decide capabilities or prove behavior.

Response shape:
{"surface_items":[{"id":"s1","kind":"command|subcommand|mode|option","name":"...","description":"...","evidence_source":"help|man|stdout","decision":"keep|defer|skip","reason":"short decision reason"}]}.

Boundary:
- Do not choose capability_id, execution args, probe inputs, verify specs, verifiers, or pass/fail outcomes.
- Surface can only decide whether an observed CLI surface is worth considering later.
- Descriptions and reasons must be grounded in observations and must not infer broader reusable capability semantics.

Internal decision process:
Before writing JSON, classify each candidate surface internally in this order:
1. Is it a documented CLI entry point in observations? If not, skip.
2. Is it only metadata, help, version, usage, self-documentation, or an alias without added operation coverage? If yes, skip.
3. Does the observed name or description provide a stable operation meaning suitable for Capability planning? If yes, keep.
4. Is it potentially useful but too shallow, ambiguous, interactive, server/listener, protocol-specific, low-level, or in need of command-specific help, man output, or safer inspection? If yes, defer.
5. Otherwise skip.
Do not output hidden reasoning steps. Output only the JSON object with one short reason per item.

Kind rules:
- Use kind only from command, subcommand, mode, option.
- Use command for primary commands and subcommand for documented nested commands.
- Use mode for documented operating modes.
- Use option when a CLI exposes useful behavior primarily through flags.
- For flag-driven CLIs, emit each documented operation flag as kind="option" instead of subcommand.

Keep rules:
- Use keep when the observation, command name, or option name gives a stable reusable operation meaning suitable for Capability planning.
- For command-list CLIs, include documented primary commands broadly up to max_surface_items.
- Prefer primary commands and command families over every algorithm, format, cipher, digest, flag variant, alias, or metadata-only entry.
- For complex command-list CLIs, do not keep every primary command.
- Keep commands with clear common operation names, clear data/object names, or explicit reusable-operation descriptions.
- Keep core state-changing operations such as install, update, upgrade, uninstall, link, unlink, pin, unpin, tap, untap, and cleanup when documented.
- Do not defer solely because a surface is state-changing, network-dependent, configuration-changing, destructive, or may require confirmation.
- Those risks belong to later Binding, Verification, or Use policy.

Defer rules:
- Use defer only when the current observation is too shallow or ambiguous to infer a stable semantic operation.
- Use defer when the surface is interactive, server/listener, protocol-specific, low-level, or clearly needs command-specific help, man output, or safer inspection.

Skip rules:
- Use skip for metadata-only, self-documentation, or alias-only entries.
- Skip individual algorithms, formats, ciphers, digests, flag variants, and metadata entries unless the CLI exposes useful behavior primarily through that option.`

const cliCapabilitySystemPrompt = `Return only JSON. Map kept CLI surfaces to provider-independent capability ids.

Goal:
Stage 2 is semantic planning. It names the reusable operation that later Binding and Evidence stages may implement and verify.

Response shape:
{"capabilities":[{"capability_id":"subject.operation","description":"...","source_surface_ids":["s1"],"confidence":"high|medium|low"}]}.

Boundary:
- Do not produce execution args, probes, verifiers, verify specs, pass/fail claims, or input schemas.
- Do not decide whether the provider-specific execution works.
- Do not output hidden reasoning steps. Output only the JSON object.

Internal decision process:
Before writing JSON, decide each capability internally in this order:
1. Identify the semantic subject of the kept surface without using provider, executable, command, flag, format, algorithm, mode, or artifact wording.
2. Identify the semantic operation direction. Opposite operations must stay separate.
3. Reuse an existing_capabilities id only when both its id and description are valid and semantically equivalent.
4. Use capability_policy.preferred_subjects and capability_policy.preferred_operations when they express the operation accurately.
5. Create a new subject or operation only when the supplied vocabulary cannot express the observed operation.
6. Merge source surfaces only when they describe the same subject and operation and can plausibly share compatible Binding inputs, execution shape, and output semantics.
7. Split or keep only the clearest directly executable source when surfaces differ by command family, required inputs, execution shape, output semantics, or operation direction.

Capability id rules:
- capability_id must be exactly two lowercase dotted parts: <subject>.<operation>.
- Choose subject from capability_policy.preferred_subjects whenever a semantically correct subject exists.
- Choose operation from capability_policy.preferred_operations whenever a semantically correct operation exists.
- Do not create protocol-specific, format-specific, algorithm-specific, or implementation-specific subjects when a supplied generic subject preserves the reusable operation meaning.
- Create a new lowercase subject or operation only when observations cannot be expressed by supplied generic terms.
- New terms must be minimal, generic, reusable, and provider-independent.
- capability_id must describe the reusable semantic operation, not the provider surface.
- Do not include provider names, executable names, command names, flags, paths, versions, formats, encodings, algorithms, modes, variants, target artifact kinds, or input/output media in capability_id.

Discriminator rules:
- Runtime-selectable or binding-specific details belong to later Binding execution inputs and Evidence checks, not capability_id.
- If the same reusable operation can vary by format, encoding, algorithm, mode, variant, artifact kind, or concrete I/O shape, keep the capability_id generic.
- Do not maintain a format-by-format or algorithm-by-algorithm whitelist. Apply the same discriminator rule to new future variants.

Grouping rules:
- Merge surfaces only when they share the same semantic subject and operation.
- Merged surfaces must plausibly share compatible Binding inputs, execution shape, and output semantics.
- Do not merge surfaces that require clearly different inputs, execution shapes, output semantics, command families, or opposite operations.
- Prefer the most direct source surface for each capability.
- Do not add extra source_surface_ids merely because another command can also participate in the broad operation.
- If the same semantic operation appears across different command families, prefer separate narrower capabilities or keep only the clearest directly executable source surface.
- source_surface_ids must only reference ids from the supplied surface_items.

Reuse and description rules:
- Reuse an existing_capabilities id only when its id and description follow these rules and are semantically equivalent.
- Create a new capability when no existing capability clearly describes the operation.
- Do not reuse invalid, provider-specific, command-specific, or discriminator-specific existing ids.
- description must be provider-independent and must describe exactly the selected semantic capability.
- description must not add unsupported operations, hide opposite operations, or narrow capability_id with a format, encoding, algorithm, mode, or target artifact discriminator.
- confidence="high" only when the kept surface name and description clearly support the semantic capability.
- Use confidence="medium" when the operation is supported but the subject, direction, or grouping is partly inferred.
- Use confidence="low" when the mapping is weak but still worth exploring; omit the capability if it is not supported by the supplied surface_items.
- If debug_filter is set, only return that exact capability_id when the supplied surfaces support it and it satisfies these rules.
- Otherwise return no capabilities.`

const cliBindingSystemPrompt = `Return only JSON. For one planned capability, produce provider-specific CLI candidate executions and controlled probe material.

Goal:
Stage 3 is Binding. It materializes one provider-independent capability into possible provider-specific CLI invocations and controlled probe material that later Evidence can use.

Response shape:
{"candidates":[{"provider_id":"optional","capability_id":"same as plan","description":"...","execution":{"kind":"cli","spec":{"args":["subcommand","{{source}}","{{target}}"],"stdout_path_input":"optional"}}}],"probe_material":[{"candidate_index":0,"inputs":{"source":"{{workdir}}/input.txt","target":"{{workdir}}/output.artifact"},"fixtures":[{"input":"source","filename":"input.txt","content":"hello"}]}]}.

Boundary:
- Only produce provider-specific candidate executions and controlled probe material.
- Do not choose, create, or rename capability_id.
- Do not verify, choose verify level, produce checks, claim success, or decide promotion.
- Do not output verifier material, separate input schemas, or hidden reasoning steps.
- Output only the JSON object.

Internal decision process:
Before writing JSON, decide each candidate internally in this order:
1. Choose the most direct observed command, subcommand, mode, or option path that supports the current capability.
2. Build execution.spec.args as a JSON string array with one shell token per array item.
3. Do not include the provider executable path or executable name in args; CAL supplies the provider path.
4. Use {{placeholder}} only for runtime-controlled values such as {{source}}, {{target}}, {{format}}, or {{algorithm}}.
5. Provide a probe input or fixture for every placeholder used in args or stdout_path_input.
6. Keep probe input paths inside {{workdir}} or provide content through fixtures; do not reference real user files, global config, network resources, or external state.
7. Use stdout_path_input only when the CLI writes the primary result to stdout.
8. Do not set stdout_path_input when args already write the target file through an output placeholder.
9. Prefer one most direct, probeable candidate.
10. Return multiple candidates only when observations clearly show distinct execution families or input modes.

Execution rules:
- For CLI providers, execution.kind must be "cli".
- execution.spec.args must be a JSON array of strings, never a shell command string.
- Split every command token into a separate array item.
- Do not join multiple shell tokens into one string.
- Do not use shell pipes, redirects, command chaining, or shell-specific syntax unless observations show the provider surface itself requires that shape and it cannot be split.
- CLI args must not include the provider executable path or executable name.
- execution.spec.stdout_path_input, when present, must be a string input name such as "target".
- Do not use an object, array, path, or {{placeholder}} for stdout_path_input.
- Use stdout_path_input only when the CLI writes the primary artifact to stdout.
- Do not set stdout_path_input when args already include an output path placeholder such as --output {{target}}, --out {{target}}, or -o {{target}}.
- Every {{placeholder}} in args or stdout_path_input must have a probe input or fixture.
- Prefer one candidate.
- Return more than one candidate only when observations clearly show different execution families or input modes.
- Never exceed max_candidates_per_capability.
- If the primary output is stdout and later verification needs a file artifact, set stdout_path_input to the output path input.
- If an output artifact is checked later, execution must produce it through an arg placeholder or stdout_path_input.
- Description must be provider-independent and no broader than the execution.`

const cliEvidenceSystemPrompt = `Return only JSON. For one candidate and probe material, plan how CAL should collect deterministic verification evidence.

Goal:
Stage 4 is Evidence. It only proposes verify method and checks so Proposal can derive the final verify level locally, then later Verification can execute the candidate and evaluate built-in checks or record weak contract evidence when execution is unsafe.

Response shape:
{"verify":{"method":"execute|contract","checks":[{"subject":{"type":"file|stdout|stderr|exit_code","input":"file input only"},"predicate":"equals|not_equals|exists|non_empty|format|contains|contains_any|regex|bytes_equal_transform|hash_line_matches","params":{}}]}}.

Boundary:
- Do not execute the candidate.
- Do not claim pass/fail or decide promotion.
- Do not write code, scripts, verifier packages, or natural-language oracles.
- Use only subjects, predicates, and params allowed by verify_subject_rules.
- For subject.type="file", subject.input must be one of available_file_inputs.
- Do not output verify.level. CAL derives level locally from method and checks.
- Output only the JSON object.

Internal decision process:
Before writing JSON, decide the VerifySpec internally in this order:
1. If the probe is safe, short, local, reads only probe fixtures, and writes only declared probe outputs inside the probe workdir, use method="execute".
2. If execution would install, remove, update, upgrade, clean, link, unlink, tap, untap, edit, start services, require network, require interaction, or change external state, use method="contract".
3. For contract, checks are advisory and are not executed.
4. For execute, choose the strongest built-in deterministic checks supported by probe_material and observations.
5. Use checks that prove the result, not the capability name or candidate description.
6. Do not use fixture-only sample content as durable expected output unless observations explicitly say the command always emits that literal.

Check rules:
- CAL executes the candidate and evaluates checks locally.
- Dry-run flags do not by themselves make package-manager update or upgrade commands safe.
- Contract verification does not execute checks and must not include pass/fail claims from execution.
- Checks must follow verify_subject_rules exactly.
- For subject.type="file", subject.input must be one of available_file_inputs.
- Do not invent subject types, file inputs, predicates, or params outside verify_subject_rules.
- A file subject input names a path input; it is not file content.
- Use file subjects for artifact path checks and file content checks.
- For generated file artifacts, prefer exists, non_empty, and format when they prove the output shape.
- Use file contains, contains_any, or regex only when observations explicitly guarantee stable literal output content.
- For stable literal output content, prefer contains over anchored full-file regex. Use anchored regex only when observations specify the exact whole file content including newline behavior.
- Fixture content is probe-only sample input. Do not use fixture content as durable expected output unless observations explicitly say the command always emits that literal.
- Use stdout, stderr, and exit_code subjects only for process result checks.`
