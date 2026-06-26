# CAL Discovery LLM Prompt Design

This document defines the prompt contract for the CAL `LLM proposer`.

The LLM is allowed to propose candidate binding material, probe plans, and
generated verifier harness packages. It is not allowed to decide verification
success or Promotion.

## Goal

The prompt must preserve a one-way inference chain:

```text
provider observations
-> candidate operations
-> capability ids
-> binding contracts
-> verifier requirements
-> existing verifier ids or generated harness packages
-> probe plans
```

The reverse direction is forbidden:

```text
easy verifier harness
-> easier-to-verify candidate
```

`existing_capability_ids` is a helper input. It must not become the source of
truth for what the provider can do.

## Prompt Inputs

The user payload may include:

```text
provider
observations
existing_capability_ids
hint
```

`observations` are the only authority for provider behavior.

`existing_capability_ids` is a bounded local lookup result for reuse. It is not
the full local capability catalog and is not proof that an observed provider can
implement the capability.

## Required Output

The model must return exactly one proposal JSON object:

```text
proposal
  verifier_packages[] optional
    id
    description
    verify_py
  candidates[]
    provider_id optional
    capability_id
    description
    input_constraints optional
    execution
    rationale optional
  probe_plans[]
    candidate_index
    inputs
    fixtures optional
    verifier
    rationale optional
```

Every candidate must have one matching probe plan by `candidate_index`.

The model must not return Markdown, prose, self-reported provenance, or a
verification result.

## Candidate Operation Rules

The model must infer candidate operations from provider observations first.

Only emit a candidate when the observations document enough behavior to build an
execution plan and a meaningful verification plan. Useful evidence includes:

```text
command or operation
required arguments
input path or input value
output path or output value
fixed result type
runtime result discriminator
documented enum values
safe probe shape
```

If observations document multiple independent operations, the model may emit
multiple candidates. For example, encode and decode are independent operations.

A provider may document many unrelated operations. Unrelated operations are not
a reason to return an empty `candidates[]` array for a different clearly
supported operation.

If observations document one operation with many runtime values, do not emit one
candidate for every value. Use the parameterized capability rule below.

Do not emit speculative candidates. Do not infer capabilities from provider
names, executable names, marketing names, flag names alone, or verifier
availability.

## Capability ID Rules

Capability ids are provider-independent semantic operation ids.

They must not include:

```text
provider names
executable names
flags
paths
versions
marketing names
temporary task names
random suffixes
```

Selection order:

```text
1. Reuse a semantically matching existing_capability_ids value.
2. Generate a new id only when no existing id matches.
```

When generating a new id, use lowercase dotted words:

```text
<domain>.<verb_object>
```

## Fixed And Parameterized Results

A candidate must not claim a broader capability than its execution can provide.

If `execution.spec.args` hard-codes a result discriminator, include that
discriminator in the capability id. A result discriminator can be an output
type, result type, format, encoding, archive type, checksum algorithm, or target
artifact kind.

Examples:

```text
document.export_pdf
document.convert_html
plist.convert_json
archive.create_zip
file.compress_gzip
text.encode_base64
file.hash_sha1
```

If the provider exposes the discriminator as a runtime input, prefer one
parameterized capability instead of many fixed-value capabilities.

The capability id for a parameterized operation must name the runtime
discriminator in the operation, such as:

```text
document.convert_format
file.hash_algorithm
```

Do not use an overly broad id such as `document.convert` when the result is
selected by `format`.

Parameterized candidates must satisfy all of these:

```text
execution uses a runtime placeholder such as {{format}}
input_constraints documents the placeholder meaning
input_constraints.enum records documented accepted values or a verified subset
description states that the result is selected by runtime input
verifier reads the same runtime input and validates the matching outcome
```

If observations support a parameterized operation but only some documented
values can be meaningfully verified, emit the parameterized candidate with
`input_constraints.enum` limited to that verified documented subset instead of
returning an empty `candidates[]` array.

If the verifier can validate only one fixed discriminator value, do not claim a
parameterized capability. Either emit a fixed-value capability whose execution
hard-codes that value, or generate a verifier harness that supports the
parameterized input.

Do not rewrite a parameterized candidate into a fixed-value candidate merely
because the fixed value has an existing verifier. Candidate shape comes before
verifier selection.

## Binding Contract Rules

The candidate execution describes how this provider implements the capability.

For CLI executions:

```text
execution.spec.args must not include the executable path
CAL supplies the provider path
runtime inputs use placeholders such as {{source}}, {{target}}, {{format}}
```

Every placeholder used in `execution.spec.args` must have a matching probe
`inputs` value or fixture input for the same candidate. For example, if args
contain `{{filter}}`, the probe plan must provide a concrete safe filter value
such as `.` or a documented field selector.

If a verifier checks a target artifact, candidate execution must produce that
target. If the CLI writes the artifact through its own arguments, `{{target}}`
must appear in `execution.spec.args`. If the CLI prints the artifact to stdout,
set `execution.spec.stdout_path_input` to the same input key, usually `target`.
Do not provide `probe_plans[].inputs.target` only for verifier use unless
execution produces it through args or `stdout_path_input`.

`input_constraints` may describe only placeholders that appear in
`execution.spec.args` or `execution.spec.stdout_path_input`.

Before returning, the model must check each candidate/probe pair against this
binding contract:

- every probe input consumed by `verify_py` is either an execution placeholder,
  a fixture-backed execution input, or a produced artifact;
- every produced artifact path checked by `verify_py` is produced by
  `execution.spec.args` containing its placeholder or by
  `execution.spec.stdout_path_input`;
- commands that print the promised result to stdout must use
  `stdout_path_input`; stdout must not be left implicit when a verifier checks a
  target file;
- if this contract cannot be satisfied, the candidate must not be emitted.

When observations document accepted values, formats, modes, or meanings for a
runtime input, copy those into `input_constraints`. Do not invent enum values.

Candidate `description` must be one concise provider-independent sentence. It
must match the execution and must not describe a broader operation than the
binding can perform.

## Verifier Rules

Derive the verifier requirement from the candidate outcome.

Generate one `verifier_packages[]` entry for each probe plan and reference its
proposal-local id from the matching `probe_plans[].verifier.id`. CAL may replace
that proposal-local id with a stable local verifier id before installing the
package.

The proposal-local id must be lowercase `snake_case`, must be semantic, and
must not start with `verifier_`. CAL installs it as:

```text
verifier_<proposal_local_id>_<hash12>
```

Prefer outcome-specific verifiers over artifact-existence verifiers. Use
presence-only verification only when artifact creation itself is the reusable
outcome.

For parameterized candidates, the verifier must inspect the same runtime
discriminator. For example, if the candidate uses `{{format}}`, the verifier
must read `inputs.format` and validate the target artifact according to that
value.

Generate one `verifier_packages[]` entry when meaningful deterministic evidence
can be checked, and reference it from the matching probe plan.

If no meaningful verifier can be selected or generated, do not emit the
candidate.

## Generated Harness Rules

Generated verifier packages are local single-file Python harnesses.

The first version supports:

```text
runtime = python3
entry = verify.py
timeout_ms = 3000
```

The model supplies:

```text
id
description
verify_py
```

The proposal-local package id must be lowercase snake_case, must not start with
`verifier_`, and must not include provider names, CLI flags, paths, hashes, or
fixture literals. CAL installs it as `verifier_<proposal_local_id>_<hash12>`.

Verifier harnesses must validate semantic outcomes, not incidental formatting.
For text encodings, serialized data, checksums, archives, and document
conversions, compare the decoded, parsed, hashed, or inspected meaning that the
capability promises. For checksum outputs, extract and compare the documented
digest value from textual output unless observations explicitly document raw
binary digest output; do not compare the entire target file to raw digest bytes
merely because the target is a file. Do not fail solely on trailing newlines,
line wrapping, labels, filenames, or ASCII whitespace when that formatting is
non-semantic for the documented output format; normalize only documented
non-semantic formatting before doing strict validation. For encode/decode
transformations, verify the produced artifact against the source bytes or
declared runtime input when those inputs are execution placeholders.

`verify_py` must read one JSON object from stdin:

```text
verifier
inputs
```

It must be reusable after Promotion. It may depend only on:

```text
candidate execution placeholders
produced artifacts such as target
fixed literals documented by observations and embedded in verify_py
```

It must not require probe-only inputs or fixture-only values during verification.
If the verifier needs a variable value, that value must also be a candidate
execution placeholder with `input_constraints` so future `run --verify` calls
can supply it.

Inside Python source, use Python literals:

```text
True
False
None
```

Do not write JSON literals such as `true`, `false`, or `null` inside Python
dictionaries passed to `json.dumps`.

Success output:

```json
{
  "passed": true,
  "evidence": [{
    "id": "<verifier id>",
    "type": "<verifier id>",
    "content": {"target": "<path or checked value>"}
  }],
  "outputs": {"target": "<path or checked value>"}
}
```

Failure output:

```json
{
  "passed": false,
  "error": {
    "code": "<snake_case_code>",
    "message": "<human-readable message>"
  }
}
```

`evidence` must always be an array of evidence objects. The harness must not
claim success without checking the output.

## Probe Plan Rules

Each probe plan must reference one candidate by `candidate_index`.

Probe `inputs` plus fixture inputs must cover the runtime placeholders used by
`execution.spec.args` and the verifier.

Path inputs must stay inside the probe work directory. Use `{{workdir}}` for
target artifacts. CAL anchors obvious path inputs such as `target`, `source`,
`input`, `output`, `*_path`, and `*_file` to the probe work directory and
rejects escaping paths.

Non-path scalar inputs, such as jq filters `.` or `.name`, are not treated as
paths unless the input key is a path key or the value is an explicit relative
path such as `./file` or `../file`.

Fixtures should provide only the minimal controlled input required for the
probe. Do not rely on external fixed paths.

## Anti-Patterns

Do not:

```text
choose candidates because they are easy to verify
drop observed candidates because no verifier harness already exists
change a parameterized candidate into a fixed-value candidate to simplify verification
use file existence verification when a stronger outcome verifier can be used
invent enum values not documented or strongly implied by observations
return candidates without probe plans
return verifier results or claim that verification passed
```

## Example: Parameterized Conversion

An observation like:

```text
-convert fmt
fmt is one of: xml1 binary1 json
-o path
```

should usually produce one parameterized capability:

```text
plist.convert_format
```

with:

```text
args = ["-convert", "{{format}}", "-o", "{{target}}", "{{source}}"]
input_constraints.format.enum = ["xml1", "binary1", "json"]
```

The verifier must read `inputs.format` and validate the target according to that
format. If the model cannot select or generate such a verifier, it must not
claim `plist.convert_format`.
