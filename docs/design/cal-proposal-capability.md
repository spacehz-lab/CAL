# CAL Proposal Capability

Proposal Capability is the second internal Proposal stage.

It receives Surface output and chooses the provider-independent capability ids
that later Binding and Evidence stages must use.

## Input

```text
Provider
kept surface_items[] projected to id, kind, name, and description
existing Capability ids and descriptions
capability policy preferred_subjects and preferred_operations
optional debug capability filter
```

`existing Capability ids and descriptions` are a bounded local lookup result.
They are reuse candidates, not proof that the provider implements those
capabilities. Reuse requires both the id and description to be valid and
semantically equivalent to the observed operation.

`capability policy` supplies preferred subject and operation vocabularies.
Capability should choose from these vocabularies whenever a semantically correct
combination exists. It may create a new subject or operation only when
observations clearly cannot be expressed by the supplied terms.

Capability does not need Surface `decision`, `evidence_source`, or `metadata`.
Those fields remain Stage1 process material and should not be sent to the
Capability prompt by default.

The debug filter is not a task hint and must not change capability id rules.

## Output

Capability outputs a global plan:

```text
capabilities[]
  capability_id
  description optional
  source_surface_ids[]
  confidence high | medium | low
```

## Capability Id Rules

Capability ids use exactly two lowercase dotted parts:

```text
<subject>.<operation>
```

They must be provider-independent and semantic. Do not include:

```text
provider names
executable names
command names
flags
paths
versions
formats
encodings
checksum algorithms
modes
target artifact kinds
```

Discriminators belong to Binding execution inputs and Evidence checks.

This is a generic discriminator rule, not a closed list of special cases. If a
future surface varies by a runtime-selectable or binding-specific detail, keep
the `capability_id` generic and let later stages express the detail through
execution inputs and verification checks.

## Decision Process

Capability should internally decide each item in this order:

```text
identify the semantic subject
identify the semantic operation direction
reuse an existing valid equivalent id when possible
use preferred subjects and operations when accurate
create a new provider-independent term only when required
merge only compatible source surfaces
split or keep the clearest surface when command family, inputs, execution,
output, or operation direction differs
```

This decision process is prompt guidance only. The model must still output only
the JSON response shape and must not expose hidden reasoning steps.

## Reuse Rules

Reuse an existing Capability id only when the observed subject and operation are
semantically equivalent.

If no existing id matches, generate a new id following the same two-part rule.

Capability is the only Proposal stage that may choose or generate
`capability_id`. Binding and Evidence must reject or normalize away any attempt
to rename it.

## Local Normalization

CAL validates Capability output before Binding:

```text
capability_id must be valid
subject and operation must be single lowercase terms
source_surface_ids must reference kept Surface items
duplicate capability ids are merged
invalid items are skipped
the stage fails only when no capability remains
```

Surfaces should be merged only when they share the same semantic subject and
operation and can plausibly share compatible Binding inputs, execution shape,
and output semantics. Capability should not merge surfaces with clearly
different inputs, execution shapes, output semantics, or opposite operations.

`confidence` is a planning confidence signal:

```text
high    surface name and description clearly support the semantic capability
medium  operation is supported but subject, direction, or grouping is partly inferred
low     weak but still supported enough to explore
```

Unsupported mappings should be omitted instead of returned with low confidence.

Out-of-policy but valid subject or operation terms may pass when the model
created them for observations that cannot be expressed by preferred terms.
Diagnostics record those cases as `out_of_policy`.

## Trace Diagnostics

Capability writes Stage2 decisions into the discovery trace:

```text
trace.proposal.stages[]
  name = capability
  summary.raw
  summary.keep
  summary.defer
  summary.skip
  summary.selected
  summary.reused
  summary.created
  summary.out_of_policy
  items[]
    id
    kind = capability
    name = capability_id
    decision
```

## Boundary

Capability can conclude:

```text
These provider surfaces should be explored under these capability ids.
```

Capability cannot conclude:

```text
The provider-specific execution is known.
The binding works.
The outcome is verified.
```
