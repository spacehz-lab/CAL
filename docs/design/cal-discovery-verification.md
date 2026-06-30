# CAL Discovery Verification

Discovery Verification is the third step of Discovery.

It executes candidate bindings from `Trace.candidates[]` with safe probes,
collects evidence, evaluates `verify.checks`, and records the outcome in
`Trace.probes[]`.

Verification does not create durable `Capability` or `Binding` records.
Promotion uses passed probes to create or update those core records.

## Input

Verification input:

```text
Trace.candidates[]
probe plan material
verify spec
temporary probe work directory
```

Fields:

```text
Trace.candidates[]
  Candidate proposals created by Proposal.

probe plan material
  Controlled inputs and fixture files for one candidate.

verify spec
  Deterministic evidence checks proposed by Proposal Evidence and validated by
  CAL.

temporary probe work directory
  Isolated working area used to materialize fixtures and obvious path inputs.
```

Verification selects one or more candidates from `Trace.candidates[]`.

## Handling

Verification handling:

```text
Trace.candidates[]
-> select candidate
-> build probe context
-> execute candidate.execution
-> collect evidence context
-> evaluate verify.checks
-> classify pass / fail / ambiguous
-> append probe result to Trace.probes[]
```

Probe context:

```text
probe_context
  candidate
  work_dir
  inputs
  fixtures optional
  execution_result
  evidence_context
  verify
```

`execution_result` includes command output, stderr, exit code, and produced
artifacts when the execution kind exposes them.

`evidence_context` is provider-specific but normalized for verification. CLI
evidence may include `stdout`, `stderr`, and path inputs such as `source` and
`target`. Future Web and GUI evidence may include `dom`, `url`, `network`,
`ax`, `screenshot`, and `app_state`.

Verification must not use an `LLM proposer` to decide success. Proposal may
produce a `verify` spec, but CAL executes the candidate and evaluates checks
locally.

## Verify Spec

`verify` is the durable verification plan attached to a passing binding:

```text
verify
  level L0 | L1 | L2 | L3
  method execute | contract
  checks[]
    subject
      type file | stdout | stderr | exit_code
      input file subjects only
    predicate
    params optional
```

`level` describes the strength of deterministic evidence:

```text
L3 semantic
  Verifies the semantic result itself.

L2 structural
  Verifies output structure, format, or key properties.

L1 behavioral
  Verifies that an action occurred or state changed.

L0 unsupported
  No reliable deterministic verification is available.
```

CAL owns final level validation. A model-suggested level is process material,
not proof.

`method` describes evidence collection. `execute` runs the probe and evaluates
built-in checks locally. `contract` records weak evidence without executing the
probe when real execution would install, remove, update, edit, start services,
require network, require interaction, or change external state. Contract
verification cannot exceed `L1` and must not include checks.

## Built-In Checks

The default verification path is built-in check evaluation, not generated code.

Initial CLI checks should stay small:

```text
exists
non_empty
format
contains
regex
bytes_equal_transform
hash_line_matches
```

Checks use typed subjects. `file` subjects read a path from a named input such
as `target`; `stdout`, `stderr`, and `exit_code` read process results. CAL uses
the same core VerifySpec rule table for Stage4 prompt injection and
`ValidateVerifySpec`, so invalid subject/predicate/parameter combinations are
rejected before probing or promotion.

Examples:

```text
file(target) bytes_equal_transform source with transform base64_decode
stdout hash_line_matches source with algorithm sha1
file(target) exists + non_empty + format pdf
```

Checks must reference only evidence subjects available in the probe context.
They must not require hidden probe-only values that future `run --verify` calls
cannot supply or reproduce.

Each execute probe has a bounded timeout and records `execution_timeout` when
the candidate command exceeds it. A single probe timeout should fail that
candidate, not cancel unrelated candidates.

## Output

Verification outputs probe results through `Trace`.

```text
Trace
  probes[]
```

Verification does not output:

```text
Capability
Binding
```

## Trace Data

Probe:

```text
probe
  candidate_index
  passed
  verify
  evidence
  reason optional
  error optional
  created_at
```

Fields:

```text
candidate_index
  Local index into Trace.candidates[].
  It is not a durable id.

passed
  Whether the candidate passed deterministic verification.

verify
  Verification level, method, and deterministic checks used for the probe.

evidence
  Evidence collected during execution and verification.
```

Evidence examples:

```text
command_exit
command_output
artifact_exists
artifact_parse
text_match
ui_state
structured_state
api_ack
```

Use `content` for small facts. Use `ref` for files or larger artifacts.

## Boundary

Verification can conclude:

```text
The candidate passed at a verified level.
The candidate failed.
The result is ambiguous.
```

Verification cannot conclude:

```text
The binding is durable.
The capability should be added to the core model.
```

Promotion makes that decision from a passed probe and its verification level.
