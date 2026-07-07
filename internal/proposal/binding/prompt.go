package binding

import (
	"encoding/json"

	"github.com/spacehz-lab/cal/internal/llm"
)

const promptKeySelectedSurfaceItems = "selected_surface_items"

const systemPrompt = `Return only JSON. For one planned capability, produce provider-specific CLI candidate executions and controlled probe material.

Goal:
Stage 3 is Binding. It materializes one provider-independent capability into possible provider-specific CLI invocations and controlled probe material that later Evidence can use.

Response shape:
{"candidates":[{"capability_id":"same as plan","description":"...","execution":{"kind":"cli","spec":{"args":["subcommand","{{source}}","{{target}}"],"stdout_path_input":"target"}}}],"probe_material":[{"candidate_index":0,"inputs":{"source":"{{workdir}}/input.txt","target":"{{workdir}}/output.artifact"},"fixtures":[{"input":"source","filename":"input.txt","content":"hello"}]}]}.

Boundary:
- Only produce provider-specific candidate executions and controlled probe material.
- Do not choose, create, or rename capability_id.
- Do not verify, choose verify level, produce checks, claim success, or decide promotion.
- Do not output verifier material, separate input schemas, or hidden reasoning steps.
- Output only the JSON object.
- selected_surface_items are the surfaces chosen by Capability for this planned capability. Do not reselect unrelated surfaces.
- If a selected surface gives a direct command, subcommand, mode, option, or usage for the planned capability, return at least one candidate and matching probe material.
- Return empty candidates only when every selected surface lacks both an executable name/path and any observed command, usage, mode, or option path that can be mapped to CLI args.

Internal decision process:
Before writing JSON, decide each candidate internally in this order:
1. Choose the most direct observed command, subcommand, mode, or option path that supports the current capability.
2. Build execution.spec.args as a JSON string array with one shell token per array item.
3. Do not include the provider executable path or executable name in args; CAL supplies the provider path.
4. Treat selected_surface_items[].usage as the strongest invocation hint when present; convert documented placeholders such as <input> and <output> into runtime placeholders such as {{source}} and {{target}}.
5. If usage is absent, fall back to the selected surface name, description, and observations.
6. Use {{placeholder}} only for runtime-controlled values such as {{source}}, {{target}}, {{format}}, or {{algorithm}}.
7. Provide a probe input or fixture for every placeholder used in args or stdout_path_input.
8. Keep probe input paths inside {{workdir}} or provide content through fixtures; do not reference real user files, global config, network resources, or external state.
9. Use stdout_path_input only when the CLI writes the primary result to stdout.
10. Do not set stdout_path_input when args already write the target file through an output placeholder.
11. Prefer one most direct, probeable candidate.
12. Return multiple candidates only when observations clearly show distinct execution families or input modes.

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
- Every candidate must have exactly one probe_material record with candidate_index set to that candidate's zero-based index.
- If a candidate has no placeholders, still include a probe_material record with "inputs":{} for that candidate_index.
- Prefer one candidate.
- Return more than one candidate only when observations clearly show different execution families or input modes.
- Never exceed max_candidates_per_capability.
- If the primary output is stdout and later verification needs a file artifact, set stdout_path_input to the output path input.
- If an output artifact is checked later, execution must produce it through an arg placeholder or stdout_path_input.
- Description must be provider-independent and no broader than the execution.`

func prompt(req *Request) *llm.Request {
	payload, _ := json.Marshal(map[string]any{
		"provider":                      req.Provider,
		"observations":                  req.Observations,
		promptKeySelectedSurfaceItems:   req.Surfaces,
		"capability_plan":               req.Capability,
		"max_candidates_per_capability": req.MaxCandidates,
	})
	return &llm.Request{System: systemPrompt, User: string(payload), JSON: true}
}
