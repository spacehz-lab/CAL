# CAL Capability Run

Capability Run is the execution access module for promoted capabilities.

It lets an agent or human request a capability by id and delegates binding selection and execution to CAL. Outcome verification is explicit.

## Role

Capability Run answers one question:

```text
Can CAL execute this promoted capability now?
```

It is the deterministic execution primitive under Capability Use and after
Capability List.

The intended semantic online path is:

```text
agent task
-> calctl use <intent> --json
-> Use selects capability and binding
-> Run executes the selected binding
-> CAL resolves, executes, and records the run
```

The deterministic lower-level path remains:

```text
agent task
-> calctl capabilities list --json
-> agent selects capability_id
-> calctl runs create --capability-id <capability-id> --inputs-json ... --json
```

When only `capability_id` is supplied, Run chooses the binding. When Use has
already selected a binding, Run should accept an optional `binding_id` execution
constraint and execute that exact promoted binding after validation.

## Input

Default command shape:

```bash
calctl runs create --capability-id <capability-id> --inputs-json <json> --json
```

Verified command shape for experiments or explicit outcome checks:

```bash
calctl runs create --capability-id <capability-id> --inputs-json <json> --verify --json
```

File-backed JSON may be used when shell quoting would be awkward:

```bash
calctl runs create --capability-id <capability-id> --inputs-file <json-file> --json
```

Optional execution constraints:

```bash
calctl runs create --capability-id <capability-id> --binding-id <binding-id> --inputs-json <json> --json
calctl runs create --capability-id <capability-id> --provider-id <provider-id> --inputs-json <json> --json
calctl runs create --capability-id <capability-id> --strategy default --inputs-json <json> --json
```

`--inputs-json` passes the capability input object inline.

`--inputs-file` reads the capability input object from a JSON file. It is not the business input file unless the capability schema defines it that way.

`--verify` runs the selected binding's verifier after execution. Without it, `run` is a lightweight capability invocation and returns `verified:false`.

`--provider` constrains binding selection to one provider.

`--binding-id` constrains execution to one promoted binding that belongs to the
requested capability. This is primarily for Capability Use, debugging, and
reproducible tests. It is not a public binding-management surface.

`--strategy` controls automatic binding resolution. The first stable strategy should be:

```text
default
```

`default` means reliable, runnable, then efficient. It is not a natural-language policy.

## Automatic Binding Resolution

Run must resolve a binding internally.

The caller should not need a separate resolve round trip for normal execution.

Current v1 resolution steps:

```text
load Capability by capability_id
-> filter bindings that are promoted
-> filter binding_id if supplied
-> filter provider_id if supplied
-> reject bindings whose verifier is missing
-> reject bindings whose execution kind is unsupported in this runtime
-> sort remaining bindings by deterministic binding id
-> select the first binding
```

Later scoring may add:

```text
recent successful run
higher historical success rate
lower average duration
lower provider startup cost
```

Run history can improve future selection, but absence of history must not make a promoted binding unrunnable.

If history-based scoring is added, it may compute history from stored Run records instead of denormalizing statistics onto Binding.

After selecting a binding, Run validates the provided input object against the
selected `Binding.execution` placeholders and `Binding.input_constraints`.
Missing runtime inputs or rejected constraint values return `invalid_run_input`
before provider execution starts.

## Execution

Run executes the selected binding's `execution` plan.

Supported execution kinds should remain small and explicit. The current v1 runtime supports:

```text
cli
```

The core model may contain future execution kinds such as `menu`, `ax_action`, and `url_open`, but unsupported runtime kinds must return a structured error instead of falling back to low-level agent behavior inside CAL.

Run must not:

```text
infer a new binding
rewrite Binding.execution
promote candidates
read Trace candidates as executable knowledge
use an `LLM proposer` to decide whether execution succeeded
```

## Verification

By default, Run does not execute the selected binding's verifier. This keeps ordinary capability reuse lightweight.

When `--verify` is supplied, Run executes the selected binding's verifier after provider execution. Verification result is part of the run result:

```text
verified true
verified false
verifier_error
```

If execution completes but verification fails in `--verify` mode, Run returns `verification_failed` unless the binding is explicitly safe to retry.

Automatic fallback after verifier failure is allowed only for bindings marked safe for retry or idempotent. Otherwise CAL should return the failed run and let the agent or user decide the next step.

## Output

Successful output shape:

```json
{
  "run_id": "run_123",
  "capability_id": "document.export_pdf",
  "binding_id": "binding_soffice_pdf",
  "provider_id": "provider_soffice",
  "resolution": {
    "strategy": "default",
    "bindings_considered": 2,
    "reason": "selected promoted binding with verifier and recent successful run"
  },
  "status": "succeeded",
  "verified": false,
  "duration_ms": 842
}
```

Successful verified output additionally includes verifier outputs and evidence:

```json
{
  "run_id": "run_123",
  "capability_id": "document.export_pdf",
  "binding_id": "binding_soffice_pdf",
  "provider_id": "provider_soffice",
  "status": "succeeded",
  "verified": true,
  "outputs": {
    "target": "report.pdf"
  },
  "evidence": [
    {
      "id": "file_parse_pdf",
      "type": "file_parse_pdf"
    }
  ],
  "duration_ms": 842
}
```

Failure output shape:

```json
{
  "run_id": "run_124",
  "capability_id": "document.export_pdf",
  "binding_id": "binding_soffice_pdf",
  "provider_id": "provider_soffice",
  "resolution": {
    "strategy": "default",
    "bindings_considered": 2,
    "reason": "selected promoted binding with verifier"
  },
  "status": "failed",
  "verified": false,
  "error": {
    "code": "verification_failed",
    "message": "output PDF did not parse successfully"
  },
  "evidence": []
}
```

Required output fields:

```text
run_id
capability_id
status
verified
resolution.strategy
```

Required when a binding was selected:

```text
binding_id
provider_id
resolution.reason
```

## Run Record

Run writes a durable Run record.

The record should capture:

```text
run id
capability id
selected binding id
selected provider id
input summary or full capability input object
outputs or outcome when available
evidence references for verified runs
verifier result when requested
timing
error code and message
```

Run records support future evaluation and automatic binding selection.

## Boundary

Capability Run owns:

```text
binding resolution for one capability_id
runtime execution of Binding.execution
deterministic verifier execution when requested
run record writing
evidence reference writing when verification runs
structured success and failure output
```

Capability Run does not own:

```text
natural-language task interpretation
Use selection
capability catalog listing
Discovery Entry
Discovery Inference
Discovery Verification
Discovery Promotion
Trace writing
manual low-level fallback after failure
```

## Agent Policy

A skill or agent policy can use Run like this:

```text
After selecting an exact capability_id from CAL Capability List, call `calctl runs create`.
Do not preselect a binding unless the user or task requires a specific provider.
Trust CAL to choose the binding according to its resolver policy.
If Run returns a verifier failure, do not assume the task succeeded.
If Run returns no runnable binding, continue normal agent behavior or trigger explicit discovery.
```

For user-facing semantic reuse, call Capability Use instead of asking the agent
to know an exact `capability_id`.
