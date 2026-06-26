# CAL Discovery Proposal

This document defines where candidate proposal generation may participate in Discovery, including the bounded places where `LLM proposer` calls are allowed.

Live LLM adapter configuration is defined in `docs/design/cal-discovery-llm.md`.
LLM prompt design is defined in `docs/design/cal-discovery-llm-prompt.md`.

Target CAL execution is service mode: `cald` owns Discovery state, reads and writes `Trace`, runs deterministic verification, and persists promoted `Capability` and `Binding` records. Codex or another agent may call `calctl`, but the target service owns the Discovery loop. The target command entry for one provider acquisition is `calctl discovery run`.

## Goal

Keep `LLM proposer` calls rare and bounded.

Target call count:

```text
Entry        0
Inference    0 or 1
Verification 0 by default, 1 only for probe-plan fallback
Promotion    0
Trace        0
```

Common case:

```text
0 calls when deterministic parsers can propose candidates.
1 call when semantic inference is needed.
2 calls only when semantic inference and probe-plan fallback are both needed.
```

## Entry

Entry must not call an `LLM proposer`.

```text
provider_sources
-> inspect configured path sources
-> create/update Provider
```

Provider records are entry facts. They must not depend on `LLM proposer` output.

## Inference

Inference is the primary candidate-proposal generation point.

Local context preparation should run before the `LLM proposer` call:

```text
provider observation
bounded existing Capability.id lookup
optional debug hint
```

If local context is insufficient to produce replayed proposal material, Inference may make one `LLM proposer` call that combines:

```text
interpret observations
lookup selected existing Capability.id or return none
propose candidate bindings
```

Inputs:

```text
Provider
observations
selected existing Capability ids
hint optional
```

Output:

```text
candidates[]
```

Each candidate must include:

```text
provider_id
capability_id
description
execution
```

`description` is required process material. It must describe the exact reusable
operation exposed by `execution`; it must be provider-independent and must not
claim a broader operation than the binding can perform. Do not include provider
names, executable names, flags, paths, versions, or marketing labels in the
description.

The proposer must not mark a candidate as verified. Its output is process material written to `Trace.candidates[]`.

Capability id selection remains lookup-first:

```text
reuse: <existing_capability_id>
none
```

Only `none` allows proposing a new `capability_id`.

The selected existing Capability ids are produced by local Capability Catalog
Lookup before the `LLM proposer` call. They are bounded topK candidates, not the
full capability catalog. This lookup does not change the target call count:
Inference still uses at most one `LLM proposer` call in the common semantic path.

## Verification

Verification must not use an `LLM proposer` to decide success.

Default path:

```text
candidate.execution
-> execute safe probe
-> collect evidence
-> deterministic verifier
-> pass / fail / ambiguous
```

Allowed fallback:

```text
If CAL cannot construct a probe context or expected outcome deterministically,
an `LLM proposer` may propose a probe plan.
```

Proposal and probe-plan fallback output:

```text
proposal
  metadata optional
  verifier_packages[] optional
    id
    description
    verify_py
  candidates[]
  probe_plans[]
    candidate_index
    inputs
    fixtures optional
    verifier
    rationale optional
```

CAL still executes the probe and runs deterministic verification. The proposer does not decide `passed`.

Replay contract:

```text
proposal
  metadata optional
    source optional
    prompt_version optional
    model optional
    schema_version optional
  verifier_packages[] optional
  candidates[]
  probe_plans[]
```

`candidates[]` contains candidate binding material. `probe_plans[]` contains
fixture inputs, runtime inputs, and one deterministic verifier per proposed
candidate. `verifier_packages[]` may carry local single-file Python harnesses
when the outcome can be checked with proposal-provided deterministic evidence.
CAL may replay this JSON as an SOP fixture, or accept the same JSON from an
`LLM proposer` adapter. The current adapter only parses this bounded proposal
contract; it does not give the model authority to verify or promote.
Replay and adapter output must go through the same execution, verifier, Trace,
and Promotion path as live Discovery.
Probe-plan path inputs must stay inside the temporary probe work directory.
Obvious relative path inputs such as `target`, `source`, `input`, `output`,
`*_path`, and `*_file` are anchored to that directory; escaping relative or
absolute paths are rejected before execution.
When a probe plan supplies a `target` artifact for verification, the candidate
execution must produce that same input either by using `{{target}}` in
`execution.spec.args` or by setting `execution.spec.stdout_path_input` to
`target`.

Candidate `input_constraints` may describe accepted values or meanings for
runtime placeholders used by the candidate execution. Promotion copies these
constraints onto the Binding after Verification passes. Capability records do
not define provider-specific input schemas.

Trace candidates created from replayed proposals record proposal provenance,
including source, prompt version, model, schema version, and a proposal hash.
These fields are evaluation evidence only. They are not proof that the
candidate works.
Live LLM adapters must not trust model-provided provenance fields. They parse the
same bounded proposal shape, but CAL supplies source, prompt version, schema
version, and model from the adapter call context before Trace writing.

## Promotion

Promotion must not call an `LLM proposer`.

```text
passed probe + candidate
-> Capability
-> Binding
```

Promotion must not rename `candidate.capability_id`, rewrite `candidate.execution`, ignore evidence, or promote failed or ambiguous probes.

## Trace

Trace writing must not call an `LLM proposer`.

Trace records the process material created by Entry, Inference, Verification, and Promotion.

## Service Mode

Service mode is the target execution shape:

```text
calctl
-> cald
-> Discovery
-> Trace
-> Provider / Capability / Binding
```

In the target service shape, `cald` owns:

```text
Discovery state
LLM proposer call boundaries
Trace writing
safe probe execution
deterministic verification
promotion
```

Agents can trigger Discovery through `calctl`, but they should not own the verification or promotion rules.

Manual acquisition requests enter CAL through `calctl`, then run inside `cald`
through the local HTTP control API. Proposal parsing, generated verifier
installation, probe execution, Trace writing, and Promotion are service-owned
work.
