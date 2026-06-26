# CAL Discovery Verification

Discovery Verification is the third step of Discovery.

It tests candidate bindings from `Trace.candidates[]` with safe probes and records the outcome in `Trace.probes[]`.

Verification does not create durable `Capability` or `Binding` records. Promotion uses passed probes to create or update those core records.

## Input

Verification input:

```text
Trace.candidates[]
probe plan material
temporary probe work directory
```

Fields:

```text
Trace.candidates[]
  Candidate proposals created by Inference.

probe plan material
  Controlled inputs, fixture files, and deterministic verifier id selected for
  one candidate.

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
-> collect evidence
-> run verifier
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
  verifier
```

Fields:

```text
candidate
  Selected candidate from Trace.candidates[].

work_dir
  Temporary directory for materialized inputs, outputs, and fixture files.

inputs
  Runtime input object passed to the candidate execution and verifier.

fixtures optional
  Controlled files or values materialized before execution.

verifier
  Deterministic verifier package id and verifier-specific config.
```

Verification executes the candidate's proposed `execution`. The raw execution result is converted into evidence and checked by a deterministic verifier.

Verification must not use an `LLM proposer` to decide success. An `LLM proposer`
may only propose a probe plan and generated verifier harness. CAL still executes
the probe and runs deterministic verification.

Probe planning and outcome verification are separate responsibilities:

```text
ProbePlanner
  builds fixture inputs and selects a deterministic verifier

runtime verifier
  executes the verifier and decides pass / fail
```

Runtime verifier implementations are script packages registered by the runtime
verifier registry. The registry is a code trust boundary: SOP or `LLM proposer`
output may reference a local verifier id or provide a new local verifier
harness package, but it cannot mark verification as passed. CAL installs
generated harnesses under `CAL_HOME/verifiers/<id>/`, executes verifier scripts
locally, and uses only the resulting evidence as proof.

Local installations and generated harnesses add verifier scripts under
`CAL_HOME/verifiers/`. Each verifier directory contains `meta.json` and an
executable script entry. Verifier scripts receive one JSON request on stdin and
must write one JSON result on stdout. CAL owns registration, timeout, JSON
decoding, evidence recording, and pass/fail handling. Generated harnesses are
local materialized code, not a security sandbox.

SOP or `LLM proposer` probe-plan output must be schema-validated and executed by
CAL; it must not set `passed` or bypass fixture safety checks. When it supplies
a generated harness, CAL still performs deterministic local verifier execution.
Probe fixture files and obvious path inputs are materialized inside the temporary
probe work directory; paths that escape that directory are rejected before
provider execution or verifier execution.

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
  verifier
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

verifier
  Deterministic check used for the probe.

evidence
  Evidence collected during execution and verification.

reason optional
  Explanation for failed or ambiguous results.

error optional
  Execution-level error, such as timeout, permission failure, or crash.

created_at
  Probe record time.
```

Verifier:

```text
verifier
  id
```

Verifier ids are lowercase snake_case package ids:

```text
^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$
```

The id should describe the evidence check, not the provider or command that
created the artifact. Do not include provider names, CLI flags, paths, random
hashes, or fixture literals in verifier ids.

Generated verifier proposal ids are proposal-local semantic ids and must not
start with `verifier_`. CAL rewrites each generated harness to a stable
installed id before installing it:

```text
verifier_<proposal_local_id>_<hash12>
```

The first generated package format is a single `python3` `verify.py` file with
a fixed timeout and no overwrite of existing verifier ids.

CAL does not ship embedded default verifier packages. Replay fixtures and live
LLM proposals must carry generated verifier packages, or the referenced verifier
must already exist under `CAL_HOME/verifiers/`.

Evidence:

```text
evidence[]
  type
  content optional
  ref optional
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
The candidate passed.
The candidate failed.
The result is ambiguous.
```

Verification cannot conclude:

```text
The binding is durable.
The capability should be added to the core model.
```

Promotion makes that decision from a passed probe.
