# CAL Discovery Trace

`Trace` records what happened during one Discovery attempt.

It is a process artifact for explanation, debugging, and evaluation. It is not a core model and must not replace `Provider`, `Capability`, or `Binding`.

## Input

Trace input comes from Discovery steps:

```text
Entry
  provider entry facts

Proposal
  observations
  candidates
  proposal stage summaries

Verification
  probes
  evidence
  verify check results

Promotion
  promoted capability and binding summaries
  failure or ambiguity reasons
```

## Handling

Trace handling:

```text
create Trace
-> append Entry details
-> append Proposal observations
-> append Proposal candidates
-> append Verification probes
-> append Promotion summaries
-> update status
```

Lifecycle:

```text
Discovery starts
-> create Trace
   -> started_at
   -> status = running
   -> hint optional
   -> provider_ids optional

Entry
-> append provider ids or entry summary

Proposal
-> append observations[]
-> append proposal stage diagnostics
-> append candidates[]

Verification
-> append probes[]

Promotion
-> append promotion summaries
-> status = completed

Failure or cancel
-> status = failed or canceled
-> error optional
```

Trace handling must not infer, verify, or promote. It only records process material.

## Output

Trace output:

```text
trace.json
```

Persisted shape:

```text
CAL_HOME/
  discovery/
    <trace-id>/
      trace.json
```

`<trace-id>` is the discovery trace artifact id returned by discovery control commands. It is not an execution run id and is not part of the core model.

Trace does not output:

```text
Provider
Capability
Binding
```

Those records are written by their owning steps:

```text
Entry      -> Provider
Promotion  -> Capability + Binding
```

## Data Structure

Trace:

```text
Trace
  started_at
  ended_at optional
  status
  hint optional
  provider_ids optional
  observations[]
  proposal optional
  candidates[]
  probes[]
  promotions[]
  error optional
```

Status:

```text
running
completed
failed
canceled
```

Observation:

```text
observation
  provider_id
  type
  source optional
  content optional
  error optional
  created_at
```

Proposal diagnostics:

```text
proposal
  schema_version optional
  prompt_version optional
  model optional
  stages[]

proposal_stage
  name
  items[]
  summary optional
  duration_ms optional

proposal_item
  id optional
  kind optional
  name optional
  decision optional
  rationale optional
```

Proposal diagnostics record parsed stage decisions that are useful for
debugging and evaluation but are not executable candidates. For Surface,
`decision` records the final keep/defer/skip result after local policy, and
`summary.selected` records how many items were passed to Capability planning.

Candidate:

```text
candidate
  provider_id
  capability_id
  description
  source optional
  provenance optional
  input_constraints optional
  execution
  rationale optional
  created_at
```

`input_constraints` describes provider-specific accepted values or meanings for
runtime placeholders used by the candidate execution. It is promoted onto the
Binding when the candidate passes Verification.

Candidate provenance:

```text
provenance
  source optional
  prompt_version optional
  model optional
  schema_version optional
  proposal_hash optional
```

Candidate provenance records where a candidate came from, such as deterministic
rules or a replayed SOP / `LLM proposer` proposal. It is process evidence for
debugging and evaluation, not proof that the candidate is correct.

Probe:

```text
probe
  candidate_index
  passed
  verify
  fallback optional
  evidence
  reason optional
  error optional
  created_at
```

Promotion summary:

```text
promotion
  candidate_index
  capability_id
  binding_id optional
  provider_id
  capability_action optional
  binding_action optional
  created_at
```

Traces write `promotions[]`. CAL does not keep a singular promotion field.

`capability_action` records whether Promotion created a new semantic
Capability or reused an existing one:

```text
created
reused
```

`binding_action` records whether Promotion added a new Binding under that
Capability or refreshed an existing Binding id:

```text
created
updated
```

Trace internals are arrays or embedded objects. Do not promote observations, candidates, probes, evidence, or promotion summaries into top-level models in the first version.

## Boundary

Trace can record:

```text
What happened.
Why a candidate was proposed.
What a probe observed.
Why a candidate passed, failed, or stayed ambiguous.
What was promoted.
```

Trace cannot be used as:

```text
The core capability model.
A runtime binding source.
A replacement for Provider, Capability, or Binding.
```
