# CAL Discovery Proposal

Discovery Proposal is the second step of Discovery.

It observes a discovered `Provider` and produces candidate binding material plus
the probe and evidence plan needed by Verification.

Discovery keeps the external lifecycle stable:

```text
Entry
-> Proposal
-> Verification
-> Promotion
```

Proposal is the only Discovery step that may use an `LLM proposer` for semantic
interpretation. Verification and Promotion remain deterministic CAL-owned work.

Live LLM adapter configuration is defined in `docs/design/cal-discovery-llm.md`.
Stage contracts are defined in:

```text
docs/design/cal-proposal-surface.md
docs/design/cal-proposal-capability.md
docs/design/cal-proposal-binding.md
docs/design/cal-proposal-evidence.md
```

## Goal

Keep Proposal bounded while still covering complex provider surfaces.

The target live path uses four internal Proposal stages:

```text
Surface
-> Capability
-> Binding
-> Evidence
```

These are internal Proposal stages, not public Discovery lifecycle stages.
Deterministic parsers, replay fixtures, and future provider-specific adapters
may skip or replace any internal stage as long as they produce the same Proposal
contract.

## Stages

### Surface

Surface reads provider observations and extracts a bounded list of documented
commands, actions, modes, or UI surfaces worth considering.

```text
Provider observations
-> surface items
```

Surface is a pruning stage. It does not propose `Capability.id`, candidate
executions, probe inputs, or verification checks.

### Capability

Capability receives the bounded surface list and plans provider-independent
capability ids.

```text
surface items
existing Capability ids
preferred subject and operation policy
optional debug capability filter
-> capability plan
```

Capability is the only Proposal stage that owns `capability_id` selection,
reuse, and deduplication. Later stages must not rename capability ids.
It should prefer configured subject and operation terms, and only create a new
term when the observed reusable capability cannot be expressed by the preferred
vocabulary.

### Binding

Binding receives one planned capability at a time and materializes
provider-specific candidate binding material.

```text
capability plan item
relevant surface items
provider observations
-> candidate execution
-> probe inputs and fixtures
```

After Capability planning, CAL may run `Binding -> Evidence` pipelines in
parallel per capability id. The default implementation should cap concurrency
instead of starting one unbounded model call per capability.

### Evidence

Evidence receives a candidate binding and proposes how CAL should verify it.

```text
candidate execution
probe inputs and fixtures
-> verify spec
```

Evidence does not execute the provider and does not decide pass/fail. It only
produces `verify.level` and `verify.checks` for later deterministic
Verification.

## Output Contract

Proposal output is process material written to `Trace`.

At minimum, Proposal produces:

```text
Trace.candidates[]
probe plans
verify specs
```

Each candidate must include:

```text
provider_id
capability_id
description
execution
```

Each probe plan must include:

```text
candidate_index
inputs
fixtures optional
verify
```

`description` is required process material. It must describe the exact reusable
operation exposed by `execution`; it must be provider-independent and must not
claim a broader operation than the binding can perform. Do not include provider
names, executable names, flags, paths, versions, or marketing labels in the
description.

`verify` is a plan for deterministic Verification. It is not proof.

## Capability Id Boundary

Capability ids are provider-independent semantic operation ids:

```text
<subject>.<operation>
```

They must match:

```text
^[a-z0-9]+\.[a-z0-9]+$
```

Capability ids must not include provider names, executable names, command names,
menu labels, flags, paths, versions, marketing names, temporary task names, or
random suffixes.

Capability ids also must not encode result discriminators such as:

```text
format
encoding
checksum algorithm
archive type
mode
target artifact kind
```

Those discriminators belong in the binding execution, input constraints, and
verify checks. For example:

```text
file.checksum
text.encode
text.decode
document.convert
archive.create
```

not:

```text
file.sha1sum
text.encodebase64
document.convertpdf
```

## Diagnostics

Proposal diagnostics are persisted under `Trace.proposal`.

`Trace.proposal.stages[]` records normalized stage decisions and summaries.
`Trace.proposal.attempts[]` records each live LLM stage call so failed live runs
can be debugged after the process exits.

Each attempt records:

```text
stage
capability_id optional
candidate_index optional
status succeeded | failed
duration_ms
error optional
raw_response optional
```

Failed attempts should preserve the raw response when the model returned one.
This is especially important for Evidence failures, where malformed or locally
invalid VerifySpec JSON must be inspectable from the stored trace without
rerunning the live model.

## Concurrency

Surface and Capability are global stages and run serially for one provider
acquisition.

After Capability has produced a deduplicated plan, CAL may run independent
per-capability pipelines:

```text
capability A -> Binding -> Evidence
capability B -> Binding -> Evidence
capability C -> Binding -> Evidence
```

Failure in one per-capability pipeline should not fail the provider acquisition
if another pipeline yields a verifiable candidate. The provider acquisition
fails only when no candidate can pass Verification and Promotion.

Binding locally filters invalid candidate executions before Evidence planning.
It should cap candidates per capability and skip outputs whose probe material
does not cover execution inputs. This keeps Evidence calls bounded and prevents
LLM-produced execution/probe mismatches from reaching Verification.

## Validation Gates

CAL must validate Proposal material before Verification:

```text
capability_id shape and ownership
candidate execution completeness
probe inputs cover execution placeholders
input_constraints only reference execution inputs
target artifacts are produced by execution before checks reference them
verify checks reference only available inputs, outputs, or evidence subjects
verify level is derived or validated locally
contract verification is capped at L1
```

The model may suggest `verify.level` and `verify.method`, but CAL owns the final
accepted verification contract.

## Replay Compatibility

Replay proposal JSON should follow the same final process contract as live
Proposal output. New replay fixtures should use `verify.level`,
`verify.method`, and built-in checks.

## Boundary

Proposal can conclude:

```text
This provider may implement these candidate bindings.
These probe inputs can test each candidate.
These verify checks can evaluate the probe result.
```

Proposal cannot conclude:

```text
The candidate passed.
The binding is durable.
The capability should be added to the core model.
```

Verification and Promotion make those decisions.
