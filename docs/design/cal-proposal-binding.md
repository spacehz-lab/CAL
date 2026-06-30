# CAL Proposal Binding

Proposal Binding is the third internal Proposal stage.

It receives one planned capability and materializes provider-specific candidate
binding material.

## Input

```text
Provider
one capability plan item
relevant surface_items[]
observations
```

After Capability planning, CAL may run Binding stages in parallel per
capability id with a bounded concurrency limit.

Binding should prefer one candidate per capability. It may return more than one
candidate only when observations clearly show different execution families or
input modes. The local profile caps candidates per capability before Evidence
planning so verification cost does not grow unbounded.

Each per-capability Binding pipeline has a bounded timeout. The default CLI
profile uses a conservative four-minute timeout so a few slow LLM calls cannot
block the whole provider acquisition. A timed-out pipeline is treated as failed,
while other capability pipelines may still contribute candidates.

## Output

Binding outputs candidate and probe material:

```text
candidates[]
  provider_id
  capability_id
  description
  execution

probe_material[]
  candidate_index
  inputs
  fixtures optional
```

The candidate `capability_id` must match the current capability plan item.

## Local Validation

CAL normalizes and validates Binding output before Evidence planning:

```text
candidate.provider_id is empty or matches Provider.id
candidate.capability_id is empty or matches the current capability
candidate.description is present or inherited from the capability plan
execution.kind is supported for the provider
CLI args are present and are strings
CLI args do not include the provider executable path or executable name
probe_material.candidate_index is valid and unique
probe inputs or fixtures cover every execution placeholder
```

Invalid candidates are skipped. A Binding stage fails only when it has no usable
candidate after local validation.

## Execution Rules

For CLI executions:

```text
execution.spec.args must not include the executable path
CAL supplies the provider path
runtime inputs use placeholders such as {{source}}, {{target}}, {{format}}
```

Every placeholder used in `execution.spec.args` must have a matching runtime
input or fixture-backed input in probe material.

If the observed CLI prints the candidate's primary result to stdout, Binding
must make stdout explicit by setting `execution.spec.stdout_path_input` to the
path input that should receive stdout, usually `target`.

If a target artifact is checked later, candidate execution must produce that
target either through an argument placeholder or through `stdout_path_input`.

Runtime discriminators such as `format`, `algorithm`, `encoding`, and `mode`
stay in execution inputs. They do not alter `capability_id`.

Binding must not output separate input schemas.

## Diagnostics

Binding records proposal diagnostics with:

```text
stage.name = binding
summary.raw
summary.keep
summary.skip
summary.defer
summary.selected
items[].reason optional
```

Diagnostics explain which candidate executions were selected for Evidence
planning. Skipped or deferred candidates may include a local `reason` such as a
missing probe input, duplicate probe material, invalid CLI args, or candidate
limit. They are not durable Binding records and are not proof of correctness.

## Boundary

Binding can conclude:

```text
This provider-specific execution may implement the planned capability.
These controlled inputs can probe it.
```

Binding cannot conclude:

```text
The execution passed.
The binding is durable.
The verification level is final.
```
