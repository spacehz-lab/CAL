package binding

import (
	"encoding/json"
	"strings"

	"github.com/spacehz-lab/cal/internal/llm"
)

const promptKeySelectedSurfaceItems = "selected_surface_items"
const promptKeyAcquisitionHint = "acquisition_hint"

const systemPrompt = `Return only JSON. For one planned capability, produce provider-specific CLI candidate executions and controlled probe material.

Goal:
Stage 3 is Binding. It materializes one provider-independent capability into possible provider-specific CLI invocations and controlled probe material that later Evidence can use.

Hard requirement:
When selected_surface_items is non-empty, candidates must be non-empty. Do not output {"candidates":[],"probe_material":[]} for a non-empty selected_surface_items payload.

Response shape:
{"candidates":[{"capability_id":"same as plan","description":"...","execution":{"kind":"cli","spec":{"args":["subcommand","{{source}}","{{target}}"],"stdout_path_input":"target"}}}],"probe_material":[{"candidate_index":0,"inputs":{"source":"{{workdir}}/input.txt","target":"{{workdir}}/output.artifact"},"fixtures":[{"input":"source","filename":"input.txt","content":"hello"}]}]}.

Boundary:
- Only produce provider-specific candidate executions and controlled probe material.
- Do not choose, create, or rename capability_id.
- Do not verify, choose verify level, produce checks, claim success, or decide promotion.
- Do not output verifier material, separate input schemas, or hidden reasoning steps.
- Output only the JSON object.
- selected_surface_items are the surfaces chosen by Capability for this planned capability. Do not reselect unrelated surfaces.
- Binding is not a filter stage. Do not reject a selected surface because the capability_id or capability_plan.description is broader, generic, or named differently than the concrete provider invocation.
- Use selected_surface_items as the primary source for candidate construction.
- acquisition_hint is an optional natural-language narrowing hint from the caller. Use it only to choose concrete runtime-controlled values that are already supported by selected_surface_items, such as format, algorithm, mode, or variant values.
- Use observations only to recover missing invocation details for selected surfaces, such as flags, operand meaning, argument names, default behavior, and stdout/file output behavior.
- Do not use acquisition_hint or observations to discover unrelated surfaces, replace selected_surface_items, or re-evaluate whether the capability should exist.
- If selected_surface_items is non-empty and any selected surface has a name, usage, mode, option, command, or observed invocation details, return at least one best-effort candidate and matching probe material.
- Do not return empty candidates merely because the invocation uses positional operands, optional operands, defaults, stdout output, or a narrower execution than the capability description.
- Return empty candidates only when selected_surface_items is empty.

Internal decision process:
Before writing JSON, decide each candidate internally in this order:
1. Choose the most direct command, subcommand, mode, option path, or usage from selected_surface_items.
2. Build execution.spec.args as a JSON string array with one shell token per array item.
3. Do not include the provider executable path or executable name in args; CAL supplies the provider path.
4. Treat selected_surface_items[].usage as the strongest invocation hint when present; remove the executable name from args and convert documented placeholders such as <input> and <output> into runtime placeholders such as {{source}} and {{target}}.
5. If usage is absent, fall back to the selected surface name, description, and observations.
6. Map documented input file, path, source, or FILE operands to {{source}}; map output, target, destination, result, or OUTPUT operands to {{target}}.
7. Use {{placeholder}} only for runtime-controlled values such as {{source}}, {{target}}, {{format}}, or {{algorithm}}.
8. If acquisition_hint names a supported runtime value for a placeholder, use that value in probe_material.inputs instead of choosing an arbitrary default.
9. Provide a probe input or fixture for every placeholder used in args or stdout_path_input.
10. Keep probe input paths inside {{workdir}} or provide content through fixtures; do not reference real user files, global config, network resources, or external state.
11. Stdout is a valid primary output; do not return empty candidates merely because the CLI writes its result to stdout.
12. Use stdout_path_input only when the CLI writes the primary result to stdout and the candidate should capture stdout to a runtime-controlled path.
13. Do not set stdout_path_input when args already write the target file through an output placeholder.
14. Prefer one most direct, probeable candidate.
15. Return multiple candidates only when observations clearly show distinct execution families or input modes.

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
- stdout_path_input must name an output path input such as "target"; it must not name an input source such as "source", "input", or "stdin".
- Do not set stdout_path_input when args already include an output path placeholder such as --output {{target}}, --out {{target}}, or -o {{target}}.
- If stdout_path_input is set, probe_material.inputs must contain the same input name.
- Every {{placeholder}} in args or stdout_path_input must have a probe input or fixture.
- Every candidate must have exactly one probe_material record with candidate_index set to that candidate's zero-based index.
- If a candidate has no placeholders, still include a probe_material record with "inputs":{} for that candidate_index.
- Prefer one candidate.
- Return more than one candidate only when observations clearly show different execution families or input modes.
- Never exceed max_candidates_per_capability.
- If an output artifact is represented by args, execution must produce it through an arg placeholder.
- Description must describe the actual execution and may be narrower than capability_plan.description.`

func prompt(req *Request) *llm.Request {
	payload, _ := json.Marshal(map[string]any{
		"provider":                      req.Provider,
		"observations":                  req.Observations,
		promptKeySelectedSurfaceItems:   req.Surfaces,
		"capability_plan":               req.Capability,
		promptKeyAcquisitionHint:        strings.TrimSpace(req.Hint),
		"max_candidates_per_capability": req.MaxCandidates,
	})
	return &llm.Request{System: systemPrompt, User: string(payload), JSON: true}
}
