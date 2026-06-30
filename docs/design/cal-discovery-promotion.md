# CAL Discovery Promotion

Discovery Promotion is the final step of Discovery.

It turns candidates that passed Verification into durable `Capability` and
`Binding` records.

Promotion is the boundary between discovery process material and the reusable
core model.

## Input

Promotion input:

```text
Trace.candidates[]
Trace.probes[]
```

Promotion uses:

```text
passed probes
candidates selected by probe.candidate_index
verify.level
evidence
```

Required candidate fields:

```text
provider_id
capability_id
description
execution
```

Required probe fields:

```text
passed
verify
evidence
```

Promotion requires:

```text
probe.passed == true
candidate.provider_id exists
candidate.capability_id is valid
candidate.description exists
candidate.execution exists
probe.verify exists
probe.evidence exists
```

Failed or ambiguous probes remain in `Trace`. They are not promoted.

## Verification Level Policy

Promotion uses the verification level as a trust gate:

```text
L3 semantic
  Default promotable.

L2 structural
  Default promotable, but ranked below L3 during Use.

L1 behavioral
  Not promoted by default. It may be promoted only behind an explicit
  experimental or user-approved policy.

L0 unsupported
  Not promoted.
```

CAL owns final level derivation. Promotion must not trust a level that has not
passed Verification.

## Handling

Promotion handling:

```text
Trace.probes[]
-> select passed L2/L3 probes by default policy
-> load candidate by candidate_index
-> validate candidate and probe
-> create or update Capability
-> create or update Binding
-> attach verify spec and compact evidence
-> persist Capability
```

Promotion must not re-infer the candidate. It uses candidate executions and
passed probe results as they are.

Promotion must not invent or rename the capability id:

```text
Capability.id = candidate.capability_id
```

If at least one candidate passes Verification and is promoted, the provider
acquisition can complete successfully. Failed probes and non-promotable L0/L1
probes remain in Trace and do not block other passing candidates.

## Output

Promotion outputs durable core records:

```text
Capability
Binding
```

Persisted shape:

```text
CAL_HOME/
  capabilities/
    <capability-id>.json
```

Bindings start embedded in the capability record:

```text
Capability
  id
  description
  bindings[]
```

Binding:

```text
Binding
  id optional
  provider_id
  capability_id
  execution
  verify
  evidence
  state
  created_at
```

## Field Mapping

Binding fields come from each candidate and passed probe:

```text
candidate.capability_id      -> Capability.id
candidate.description        -> Capability.description

candidate.provider_id        -> Binding.provider_id
candidate.capability_id      -> Binding.capability_id
candidate.execution          -> Binding.execution

probe.verify                 -> Binding.verify
probe.evidence               -> Binding.evidence
```

Binding state:

```text
promoted
```

`Binding.id` is optional at the concept level. If the implementation needs a
stable selector or file key, derive it from:

```text
binding_<short_hash(capability_id|provider_id|canonical_execution)>
```

The conceptual identity of a binding is:

```text
capability_id
provider_id
canonical_execution
```

## Boundary

Promotion can conclude:

```text
The Capability has a verified Binding.
The Binding can be used by runtime according to its verification level.
```

Promotion cannot:

```text
Invent a new capability_id.
Rewrite candidate.execution.
Ignore verify level or evidence.
Promote failed or ambiguous probes.
Promote L0 probes.
Promote L1 probes without explicit experimental policy.
```
