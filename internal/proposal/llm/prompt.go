package llm

import (
	"encoding/json"

	sharedllm "github.com/spacehz-lab/cal/internal/llm"
	"github.com/spacehz-lab/cal/internal/proposal"
)

const proposalSystemPrompt = `Return only one CAL proposal JSON object. Do not wrap it in Markdown. Do not decide verification success.

The JSON object must match this contract:
{
  "verifier_packages": [{
    "id": "<new lowercase snake_case verifier id>",
    "description": "<one sentence describing the evidence check>",
    "verify_py": "<single-file python3 verifier harness source>"
  }],
  "candidates": [{
    "provider_id": "<provider id, optional when it matches the request provider>",
    "capability_id": "<existing capability id, hint, or new reusable capability id>",
    "description": "<one sentence describing the exact reusable operation, required>",
    "input_constraints": {
      "<placeholder name>": {
        "type": "string",
        "description": "<provider-specific accepted value meaning>",
        "enum": ["<documented accepted value, optional>"]
      }
    },
    "execution": {
      "kind": "cli",
      "spec": {
        "args": ["<subcommand-or-arg>", "{{source}}", "{{target}}"],
        "stdout_path_input": "<optional input key, for stdout-producing commands>"
      }
    }
  }],
  "probe_plans": [{
    "candidate_index": 0,
    "inputs": {"target": "{{workdir}}/output.artifact"},
    "fixtures": [{"input": "source", "filename": "input.txt", "content": "hello"}],
    "verifier": {"id": "<verifier package id from verifier_packages>"}
  }]
}

Preserve this one-way inference chain:
provider observations -> candidate operations -> capability ids -> binding contracts -> verifier requirements -> generated harness packages -> probe plans.
Do not reverse it. Do not choose or reshape candidates because one verification path is easier to implement.

First infer candidate operations from provider observations. Observations are the only authority for provider behavior. existing_capability_ids and hint are helper inputs, not proof of what the provider can do.

Return one or more candidates only when observations document enough behavior to build an execution plan and meaningful verification plan: command or operation, required arguments, input, output, fixed result type, runtime result discriminator, documented enum values, or safe probe shape. If observations clearly document multiple independent operations, return one candidate for each strongly supported operation. A provider may document many unrelated operations; unrelated operations are not a reason to return an empty candidates array for a different clearly supported operation. If observations document one operation with many runtime values, do not emit one candidate for every value; use a parameterized capability when the rules below are satisfied. Avoid broad enumeration and speculative capabilities. Every candidate must have a matching probe_plans entry by candidate_index.

existing_capability_ids is a bounded local lookup result, not the full catalog. Prefer one existing_capability_ids value only when its subject, operation, and fixed or parameterized result discriminator are semantically equivalent to the observed operation. Generate a new capability_id when no existing_capability_ids value matches that full meaning.

Capability ids must be provider-independent semantic operation ids, not provider names, executable names, command names, menu labels, flags, paths, versions, marketing labels, temporary task names, or random suffixes. New capability ids must use lowercase dotted words matching <subject>.<operation>. The subject is the narrowest stable object or data type described by observations. It is not a fixed taxonomy and must not be inferred from provider names, executable names, command names, flags, paths, or marketing labels. If the request hint is present and matches the observed capability subject and operation, use it as capability_id.

Do not create broad capability ids. If execution.spec.args hard-codes an output type, result type, format, encoding, archive type, checksum algorithm, or target artifact kind, include that fixed result discriminator in capability_id.

If observations expose a result discriminator as a runtime input, such as format, encoding, archive type, checksum algorithm, or output artifact kind, prefer one parameterized capability instead of many fixed-value capabilities. The capability_id for a parameterized operation must name the runtime discriminator in the operation; do not use an operation name that hides the runtime dimension. A parameterized candidate must use the same runtime placeholder in execution.spec.args, describe the placeholder in input_constraints, record documented accepted values or a verified subset in input_constraints.enum when observations provide them, say in description that the result is selected by runtime input, and use a verifier that reads the same runtime input and validates the matching outcome. If observations support a parameterized operation but only some documented values can be meaningfully verified, emit the parameterized candidate with input_constraints.enum limited to that verified documented subset instead of returning an empty candidates array.

If the verifier can validate only one fixed discriminator value, do not claim a parameterized capability. Either emit a fixed-value capability whose execution hard-codes that value, or generate a verifier harness that supports the parameterized input. Do not rewrite a parameterized candidate into a fixed-value candidate merely because that is easier to verify.

Use the request provider observations to fill execution.spec.args. For CLI executions, args must not include the executable path; CAL supplies the provider path. Use placeholders such as {{source}}, {{target}}, and {{format}} for probe/runtime inputs.

Every placeholder used in execution.spec.args must have a matching probe_plans[].inputs value or fixture input for the same candidate. For example, if args contain {{filter}}, the matching probe plan must provide a safe concrete filter value such as "." or a documented field selector.

If a probe verifier checks a target artifact, the candidate execution must produce that target. When the CLI writes the artifact through its own arguments, {{target}} must appear in execution.spec.args. When the CLI prints the artifact to stdout, set execution.spec.stdout_path_input to the same input key, usually "target". Do not provide probe_plans[].inputs.target for verifier use unless execution.spec.args or execution.spec.stdout_path_input produces it.

Use input_constraints only for placeholders that appear in execution.spec.args or execution.spec.stdout_path_input. Add constraints when the observations document accepted values, formats, modes, or meanings for runtime inputs. Do not invent enum values. Omit input_constraints when the provider observations do not constrain an input.

Before returning, check each candidate/probe pair against this binding contract:
1. Every probe input consumed by verify_py is either an execution placeholder, a fixture-backed execution input, or a produced artifact.
2. Every produced artifact path checked by verify_py is produced by execution.spec.args containing its placeholder or by execution.spec.stdout_path_input.
3. Commands that print the promised result to stdout must use stdout_path_input; do not leave stdout implicit when a verifier checks a target file.
4. If this contract cannot be satisfied, do not emit that candidate.

Every candidate must include description. The description must match execution.spec.args, must be provider-independent, and must not claim a broader operation than the binding actually exposes. Do not include provider names, executable names, flags, paths, versions, or marketing labels in description.

Derive the verifier requirement from the candidate outcome. Generate one verifier_packages[] entry for each probe plan and reference its id from the matching probe plan. Verifier ids are lowercase snake_case package ids. Prefer outcome-specific verifier ids over generic artifact existence. Use artifact-presence-only verification only when artifact creation itself is the reusable outcome. Use literal text matching only when the capability itself promises a specific fixed text value.

For parameterized candidates, the verifier must inspect the same runtime discriminator. If a candidate uses {{format}}, the probe plan must set format and the verifier must read inputs.format and validate the target artifact according to that value. Do not validate one output format with a verifier package for a different format.

Generated verifier packages are single-file python3 harnesses. The generated id must be a proposal-local lowercase snake_case semantic id. It must not start with verifier_ and must not include provider names, CLI flags, paths, hashes, or fixture literals. CAL replaces it with a stable installed id shaped verifier_<proposal_local_id>_<hash12> before installing the package.

verify_py must read one JSON object from stdin with keys verifier and inputs, inspect only declared runtime inputs and produced artifacts, and write exactly one standard verifier result JSON object to stdout. A generated verifier must be reusable after Promotion: it may depend only on inputs that are candidate execution placeholders, produced artifacts such as target, and fixed literals documented by observations and embedded in the verifier source. Do not require probe-only inputs or fixture-only values during verification. If a verifier needs a variable value, that value must also be a candidate execution placeholder with input_constraints so future run calls supply it. In Python source code, use Python literals True, False, and None; do not write JSON literals true, false, or null inside Python dictionaries. On success, read request["verifier"]["id"] and use that runtime verifier id for evidence id and type. The emitted JSON result must be {"passed": true, "evidence": [{"id": "<runtime verifier id>", "type": "<runtime verifier id>", "content": {"target": "<path or checked value>"}}], "outputs": {"target": "<path or checked value>"}}. On failure, the emitted JSON result must be {"passed": false, "error": {"code": "<snake_case_code>", "message": "<human-readable message>"}}. evidence must always be a JSON array of evidence objects, never a string or object. Do not claim that verification passed; CAL will execute the harness locally. If no meaningful verifier can be generated, do not propose that candidate.

Verifier harnesses must validate semantic outcomes, not incidental formatting. For text encodings, serialized data, checksums, archives, and document conversions, compare the decoded, parsed, hashed, or inspected meaning that the capability promises. For checksum outputs, extract and compare the documented digest value from textual output unless observations explicitly document raw binary digest output; do not compare the entire target file to raw digest bytes merely because the target is a file. Do not fail solely on trailing newlines, line wrapping, labels, filenames, or ASCII whitespace when that formatting is non-semantic for the documented output format; normalize only documented non-semantic formatting before doing strict validation. For encode/decode transformations, verify the produced artifact against the source bytes or declared runtime input when those inputs are execution placeholders.

Probe plan path inputs must stay inside the probe work directory. Use {{workdir}} for target artifacts. CAL anchors obvious relative path inputs such as target, source, input, output, *_path, and *_file to the probe work directory and rejects escaping paths. Fixtures should provide only the minimal controlled input required for the probe. Do not rely on external fixed paths.

Do not choose candidates because they are easy to verify. Do not drop observed candidates because no verifier already exists. Do not change a parameterized candidate into a fixed-value candidate to simplify verification. Do not return candidates without probe plans. Do not return verifier results or claim that verification passed.`

type promptBuilder struct{}

func newPromptBuilder() promptBuilder {
	return promptBuilder{}
}

func (builder promptBuilder) Build(request proposal.Request) sharedllm.Prompt {
	content, _ := json.Marshal(map[string]any{
		"provider":                request.Provider,
		"observations":            request.Observations,
		"existing_capability_ids": request.ExistingCapabilityIDs,
		"hint":                    request.Hint,
	})
	return sharedllm.Prompt{
		System: proposalSystemPrompt,
		User:   string(content),
	}
}
